package mux

import (
	"context"
	"errors"
	"mondaynightpoker-server/pkg/model"
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

		player := r.Context().Value(ctxPlayerKey).(*model.Player)
		tables, err := m.repos.Players.GetTables(r.Context(), player, offset, limit)
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

var tableNameWordChar = regexp.MustCompile(`\w`)

// validTableName reports whether name is an acceptable table name
func validTableName(name string) bool {
	return tableNameWordChar.MatchString(name) && len(name) >= 3 && len(name) <= 40
}

func (m *Mux) postTable() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var pp postTablePayload
		if !decodeRequest(w, r, &pp) {
			return
		}

		if !validTableName(pp.Name) {
			writeJSONError(w, http.StatusBadRequest, errors.New("name must be 3-40 characters"))
			return
		}

		player := r.Context().Value(ctxPlayerKey).(*model.Player)
		tbl, err := m.repos.Tables.CreateTable(r.Context(), player, pp.Name)
		if err != nil {
			var ue model.UserError
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
	*model.Table
	Players []*model.PlayerTable `json:"players"`
}

func (m *Mux) getTableUUID() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tbl := r.Context().Value(ctxTableKey).(*model.Table)
		players, err := m.repos.Tables.GetPlayers(r.Context(), tbl)
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
		player := r.Context().Value(ctxPlayerKey).(*model.Player)
		tbl := r.Context().Value(ctxTableKey).(*model.Table)

		playerTable, err := m.repos.Tables.Join(r.Context(), player, tbl)
		if err != nil {
			if err == model.ErrDuplicateKey {
				writeJSONError(w, http.StatusBadRequest, errors.New("player is already at the table"))
			} else {
				writeJSONError(w, http.StatusInternalServerError, err)
			}

			return
		}

		writeJSON(w, http.StatusCreated, playerTable)
	})
}

func (m *Mux) postTableUUIDClone() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var pp postTablePayload
		if !decodeRequest(w, r, &pp) {
			return
		}

		if !validTableName(pp.Name) {
			writeJSONError(w, http.StatusBadRequest, errors.New("name must be 3-40 characters"))
			return
		}

		player := r.Context().Value(ctxPlayerKey).(*model.Player)
		tbl := r.Context().Value(ctxTableKey).(*model.Table)

		isAdmin, err := m.isTableAdmin(r.Context(), player, tbl)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err)
			return
		}

		if !isAdmin {
			writeJSONError(w, http.StatusBadRequest, model.UserError("only a table admin can clone a table"))
			return
		}

		newTable, err := m.repos.Tables.CloneTable(r.Context(), player, tbl, pp.Name)
		if err != nil {
			var ue model.UserError
			if errors.As(err, &ue) {
				writeJSONError(w, http.StatusBadRequest, err)
			} else {
				writeJSONError(w, http.StatusInternalServerError, err)
			}
			return
		}

		writeJSON(w, http.StatusCreated, newTable)
	})
}

func (m *Mux) postTableUUIDName() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var pp postTablePayload
		if !decodeRequest(w, r, &pp) {
			return
		}

		if !validTableName(pp.Name) {
			writeJSONError(w, http.StatusBadRequest, errors.New("name must be 3-40 characters"))
			return
		}

		player := r.Context().Value(ctxPlayerKey).(*model.Player)
		tbl := r.Context().Value(ctxTableKey).(*model.Table)

		isAdmin, err := m.isTableAdmin(r.Context(), player, tbl)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err)
			return
		}

		if !isAdmin {
			writeJSONError(w, http.StatusBadRequest, model.UserError("only a table admin can rename a table"))
			return
		}

		tbl.Name = pp.Name
		if err := m.repos.Tables.Save(r.Context(), tbl); err != nil {
			writeJSONError(w, http.StatusInternalServerError, err)
			return
		}

		// re-fetch so the response carries the updated modified timestamp
		saved, err := m.repos.Tables.GetTableByUUID(r.Context(), tbl.UUID)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err)
			return
		}

		writeJSON(w, http.StatusOK, saved)
	})
}

// isTableAdmin reports whether the player may perform table-admin actions on
// the table. Site admins always may, even if they are not seated.
func (m *Mux) isTableAdmin(ctx context.Context, player *model.Player, tbl *model.Table) (bool, error) {
	if player.IsSiteAdmin {
		return true, nil
	}

	pt, err := m.repos.Tables.GetPlayerTable(ctx, player, tbl)
	if err != nil && !errors.Is(err, model.ErrPlayerNotAtTable) {
		return false, err
	}

	return pt != nil && pt.IsTableAdmin, nil
}

func (m *Mux) tableMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uuid := mux.Vars(r)["uuid"]
		tbl, err := m.repos.Tables.GetTableByUUID(r.Context(), uuid)
		if err != nil {
			writeMaybeNotFoundError(w, err)
			return
		}

		newCtx := context.WithValue(r.Context(), ctxTableKey, tbl)

		next.ServeHTTP(w, r.WithContext(newCtx))
	})
}
