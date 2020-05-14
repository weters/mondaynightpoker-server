package mux

import "net/http"

type healthResponse struct {
	Status string `json:"status"`
	Version string `json:"version"`
}

func (m *Mux) getHealth() http.HandlerFunc {
	payload := healthResponse{
		Status: "OK",
		Version: m.version,
	}

	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, payload)
	}
}
