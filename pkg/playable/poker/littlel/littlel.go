package littlel

import (
	"errors"
	"fmt"
	"math"
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable"
	"sort"
	"strings"
)

// ErrNotPlayersTurn is an error when a player attempts to act out of turn
var ErrNotPlayersTurn = errors.New("it is not your turn")

const maxParticipants = 10

type round int

const (
	roundTradeIn round = iota // nolint
	roundBeforeFirstTurn
	roundBeforeSecondTurn
	roundBeforeThirdTurn
	roundFinalBettingRound
	roundRevealWinner
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
	round              round
	community          []*deck.Card
	discards           []*deck.Card

	done    bool
	winners []*Participant

	lastAdjustmentRound round // the last round an adjustment ran
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
		options:             options,
		playerIDs:           append([]int64{}, playerIDs...), // make a copy
		idToParticipant:     idToParticipant,
		deck:                d,
		pot:                 len(idToParticipant) * options.Ante,
		discards:            []*deck.Card{},
		lastAdjustmentRound: round(-1),
		logChan:             make(chan []*playable.LogMessage, 256),
	}

	if err := g.buildTradeInsBitField(options.TradeIns); err != nil {
		return nil, err
	}

	g.logChan <- playable.SimpleLogMessageSlice(0, "New game of Little L started (ante: ${%d}; trades: %s)", g.options.Ante, g.GetAllowedTradeIns().String())

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

// buildTradeInsBitField converts an int array into a bitwise int
func (g *Game) buildTradeInsBitField(values []int) error {
	// treat empty list as no trades
	if len(values) == 0 {
		g.tradeInsBitField = 1
		return nil
	}

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
	if g.round > roundBeforeFirstTurn {
		cards[0] = g.community[0]
	}

	if g.round > roundBeforeSecondTurn {
		cards[1] = g.community[1]
	}

	if g.round > roundBeforeThirdTurn {
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
	if g.round > roundFinalBettingRound {
		return nil
	}

	index := (g.decisionStartIndex + g.decisionCount) % len(g.playerIDs)
	p := g.idToParticipant[g.playerIDs[index]]
	if p.didFold {
		panic("decision index is on a folded player")
	}

	return p
}

// IsRoundOver returns true if all participants have had a turn
func (g *Game) IsRoundOver() bool {
	return g.GetCurrentTurn() == nil
}

// IsGameOver returns true if the game is over
func (g *Game) IsGameOver() bool {
	return g.winners != nil
}

// NextRound will advance the game to the next round
func (g *Game) NextRound() error {
	if !g.IsRoundOver() {
		return errors.New("round is not over")
	}

	if g.round == roundRevealWinner {
		return errors.New("cannot advance the round")
	}

	g.endOfRoundAdjustments()

	g.round++
	g.reset()

	if g.round == roundRevealWinner {
		g.endGame()
	}

	return nil
}

func (g *Game) endOfRoundAdjustments() {
	if g.lastAdjustmentRound == g.round {
		panic(fmt.Sprintf("already ran endOfRoundAdjustments() for round: %d", g.round))
	}

	g.lastAdjustmentRound = g.round

	for _, p := range g.idToParticipant {
		p.balance -= p.currentBet
	}
}

func (g *Game) getPotLimit() int {
	return g.pot + g.currentBet
}

// ParticipantBets handles both bets and raises
func (g *Game) ParticipantBets(p *Participant, bet int) error {
	term := strings.ToLower(string(ActionBet))
	if g.currentBet > 0 {
		term = strings.ToLower(string(ActionRaise))
	}

	if g.GetCurrentTurn() != p {
		return ErrNotPlayersTurn
	}

	if bet%g.options.Ante > 0 {
		return fmt.Errorf("your %s must be in multiples of the ante (%d¢)", term, g.options.Ante)
	}

	if bet > g.getPotLimit() {
		return fmt.Errorf("your %s (%d¢) must not exceed the pot limit (%d¢)", term, bet, g.getPotLimit())
	}

	if bet < g.options.Ante {
		return fmt.Errorf("your %s must at least match the ante (%d¢)", term, g.options.Ante)
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
		g.endOfRoundAdjustments()
		g.endGame()
		return nil
	}

	g.advanceDecision()
	return nil
}

// reset should be called when we enter a new round
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
	if g.round != 0 {
		return errors.New("we are not in the trade-in round")
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

	p.traded = len(cards)
	g.advanceDecision()
	return nil
}

// CanRevealCards returns true if all cards are flipped
func (g *Game) CanRevealCards() bool {
	return g.round >= roundRevealWinner
}

func (g *Game) getActionsForPlayer(playerID int64) []Action {
	p, ok := g.idToParticipant[playerID]
	if !ok {
		// viewer
		return nil
	}

	actions := make([]Action, 0)
	if p == g.GetCurrentTurn() {
		if g.round == roundTradeIn {
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
	} else if g.IsRoundOver() {
		actions = append(actions, ActionNextRound)
	}

	return actions
}

// endGame will handle any end of game actions, calculate winners, etc.
func (g *Game) endGame() {
	if g.winners != nil {
		panic("endGame already called")
	}

	g.round = roundRevealWinner

	winners := make([]*Participant, 0, 1)
	best := math.MinInt32
	community := g.GetCommunityCards()
	for _, id := range g.playerIDs {
		p := g.idToParticipant[id]
		if p.didFold {
			continue
		}

		bestHand := p.GetBestHand(community)
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
		winner.didWin = true
	}

	if mod := g.pot % len(winners); mod > 0 {
		winners[0].balance += mod
	}

	g.sendEndOfGameLogMessages()
}

func (g *Game) sendEndOfGameLogMessages() {
	community := g.GetCommunityCards()

	lms := make([]*playable.LogMessage, 0, len(g.idToParticipant))
	for _, winner := range g.winners {
		hand := winner.GetBestHand(community).analyzer.GetHand().String()
		lms = append(lms, playable.SimpleLogMessage(winner.PlayerID, "{} had a %s and won ${%d}", hand, winner.balance))
	}

	for _, playerID := range g.playerIDs {
		p := g.idToParticipant[playerID]
		if p.didWin {
			continue
		}

		if p.didFold {
			lms = append(lms, playable.SimpleLogMessage(p.PlayerID, "{} folded and lost ${%d}", -1*p.balance))
		} else {
			hand := p.GetBestHand(community).analyzer.GetHand().String()
			lms = append(lms, playable.SimpleLogMessage(p.PlayerID, "{} had a %s and lost ${%d}", hand, -1*p.balance))
		}
	}

	g.logChan <- lms
}
