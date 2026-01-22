package sevencard

import (
	"testing"

	"mondaynightpoker-server/pkg/deck"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestHighChicago_Name(t *testing.T) {
	opts := DefaultOptions()
	opts.Variant = &HighChicago{}
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2}, opts)
	assert.Equal(t, "High Chicago", game.Name())
}

func TestHighChicago_GetSplitPotWinners(t *testing.T) {
	a := assert.New(t)
	hc := &HighChicago{}

	c := func(s string, isFaceUp bool) *deck.Card {
		card := deck.CardFromString(s)
		if isFaceUp {
			card.SetBit(faceUp)
		}
		return card
	}

	opts := DefaultOptions()
	opts.Variant = hc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	p := createParticipantGetter(game)

	// Setup hands with hole cards (first 2 and last 1) and face-up cards (middle 4)
	// Player 1: Ace of Spades in hole - highest
	p(1).hand = deck.Hand{
		c("14s", false), // Ace of Spades - hole card (highest spade)
		c("2c", false),  // 2 of Clubs - hole card
		c("3c", true),   // face-up
		c("4c", true),   // face-up
		c("5c", true),   // face-up
		c("6c", true),   // face-up
		c("7c", false),  // hole card
	}

	// Player 2: King of Spades in hole
	p(2).hand = deck.Hand{
		c("13s", false), // King of Spades - hole card
		c("2d", false),  // 2 of Diamonds - hole card
		c("3d", true),   // face-up
		c("4d", true),   // face-up
		c("5d", true),   // face-up
		c("6d", true),   // face-up
		c("7d", false),  // hole card
	}

	// Player 3: No spades in hole
	p(3).hand = deck.Hand{
		c("2h", false), // hole card
		c("3h", false), // hole card
		c("4h", true),  // face-up
		c("5h", true),  // face-up
		c("6h", true),  // face-up
		c("7h", true),  // face-up
		c("8h", false), // hole card
	}

	winners, card, desc := hc.GetSplitPotWinners(game)

	a.Len(winners, 1)
	a.Equal(int64(1), winners[0].PlayerID)
	a.Equal(deck.Ace, card.Rank)
	a.Equal(deck.Spades, card.Suit)
	a.Equal("high spade in the hole", desc)
}

func TestHighChicago_GetSplitPotWinners_FaceUpSpadeDoesNotCount(t *testing.T) {
	a := assert.New(t)
	hc := &HighChicago{}

	c := func(s string, isFaceUp bool) *deck.Card {
		card := deck.CardFromString(s)
		if isFaceUp {
			card.SetBit(faceUp)
		}
		return card
	}

	opts := DefaultOptions()
	opts.Variant = hc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2}, opts)
	p := createParticipantGetter(game)

	// Player 1: Ace of Spades face-up (should NOT count), King of Spades in hole
	p(1).hand = deck.Hand{
		c("13s", false), // King of Spades - hole card
		c("2c", false),  // hole card
		c("14s", true),  // Ace of Spades - face-up (doesn't count!)
		c("4c", true),   // face-up
		c("5c", true),   // face-up
		c("6c", true),   // face-up
		c("7c", false),  // hole card
	}

	// Player 2: Queen of Spades in hole
	p(2).hand = deck.Hand{
		c("12s", false), // Queen of Spades - hole card
		c("2d", false),  // hole card
		c("3d", true),   // face-up
		c("4d", true),   // face-up
		c("5d", true),   // face-up
		c("6d", true),   // face-up
		c("7d", false),  // hole card
	}

	winners, card, _ := hc.GetSplitPotWinners(game)

	a.Len(winners, 1)
	a.Equal(int64(1), winners[0].PlayerID)
	a.Equal(deck.King, card.Rank, "King in hole beats Queen in hole, Ace face-up doesn't count")
}

func TestHighChicago_GetSplitPotWinners_NoSpadesInHole(t *testing.T) {
	a := assert.New(t)
	hc := &HighChicago{}

	c := func(s string, isFaceUp bool) *deck.Card {
		card := deck.CardFromString(s)
		if isFaceUp {
			card.SetBit(faceUp)
		}
		return card
	}

	opts := DefaultOptions()
	opts.Variant = hc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2}, opts)
	p := createParticipantGetter(game)

	// Player 1: No spades in hole
	p(1).hand = deck.Hand{
		c("2c", false), // hole card
		c("3c", false), // hole card
		c("4c", true),  // face-up
		c("5c", true),  // face-up
		c("6c", true),  // face-up
		c("7c", true),  // face-up
		c("8c", false), // hole card
	}

	// Player 2: No spades in hole
	p(2).hand = deck.Hand{
		c("2d", false), // hole card
		c("3d", false), // hole card
		c("4d", true),  // face-up
		c("5d", true),  // face-up
		c("6d", true),  // face-up
		c("7d", true),  // face-up
		c("8d", false), // hole card
	}

	winners, card, desc := hc.GetSplitPotWinners(game)

	a.Nil(winners)
	a.Nil(card)
	a.Empty(desc)
}

func TestHighChicago_GetSplitPotWinners_Tie(t *testing.T) {
	a := assert.New(t)
	hc := &HighChicago{}

	c := func(s string, isFaceUp bool) *deck.Card {
		card := deck.CardFromString(s)
		if isFaceUp {
			card.SetBit(faceUp)
		}
		return card
	}

	opts := DefaultOptions()
	opts.Variant = hc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2}, opts)
	p := createParticipantGetter(game)

	// Both players have King of Spades in hole (impossible in real game, but test tie logic)
	p(1).hand = deck.Hand{
		c("13s", false), // King of Spades - hole card
		c("2c", false),  // hole card
		c("3c", true),   // face-up
		c("4c", true),   // face-up
		c("5c", true),   // face-up
		c("6c", true),   // face-up
		c("7c", false),  // hole card
	}

	p(2).hand = deck.Hand{
		c("13s", false), // King of Spades - hole card (tie)
		c("2d", false),  // hole card
		c("3d", true),   // face-up
		c("4d", true),   // face-up
		c("5d", true),   // face-up
		c("6d", true),   // face-up
		c("7d", false),  // hole card
	}

	winners, card, _ := hc.GetSplitPotWinners(game)

	a.Len(winners, 2)
	a.Equal(deck.King, card.Rank)
}

func TestHighChicago_GetSplitPotWinners_FoldedPlayersExcluded(t *testing.T) {
	a := assert.New(t)
	hc := &HighChicago{}

	c := func(s string, isFaceUp bool) *deck.Card {
		card := deck.CardFromString(s)
		if isFaceUp {
			card.SetBit(faceUp)
		}
		return card
	}

	opts := DefaultOptions()
	opts.Variant = hc
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2}, opts)
	p := createParticipantGetter(game)

	// Player 1: Ace of Spades in hole but folded
	p(1).hand = deck.Hand{
		c("14s", false), // Ace of Spades - hole card
		c("2c", false),  // hole card
		c("3c", true),   // face-up
		c("4c", true),   // face-up
		c("5c", true),   // face-up
		c("6c", true),   // face-up
		c("7c", false),  // hole card
	}
	p(1).didFold = true

	// Player 2: King of Spades in hole
	p(2).hand = deck.Hand{
		c("13s", false), // King of Spades - hole card
		c("2d", false),  // hole card
		c("3d", true),   // face-up
		c("4d", true),   // face-up
		c("5d", true),   // face-up
		c("6d", true),   // face-up
		c("7d", false),  // hole card
	}

	winners, card, _ := hc.GetSplitPotWinners(game)

	a.Len(winners, 1)
	a.Equal(int64(2), winners[0].PlayerID, "Player 1 folded, so Player 2 wins")
	a.Equal(deck.King, card.Rank)
}

func TestHighChicago_endGame_SplitsPot(t *testing.T) {
	a := assert.New(t)

	c := func(s string, isFaceUp bool) *deck.Card {
		card := deck.CardFromString(s)
		if isFaceUp {
			card.SetBit(faceUp)
		}
		return card
	}

	opts := DefaultOptions()
	opts.Variant = &HighChicago{}
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2, 3}, opts)
	a.NoError(game.Start())
	p := createParticipantGetter(game)

	// Player 1: Best hand (flush) but no spade in hole
	p(1).hand = deck.Hand{
		c("2c", false), // hole card
		c("3c", false), // hole card
		c("4c", true),  // face-up
		c("5c", true),  // face-up
		c("6c", true),  // face-up
		c("7c", true),  // face-up
		c("8c", false), // hole card - Flush in clubs
	}

	// Player 2: Worse hand but Ace of Spades in hole
	p(2).hand = deck.Hand{
		c("14s", false), // Ace of Spades - hole card
		c("2d", false),  // hole card
		c("3h", true),   // face-up
		c("4d", true),   // face-up
		c("9h", true),   // face-up
		c("10d", true),  // face-up
		c("11d", false), // hole card - high card only
	}

	// Player 3: folded
	p(3).didFold = true

	game.pot = 150 // 3 players x $25 ante + some betting
	game.endGame()

	// Pot is 150, split in half: 75 for hand, 75 for high spade
	a.Equal(75, game.winners[p(1)], "Player 1 wins half pot for best hand")
	a.Equal(75, game.winners[p(2)], "Player 2 wins half pot for high spade in hole")

	// Check balances (ante was 25)
	a.Equal(50, p(1).balance)  // Won 75, paid 25 ante = +50
	a.Equal(50, p(2).balance)  // Won 75, paid 25 ante = +50
	a.Equal(-25, p(3).balance) // Lost ante
}

func TestHighChicago_endGame_SamePlayerWinsBoth(t *testing.T) {
	a := assert.New(t)

	c := func(s string, isFaceUp bool) *deck.Card {
		card := deck.CardFromString(s)
		if isFaceUp {
			card.SetBit(faceUp)
		}
		return card
	}

	opts := DefaultOptions()
	opts.Variant = &HighChicago{}
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2}, opts)
	a.NoError(game.Start())
	p := createParticipantGetter(game)

	// Player 1: Best hand AND Ace of Spades in hole
	p(1).hand = deck.Hand{
		c("14s", false), // Ace of Spades - hole card
		c("14c", false), // Ace of Clubs - hole card
		c("14d", true),  // Ace of Diamonds - face-up
		c("14h", true),  // Ace of Hearts - face-up (Four of a kind!)
		c("2c", true),   // face-up
		c("3c", true),   // face-up
		c("4c", false),  // hole card
	}

	// Player 2: Worse hand, lower spade in hole
	p(2).hand = deck.Hand{
		c("2s", false),  // 2 of Spades - hole card
		c("2d", false),  // hole card
		c("3h", true),   // face-up
		c("4d", true),   // face-up
		c("9h", true),   // face-up
		c("10d", true),  // face-up
		c("11d", false), // hole card
	}

	game.pot = 50 // 2 players x $25 ante
	game.endGame()

	// Player 1 wins everything
	a.Equal(50, game.winners[p(1)], "Player 1 wins entire pot")
	a.Equal(0, game.winners[p(2)], "Player 2 wins nothing")

	a.Equal(25, p(1).balance)  // Won 50, paid 25 ante = +25
	a.Equal(-25, p(2).balance) // Lost ante
}

func TestHighChicago_LogMessages(t *testing.T) {
	a := assert.New(t)

	c := func(s string, isFaceUp bool) *deck.Card {
		card := deck.CardFromString(s)
		if isFaceUp {
			card.SetBit(faceUp)
		}
		return card
	}

	opts := DefaultOptions()
	opts.Variant = &HighChicago{}
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2}, opts)
	a.NoError(game.Start())
	p := createParticipantGetter(game)

	// Player 1: Best hand (pair of aces)
	p(1).hand = deck.Hand{
		c("14c", false), // Ace of Clubs - hole card
		c("14d", false), // Ace of Diamonds - hole card
		c("2c", true),   // face-up
		c("3c", true),   // face-up
		c("4c", true),   // face-up
		c("5c", true),   // face-up
		c("6c", false),  // hole card
	}

	// Player 2: Ace of Spades in hole but worse hand
	p(2).hand = deck.Hand{
		c("14s", false), // Ace of Spades - hole card
		c("2d", false),  // hole card
		c("3h", true),   // face-up
		c("4d", true),   // face-up
		c("9h", true),   // face-up
		c("10d", true),  // face-up
		c("11d", false), // hole card
	}

	game.pot = 50
	game.endGame()

	logs := game.pendingLogs

	// Find the log message for player 2 (split pot winner)
	var foundSpadeLog bool
	for _, log := range logs {
		if len(log.PlayerIDs) == 1 && log.PlayerIDs[0] == 2 {
			if len(log.Cards) == 1 {
				a.Equal(deck.Ace, log.Cards[0].Rank)
				a.Equal(deck.Spades, log.Cards[0].Suit)
				a.Contains(log.Message, "high spade in the hole")
				a.Contains(log.Message, "A♠")
				foundSpadeLog = true
			}
		}
	}
	a.True(foundSpadeLog, "Should have log message for high spade winner with card")
}

func TestHighChicago_NoSpadesInHole_FullPotToHandWinner(t *testing.T) {
	a := assert.New(t)

	c := func(s string, isFaceUp bool) *deck.Card {
		card := deck.CardFromString(s)
		if isFaceUp {
			card.SetBit(faceUp)
		}
		return card
	}

	opts := DefaultOptions()
	opts.Variant = &HighChicago{}
	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2}, opts)
	a.NoError(game.Start())
	p := createParticipantGetter(game)

	// Both players have no spades in hole
	// Player 1 has pair of aces
	p(1).hand = deck.Hand{
		c("14c", false), // Ace of Clubs - hole card (pair of aces = best hand)
		c("14d", false), // Ace of Diamonds - hole card
		c("2c", true),   // face-up
		c("3h", true),   // face-up
		c("4d", true),   // face-up
		c("5h", true),   // face-up
		c("6c", false),  // hole card
	}

	// Player 2 has high card only (no pairs, straights, or flushes)
	p(2).hand = deck.Hand{
		c("2h", false),  // hole card
		c("4d", false),  // hole card
		c("6c", true),   // face-up
		c("8d", true),   // face-up
		c("10h", true),  // face-up
		c("12c", true),  // face-up (Queen)
		c("13d", false), // hole card (King) - high card only
	}

	game.pot = 50
	game.endGame()

	// No one has a spade in hole, so full pot goes to hand winner
	a.Equal(50, game.winners[p(1)], "Player 1 wins full pot for best hand")
	a.Equal(0, game.winners[p(2)], "Player 2 wins nothing")
}
