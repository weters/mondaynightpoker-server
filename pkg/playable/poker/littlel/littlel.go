package littlel

import (
	"errors"
	"fmt"
	"math"
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable"
	"sort"
)

// ErrNotPlayersTurn is an error when a player attempts to act out of turn
var ErrNotPlayersTurn = errors.New("it is not your turn")

const maxParticipants = 10

type stage int

const (
	stageTradeIn stage = iota // nolint
	stageBeforeFirstTurn
	stageBeforeSecondTurn
	stageBeforeThirdTurn
	stageFinalBettingRound // nolint
	stageRevealWinner
)

// seed of 0 means truly random shuffle
// setting to a global so we can override in a test
var seed int64 = 0

// Game represents an individual game of Little L
type Game struct {
	playerIDs          []int64
	idToParticipant    map[int64]*Participant
	options            Options
	logChan            chan []*playable.LogMessage
	tradeInsBitField   int
	deck               *deck.Deck
	decisionStartIndex int
	decisionCount      int
	pot                int
	currentBet         int
	stage              stage
	community          []*deck.Card
	discards           []*deck.Card

	done    bool
	winners []*Participant
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

	for _, p := range g.idToParticipant {
		sort.Sort(p.hand)
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
func (g *Game) GetAllowedTradeIns() TradeIns {
	tradeIns := make([]int, 0, len(g.options.TradeIns))
	for i := 0; i < g.options.InitialDeal; i++ {
		if g.tradeInsBitField&(1<<i) > 0 {
			tradeIns = append(tradeIns, i)
		}
	}

	return tradeIns
}

// GetCommunityCards will return the community cards
// A card will be nil if we have not progressed far enough in the game
func (g *Game) GetCommunityCards() []*deck.Card {
	cards := make([]*deck.Card, 3)
	if g.stage > stageBeforeFirstTurn {
		cards[0] = g.community[0]
	}

	if g.stage > stageBeforeSecondTurn {
		cards[1] = g.community[1]
	}

	if g.stage > stageBeforeThirdTurn {
		cards[2] = g.community[2]
	}

	return cards
}

// GetCurrentTurn returns the current participant who needs to make a decision
func (g *Game) GetCurrentTurn() *Participant {
	if g.decisionCount >= len(g.playerIDs) {
		return nil
	}

	// no more actions
	if g.stage > stageFinalBettingRound {
		return nil
	}

	index := (g.decisionStartIndex + g.decisionCount) % len(g.playerIDs)
	p := g.idToParticipant[g.playerIDs[index]]
	if p.didFold {
		panic("decision index is on a folded player")
	}

	return p
}

// IsStageOver returns true if all participants have had a turn
func (g *Game) IsStageOver() bool {
	return g.GetCurrentTurn() == nil
}

// IsGameOver returns true if the game is over
func (g *Game) IsGameOver() bool {
	return g.winners != nil
}

// NextStage will advance the game to the next stage
func (g *Game) NextStage() error {
	if !g.IsStageOver() {
		return errors.New("stage is not over")
	}

	if g.stage == stageRevealWinner {
		return errors.New("cannot advance the stage")
	}

	for _, p := range g.idToParticipant {
		p.balance -= p.currentBet
	}

	g.stage++
	g.reset()

	if g.stage == stageRevealWinner {
		g.endGame()
	}

	return nil
}

// ParticipantBets handles both bets and raises
func (g *Game) ParticipantBets(p *Participant, bet int) error {
	if g.GetCurrentTurn() != p {
		return ErrNotPlayersTurn
	}

	if bet > g.pot {
		return fmt.Errorf("your bet (%d¢) must not exceed the current pot (%d¢)", bet, g.pot)
	}

	if bet < g.options.Ante {
		return fmt.Errorf("your bet must at least match the ante (%d¢)", g.options.Ante)
	}

	if g.currentBet > 0 && bet < g.currentBet*2 {
		return fmt.Errorf("your raise (%d¢) must be at least equal to double the previous bet (%d¢)", bet, g.currentBet*2)
	}

	diff := bet - p.currentBet
	p.currentBet = bet
	g.currentBet = bet
	g.pot += diff

	g.setDecisionIndexToCurrentTurn()
	g.advanceDecision()

	return nil
}

// setDecisionIndexToCurrentTurn will update the decision index to the current player's turn
// This will happen when a player raises because we'll need to go around the table again
func (g *Game) setDecisionIndexToCurrentTurn() {
	currentIndex := (g.decisionStartIndex + g.decisionCount) % len(g.playerIDs)
	g.decisionStartIndex = currentIndex
	g.decisionCount = 0
}

// ParticipantChecks will check for the participant as long as there's no active bet
func (g *Game) ParticipantChecks(p *Participant) error {
	if g.GetCurrentTurn() != p {
		return ErrNotPlayersTurn
	}

	if g.currentBet != 0 {
		return errors.New("you cannot check with an active bet")
	}

	g.advanceDecision()
	return nil
}

// ParticipantCalls handles when the player calls the action
func (g *Game) ParticipantCalls(p *Participant) error {
	if g.GetCurrentTurn() != p {
		return ErrNotPlayersTurn
	}

	if g.currentBet == 0 {
		return errors.New("you cannot call without an active bet")
	}

	diff := g.currentBet - p.currentBet
	p.currentBet = g.currentBet
	g.pot += diff

	g.advanceDecision()
	return nil
}

// ParticipantFolds handles when a player folds their hand
func (g *Game) ParticipantFolds(p *Participant) error {
	if g.GetCurrentTurn() != p {
		return ErrNotPlayersTurn
	}

	p.didFold = true

	stillAlive := 0
	for _, p := range g.idToParticipant {
		if !p.didFold {
			stillAlive++
		}
	}

	if stillAlive == 0 {
		panic("too many players folded")
	} else if stillAlive == 1 {
		g.endGame()
		return nil
	}

	g.advanceDecision()
	return nil
}

// reset should be called when we enter a new stage
func (g *Game) reset() {
	for _, p := range g.idToParticipant {
		p.reset()
	}

	// find first live player
	g.decisionStartIndex = 0
	g.decisionCount = 0
	g.advanceDecisionIfPlayerFolded()
	g.currentBet = 0
}

func (g *Game) advanceDecision() {
	g.decisionCount++
	g.advanceDecisionIfPlayerFolded()
}

func (g *Game) advanceDecisionIfPlayerFolded() {
	nPlayers := len(g.playerIDs)
	for ; g.decisionCount < nPlayers; g.decisionCount++ {
		playerIndex := (g.decisionStartIndex + g.decisionCount) % nPlayers
		p := g.idToParticipant[g.playerIDs[playerIndex]]
		if !p.didFold {
			break
		}
	}
}

func (g *Game) tradeCardsForParticipant(p *Participant, cards []*deck.Card) error {
	if g.stage != 0 {
		return errors.New("we are not in the trade-in stage")
	}

	if g.GetCurrentTurn() != p {
		return ErrNotPlayersTurn
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

	sort.Sort(p.hand)

	g.advanceDecision()
	return nil
}

// CanRevealCards returns true if all cards are flipped
func (g *Game) CanRevealCards() bool {
	return g.stage >= stageRevealWinner
}

func (g *Game) getActionsForPlayer(playerID int64) []Action {
	p, ok := g.idToParticipant[playerID]
	if !ok {
		// viewer
		return nil
	}

	actions := make([]Action, 0)
	if p == g.GetCurrentTurn() {
		if g.stage == stageTradeIn {
			actions = append(actions, ActionTrade)
		} else {
			if g.currentBet == 0 {
				actions = append(actions, ActionCheck, ActionBet, ActionFold)
			} else {
				actions = append(actions, ActionCall, ActionRaise, ActionFold)
			}
		}
	}

	if g.IsGameOver() {
		actions = append(actions, ActionEndGame)
	} else if g.IsStageOver() {
		actions = append(actions, ActionNextStage)
	}

	return actions
}

// endGame will handle any end of game actions, calculate winners, etc.
func (g *Game) endGame() {
	if g.winners != nil {
		panic("endGame already called")
	}

	g.stage = stageRevealWinner

	winners := make([]*Participant, 0, 1)
	best := math.MinInt32
	for _, id := range g.playerIDs {
		p := g.idToParticipant[id]
		if p.didFold {
			continue
		}

		bestHand := p.GetBestHand(g.GetCommunityCards())
		strength := bestHand.analyzer.GetStrength()
		if strength == best {
			winners = append(winners, p)
		} else if strength > best {
			winners = []*Participant{p}
			best = strength
		}
	}

	g.winners = winners
	for _, winner := range winners {
		winner.balance += g.pot / len(winners)
	}

	if mod := g.pot % len(winners); mod > 0 {
		winners[0].balance += mod
	}
}
