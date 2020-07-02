package mux

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const maxRows = 100
const defaultRows = 100

func parsePaginationOptions(r *http.Request) (int64, int, error) {
	start := int64(0)
	rows := defaultRows

	if startStr := r.FormValue("start"); startStr != "" {
		val, err := strconv.ParseInt(startStr, 10, 64)
		if err != nil {
			return 0, 0, err
		}

		if val < 0 {
			return 0, 0, errors.New("start cannot be less than zero")
		}

		start = val
	}

	if rowsStr := r.FormValue("rows"); rowsStr != "" {
		val, err := strconv.Atoi(rowsStr)
		if err != nil {
			return 0, 0, err
		}

		if val <= 0 {
			return 0, 0, errors.New("rows must be greater than zero")
		}

		if val > maxRows {
			return 0, 0, fmt.Errorf("rows cannot be greater than %d", maxRows)
		}

		rows = val
	}

	return start, rows, nil
}

func remoteAddr(r *http.Request) string {
	parts := strings.Split(r.RemoteAddr, ":")
	if len(parts) == 1 {
		return parts[0]
	}

	return strings.Join(parts[0:len(parts)-1], ":")
}

func decodeRequest(w http.ResponseWriter, r *http.Request, payload interface{}) bool {
	if ct := r.Header.Get("Content-Type"); ct != "application/json" && ct != "text/json" {
		writeJSONError(w, http.StatusUnsupportedMediaType, nil)
		return false
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSONError(w, http.StatusBadRequest, err)
		return false
	}

	return true
}

func writeJSON(w http.ResponseWriter, statusCode int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		logrus.WithError(err).Error("could not write JSON response")
	}
}

type errorResponse struct {
	Message    string `json:"message"`
	StatusCode int    `json:"statusCode"`
}

// if err is sql.ErrNoRows, treat as 404, otherwise treat as a 500
func writeMaybeNotFoundError(w http.ResponseWriter, err error) {
	if err == sql.ErrNoRows {
		writeJSONError(w, http.StatusNotFound, nil)
		return
	}

	writeJSONError(w, http.StatusInternalServerError, err)
}

func writeJSONError(w http.ResponseWriter, statusCode int, err error) {
	var msg string

	if statusCode < 500 && err != nil {
		msg = err.Error()
	} else {
		msg = http.StatusText(statusCode)
	}

	if statusCode >= 500 {
		logrus.WithField("statusCode", statusCode).Error(err)
	}

	writeJSON(w, statusCode, errorResponse{
		Message:    msg,
		StatusCode: statusCode,
	})
}

func assertDo(t *testing.T, req *http.Request, respObj interface{}, statusCode int, signedJWT ...string) *http.Response {
	t.Helper()

	if len(signedJWT) > 0 {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", signedJWT[0]))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
		return nil
	}
	defer resp.Body.Close()

	if statusCode != resp.StatusCode {
		b, _ := ioutil.ReadAll(resp.Body)
		t.Log(string(b))
		assert.Equal(t, statusCode, resp.StatusCode)
		return nil
	}

	if respObj != nil {
		if err := json.NewDecoder(resp.Body).Decode(respObj); err != nil {
			t.Error(err)
			return nil
		}
	}

	return resp
}

func assertGetWithResp(t *testing.T, ts *httptest.Server, path string, respObj interface{}, statusCode int, signedJWT ...string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, ts.URL+path, nil)
	if err != nil {
		t.Error(err)
		return nil
	}

	return assertDo(t, req, respObj, statusCode, signedJWT...)
}

func assertGet(t *testing.T, ts *httptest.Server, path string, respObj interface{}, statusCode int, signedJWT ...string) {
	t.Helper()
	resp := assertGetWithResp(t, ts, path, respObj, statusCode, signedJWT...)
	_ = resp.Body.Close()
}

func assertPostWithResp(t *testing.T, ts *httptest.Server, path string, payload interface{}, respObj interface{}, statusCode int, signedJWT ...string) *http.Response {
	t.Helper()

	var body io.Reader
	switch val := payload.(type) {
	case string:
		body = strings.NewReader(val)
	default:
		b, err := json.Marshal(val)
		if err != nil {
			t.Error(err)
			return nil
		}
		body = bytes.NewReader(b)
	}

	req, err := http.NewRequest(http.MethodPost, ts.URL+path, body)
	if err != nil {
		t.Error(err)
		return nil
	}
	req.Header.Set("Content-Type", "application/json")

	return assertDo(t, req, respObj, statusCode, signedJWT...)
}

func assertPost(t *testing.T, ts *httptest.Server, path string, payload interface{}, respObj interface{}, statusCode int, signedJWT ...string) {
	t.Helper()
	resp := assertPostWithResp(t, ts, path, payload, respObj, statusCode, signedJWT...)
	resp.Body.Close()
}
