package mux

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"mondaynightpoker-server/internal/util"
	"mondaynightpoker-server/pkg/model"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/badoux/checkmail"
	"github.com/gorilla/mux"
)

type postPlayerPayload struct {
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	Token       string `json:"token"`
}

// playerWithEmail should only be return in an admin context, or for the requesting player
type playerWithEmail struct {
	*model.Player
	Email string `json:"email"`
}

var validDisplayNameRx = regexp.MustCompile(`^[\p{L}\p{N} ]{0,40}\z`)
var statusOK = map[string]string{
	"status": "OK",
}

func (m *Mux) postPlayer() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var pp postPlayerPayload
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

		if err := validatePassword(pp.Password); err != nil {
			writeJSONError(w, http.StatusBadRequest, err)
			return
		}

		addr := remoteAddr(r)
		at, err := m.repos.Players.LastPlayerCreatedAt(r.Context(), addr)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err)
			return
		}

		playerCreateDelay := time.Second * time.Duration(m.cfg.PlayerCreateDelay)
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

		player, err := m.repos.Players.CreatePlayer(r.Context(), pp.Email, displayName, pp.Password, addr)
		if err != nil {
			if err == model.ErrDuplicateKey {
				writeJSONError(w, http.StatusBadRequest, errors.New("email address is already taken"))
				return
			}

			writeJSONError(w, http.StatusInternalServerError, err)
			return
		}

		verifyToken, err := m.repos.Players.CreateAccountVerificationToken(context.Background(), player)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, err)
			return
		}

		go m.sendAccountVerificationEmail(player, verifyToken)

		writeJSON(w, http.StatusCreated, &playerWithEmail{
			Player: player,
			Email:  player.Email,
		})
	}
}

func (m *Mux) sendAccountVerificationEmail(player *model.Player, verifyToken string) {
	if m.cfg.Email.Disable {
		return
	}

	log := logrus.WithField("playerId", player.ID)

	body, err := m.emailTemplates.RenderTemplate("verify_account.html", map[string]string{
		"url":   fmt.Sprintf("%s/verify-account/%s", m.cfg.Host, verifyToken),
		"email": player.Email,
	})

	if err != nil {
		log.WithError(err).Error("could not render template")
		return
	}

	if err := m.email.SendSimple(player.Email, "Verify Your Account", body); err != nil {
		log.WithError(err).Error("could not send account verification email")
	}
}

type postPlayerIDPayload struct {
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

func (m *Mux) postPlayerID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		playerID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err)
			return
		}

		// prevent a player from updating another player
		player := r.Context().Value(ctxPlayerKey).(*model.Player)
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

		if newPassword := pp.NewPassword; newPassword != "" {
			if err := validatePassword(newPassword); err != nil {
				writeJSONError(w, http.StatusBadRequest, err)
				return
			}

			if err := player.ValidatePassword(pp.OldPassword); err != nil {
				writeJSONError(w, http.StatusBadRequest, errors.New("old password does not match"))
				return
			}

			if err := player.SetPassword(newPassword); err != nil {
				writeJSONError(w, http.StatusInternalServerError, err)
				return
			}

			update = true
		}

		if update {
			if err := m.repos.Players.Save(r.Context(), player); err != nil {
				writeJSONError(w, http.StatusInternalServerError, err)
				return
			}
		}

		writeJSON(w, http.StatusOK, statusOK)
	}
}

func (m *Mux) deletePlayerID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		playerID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err)
			return
		}

		// prevent a player from updating another player
		player := r.Context().Value(ctxPlayerKey).(*model.Player)
		if player.ID != playerID {
			writeJSONError(w, http.StatusForbidden, err)
			return
		}

		if err := m.repos.Players.Delete(r.Context(), player); err != nil {
			writeJSON(w, http.StatusInternalServerError, err)
			return
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
		var pp postPlayerPayload
		if !decodeRequest(w, r, &pp) {
			return
		}

		player, err := m.repos.Players.GetPlayerByEmailAndPassword(r.Context(), pp.Email, pp.Password)
		if err != nil {
			var ue model.UserError
			if errors.As(err, &ue) {
				writeJSONError(w, http.StatusUnauthorized, err)
				return
			}

			writeJSONError(w, http.StatusInternalServerError, err)
			return
		}

		signedToken, err := m.tokens.Sign(player.ID)
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

// postPlayerAuthRefresh exchanges a valid token for a freshly issued one so
// active players roll onto new expiries without logging in again
func (m *Mux) postPlayerAuthRefresh() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		player := r.Context().Value(ctxPlayerKey).(*model.Player)

		signedToken, err := m.tokens.Sign(player.ID)
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
		userID, err := m.tokens.ValidUserID(signedToken)
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, err)
			return
		}

		player, err := m.repos.Players.GetPlayerByID(r.Context(), userID)
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

		players, err := m.repos.Players.GetPlayersWithSearch(r.Context(), r.FormValue("search"), offset, limit)
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
		player, err := m.repos.Players.GetPlayerByID(r.Context(), playerID)
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

			if err := m.repos.Players.Save(r.Context(), player); err != nil {
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

		if player, _ := m.repos.Players.GetPlayerByEmail(r.Context(), payload.Email); player != nil {
			token, err := m.repos.Players.CreatePasswordResetRequest(r.Context(), player)
			if err != nil {
				writeJSONError(w, http.StatusInternalServerError, err)
				return
			}

			go func() {
				if m.cfg.Email.Disable {
					return
				}

				data := map[string]string{
					"url":   fmt.Sprintf("%s/reset-password/%s", m.cfg.Host, token),
					"email": player.Email,
					"host":  m.cfg.Host,
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
		if err := m.repos.Players.IsPasswordResetTokenValid(r.Context(), token); err != nil {
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

		if err := m.repos.Players.IsPasswordResetTokenValid(r.Context(), token); err != nil {
			writeJSONError(w, http.StatusNotFound, nil)
			return
		}

		if payload.Email == "" {
			writeJSONError(w, http.StatusBadRequest, errors.New("email is required"))
			return
		}

		if err := validatePassword(payload.Password); err != nil {
			writeJSONError(w, http.StatusBadRequest, err)
			return
		}

		player, err := m.repos.Players.GetPlayerByEmail(r.Context(), payload.Email)
		if err != nil {
			if err != sql.ErrNoRows {
				writeJSONError(w, http.StatusInternalServerError, err)
			} else {
				writeJSONError(w, http.StatusBadRequest, nil)
			}
			return
		}

		if err := m.repos.Players.ResetPassword(r.Context(), player, payload.Password, token); err != nil {
			writeJSONError(w, http.StatusBadRequest, nil)
			return
		}

		writeJSON(w, http.StatusOK, statusOK)
	}
}

func (m *Mux) postPlayerVerifyAccountToken() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := mux.Vars(r)["token"]
		if err := m.repos.Players.VerifyAccount(r.Context(), token); err != nil {
			if errors.Is(err, model.ErrTokenExpired) {
				writeJSONError(w, http.StatusBadRequest, err)
			} else {
				writeJSONError(w, http.StatusInternalServerError, err)
			}

			return
		}

		writeJSON(w, http.StatusOK, statusOK)
	}
}

type postAdminTestPlayerPayload struct {
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	Password    string `json:"password"`
}

type postAdminTestPlayerResponse struct {
	PlayerID int64  `json:"playerId"`
	Email    string `json:"email"`
}

func (m *Mux) postAdminTestPlayer() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var pp postAdminTestPlayerPayload
		if !decodeRequest(w, r, &pp) {
			return
		}

		if pp.Email == "" {
			writeJSONError(w, http.StatusBadRequest, errors.New("email is required"))
			return
		}

		if err := validatePassword(pp.Password); err != nil {
			writeJSONError(w, http.StatusBadRequest, err)
			return
		}

		displayName := pp.DisplayName
		if displayName == "" {
			displayName = util.GetRandomName()
		}

		player, err := m.repos.Players.CreatePlayer(r.Context(), pp.Email, displayName, pp.Password, "")
		if err != nil {
			if err == model.ErrDuplicateKey {
				writeJSONError(w, http.StatusBadRequest, errors.New("email address is already taken"))
				return
			}

			writeJSONError(w, http.StatusInternalServerError, err)
			return
		}

		player.Status = model.PlayerStatusVerified
		if err := m.repos.Players.Save(r.Context(), player); err != nil {
			writeJSONError(w, http.StatusInternalServerError, err)
			return
		}

		writeJSON(w, http.StatusCreated, postAdminTestPlayerResponse{
			PlayerID: player.ID,
			Email:    player.Email,
		})
	}
}

func (m *Mux) getPlayerProfile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		player := r.Context().Value(ctxPlayerKey).(*model.Player)
		m.writePlayerProfile(w, r, player.ID)
	}
}

func (m *Mux) getPlayerIDProfile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		playerID, _ := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
		m.writePlayerProfile(w, r, playerID)
	}
}

func (m *Mux) writePlayerProfile(w http.ResponseWriter, r *http.Request, playerID int64) {
	start, rows, err := parsePaginationOptions(r, 1000)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err)
		return
	}

	from := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Now().In(time.UTC).Add(24 * time.Hour)

	if fromStr := r.FormValue("from"); fromStr != "" {
		parsed, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, errors.New("invalid 'from' date format, use ISO 8601"))
			return
		}
		from = parsed.In(time.UTC)
	}

	if toStr := r.FormValue("to"); toStr != "" {
		parsed, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, errors.New("invalid 'to' date format, use ISO 8601"))
			return
		}
		to = parsed.In(time.UTC)
	}

	profile, err := m.repos.Players.GetPlayerProfile(r.Context(), playerID, from, to, start, rows)
	if err != nil {
		writeMaybeNotFoundError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, profile)
}

func validatePassword(password string) error {
	if len(password) < 6 {
		return errors.New("password must be at least six characters")
	}

	return nil
}
