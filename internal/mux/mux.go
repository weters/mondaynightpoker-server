package mux

import (
	"context"
	"mondaynightpoker-server/internal/jwt"
	"mondaynightpoker-server/pkg/room"
	"mondaynightpoker-server/pkg/table"
	"net/http"
	"strconv"
	"strings"
	"time"

	gmux "github.com/gorilla/mux"
)

type ctxKey int

const (
	ctxPlayerKey ctxKey = iota
	ctxTableKey
)

// Mux handles HTTP requests
type Mux struct {
	*gmux.Router
	config    config
	version   string
	recaptcha recaptcha
	pitBoss   *room.PitBoss

	// store for testing purposes
	authRouter  *gmux.Router
	adminRouter *gmux.Router
}

type config struct {
	// playerCreateDelay is the minimum duration between two player create events from a single remote address
	playerCreateDelay time.Duration
}

// NewMux returns a new HTTP mux
func NewMux(version string) *Mux {
	pitBoss := room.NewPitBoss()
	pitBoss.StartShift()

	this := &Mux{
		Router:  gmux.NewRouter(),
		version: version,
		pitBoss: pitBoss,
		config: config{
			playerCreateDelay: time.Minute,
		},
		recaptcha: newRecaptcha(),
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
	}

	// requires bearer authorization
	{
		r := this.authRouter

		r.Methods(http.MethodPost).Path("/player/{id:[0-9]+}").Handler(this.postPlayerID())

		r.Methods(http.MethodGet).Path("/table").Handler(this.getTable())

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
		r.Methods(http.MethodPost).Path("/table").Handler(this.postTable())
		r.Methods(http.MethodGet).Path("/player").Handler(this.getPlayer())
		r.Methods(http.MethodGet).Path("/player/{id:[0-9]+}/table").Handler(this.getPlayerIDTable())
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

		player, err := table.GetPlayerByID(r.Context(), id)
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
		player := r.Context().Value(ctxPlayerKey).(*table.Player)
		if !player.IsSiteAdmin {
			writeJSONError(w, http.StatusForbidden, nil)
			return
		}

		next.ServeHTTP(w, r)
	})
}
