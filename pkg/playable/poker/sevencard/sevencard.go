package sevencard

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"math"
	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/playable/poker/handanalyzer"
	"time"
)

const maxParticipants = 7

var errNotPlayersTurn = errors.New("it is not your turn")

// lastAction tracks the most recent action taken by a player
type lastAction struct {
	PlayerID int64  `json:"playerId"`
	Action   Action `json:"action"`
}

// Game is a single game of seven-card poker
type Game struct {
	deck            *deck.Deck
	round           round
	options         Options
	playerIDs       []int64
	idToParticipant map[int64]*participant

	// nolint: godox
	// TODO: possibly refactor with Little L
	decisionStartIndex int
	decisionCount      int
	dealerIndex        int

	currentBet int
	pot        int

	lastAction *lastAction
	winners    map[*participant]int

	pendingLogs []*playable.LogMessage
	logChan     chan []*playable.LogMessage
	logger      logrus.FieldLogger

	setDoneAt time.Time // set done to true after this time

	// done will be set to true if the game has ended, and the players advance
	done bool
}

// NewGame returns a new seven-card poker Game
// Deprecated: Use NewGameV2 instead for proper table stake support
func NewGame(logger logrus.FieldLogger, playerIDs []int64, options Options) (*Game, error) {
	// Create players with 0 table stake for backwards compatibility
	players := make([]playable.Player, len(playerIDs))
	for i, id := range playerIDs {
		players[i] = &simplePlayer{id: id, tableStake: 0}
	}
	return NewGameV2(logger, players, options)
}

// simplePlayer implements playable.Player for backwards compatibility
type simplePlayer struct {
	id         int64
	tableStake int
}

func (s *simplePlayer) GetPlayerID() int64  { return s.id }
func (s *simplePlayer) GetTableStake() int { return s.tableStake }

// NewGameV2 returns a new seven-card poker Game with table stake support
func NewGameV2(logger logrus.FieldLogger, players []playable.Player, options Options) (*Game, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	if len(players) < 2 {
		return nil, errors.New("you must have at least two participants")
	}

	if len(players) > maxParticipants {
		return nil, fmt.Errorf("seven-card allows at most %d participants", maxParticipants)
	}

	d := deck.New()
	d.Shuffle()

	options.Variant.Start()

	playerIDs := make([]int64, len(players))
	idToParticipant := make(map[int64]*participant)
	for i, player := range players {
		playerIDs[i] = player.GetPlayerID()
		idToParticipant[player.GetPlayerID()] = newParticipant(
			player.GetPlayerID(),
			player.GetTableStake(),
			options.Ante,
		)
	}

	game := &Game{
		deck:            d,
		options:         options,
		playerIDs:       playerIDs,
		idToParticipant: idToParticipant,
		dealerIndex:     len(players) - 1, // dealer is last player
		pot:             options.Ante * len(players),
		pendingLogs:     make([]*playable.LogMessage, 0),
		logChan:         make(chan []*playable.LogMessage, 256),
		logger:          logger,
	}

	return game, nil
}

// Start starts the game of Seven-Card poker
func (g *Game) Start() error {
	if g.round != beforeDeal {
		return errors.New("the game has already started")
	}

	// deal two face-down, one face-up
	for _, faceDown := range []bool{true, true, false} {
		if err := g.dealCards(faceDown); err != nil {
			return err
		}
	}

	g.pendingLogs = append(g.pendingLogs, playable.SimpleLogMessage(0, "New game of %s started (ante: ${%d})", g.Name(), g.options.Ante))

	g.determineFirstToAct()
	g.round++

	g.logChan <- g.pendingLogs
	g.pendingLogs = make([]*playable.LogMessage, 0)

	return nil
}

func (g *Game) nextRound() {
	g.round++
	g.currentBet = 0

	for _, p := range g.idToParticipant {
		p.resetForNewRound()
	}

	var cardName string
	var err error
	switch g.round {
	case secondBettingRound:
		cardName = "fourth street"
		err = g.dealCards(false)
	case thirdBettingRound:
		cardName = "fifth street"
		err = g.dealCards(false)
	case fourthBettingRound:
		cardName = "sixth street"
		err = g.dealCards(false)
	case finalBettingRound:
		cardName = "river"
		err = g.dealCards(true)
	case revealWinner:
		g.endGame()
		return
	default:
		panic(fmt.Sprintf("round %d is not implemented", g.round))
	}

	if cardName != "" {
		g.pendingLogs = append(g.pendingLogs, playable.SimpleLogMessage(0, "Dealt %s", cardName))
	}

	if err != nil {
		panic(fmt.Sprintf("could not deal cards: %v", err))
	}

	g.determineFirstToAct()
}

func (g *Game) isRoundOver() bool {
	return g.getCurrentTurn() == nil
}

// advanceDecision moves the decision to the next participant still active
func (g *Game) advanceDecision() {
	g.decisionCount++
	g.advanceDecisionIfPlayerDidFold()

	if g.isRoundOver() {
		g.nextRound()
	}
}

func (g *Game) setDecisionIndexToCurrentTurn() {
	currentIndex := (g.decisionStartIndex + g.decisionCount) % len(g.playerIDs)
	g.decisionStartIndex = currentIndex
	g.decisionCount = 0
}

// advanceDecisionIfPlayerDidFold will advance the decision to the next participant still active
// if the current decision is with a folded participant
func (g *Game) advanceDecisionIfPlayerDidFold() {
	nPlayers := len(g.playerIDs)
	for ; g.decisionCount < nPlayers; g.decisionCount++ {
		index := (g.decisionStartIndex + g.decisionCount) % nPlayers
		p := g.idToParticipant[g.playerIDs[index]]
		if !p.didFold {
			break
		}
	}
}

// getCurrentTurn returns the participant who needs to make a decision
// A nil value is returned if all players have made their decision for the current round,
// or if we reached the end of the game
func (g *Game) getCurrentTurn() *participant {
	if g.decisionCount >= len(g.playerIDs) {
		return nil
	}

	if g.round > finalBettingRound {
		return nil
	}

	index := (g.decisionStartIndex + g.decisionCount) % len(g.playerIDs)
	p := g.idToParticipant[g.playerIDs[index]]
	if p.didFold {
		panic("decision is on a player who folded")
	}

	return p
}

// determineFirstToAct will set the decisionStartIndex to the best visible hand who hasn't folded
func (g *Game) determineFirstToAct() {
	bestStrength := math.MinInt64
	var handName string
	bestIndex := 0

	for index, id := range g.playerIDs {
		p := g.idToParticipant[id]
		if p.didFold {
			continue
		}

		hand := make(deck.Hand, 0, len(p.hand))
		for _, card := range p.hand {
			if card.IsBitSet(faceUp) {
				if card.IsBitSet(privateWild) && card.IsWild {
					card = card.Clone()
					card.IsWild = false
				}

				hand = append(hand, card)
			}
		}

		ha := handanalyzer.New(5, hand)
		strength := ha.GetStrength()
		if strength > bestStrength {
			bestStrength = strength
			bestIndex = index
			handName = ha.GetHand().String()
		}
	}

	g.decisionStartIndex = bestIndex
	g.decisionCount = 0

	id := g.playerIDs[bestIndex]
	g.pendingLogs = append(g.pendingLogs, playable.SimpleLogMessage(id, "{} is first to act (%s)", handName))
}

func (g *Game) dealCards(faceDown bool) error {
	for _, pid := range g.playerIDs {
		participant := g.idToParticipant[pid]
		if !participant.didFold {
			card, err := g.deck.Draw()
			if err != nil {
				return err
			}

			if !faceDown {
				card.SetBit(faceUp)
			}

			participant.hand.AddCard(card)
			g.options.Variant.ParticipantReceivedCard(g, participant, card)
		}
	}

	return nil
}

func (g *Game) isGameOver() bool {
	return g.winners != nil
}

// GetDealerID returns the player ID of the dealer
func (g *Game) GetDealerID() int64 {
	return g.playerIDs[g.dealerIndex]
}

func (g *Game) endGame() {
	if g.winners != nil {
		panic("endGame() already called")
	}

	g.round = revealWinner

	winnerList := make([]*participant, 0)
	bestStrength := math.MinInt64

	for _, id := range g.playerIDs {
		p := g.idToParticipant[id]
		if p.didFold {
			continue
		}

		ha := handanalyzer.New(5, p.hand)
		if s := ha.GetStrength(); s > bestStrength {
			winnerList = []*participant{p}
			bestStrength = s
		} else if s == bestStrength {
			winnerList = append(winnerList, p)
		}
	}

	// Calculate winnings per winner
	winAmount := g.pot / len(winnerList)
	g.winners = make(map[*participant]int)

	for _, winner := range winnerList {
		winner.didWin = true
		winner.balance += winAmount
		g.winners[winner] = winAmount
	}

	// Handle remainder
	if remainder := g.pot % len(winnerList); remainder > 0 {
		winnerList[0].balance += remainder
		g.winners[winnerList[0]] += remainder
	}

	g.sendEndOfGameLogMessages()
}

func (g *Game) sendEndOfGameLogMessages() {
	lms := make([]*playable.LogMessage, 0, len(g.idToParticipant))
	for winner, amount := range g.winners {
		hand := winner.getHandAnalyzer().GetHand().String()
		lms = append(lms, playable.SimpleLogMessage(winner.PlayerID, "{} had a %s and won ${%d}", hand, amount))
	}

	for _, playerID := range g.playerIDs {
		p := g.idToParticipant[playerID]
		if p.didWin {
			continue
		}

		if p.didFold {
			lms = append(lms, playable.SimpleLogMessage(p.PlayerID, "{} folded and lost ${%d}", -1*p.balance))
		} else {
			hand := p.getHandAnalyzer().GetHand().String()
			lms = append(lms, playable.SimpleLogMessage(p.PlayerID, "{} had a %s and lost ${%d}", hand, -1*p.balance))
		}
	}

	g.pendingLogs = append(g.pendingLogs, lms...)
}
