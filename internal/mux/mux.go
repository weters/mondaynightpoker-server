package mux

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"mondaynightpoker-server/internal/config"
	"mondaynightpoker-server/internal/email"
	"mondaynightpoker-server/internal/oauth"
	"mondaynightpoker-server/pkg/model"
	"mondaynightpoker-server/pkg/room"

	gmux "github.com/gorilla/mux"
)

type ctxKey int

const (
	ctxPlayerKey ctxKey = iota
	ctxTableKey
)

// TokenService signs and validates player JWTs
type TokenService interface {
	Sign(userID int64) (string, error)
	ValidUserID(signedString string) (int64, error)
}

// Deps are the dependencies a Mux requires
type Deps struct {
	Version        string
	Config         config.Config
	Repos          *model.Repositories
	PitBoss        *room.PitBoss
	Tokens         TokenService
	Email          *email.Client
	EmailTemplates *email.Template
	// Recaptcha is optional; when nil, a verifier is built from Config
	Recaptcha recaptcha
	// OAuth is optional; when set alongside MCPHandler, the OAuth 2.1
	// authorization server and MCP resource-server middleware are wired up
	OAuth *oauth.Server
	// MCPHandler is optional; when set alongside OAuth, it is mounted at /mcp
	// behind OAuth's bearer-token middleware
	MCPHandler http.Handler
}

// Mux handles HTTP requests
type Mux struct {
	*gmux.Router
	version        string
	cfg            config.Config
	repos          *model.Repositories
	tokens         TokenService
	recaptcha      recaptcha
	pitBoss        *room.PitBoss
	email          *email.Client
	emailTemplates *email.Template

	// store for testing purposes
	authRouter  *gmux.Router
	adminRouter *gmux.Router
}

// NewMux returns a new HTTP mux
func NewMux(deps Deps) *Mux {
	captcha := deps.Recaptcha
	if captcha == nil {
		captcha = newRecaptcha(deps.Config.RecaptchaSecret)
	}

	this := &Mux{
		Router:         gmux.NewRouter(),
		version:        deps.Version,
		cfg:            deps.Config,
		repos:          deps.Repos,
		tokens:         deps.Tokens,
		pitBoss:        deps.PitBoss,
		email:          deps.Email,
		emailTemplates: deps.EmailTemplates,
		recaptcha:      captcha,
	}

	this.authRouter = this.Router.NewRoute().Subrouter()
	this.authRouter.Use(this.authMiddleware)

	this.adminRouter = this.authRouter.NewRoute().Subrouter()
	this.adminRouter.Use(this.adminMiddleware)

	// unauthorized endpoints
	{
		r := this.Router
		r.Methods(http.MethodGet).Path("/health").Handler(this.getHealth())
		r.Methods(http.MethodPost).Path("/player").Handler(this.postPlayer())
		r.Methods(http.MethodPost).Path("/player/auth").Handler(this.postPlayerAuth())
		r.Methods(http.MethodGet).Path("/player/auth/{jwt:.*}").Handler(this.getPlayerAuthJWT())
		r.Methods(http.MethodPost).Path("/player/verify/{token:[a-zA-Z0-9_-]{20}}").Handler(this.postPlayerVerifyAccountToken())
		r.Methods(http.MethodPost).Path("/player/reset-password-request").Handler(this.postPlayerResetPasswordRequest())
		r.Methods(http.MethodPost).Path("/player/reset-password/{token:[a-zA-Z0-9_-]{20}}").Handler(this.postPlayerResetPasswordToken())
		r.Methods(http.MethodGet).Path("/player/reset-password/{token:[a-zA-Z0-9_-]{20}}").Handler(this.getPlayerResetPasswordToken())
	}

	// OAuth 2.1 authorization server + MCP endpoint. These are unauthenticated
	// at the router level: /oauth/* uses PKCE + its own login flow, and /mcp is
	// guarded by OAuth's own bearer-token middleware rather than authMiddleware.
	// Both deps ship together; if only one is set, nothing is registered.
	if deps.OAuth != nil && deps.MCPHandler != nil {
		r := this.Router
		r.Methods(http.MethodGet).Path("/.well-known/oauth-protected-resource").Handler(deps.OAuth.ProtectedResourceMetadata())
		r.Methods(http.MethodGet).Path("/.well-known/oauth-authorization-server").Handler(deps.OAuth.AuthorizationServerMetadata())
		r.Methods(http.MethodGet).Path("/oauth/authorize").Handler(deps.OAuth.Authorize())
		r.Methods(http.MethodPost).Path("/oauth/authorize").Handler(deps.OAuth.AuthorizePost())
		r.Methods(http.MethodPost).Path("/oauth/token").Handler(deps.OAuth.Token())
		r.Methods(http.MethodPost).Path("/oauth/register").Handler(deps.OAuth.Register())
		r.PathPrefix("/mcp").Handler(deps.OAuth.RequireMCPAuth(http.StripPrefix("/mcp", deps.MCPHandler)))
	}

	// requires bearer authorization
	{
		r := this.authRouter

		r.Methods(http.MethodPost).Path("/player/auth/refresh").Handler(this.postPlayerAuthRefresh())
		r.Methods(http.MethodPost).Path("/player/{id:[0-9]+}").Handler(this.postPlayerID())
		r.Methods(http.MethodDelete).Path("/player/{id:[0-9]+}").Handler(this.deletePlayerID())
		r.Methods(http.MethodGet).Path("/player/profile").Handler(this.getPlayerProfile())
		r.Methods(http.MethodGet).Path("/table").Handler(this.getTable())
		r.Methods(http.MethodPost).Path("/table").Handler(this.postTable())

		tr := r.PathPrefix("/table/{uuid:(?i)[a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12}}").Subrouter()
		tr.Use(this.tableMiddleware)

		tr.Methods(http.MethodGet).Path("").Handler(this.getTableUUID())
		tr.Methods(http.MethodGet).Path("/ws").Handler(this.getTableUUIDWS())
		tr.Methods(http.MethodPost).Path("/seat").Handler(this.postTableUUIDSeat())
		tr.Methods(http.MethodPost).Path("/clone").Handler(this.postTableUUIDClone())
		tr.Methods(http.MethodPost).Path("/name").Handler(this.postTableUUIDName())
	}

	// requires admin access
	// depends on authMiddlemare
	{
		r := this.adminRouter
		r.Methods(http.MethodGet).Path("/player").Handler(this.getPlayer())
		r.Methods(http.MethodGet).Path("/player/{id:[0-9]+}/profile").Handler(this.getPlayerIDProfile())

		r.Methods(http.MethodPost).Path("/admin/test-player").Handler(this.postAdminTestPlayer())
		r.Methods(http.MethodPost).Path("/admin/player/{id:[0-9]+}").Handler(this.postAdminPlayerID())
		r.Methods(http.MethodGet).Path("/admin/table").Handler(this.getAdminTable())
		r.Methods(http.MethodPost).Path("/admin/table/{uuid:(?i)[a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12}}").Handler(this.postAdminTableUUID())
	}

	return this
}

func (m *Mux) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.FormValue("access_token")
		if token == "" {
			authHeader := strings.Split(r.Header.Get("Authorization"), " ")
			if len(authHeader) != 2 || strings.ToLower(authHeader[0]) != "bearer" {
				writeJSONError(w, http.StatusUnauthorized, nil)
				return
			}

			token = authHeader[1]
		}

		id, err := m.tokens.ValidUserID(token)
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, nil)
			return
		}

		player, err := m.repos.Players.GetPlayerByID(r.Context(), id)
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, nil)
			return
		}

		newCtx := context.WithValue(r.Context(), ctxPlayerKey, player)
		w.Header().Set("MondayNightPoker-UserID", strconv.FormatInt(player.ID, 10))
		next.ServeHTTP(w, r.WithContext(newCtx))
	})
}

// adminMiddleware requires authMiddleware to execute first
func (m *Mux) adminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		player := r.Context().Value(ctxPlayerKey).(*model.Player)
		if !player.IsSiteAdmin {
			writeJSONError(w, http.StatusForbidden, nil)
			return
		}

		next.ServeHTTP(w, r)
	})
}
