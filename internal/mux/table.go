package mux

import (
	"context"
	"errors"
	"mondaynightpoker-server/pkg/table"
	"net/http"
	"regexp"

	"github.com/gorilla/mux"
)

func (m *Mux) getTable() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		offset, limit, err := parsePaginationOptions(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err)
			return
		}

		player := r.Context().Value(ctxPlayerKey).(*table.Player)
		tables, err := player.GetTables(r.Context(), offset, limit)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err)
			return
		}

		writeJSON(w, http.StatusOK, tables)
	})
}

type postTablePayload struct {
	Name string `json:"name"`
}

func (m *Mux) postTable() http.HandlerFunc {
	var wordChar = regexp.MustCompile(`\w`)
	return func(w http.ResponseWriter, r *http.Request) {
		var pp postTablePayload
		if !decodeRequest(w, r, &pp) {
			return
		}

		if !wordChar.MatchString(pp.Name) || len(pp.Name) < 3 || len(pp.Name) > 40 {
			writeJSONError(w, http.StatusBadRequest, errors.New("name must be 3-40 characters"))
			return
		}

		player := r.Context().Value(ctxPlayerKey).(*table.Player)
		tbl, err := player.CreateTable(r.Context(), pp.Name)
		if err != nil {
			var ue table.UserError
			if errors.As(err, &ue) {
				writeJSONError(w, http.StatusBadRequest, err)
			} else {
				writeJSONError(w, http.StatusInternalServerError, err)
			}
			return
		}

		writeJSON(w, http.StatusCreated, tbl)
	}
}

type getTableUUIDResponse struct {
	*table.Table
	Players []*table.PlayerTable `json:"players"`
}

func (m *Mux) getTableUUID() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tbl := r.Context().Value(ctxTableKey).(*table.Table)
		players, err := tbl.GetPlayers(r.Context())
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err)
			return
		}

		writeJSON(w, http.StatusOK, getTableUUIDResponse{
			Table:   tbl,
			Players: players,
		})
	})
}

func (m *Mux) postTableUUIDSeat() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		player := r.Context().Value(ctxPlayerKey).(*table.Player)
		tbl := r.Context().Value(ctxTableKey).(*table.Table)

		playerTable, err := player.Join(r.Context(), tbl)
		if err != nil {
			if err == table.ErrDuplicateKey {
				writeJSONError(w, http.StatusBadRequest, errors.New("player is already at the table"))
			} else {
				writeJSONError(w, http.StatusInternalServerError, err)
			}

			return
		}

		writeJSON(w, http.StatusCreated, playerTable)
	})
}

func (m *Mux) tableMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uuid := mux.Vars(r)["uuid"]
		tbl, err := table.GetTableByUUID(r.Context(), uuid)
		if err != nil {
			writeMaybeNotFoundError(w, err)
			return
		}

		newCtx := context.WithValue(r.Context(), ctxTableKey, tbl)

		next.ServeHTTP(w, r.WithContext(newCtx))
	})
}
