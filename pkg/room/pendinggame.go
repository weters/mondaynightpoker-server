package room

import (
	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/room/gamefactory"
	"time"
)

const secondsUntilStart = time.Second * 10

type pendingGame struct {
	Name     string    `json:"name"`
	Ante     int       `json:"ante"`
	Start    time.Time `json:"start"`
	PlayerID int64     `json:"playerId"`
	client   *Client
	message  *playable.PayloadIn
	timer    *time.Timer
}

func newPendingGame(c *Client, msg *playable.PayloadIn) (*pendingGame, error) {
	factory, err := gamefactory.Get(msg.Subject)
	if err != nil {
		return nil, err
	}

	name, ante, err := factory.Details(msg.AdditionalData)
	if err != nil {
		return nil, err
	}

	start := time.Now().Add(secondsUntilStart)
	timer := time.NewTimer(time.Until(start))

	return &pendingGame{
		client:   c,
		message:  msg,
		Name:     name,
		Ante:     ante,
		Start:    start,
		PlayerID: c.player.ID,
		timer:    timer,
	}, nil
}
