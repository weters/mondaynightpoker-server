package littlel

import (
	"errors"
	"fmt"
	"mondaynightpoker-server/pkg/playable"
)

// Game represents an individual game of Little L
type Game struct {
	options          Options
	logChan          chan []*playable.LogMessage
	tradeInsBitField int
}

// New returns a new instance of the game
func New(options Options) (*Game, error) {
	if options.Ante <= 0 {
		return nil, errors.New("ante must be greater than zero")
	}

	if options.InitialDeal < 3 || options.InitialDeal > 5 {
		return nil, errors.New("the initial deal must be between 3 and 5 cards")
	}

	g := &Game{
		options: options,
	}

	if err := g.parseTradeIns(options.TradeIns); err != nil {
		return nil, err
	}

	return g, nil
}

// Action performs a game action on behalf of the player
func (g *Game) Action(playerID int64, message *playable.PayloadIn) (playerResponse *playable.Response, updateState bool, err error) {
	panic("implement me")
}

// GetPlayerState returns the state of the player
func (g *Game) GetPlayerState(playerID int64) (*playable.Response, error) {
	panic("implement me")
}

// GetEndOfGameDetails returns the details at the end of a game
func (g *Game) GetEndOfGameDetails() (gameOverDetails *playable.GameOverDetails, isGameOver bool) {
	panic("implement me")
}

// Name returns the name of the game
func (g *Game) Name() string {
	return "Little L"
}

// LogChan returns a channel that can receive log messages
func (g *Game) LogChan() <-chan []*playable.LogMessage {
	return g.logChan
}

// parseTradeins converts an int array into a bitwise int
func (g *Game) parseTradeIns(values []int) error {
	tradeIns := 0
	for _, val := range values {
		if val < 0 || val > g.options.InitialDeal {
			return fmt.Errorf("invalid trade-in option: %d", val)
		}

		tradeIns |= 1 << val
	}

	g.tradeInsBitField = tradeIns
	return nil
}

// CanTrade returns true if the player can trade the supplied count of cards
func (g *Game) CanTrade(count int) bool {
	val := 1 << count
	return g.tradeInsBitField&val > 0
}

// GetAllowedTradeIns returns the an integer slice of allowed trade-ins
func (g *Game) GetAllowedTradeIns() []int {
	tradeIns := make([]int, 0, len(g.options.TradeIns))
	for i := 0; i < g.options.InitialDeal; i++ {
		if g.tradeInsBitField&(1<<i) > 0 {
			tradeIns = append(tradeIns, i)
		}
	}

	return tradeIns
}
