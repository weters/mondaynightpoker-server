package sevencard

import (
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable/poker/handanalyzer"
)

// participant is an individual player in seven-card poker
type participant struct {
	PlayerID int64
	hand     deck.Hand
	didFold  bool

	balance    int
	currentBet int

	didWin bool

	handAnalyzer         *handanalyzer.HandAnalyzer
	handAnalyzerCacheKey string
}

func newParticipant(playerID int64, ante int) *participant {
	return &participant{
		PlayerID: playerID,
		hand:     make(deck.Hand, 0, 11), // room for 7 cards, plus potential 4 extras (e.g., Baseball)
		balance:  -1 * ante,
	}
}

func (p *participant) resetForNewRound() {
	p.currentBet = 0
}

func (p *participant) getHandAnalyzer() *handanalyzer.HandAnalyzer {
	key := p.hand.String()
	if p.handAnalyzerCacheKey != key {
		p.handAnalyzer = handanalyzer.New(5, p.hand)
	}

	return p.handAnalyzer
}
