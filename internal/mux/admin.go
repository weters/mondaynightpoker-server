package mux

import (
	"mondaynightpoker-server/pkg/table"
	"net/http"
)

func (m *Mux) getAdminTable() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start, rows, err := parsePaginationOptions(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err)
			return
		}

		tables, err := table.GetTables(r.Context(), start, rows)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err)
			return
		}

		writeJSON(w, http.StatusOK, tables)
	}
}
