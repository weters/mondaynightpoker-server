package mux

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"mondaynightpoker-server/pkg/model"
	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/room"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

const writeWait = time.Second * 10
const pongWait = time.Second * 60
const pingPeriod = pongWait * 9 / 10

// checkOrigin reports whether the request may open a WebSocket connection.
// Requests without an Origin header (non-browser clients) are allowed; browsers
// must match one of the configured origins.
func (m *Mux) checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}

	originURL, err := url.Parse(origin)
	if err != nil {
		logrus.WithField("origin", origin).Warn("rejecting WebSocket connection with unparseable origin")
		return false
	}

	for _, allowed := range m.cfg.WebSocketOrigins() {
		allowedURL, err := url.Parse(allowed)
		if err != nil {
			continue
		}

		if strings.EqualFold(originURL.Scheme, allowedURL.Scheme) && strings.EqualFold(originURL.Host, allowedURL.Host) {
			return true
		}
	}

	logrus.WithField("origin", origin).Warn("rejecting WebSocket connection from disallowed origin")
	return false
}

func (m *Mux) getTableUUIDWS() http.HandlerFunc {
	upgrader := &websocket.Upgrader{
		CheckOrigin: m.checkOrigin,
	}

	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logrus.WithError(err).Error("could not upgrade connected")
			return
		}

		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		conn.SetPongHandler(func(string) error {
			_ = conn.SetReadDeadline(time.Now().Add(pongWait))
			return nil
		})

		tbl := r.Context().Value(ctxTableKey).(*model.Table)
		player := r.Context().Value(ctxPlayerKey).(*model.Player)
		client := room.NewClient(conn, player, tbl)

		m.pitBoss.ClientConnected(client)

		defer func() {
			m.pitBoss.ClientDisconnected(client)
			_ = conn.Close()
		}()

		go m.webSocketWriteLoop(client)
		m.webSocketReadLoop(client)
	}
}

func (m *Mux) webSocketWriteLoop(client *room.Client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = client.Conn.Close()
	}()

	for {
		select {
		case <-ticker.C:
			_ = client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case msg, ok := <-client.SendChan():
			if !ok {
				return
			}

			msgBytes, _ := json.Marshal(msg)
			logrus.WithField("message", string(msgBytes)).WithField("client", client.String()).Trace("sending message to client")

			_ = client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := client.Conn.WriteJSON(msg); err != nil {
				logrus.WithError(err).WithField("client", client.String()).Error("could not write message")
				return
			}
		}
	}
}

func (m *Mux) webSocketReadLoop(client *room.Client) {
	for {
		var msg playable.PayloadIn
		if err := client.Conn.ReadJSON(&msg); err != nil {
			if !websocket.IsUnexpectedCloseError(err) {
				logrus.WithError(err).Error("could not read JSON")
			} else if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure) {
				logrus.WithError(err).Error("could not read onMessage")
			}

			return
		}

		client.ReceivedMessage(&msg)
	}
}
