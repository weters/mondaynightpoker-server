package littlel

import (
	"errors"
	"fmt"
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable"
	"strconv"
	"strings"
)

const maxParticipants = 10

// seed of 0 means truly random shuffle
// setting to a global so we can override in a test
var seed int64 = 0

// Game represents an individual game of Little L
type Game struct {
	playerIDs        []int64
	idToParticipant  map[int64]*Participant
	options          Options
	logChan          chan []*playable.LogMessage
	tradeInsBitField int
	deck             *deck.Deck
	decisionIndex    int
	pot              int
	currentBet       int
	// stage 0 = trade-in
	// stage 1 pre-turn
	// stage 2 after card 1 flip
	// stage 3 after card 2 flip
	// stage 4 final betting round after card 3 flip
	stage     int
	community []*deck.Card
	discards  []*deck.Card
}

// NewGame returns a new instance of the game
func NewGame(tableUUID string, playerIDs []int64, options Options) (*Game, error) {
	if options.Ante <= 0 {
		return nil, errors.New("ante must be greater than zero")
	}

	if options.InitialDeal < 3 || options.InitialDeal > 5 {
		return nil, errors.New("the initial deal must be between 3 and 5 cards")
	}

	if len(playerIDs) < 2 {
		return nil, errors.New("you must have at least two participants")
	}

	if len(playerIDs) > maxParticipants {
		return nil, fmt.Errorf("you cannot have more than %d participants", maxParticipants)
	}

	d := deck.New()
	d.Shuffle(seed)

	idToParticipant := make(map[int64]*Participant)
	for _, id := range playerIDs {
		idToParticipant[id] = newParticipant(id, options.Ante)
	}

	g := &Game{
		options:         options,
		playerIDs:       append([]int64{}, playerIDs...), // make a copy
		idToParticipant: idToParticipant,
		deck:            d,
		pot:             len(idToParticipant) * options.Ante,
		discards:        []*deck.Card{},
	}

	if err := g.parseTradeIns(options.TradeIns); err != nil {
		return nil, err
	}

	return g, nil
}

// DealCards will deal the cards to each player
func (g *Game) DealCards() error {
	// if we really wanted to, we could make this more efficient
	// doing it this way though mimics how deals are handled in real-life
	for i := 0; i < g.options.InitialDeal; i++ {
		for _, id := range g.playerIDs {
			card, err := g.deck.Draw()
			if err != nil {
				return err
			}

			g.idToParticipant[id].hand.AddCard(card)
		}
	}

	community := make([]*deck.Card, 3)
	for i := 0; i < 3; i++ {
		card, err := g.deck.Draw()
		if err != nil {
			return err
		}
		community[i] = card
	}
	g.community = community

	return nil
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
func (g *Game) GetAllowedTradeIns() string {
	tradeIns := make([]string, 0, len(g.options.TradeIns))
	for i := 0; i < g.options.InitialDeal; i++ {
		if g.tradeInsBitField&(1<<i) > 0 {
			tradeIns = append(tradeIns, strconv.Itoa(i))
		}
	}

	return strings.Join(tradeIns, ", ")
}

// GetCurrentTurn returns the current participant who needs to make a decision
func (g *Game) GetCurrentTurn() *Participant {
	if g.decisionIndex >= len(g.playerIDs) {
		return nil
	}

	p := g.idToParticipant[g.playerIDs[g.decisionIndex]]
	if p.didFold {
		panic("decision index is on a folded player")
	}

	return p
}

// IsStageOver returns true if all participants have had a turn
func (g *Game) IsStageOver() bool {
	return g.GetCurrentTurn() == nil
}

// reset should be called when we enter a new stage
func (g *Game) reset() {
	// find first live player
	g.decisionIndex = -1
	g.advanceDecision()

	g.currentBet = 0
}

func (g *Game) advanceDecision() {
	// increment the decision index until we find the first player who hasn't folded, or we reached the end
	i := 0
	for i = g.decisionIndex + 1; i < len(g.playerIDs) && g.idToParticipant[g.playerIDs[i]].didFold; i++ {
		// noop
	}
	g.decisionIndex = i
}

func (g *Game) tradeCardsForParticipant(p *Participant, cards []*deck.Card) error {
	if g.stage != 0 {
		return errors.New("we are not in the trade-in stage")
	}

	if g.GetCurrentTurn() != p {
		return errors.New("it is not your turn")
	}

	if !g.CanTrade(len(cards)) {
		return fmt.Errorf("the valid trade-ins are: %s; you tried to trade %d", g.GetAllowedTradeIns(), len(cards))
	}

	uniq := make(map[string]bool)
	for _, card := range cards {
		if !p.hand.HasCard(card) {
			return fmt.Errorf("you do not have %s in your hand", card.String())
		}

		uniq[card.String()] = true
	}

	if len(uniq) != len(cards) {
		return errors.New("invalid trade-in")
	}

	discards := make([]*deck.Card, 0, len(cards))
	for _, card := range cards {
		p.hand.Discard(card)
		discards = append(discards, card)

		if !g.deck.CanDraw(1) {
			g.deck.ShuffleDiscards(g.discards)
			g.discards = []*deck.Card{}
		}

		card, err := g.deck.Draw()
		if err != nil {
			return err
		}

		p.hand.AddCard(card)
	}
	g.discards = append(g.discards, discards...)

	g.advanceDecision()
	return nil
}
