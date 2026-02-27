package main

import (
	"encoding/json"
	"fmt"
)

// GameState is a unified representation of valid actions across game types.
type GameState struct {
	GameName     string
	ValidActions []ValidAction
	Hand         []CardInfo
	MinBet       int
	MaxBet       int
	Balance      int // player's remaining balance (for capping bets)
	CurrentTurn  int64
	Pot          int
	Community    []CardInfo
}

// ValidAction represents a single action a bot can take.
type ValidAction struct {
	Action      string
	Name        string
	Cards       []CardInfo // for card-based actions (playCard, discard, trade)
	NeedsAmount bool       // whether this action requires a bet amount
}

// CardInfo is a simplified card representation for display and action selection.
type CardInfo struct {
	Rank int    `json:"rank"`
	Suit string `json:"suit"`
}

func (c CardInfo) String() string {
	rankStr := map[int]string{
		14: "A", 13: "K", 12: "Q", 11: "J", 10: "10",
		9: "9", 8: "8", 7: "7", 6: "6", 5: "5",
		4: "4", 3: "3", 2: "2",
	}
	suitStr := map[string]string{
		"hearts": "\u2665", "diamonds": "\u2666",
		"clubs": "\u2663", "spades": "\u2660",
	}

	r := rankStr[c.Rank]
	if r == "" {
		r = fmt.Sprintf("%d", c.Rank)
	}
	s := suitStr[c.Suit]
	if s == "" {
		s = c.Suit
	}
	return r + s
}

// DeckString returns the card in deck.CardFromString format, e.g. "14h" for Ace of hearts.
func (c CardInfo) DeckString() string {
	suitChar := map[string]string{
		"hearts": "h", "diamonds": "d",
		"clubs": "c", "spades": "s", "stars": "t",
	}
	s := suitChar[c.Suit]
	if s == "" {
		s = c.Suit[:1]
	}
	return fmt.Sprintf("%d%s", c.Rank, s)
}

// ParseGameState parses the game response data based on game name and extracts valid actions.
func ParseGameState(gameName string, rawData json.RawMessage, playerID int64) (*GameState, error) {
	switch gameName {
	case gameTexasHoldEm, gameTexasHoldEmPLO:
		return parseTexasHoldEm(rawData, gameName)
	case gameSevenCard:
		return parseSevenCard(rawData, gameName)
	case gameLittleL:
		return parseLittleL(rawData, gameName)
	case gameBourre:
		return parseBourre(rawData, playerID)
	case gameGuts:
		return parseGuts(rawData)
	case gamePassThePoop:
		return parsePassThePoop(rawData)
	case gameAceyDeucey:
		return parseAceyDeucey(rawData)
	default:
		return &GameState{GameName: gameName}, nil
	}
}

// Poker games (Texas Hold'em, Seven Card, Little L) share similar action patterns.

type pokerAction struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type rawCard struct {
	Rank int    `json:"rank"`
	Suit string `json:"suit"`
}

func toCardInfo(cards []rawCard) []CardInfo {
	out := make([]CardInfo, len(cards))
	for i, c := range cards {
		out[i] = CardInfo(c)
	}
	return out
}

func parseTexasHoldEm(data json.RawMessage, gameName string) (*GameState, error) {
	var raw struct {
		Actions    []pokerAction   `json:"actions"`
		GameState  json.RawMessage `json:"gameState"`
		PokerState *struct {
			MinBet    int             `json:"minBet"`
			MaxBet    int             `json:"maxBet"`
			Community []rawCard       `json:"community"`
			Pots      json.RawMessage `json:"pots"`
		} `json:"pokerState"`
		Participant *struct {
			Hand    []rawCard `json:"hand"`
			Balance int       `json:"balance"`
			Bet     int       `json:"currentBet"`
		} `json:"participant"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	gs := &GameState{GameName: gameName}

	if raw.Participant != nil {
		gs.Hand = toCardInfo(raw.Participant.Hand)
		gs.Balance = raw.Participant.Balance + raw.Participant.Bet
	}
	if raw.PokerState != nil {
		gs.MinBet = raw.PokerState.MinBet
		gs.MaxBet = raw.PokerState.MaxBet
		gs.Community = toCardInfo(raw.PokerState.Community)
	}

	for _, a := range raw.Actions {
		gs.ValidActions = append(gs.ValidActions, ValidAction{Action: a.ID, Name: a.Name})
	}

	return gs, nil
}

func parseSevenCard(data json.RawMessage, gameName string) (*GameState, error) {
	return parseSimplePoker(data, gameName)
}

func parseLittleL(data json.RawMessage, gameName string) (*GameState, error) {
	return parseSimplePoker(data, gameName)
}

func parseSimplePoker(data json.RawMessage, gameName string) (*GameState, error) {
	var raw struct {
		Actions    []pokerAction `json:"actions"`
		PokerState *struct {
			MinBet int `json:"minBet"`
			MaxBet int `json:"maxBet"`
		} `json:"pokerState"`
		Participant *struct {
			Hand       []rawCard `json:"hand"`
			Balance    int       `json:"balance"`
			CurrentBet int       `json:"currentBet"`
		} `json:"participant"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	gs := &GameState{GameName: gameName}
	if raw.Participant != nil {
		gs.Hand = toCardInfo(raw.Participant.Hand)
		gs.Balance = raw.Participant.Balance + raw.Participant.CurrentBet
	}
	if raw.PokerState != nil {
		gs.MinBet = raw.PokerState.MinBet
		gs.MaxBet = raw.PokerState.MaxBet
	}
	for _, a := range raw.Actions {
		gs.ValidActions = append(gs.ValidActions, ValidAction{Action: a.ID, Name: a.Name})
	}
	return gs, nil
}

func parseBourre(data json.RawMessage, playerID int64) (*GameState, error) {
	var raw struct {
		GameState *struct {
			CurrentTurn int64 `json:"currentTurn"`
			Round       int   `json:"round"`
			Pot         int   `json:"pot"`
		} `json:"gameState"`
		Hand       []rawCard `json:"hand"`
		ValidMoves []rawCard `json:"validMoves"`
		MaxDraw    int       `json:"maxDraw"`
		Folded     bool      `json:"folded"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	gs := &GameState{GameName: gameBourre}
	gs.Hand = toCardInfo(raw.Hand)

	if raw.GameState != nil {
		gs.CurrentTurn = raw.GameState.CurrentTurn
		gs.Pot = raw.GameState.Pot

		if raw.GameState.CurrentTurn == playerID && !raw.Folded {
			if raw.GameState.Round == 0 {
				// Discard phase: can discard up to maxDraw cards
				gs.ValidActions = append(gs.ValidActions, ValidAction{
					Action: "discard",
					Name:   fmt.Sprintf("Discard (max %d)", raw.MaxDraw),
				})
			} else {
				// Play phase: play a valid card
				for _, c := range raw.ValidMoves {
					ci := CardInfo(c)
					gs.ValidActions = append(gs.ValidActions, ValidAction{
						Action: "playCard",
						Name:   fmt.Sprintf("Play %s", ci),
						Cards:  []CardInfo{ci},
					})
				}
			}
		}
	}

	return gs, nil
}

func parseGuts(data json.RawMessage) (*GameState, error) {
	var raw struct {
		GameState *struct {
			Pot   int    `json:"pot"`
			Phase string `json:"phase"`
		} `json:"gameState"`
		Hand      []rawCard `json:"hand"`
		CanDecide bool      `json:"canDecide"`
		CanTrade  bool      `json:"canTrade"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	gs := &GameState{GameName: gameGuts}
	gs.Hand = toCardInfo(raw.Hand)

	if raw.GameState != nil {
		gs.Pot = raw.GameState.Pot
	}

	if raw.CanDecide {
		gs.ValidActions = append(gs.ValidActions,
			ValidAction{Action: "decide", Name: "Go In"},
			ValidAction{Action: "decide-out", Name: "Go Out"},
		)
	}
	if raw.CanTrade {
		gs.ValidActions = append(gs.ValidActions, ValidAction{
			Action: "trade",
			Name:   "Trade Cards",
		})
	}

	return gs, nil
}

func parsePassThePoop(data json.RawMessage) (*GameState, error) {
	var raw struct {
		GameState *struct {
			CurrentTurn int64 `json:"currentTurn"`
			Pot         int   `json:"pot"`
		} `json:"gameState"`
		Card             *rawCard `json:"card"`
		AvailableActions []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"availableActions"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	gs := &GameState{GameName: gamePassThePoop}
	if raw.Card != nil {
		gs.Hand = []CardInfo{{Rank: raw.Card.Rank, Suit: raw.Card.Suit}}
	}
	if raw.GameState != nil {
		gs.CurrentTurn = raw.GameState.CurrentTurn
		gs.Pot = raw.GameState.Pot
	}
	for _, a := range raw.AvailableActions {
		gs.ValidActions = append(gs.ValidActions, ValidAction{
			Action: fmt.Sprintf("%d", a.ID),
			Name:   a.Name,
		})
	}
	return gs, nil
}

const aceyDeuceyActionBet = 3 // ActionBet in acey deucey

func parseAceyDeucey(data json.RawMessage) (*GameState, error) {
	var raw struct {
		GameState *struct {
			CurrentTurn int64 `json:"currentTurn"`
			MaxBet      int   `json:"maxBet"`
		} `json:"gameState"`
		Actions []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"actions"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	gs := &GameState{GameName: gameAceyDeucey}
	if raw.GameState != nil {
		gs.CurrentTurn = raw.GameState.CurrentTurn
		gs.MinBet = 25 // acey deucey minimum bet is always $25
		gs.MaxBet = raw.GameState.MaxBet
	}
	for _, a := range raw.Actions {
		gs.ValidActions = append(gs.ValidActions, ValidAction{
			Action:      fmt.Sprintf("%d", a.ID),
			Name:        a.Name,
			NeedsAmount: a.ID == aceyDeuceyActionBet,
		})
	}
	return gs, nil
}
