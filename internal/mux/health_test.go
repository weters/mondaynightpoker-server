package mux

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealthHandler(t *testing.T) {
	deps := testDeps()
	deps.Version = "v1.2.3"

	ts := httptest.NewServer(NewMux(deps))
	defer ts.Close()

	var expects healthResponse
	assertGet(t, ts, "/health", &expects, 200)
	assert.Equal(t, "OK", expects.Status)
	assert.Equal(t, "v1.2.3", expects.Version)
}
