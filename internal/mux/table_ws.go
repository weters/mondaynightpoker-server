package mux

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"net/http"
	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/room"
	"mondaynightpoker-server/pkg/table"
	"time"
)

const writeWait = time.Second * 10
const pongWait = time.Second * 60
const pingPeriod = pongWait * 9 / 10

func (m *Mux) getTableUUIDWS() http.HandlerFunc {
	upgrader := &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
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

		tbl := r.Context().Value(ctxTableKey).(*table.Table)
		player := r.Context().Value(ctxPlayerKey).(*table.Player)
		client := room.NewClient(conn, player, tbl)

		m.pitBoss.ClientConnected(client)

		waitForCloseFrame := make(chan bool)
		defer func() {
			m.pitBoss.ClientDisconnected(client)
			_ = conn.Close()
			close(waitForCloseFrame)
		}()

		go m.webSocketWriteLoop(client, waitForCloseFrame)
		m.webSocketReadLoop(client)
	}
}

func (m *Mux) webSocketWriteLoop(client *room.Client, waitForCloseFrame chan bool) {
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
		case reason := <-client.Close:
			_ = client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			_ = client.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, reason))

			// wait for the close frame
			select {
			case <-waitForCloseFrame:
			case <-time.After(time.Second):
			}
			return
		case msg, ok := <-client.Send:
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

			client.CloseError = err
			return
		}

		client.ReceivedMessage(&msg)
	}
}
