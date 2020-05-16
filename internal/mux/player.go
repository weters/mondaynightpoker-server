package mux

import (
	"database/sql"
	"errors"
	"github.com/badoux/checkmail"
	"github.com/gorilla/mux"
	"mondaynightpoker-server/internal/jwt"
	"mondaynightpoker-server/pkg/table"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

type playerPayload struct {
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	Password    string `json:"password"`
}

var validDisplayNameRx = regexp.MustCompile(`^[\p{L}\p{N} ]*\z`)
var statusOK = map[string]string{
	"status": "OK",
}


func (m *Mux) postPlayer() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var pp playerPayload
		if !decodeRequest(w, r, &pp) {
			return
		}

		if !validDisplayNameRx.MatchString(pp.DisplayName) {
			writeJSONError(w, http.StatusBadRequest, errors.New("display name must only contain letters, numbers, and spaces"))
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

		if time.Now().Sub(at) < m.config.playerCreateDelay {
			writeJSONError(w, http.StatusBadRequest, errors.New("please wait before creating another player"))
			return
		}

		displayName := pp.Email
		if pp.DisplayName != "" {
			displayName = pp.DisplayName
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

		writeJSON(w, http.StatusCreated, player)
		return
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
	JWT    string        `json:"jwt"`
	Player *table.Player `json:"player"`
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
			JWT:    signedToken,
			Player: player,
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

		writeJSON(w, http.StatusOK, player)
	}
}

func (m *Mux) getPlayer() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		offset, limit, err := parsePaginationOptions(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err)
			return
		}

		players, err := table.GetPlayers(r.Context(), offset, limit)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err)
			return
		}

		writeJSON(w, http.StatusOK, players)
	}
}
