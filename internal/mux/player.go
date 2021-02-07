package mux

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"mondaynightpoker-server/internal/config"
	"mondaynightpoker-server/internal/jwt"
	"mondaynightpoker-server/internal/util"
	"mondaynightpoker-server/pkg/table"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/badoux/checkmail"
	"github.com/gorilla/mux"
)

type playerPayload struct {
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	Token       string `json:"token"`
}

// playerWithEmail should only be return in an admin context, or for the requesting player
type playerWithEmail struct {
	*table.Player
	Email string `json:"email"`
}

var validDisplayNameRx = regexp.MustCompile(`^[\p{L}\p{N} ]{0,40}\z`)
var statusOK = map[string]string{
	"status": "OK",
}

func (m *Mux) postPlayer() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var pp playerPayload
		if !decodeRequest(w, r, &pp) {
			return
		}

		if err := m.recaptcha.Verify(pp.Token); err != nil {
			writeJSONError(w, http.StatusBadRequest, err)
			return
		}

		if !validDisplayNameRx.MatchString(pp.DisplayName) {
			writeJSONError(w, http.StatusBadRequest, errors.New("display name must only contain letters, numbers, and spaces, and be 40 characters or less"))
			return
		}

		if err := checkmail.ValidateFormat(pp.Email); err != nil {
			writeJSONError(w, http.StatusBadRequest, errors.New("missing or invalid email address"))
			return
		}

		if len(pp.Password) < 6 {
			writeJSONError(w, http.StatusBadRequest, errors.New("password must be 6 or more characters"))
			return
		}

		addr := remoteAddr(r)
		at, err := table.LastPlayerCreatedAt(r.Context(), addr)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err)
			return
		}

		playerCreateDelay := time.Second * time.Duration(config.Instance().PlayerCreateDelay)
		if time.Since(at) < playerCreateDelay {
			writeJSONError(w, http.StatusBadRequest, errors.New("please wait before creating another player"))
			return
		}

		var displayName string
		if pp.DisplayName != "" {
			displayName = pp.DisplayName
		} else {
			displayName = util.GetRandomName()
		}

		player, err := table.CreatePlayer(r.Context(), pp.Email, displayName, pp.Password, addr)
		if err != nil {
			if err == table.ErrDuplicateKey {
				writeJSONError(w, http.StatusBadRequest, errors.New("email address is already taken"))
				return
			}

			writeJSONError(w, http.StatusInternalServerError, err)
			return
		}

		writeJSON(w, http.StatusCreated, &playerWithEmail{
			Player: player,
			Email:  player.Email,
		})
	}
}

// note: this requires admin auth
func (m *Mux) getPlayerIDTable() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// ParseInt will always succeed
		playerID, _ := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)

		player, err := table.GetPlayerByID(r.Context(), playerID)
		if err != nil {
			if err == sql.ErrNoRows {
				writeJSONError(w, http.StatusNotFound, nil)
			} else {
				writeJSONError(w, http.StatusInternalServerError, err)
			}
			return
		}

		start, rows, err := parsePaginationOptions(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err)
			return
		}

		tables, err := player.GetTables(r.Context(), start, rows)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err)
			return
		}

		writeJSON(w, http.StatusOK, tables)
	}
}

type postPlayerIDPayload struct {
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
}

func (m *Mux) postPlayerID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		playerID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err)
			return
		}

		// prevent a player from updating another player
		player := r.Context().Value(ctxPlayerKey).(*table.Player)
		if player.ID != playerID {
			writeJSONError(w, http.StatusForbidden, err)
			return
		}

		var pp postPlayerIDPayload
		if !decodeRequest(w, r, &pp) {
			return
		}

		update := false

		if displayName := pp.DisplayName; displayName != "" {
			if !validDisplayNameRx.MatchString(displayName) {
				writeJSONError(w, http.StatusBadRequest, errors.New("display name must only contain letters, numbers, and spaces"))
				return
			}

			player.DisplayName = displayName
			update = true
		}

		if email := pp.Email; email != "" {
			if err := checkmail.ValidateFormat(email); err != nil {
				writeJSONError(w, http.StatusBadRequest, errors.New("invalid email address"))
				return
			}

			player.Email = email
			update = true
		}

		if update {
			if err := player.Save(r.Context()); err != nil {
				writeJSONError(w, http.StatusInternalServerError, err)
				return
			}
		}

		writeJSON(w, http.StatusOK, statusOK)
	}
}

type postPlayerAuthResponse struct {
	JWT    string          `json:"jwt"`
	Player playerWithEmail `json:"player"`
}

func (m *Mux) postPlayerAuth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var pp playerPayload
		if !decodeRequest(w, r, &pp) {
			return
		}

		player, err := table.GetPlayerByEmailAndPassword(r.Context(), pp.Email, pp.Password)
		if err != nil {
			if err == table.ErrInvalidEmailOrPassword {
				writeJSONError(w, http.StatusUnauthorized, err)
				return
			}

			writeJSONError(w, http.StatusInternalServerError, err)
			return
		}

		signedToken, err := jwt.Sign(player.ID)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err)
			return
		}

		writeJSON(w, http.StatusOK, postPlayerAuthResponse{
			JWT: signedToken,
			Player: playerWithEmail{
				Player: player,
				Email:  player.Email,
			},
		})
	}
}

func (m *Mux) getPlayerAuthJWT() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		signedToken := mux.Vars(r)["jwt"]
		userID, err := jwt.ValidUserID(signedToken)
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, err)
			return
		}

		player, err := table.GetPlayerByID(r.Context(), userID)
		if err != nil {
			if err == sql.ErrNoRows {
				writeJSONError(w, http.StatusNotFound, errors.New("player does not exist"))
			} else {
				writeJSONError(w, http.StatusInternalServerError, err)
			}

			return
		}

		writeJSON(w, http.StatusOK, playerWithEmail{
			Player: player,
			Email:  player.Email,
		})
	}
}

func (m *Mux) getPlayer() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		offset, limit, err := parsePaginationOptions(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err)
			return
		}

		players, err := table.GetPlayersWithSearch(r.Context(), r.FormValue("search"), offset, limit)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err)
			return
		}

		adminPlayers := make([]*playerWithEmail, len(players))
		for i, p := range players {
			adminPlayers[i] = &playerWithEmail{
				Player: p,
				Email:  p.Email,
			}
		}

		writeJSON(w, http.StatusOK, adminPlayers)
	}
}

type adminPostPlayerIDRequest struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

func (m *Mux) postAdminPlayerID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		playerID, _ := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
		player, err := table.GetPlayerByID(r.Context(), playerID)
		if err != nil {
			if err == sql.ErrNoRows {
				writeJSONError(w, http.StatusNotFound, nil)
				return
			}

			writeJSONError(w, http.StatusInternalServerError, err)
			return
		}

		if r.Header.Get("content-type") != "application/json" {
			writeJSONError(w, http.StatusUnsupportedMediaType, nil)
			return
		}

		var payload adminPostPlayerIDRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeJSONError(w, http.StatusBadRequest, err)
			return
		}

		switch payload.Key {
		case "password":
			value, ok := payload.Value.(string)
			if !ok {
				writeJSONError(w, http.StatusBadRequest, errors.New("password must be a string"))
				return
			}

			if err := player.SetPassword(value); err != nil {
				writeJSONError(w, http.StatusInternalServerError, err)
				return
			}
		default:
			writeJSONError(w, http.StatusBadRequest, errors.New("bad payload"))
			return
		}

		writeJSON(w, http.StatusOK, statusOK)
	}
}

type postPlayerResetPasswordRequestPayload struct {
	Email string `json:"email"`
}

func (m *Mux) postPlayerResetPasswordRequest() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload postPlayerResetPasswordRequestPayload
		if ok := decodeRequest(w, r, &payload); !ok {
			return
		}

		if payload.Email == "" {
			writeJSONError(w, http.StatusBadRequest, errors.New("missing email"))
			return
		}

		if player, _ := table.GetPlayerByEmail(r.Context(), payload.Email); player != nil {
			token, err := player.CreatePasswordResetRequest(r.Context())
			if err != nil {
				writeJSONError(w, http.StatusInternalServerError, err)
				return
			}

			go func() {
				if config.Instance().Email.Disable {
					return
				}

				data := map[string]string{
					"url":   fmt.Sprintf("%s/reset-password/%s", config.Instance().Host, token),
					"email": player.Email,
				}

				msg, err := m.emailTemplates.RenderTemplate("password_reset.html", data)
				if err != nil {
					logrus.WithError(err).Error("could not render the template")
					return
				}

				log := logrus.WithField("to", player.Email)
				if err := m.email.SendSimple(player.Email, "Password Reset Request", msg); err != nil {
					log.WithError(err).Error("could not send email")
				} else {
					log.Info("sent password reset email")
				}
			}()
		}

		writeJSON(w, http.StatusOK, statusOK)
	}
}

func (m *Mux) getPlayerResetPasswordToken() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := mux.Vars(r)["token"]
		if err := table.IsPasswordResetTokenValid(r.Context(), token); err != nil {
			writeJSONError(w, http.StatusNotFound, nil)
			return
		}

		writeJSON(w, http.StatusOK, statusOK)
	}
}

type postPlayerResetPasswordPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (m *Mux) postPlayerResetPasswordToken() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := mux.Vars(r)["token"]

		var payload postPlayerResetPasswordPayload
		if ok := decodeRequest(w, r, &payload); !ok {
			return
		}

		if err := table.IsPasswordResetTokenValid(r.Context(), token); err != nil {
			writeJSONError(w, http.StatusNotFound, nil)
			return
		}

		if payload.Email == "" {
			writeJSONError(w, http.StatusBadRequest, errors.New("email is required"))
			return
		}

		if len(payload.Password) < 6 {
			writeJSONError(w, http.StatusBadRequest, errors.New("password must be at least 6 characters"))
			return
		}

		player, err := table.GetPlayerByEmail(r.Context(), payload.Email)
		if err != nil {
			if err != sql.ErrNoRows {
				writeJSONError(w, http.StatusInternalServerError, err)
			} else {
				writeJSONError(w, http.StatusBadRequest, nil)
			}
			return
		}

		if err := player.ResetPassword(r.Context(), payload.Password, token); err != nil {
			writeJSONError(w, http.StatusBadRequest, nil)
			return
		}

		writeJSON(w, http.StatusOK, statusOK)
	}
}
