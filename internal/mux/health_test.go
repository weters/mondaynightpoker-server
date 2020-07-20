package mux

import (
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	ts := httptest.NewServer(NewMux("v1.2.3"))
	defer ts.Close()

	var expects healthResponse
	assertGet(t, ts, "/health", &expects, 200)
	assert.Equal(t, "OK", expects.Status)
	assert.Equal(t, "v1.2.3", expects.Version)
}
