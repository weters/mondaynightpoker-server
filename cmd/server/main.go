package main

import (
	"flag"
	"mondaynightpoker-server/internal/config"
	"mondaynightpoker-server/internal/jwt"
	"mondaynightpoker-server/internal/mux"
	"mondaynightpoker-server/pkg/db"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
)

const readTimeout = time.Second * 5
const writeTimeout = time.Second * 10

// Version is the server version
var Version = "v0.0.0-dev"

var addr = flag.String("addr", ":5000", "the listen address")

func main() {
	flag.Parse()
	setupLogger()

	// fail fast
	jwt.LoadKeys()

	if config.Instance().RecaptchaSecret == "" {
		logrus.Fatal("missing recaptcha secret in configuration")
	}

	// run the db migrations
	db.Migrate()

	c := cors.New(cors.Options{
		AllowedHeaders: []string{"Origin", "Accept", "Content-Type", "X-Requested-With", "Authorization"},
		AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodDelete},
	})

	srv := &http.Server{
		Addr:         *addr,
		Handler:      loggingHandler(c.Handler(mux.NewMux(Version))),
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}

	logrus.WithField("addr", srv.Addr).Info("listening")
	logrus.Fatal(srv.ListenAndServe())
}

func loggingHandler(next http.Handler) http.Handler {
	if config.Instance().Log.DisableAccessLogs {
		return next
	}

	return handlers.CombinedLoggingHandler(os.Stdout, next)
}

func setupLogger() {
	if lvl := config.Instance().Log.Level; lvl != "" {
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
