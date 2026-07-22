package main

import (
	"flag"
	"net/http"
	"os"
	"strings"
	"time"

	"mondaynightpoker-server/internal/config"
	"mondaynightpoker-server/internal/email"
	"mondaynightpoker-server/internal/jwt"
	"mondaynightpoker-server/internal/mcpserver"
	"mondaynightpoker-server/internal/mux"
	"mondaynightpoker-server/internal/oauth"
	"mondaynightpoker-server/pkg/db"
	"mondaynightpoker-server/pkg/model"
	"mondaynightpoker-server/pkg/room"

	"github.com/gorilla/handlers"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
)

const readTimeout = time.Second * 5
const writeTimeout = time.Second * 10

// tokenTTL is how long issued player JWTs remain valid. Active clients
// refresh past the half-life via POST /player/auth/refresh.
const tokenTTL = 30 * 24 * time.Hour

// Version is the server version
var Version = "v0.0.0-dev"

var addr = flag.String("addr", ":5080", "the listen address")

func main() {
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		logrus.WithError(err).Fatal("could not load configuration")
	}

	setupLogger(cfg)

	if cfg.RecaptchaSecret == "" {
		logrus.Fatal("missing recaptcha secret in configuration")
	}

	signer, err := jwt.NewSigner(cfg.JWT, tokenTTL)
	if err != nil {
		logrus.WithError(err).Fatal("could not load JWT keys")
	}

	database, err := db.Connect(cfg.Database.DSN)
	if err != nil {
		logrus.WithError(err).Fatal("could not connect to the database")
	}

	if err := db.Migrate(database, cfg.Database.MigrationsPath); err != nil {
		logrus.WithError(err).Fatal("could not run migrations")
	}

	repos := model.NewRepositories(database)

	pitBoss := room.NewPitBoss(repos, room.PitBossOptions{
		StartGameDelay: time.Duration(cfg.StartGameDelay) * time.Second,
	})
	pitBoss.StartShift()

	emailClient, err := email.NewClient(cfg.Email.From, cfg.Email.Sender, cfg.Email.Username, cfg.Email.Password, cfg.Email.Host)
	if err != nil {
		logrus.WithError(err).Fatal("could not create email client")
	}

	emailTemplates, err := email.NewTemplate(cfg.Email.TemplatesDir)
	if err != nil {
		logrus.WithError(err).Fatal("could not load email templates")
	}

	privKey, pubKey, err := jwt.LoadKeyPair(cfg.JWT)
	if err != nil {
		logrus.WithError(err).Fatal("could not load JWT keypair for OAuth server")
	}

	apiBaseURL := cfg.APIBaseURL()
	oauthServer := oauth.New(repos, privKey, pubKey, oauth.Config{
		Issuer:   apiBaseURL,
		Resource: apiBaseURL + "/mcp",
	})
	mcpHandler := mcpserver.New(repos, Version)

	m := mux.NewMux(mux.Deps{
		Version:        Version,
		Config:         cfg,
		Repos:          repos,
		PitBoss:        pitBoss,
		Tokens:         signer,
		Email:          emailClient,
		EmailTemplates: emailTemplates,
		OAuth:          oauthServer,
		MCPHandler:     mcpHandler,
	})

	// Share the same origin allowlist as the WebSocket upgrade check so the two
	// layers cannot diverge. Previously CORS was left unset (allow-all), which
	// let REST requests through from origins the WebSocket check would reject.
	restricted := cors.New(cors.Options{
		AllowedOrigins: cfg.BrowserOrigins(),
		AllowedHeaders: []string{"Origin", "Accept", "Content-Type", "X-Requested-With", "Authorization"},
		AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodDelete},
	})

	// OAuth/MCP endpoints (other than /oauth/authorize, a top-level browser
	// navigation) are called cross-origin by arbitrary MCP clients (e.g.
	// claude.ai) that are not in cfg.BrowserOrigins(). They are not
	// cookie-authenticated -- auth is PKCE-derived bearer tokens -- so an
	// open CORS policy on just these paths is safe and lets any client
	// discover metadata, register, and exchange/use tokens.
	open := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedHeaders: []string{"*"},
		AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodOptions},
	})

	srv := &http.Server{
		Addr:         *addr,
		Handler:      loggingHandler(cfg, corsRouter(restricted, open, m)),
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}

	logrus.WithField("addr", srv.Addr).Info("listening")
	logrus.Fatal(srv.ListenAndServe())
}

// corsRouter dispatches a request to open or restricted depending on whether
// the request path is one of the unauthenticated OAuth/MCP endpoints (see the
// comment above where `open` is built) or one of the cookie-authenticated
// REST/WebSocket endpoints.
func corsRouter(restricted, open *cors.Cors, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isOpenCORSPath(r.URL.Path) {
			open.Handler(next).ServeHTTP(w, r)
			return
		}

		restricted.Handler(next).ServeHTTP(w, r)
	})
}

// isOpenCORSPath reports whether path belongs to an OAuth/MCP endpoint that
// must remain reachable cross-origin from arbitrary (non-browser-allowlisted)
// clients. /oauth/authorize is intentionally excluded: it is a top-level
// browser navigation and needs no CORS grant.
func isOpenCORSPath(path string) bool {
	switch {
	case path == "/mcp", strings.HasPrefix(path, "/mcp/"):
		return true
	case path == "/oauth/token", path == "/oauth/register":
		return true
	case strings.HasPrefix(path, "/.well-known/oauth-"):
		return true
	default:
		return false
	}
}

func loggingHandler(cfg config.Config, next http.Handler) http.Handler {
	if cfg.Log.DisableAccessLogs {
		return next
	}

	return handlers.CombinedLoggingHandler(os.Stdout, next)
}

func setupLogger(cfg config.Config) {
	if lvl := cfg.Log.Level; lvl != "" {
		level, err := logrus.ParseLevel(lvl)
		if err != nil {
			logrus.WithError(err).Fatal("could not parse level")
		}

		logrus.SetLevel(level)
	}

	if strings.ToLower(os.Getenv("LOG_FORMAT")) == "json" {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	}
}
