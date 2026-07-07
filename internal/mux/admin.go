package mux

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gorilla/mux"
)

func (m *Mux) getAdminTable() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start, rows, err := parsePaginationOptions(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err)
			return
		}

		tables, err := m.repos.Tables.GetTables(r.Context(), start, rows)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err)
			return
		}

		writeJSON(w, http.StatusOK, tables)
	}
}

type postAdminTableUUIDPayload struct {
	Deleted bool `json:"deleted"`
}

func (m *Mux) postAdminTableUUID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uuid := mux.Vars(r)["uuid"]
		table, err := m.repos.Tables.GetTableByUUID(r.Context(), uuid)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeJSONError(w, http.StatusNotFound, nil)
			} else {
				writeJSONError(w, http.StatusInternalServerError, err)
			}

			return
		}

		var payload postAdminTableUUIDPayload
		if !decodeRequest(w, r, &payload) {
			return
		}

		if table.Deleted == payload.Deleted {
			return
		}

		table.Deleted = payload.Deleted
		if err := m.repos.Tables.Save(r.Context(), table); err != nil {
			writeJSONError(w, http.StatusInternalServerError, err)
			return
		}

		writeJSON(w, http.StatusOK, table)
	}
}
