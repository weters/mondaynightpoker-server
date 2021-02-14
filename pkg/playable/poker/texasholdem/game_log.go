package texasholdem

import "mondaynightpoker-server/pkg/deck"

type gameLog struct {
	Participants []*participantJSON `json:"participants"`
	Community    deck.Hand          `json:"community"`
	Pot          int                `json:"pot"`
}

func (g *Game) gameLog() *gameLog {
	p := make([]*participantJSON, len(g.participantOrder))
	for i, id := range g.participantOrder {
		p[i] = g.participants[id].participantJSON(g, true)
	}

	return &gameLog{
		Participants: p,
		Community:    g.community,
		Pot:          g.pot,
	}
}
