package mux

import (
	"context"
	gmux "github.com/gorilla/mux"
	"mondaynightpoker-server/internal/config"
	"mondaynightpoker-server/internal/email"
	"mondaynightpoker-server/internal/jwt"
	"mondaynightpoker-server/pkg/model"
	"mondaynightpoker-server/pkg/room"
	"net/http"
	"strconv"
	"strings"
)

type ctxKey int

const (
	ctxPlayerKey ctxKey = iota
	ctxTableKey
)

// Mux handles HTTP requests
type Mux struct {
	*gmux.Router
	version   string
	recaptcha recaptcha
	pitBoss   *room.PitBoss

	// XXX: refactor this?
	email          *email.Client
	emailTemplates *email.Template

	// store for testing purposes
	authRouter  *gmux.Router
	adminRouter *gmux.Router
}

// NewMux returns a new HTTP mux
func NewMux(version string) *Mux {
	pitBoss := room.NewPitBoss()
	pitBoss.StartShift()

	e, err := emailClient()
	if err != nil {
		panic(err)
	}

	tpl, err := email.NewTemplate(config.Instance().Email.TemplatesDir)
	if err != nil {
		panic(err)
	}

	this := &Mux{
		Router:         gmux.NewRouter(),
		version:        version,
		pitBoss:        pitBoss,
		email:          e,
		emailTemplates: tpl,
		recaptcha:      newRecaptcha(),
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

	// requires bearer authorization
	{
		r := this.authRouter

		r.Methods(http.MethodPost).Path("/player/{id:[0-9]+}").Handler(this.postPlayerID())
		r.Methods(http.MethodDelete).Path("/player/{id:[0-9]+}").Handler(this.deletePlayerID())

		r.Methods(http.MethodGet).Path("/table").Handler(this.getTable())
		r.Methods(http.MethodPost).Path("/table").Handler(this.postTable())

		tr := r.PathPrefix("/table/{uuid:(?i)[a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12}}").Subrouter()
		tr.Use(this.tableMiddleware)

		tr.Methods(http.MethodGet).Path("").Handler(this.getTableUUID())
		tr.Methods(http.MethodGet).Path("/ws").Handler(this.getTableUUIDWS())
		tr.Methods(http.MethodPost).Path("/seat").Handler(this.postTableUUIDSeat())
	}

	// requires admin access
	// depends on authMiddlemare
	{
		r := this.adminRouter
		r.Methods(http.MethodGet).Path("/player").Handler(this.getPlayer())
		r.Methods(http.MethodGet).Path("/player/{id:[0-9]+}/table").Handler(this.getPlayerIDTable())

		r.Methods(http.MethodPost).Path("/admin/player/{id:[0-9]+}").Handler(this.postAdminPlayerID())
		r.Methods(http.MethodGet).Path("/admin/table").Handler(this.getAdminTable())
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

		id, err := jwt.ValidUserID(token)
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, nil)
			return
		}

		player, err := model.GetPlayerByID(r.Context(), id)
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

func emailClient() (*email.Client, error) {
	cfg := config.Instance().Email
	return email.NewClient(cfg.From, cfg.Sender, cfg.Username, cfg.Password, cfg.Host)
}
