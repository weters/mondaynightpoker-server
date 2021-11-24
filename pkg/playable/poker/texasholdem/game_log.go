package texasholdem

import "mondaynightpoker-server/pkg/deck"

type gameLog struct {
	Participants []*participantJSON `json:"participants"`
	Community    deck.Hand          `json:"community"`
	Pot          int                `json:"pot"`
}

func (g *Game) gameLog() *gameLog {
	p := make([]*participantJSON, len(g.participantOrder))
	for i, pt := range g.participantOrder {
		p[i] = pt.participantJSON(g, true)
	}

	return &gameLog{
		Participants: p,
		Community:    g.community,
		Pot:          g.potManager.Pots().Total(),
	}
}
