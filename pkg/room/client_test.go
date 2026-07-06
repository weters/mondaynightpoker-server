package room

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mondaynightpoker-server/pkg/model"
)

func TestClient_sendBufferOverflowDisconnects(t *testing.T) {
	c := NewClient(nil, &model.Player{Email: "test@example.com"}, &model.Table{UUID: "test-uuid"})

	for i := 0; i < 256; i++ {
		assert.True(t, c.Send(i))
	}

	assert.False(t, c.Send(256), "send into a full buffer must fail")

	// Disconnect with a nil connection must be a safe no-op, and repeat calls must not panic
	c.Disconnect("still a no-op")
}

func TestClient_disconnectSendsCloseFrame(t *testing.T) {
	upgrader := websocket.Upgrader{}
	serverConn := make(chan *websocket.Conn, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		serverConn <- conn
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	peer, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer func() {
		_ = resp.Body.Close()
		_ = peer.Close()
	}()

	c := NewClient(<-serverConn, &model.Player{Email: "test@example.com"}, &model.Table{UUID: "test-uuid"})
	c.Disconnect("too slow")
	c.Disconnect("second call is ignored")

	_, _, err = peer.ReadMessage()
	var closeErr *websocket.CloseError
	require.ErrorAs(t, err, &closeErr)
	assert.Equal(t, disconnectCloseCode, closeErr.Code)
	assert.Equal(t, "too slow", closeErr.Text)
}

func TestClient_String(t *testing.T) {
	assert.Equal(t, "?:?", NewClient(nil, nil, nil).String())
	assert.Equal(
		t,
		"test@example.com:test-uuid",
		NewClient(nil, &model.Player{Email: "test@example.com"}, &model.Table{UUID: "test-uuid"}).String(),
	)
}
