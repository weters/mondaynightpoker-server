package littlel

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/playable/poker/action"
	"mondaynightpoker-server/pkg/playable/poker/potmanager"
	"sort"
	"strings"
	"time"
)

const maxParticipants = 10

type round int

const (
	roundTradeIn round = iota // nolint
	roundBeforeFirstTurn
	roundBeforeSecondTurn
	roundBeforeThirdTurn
	roundFinalBettingRound // nolint:deadcode,varcheck
	roundRevealWinner
)

// seed of -1 means truly crypto-random shuffle
// setting to a global so we can override in a test
var seed int64 = -1

// Game represents an individual game of Little L
type Game struct {
	playerIDs       []int64
	idToParticipant map[int64]*Participant
	options         Options
	logger          logrus.FieldLogger
	logChan         chan []*playable.LogMessage
	tradeIns        *TradeIns
	deck            *deck.Deck
	potManager      *potmanager.PotManager
	round           round
	community       []*deck.Card
	discards        []*deck.Card

	done    bool
	winners map[*Participant]int

	endGameAt time.Time
}

// NewGameV2 returns a new instance of the game
func NewGameV2(logger logrus.FieldLogger, players []playable.Player, options Options) (*Game, error) {
	if options.Ante <= 0 {
		return nil, errors.New("ante must be greater than zero")
	}

	if options.InitialDeal < 3 || options.InitialDeal > 5 {
		return nil, errors.New("the initial deal must be between 3 and 5 cards")
	}

	if len(players) < 2 {
		return nil, errors.New("you must have at least two participants")
	}

	if len(players) > maxParticipants {
		return nil, fmt.Errorf("you cannot have more than %d participants", maxParticipants)
	}

	d := deck.New()
	d.SetSeed(seed)
	d.Shuffle()

	pm := potmanager.New(options.Ante)

	idToParticipant := make(map[int64]*Participant)
	playerIDs := make([]int64, len(players))
	for i, player := range players {
		playerIDs[i] = player.GetPlayerID()
		participant := newParticipant(player.GetPlayerID(), player.GetTableStake())
		idToParticipant[player.GetPlayerID()] = participant

		if err := pm.SeatParticipant(participant); err != nil {
			return nil, err
		}
	}
	pm.FinishSeatingParticipants()

	tradeIns, err := NewTradeIns(options.TradeIns, options.InitialDeal)
	if err != nil {
		return nil, err
	}

	g := &Game{
		options:         options,
		playerIDs:       playerIDs,
		idToParticipant: idToParticipant,
		deck:            d,
		potManager:      pm,
		discards:        []*deck.Card{},
		logChan:         make(chan []*playable.LogMessage, 256),
		logger:          logger,
		tradeIns:        tradeIns,
	}

	g.logChan <- playable.SimpleLogMessageSlice(0, "New game of Little L started (ante: ${%d}; trades: %s)", g.options.Ante, g.GetAllowedTradeIns().String())

	// decision round allows all-in participants to participate with trade-ins
	g.potManager.StartDecisionRound()

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

// CanTrade returns true if the player can trade the supplied count of cards
func (g *Game) CanTrade(count int) bool {
	return g.tradeIns.CanTrade(count)
}

// GetAllowedTradeIns returns the an integer slice of allowed trade-ins
func (g *Game) GetAllowedTradeIns() *TradeIns {
	return g.tradeIns
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
	p, err := g.potManager.GetInTurnParticipant()
	if err != nil {
		return nil
	}

	return g.idToParticipant[p.ID()]
}

// IsRoundOver returns true if all participants have had a turn
func (g *Game) IsRoundOver() bool {
	return g.potManager.IsRoundOver()
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

	g.round++

	if err := g.reset(); err != nil {
		return err
	}

	if g.round == roundRevealWinner {
		return g.endGame()
	}

	return nil
}

// ParticipantBets handles both bets and raises
func (g *Game) ParticipantBets(p *Participant, bet int) error {
	term := strings.ToLower(string(action.Bet))

	currentBet := g.potManager.GetBet()
	if currentBet > 0 {
		term = strings.ToLower(string(action.Raise))
	}

	if maxBet := g.potManager.GetPotLimitMaxBet(); bet > maxBet {
		return fmt.Errorf("your %s (${%d}) must not exceed the pot limit (${%d})", term, bet, maxBet)
	}

	allInAmount := g.potManager.GetParticipantAllInAmount(p)

	// only check the following logic IF the participant is not going all-in
	if bet != allInAmount {
		if bet%25 > 0 {
			return fmt.Errorf("your %s must be in multiples of ${25}", term)
		}

		if bet < g.options.Ante {
			return fmt.Errorf("your %s must at least match the ante (${%d})", term, g.options.Ante)
		}

		minRaiseTo := currentBet + g.potManager.GetRaise()
		if currentBet > 0 && bet < minRaiseTo {
			return fmt.Errorf("your raise of ${%d} must be at least equal to double the previous raise of ${%d}", bet-currentBet, g.potManager.GetRaise())
		}
	}

	return g.potManager.ParticipantBetsOrRaises(p, bet)
}

// ParticipantChecks will check for the participant as long as there's no active bet
func (g *Game) ParticipantChecks(p *Participant) error {
	return g.potManager.ParticipantChecks(p)
}

// ParticipantCalls handles when the player calls the action
func (g *Game) ParticipantCalls(p *Participant) error {
	return g.potManager.ParticipantCalls(p)
}

// ParticipantFolds handles when a player folds their hand
func (g *Game) ParticipantFolds(p *Participant) error {
	if err := g.potManager.ParticipantFolds(p); err != nil {
		return err
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
		return g.endGame()
	}

	return nil
}

// reset should be called when we enter a new round
func (g *Game) reset() error {
	if err := g.potManager.NextRound(); err != nil {
		return err
	}

	for _, p := range g.idToParticipant {
		p.reset()
	}

	return nil
}

func (g *Game) tradeCardsForParticipant(p *Participant, cards []*deck.Card) error {
	if g.round != 0 {
		return errors.New("we are not in the trade-in round")
	}

	if g.GetCurrentTurn() != p {
		return potmanager.ErrParticipantCannotAct
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
	return g.potManager.AdvanceDecision()
}

// CanRevealCards returns true if all cards are flipped
func (g *Game) CanRevealCards() bool {
	return g.round >= roundRevealWinner
}

func (g *Game) getFutureActionsForPlayer(playerID int64) []action.Action {
	if p, ok := g.idToParticipant[playerID]; !ok {
		return nil
	} else if !g.potManager.IsParticipantYetToAct(p) {
		return nil
	}

	if g.round == roundTradeIn {
		return []action.Action{action.Trade}
	}

	if g.potManager.GetBet() == 0 {
		return []action.Action{action.Check, action.Fold}
	}

	return []action.Action{action.Call, action.Fold}
}

func (g *Game) getActionsForPlayer(playerID int64) []action.Action {
	p, ok := g.idToParticipant[playerID]
	if !ok {
		// viewer
		return nil
	}

	actions := make([]action.Action, 0)
	if p == g.GetCurrentTurn() {
		if g.round == roundTradeIn {
			actions = append(actions, action.Trade)
		} else {
			if bet := g.potManager.GetBet(); bet == 0 {
				actions = append(actions, action.Check, action.Bet, action.Fold)
			} else if g.potManager.GetParticipantAllInAmount(p) < bet {
				actions = append(actions, action.Call, action.Fold)
			} else {
				actions = append(actions, action.Call, action.Raise, action.Fold)
			}
		}
	}

	return actions
}

// endGame will handle any end of game actions, calculate winners, etc.
func (g *Game) endGame() error {
	if g.winners != nil {
		panic("endGame already called")
	}

	g.potManager.EndGame()

	tiers := make(tieredHands)

	community := g.GetCommunityCards()
	for _, id := range g.playerIDs {
		p := g.idToParticipant[id]
		if p.didFold {
			continue
		}

		bestHand := p.GetBestHand(community)
		strength := bestHand.analyzer.GetStrength()

		tier, ok := tiers[strength]
		if !ok {
			tier = &strengthTier{
				strength:     strength,
				participants: make([]potmanager.Participant, 0),
			}
			tiers[strength] = tier
		}

		tier.participants = append(tier.participants, p)
	}

	winners := make(map[*Participant]int)
	payouts, err := g.potManager.PayWinners(tiers.getSortedTiers())
	if err != nil {
		return err
	}

	for participant, amount := range payouts {
		p := g.idToParticipant[participant.ID()]
		winners[p] = amount
	}
	g.winners = winners

	g.round = roundRevealWinner
	g.sendEndOfGameLogMessages()

	return nil
}

func (g *Game) sendEndOfGameLogMessages() {
	community := g.GetCommunityCards()

	lms := make([]*playable.LogMessage, 0, len(g.idToParticipant))
	for winner, amount := range g.winners {
		hand := winner.GetBestHand(community).analyzer.GetHand().String()
		lms = append(lms, playable.SimpleLogMessage(winner.PlayerID, "{} had a %s and won ${%d}", hand, amount))
	}

	for _, playerID := range g.playerIDs {
		p := g.idToParticipant[playerID]
		if _, ok := g.winners[p]; ok {
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

// NameFromOptions return names for the given options
func NameFromOptions(options Options) (string, error) {
	tradeIns, err := NewTradeIns(options.TradeIns, options.InitialDeal)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%d-Card Little L (trade: %s)", options.InitialDeal, tradeIns), nil
}
