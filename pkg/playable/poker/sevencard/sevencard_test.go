package sevencard

import (
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/deck"
	"testing"
)

func TestNewGame(t *testing.T) {
	a := assert.New(t)
	game, err := NewGame("", nil, Options{})
	a.EqualError(err, "ante must be greater than zero")
	a.Nil(game)

	game, err = NewGame("", nil, DefaultOptions())
	a.EqualError(err, "you must have at least two participants")
	a.Nil(game)

	p := make([]int64, 8)
	game, err = NewGame("", p, DefaultOptions())
	a.EqualError(err, "seven-card allows at most 7 participants")
	a.Nil(game)

	game, err = NewGame("", []int64{1, 2}, DefaultOptions())
	a.NoError(err)
	a.NotNil(game)

	game, err = NewGame("", []int64{1, 2, 3, 4, 5, 6, 7}, DefaultOptions())
	a.NoError(err)
	a.NotNil(game)
	a.Equal(7*25, game.pot)
	a.Equal(-25, game.idToParticipant[1].balance)
}

func TestGame_Start(t *testing.T) {
	a := assert.New(t)

	game, err := NewGame("", []int64{1, 2}, DefaultOptions())
	a.NoError(err)
	a.NotNil(game)

	game.deck.Cards = deck.CardsFromString("2c,3c,4c,5c,6c,7c")

	a.NoError(game.Start())
	a.Equal("2c,4c,6c", game.idToParticipant[1].hand.String())
	a.Equal("3c,5c,7c", game.idToParticipant[2].hand.String())
	a.Equal(1, game.decisionStartIndex)

	a.False(game.idToParticipant[1].hand[0].IsBitSet(faceUp))
	a.False(game.idToParticipant[1].hand[1].IsBitSet(faceUp))
	a.True(game.idToParticipant[1].hand[2].IsBitSet(faceUp))

	a.EqualError(game.Start(), "the game has already started")
}

func TestGame_New_notEnoughCards(t *testing.T) {
	a := assert.New(t)

	game, err := NewGame("", []int64{1, 2}, DefaultOptions())
	a.NoError(err)
	a.NotNil(game)

	game.deck.Cards = deck.CardsFromString("2c")

	a.EqualError(game.Start(), "end of deck reached")
}

func TestGame_setFirstToAct(t *testing.T) {
	a := assert.New(t)
	game, _ := NewGame("", []int64{1, 2, 3}, DefaultOptions())
	game.idToParticipant[1].hand = deck.CardsFromString("14c,14c,4c")
	game.idToParticipant[2].hand = deck.CardsFromString("2c,3c,14c")
	game.idToParticipant[3].hand = deck.CardsFromString("14c,14c,14c")

	for _, p := range game.idToParticipant {
		p.hand[2].SetBit(faceUp)
	}

	game.determineFirstToAct()
	a.Equal(1, game.decisionStartIndex)

	// folded players don't count
	game.idToParticipant[2].didFold = true
	game.determineFirstToAct()
	a.Equal(2, game.decisionStartIndex)
}

func TestGame_setFirstToAct_withMoreCards(t *testing.T) {
	a := assert.New(t)
	game, _ := NewGame("", []int64{1, 2, 3}, DefaultOptions())
	game.idToParticipant[1].hand = deck.CardsFromString("14c,14c,4c,5d")
	game.idToParticipant[2].hand = deck.CardsFromString("2c,3c,14c,3c")
	game.idToParticipant[3].hand = deck.CardsFromString("14c,14c,8c,8d")

	for _, p := range game.idToParticipant {
		p.hand[2].SetBit(faceUp)
		p.hand[3].SetBit(faceUp)
	}

	game.determineFirstToAct()
	a.Equal(2, game.decisionStartIndex)
}

func TestGame_turns(t *testing.T) {
	a := assert.New(t)

	game, _ := NewGame("", []int64{1, 2, 3}, DefaultOptions())
	a.NoError(game.Start())
	game.decisionStartIndex = 0

	a.Equal(int64(1), game.getCurrentTurn().PlayerID)
	game.advanceDecisionIfPlayerDidFold() // no movement
	a.Equal(int64(1), game.getCurrentTurn().PlayerID)

	game.advanceDecision()
	a.Equal(int64(2), game.getCurrentTurn().PlayerID)

	game.advanceDecision()
	a.Equal(int64(3), game.getCurrentTurn().PlayerID)

	// reset to test for folds
	game.decisionStartIndex = 0
	game.decisionCount = 0
	a.Equal(int64(1), game.getCurrentTurn().PlayerID) // ensure good state
	game.idToParticipant[1].didFold = true
	game.idToParticipant[2].didFold = true
	game.advanceDecisionIfPlayerDidFold()
	a.Equal(int64(3), game.getCurrentTurn().PlayerID)

	// test what happens if we make a coding error and don't advance beyond a folded player
	game.decisionCount = 0
	a.PanicsWithValue("decision is on a player who folded", func() {
		game.getCurrentTurn()
	})

	// test end of game
	game.round = finalBettingRound + 1
	a.Nil(game.getCurrentTurn(), "no turn if we finished game")
}

func TestGame_happyPath(t *testing.T) {
	a := assert.New(t)

	game, err := NewGame("", []int64{1, 2, 3}, DefaultOptions())
	a.NoError(err)
	a.NotNil(game)

	//                                         1  2  3| 1  2  3| 1  2  3
	game.deck.Cards = deck.CardsFromString("2c,3c,4c,5c,6c,7c,8c,9c,10c")

	// next 3
	game.deck.Cards = append(game.deck.Cards, deck.CardsFromString("2h,9h,4h")...) // p2 with pair of 9s
	game.deck.Cards = append(game.deck.Cards, deck.CardsFromString("2d,8d")...)    // p3 folded
	game.deck.Cards = append(game.deck.Cards, deck.CardsFromString("2s,3s")...)    // 2 has trips showing
	// final
	game.deck.Cards = append(game.deck.Cards, deck.CardsFromString("5s,6s")...)

	a.NoError(game.Start())

	a.Equal(75, game.pot)

	p := createParticipantGetter(game)

	a.Equal(firstBettingRound, game.round)
	a.NoError(game.participantChecks(p(3))) // 3 is first place because they have high-card
	a.NoError(game.participantChecks(p(1)))
	a.NoError(game.participantChecks(p(2)))

	a.Equal(secondBettingRound, game.round)
	a.NoError(game.participantBets(p(2), 25))
	a.NoError(game.participantFolds(p(3)))
	a.NoError(game.participantRaises(p(1), 125)) // $1.25 is the max bet
	a.NoError(game.participantCalls(p(2)))
	a.Equal(325, game.pot)

	a.Equal(thirdBettingRound, game.round)
	a.NoError(game.participantChecks(p(2)))
	a.NoError(game.participantChecks(p(1)))

	a.Equal(fourthBettingRound, game.round)
	a.NoError(game.participantChecks(p(1)))
	a.NoError(game.participantChecks(p(2)))

	a.False(game.isGameOver())
	a.Equal(finalBettingRound, game.round)
	a.NoError(game.participantChecks(p(1)))
	a.NoError(game.participantChecks(p(2)))

	a.Equal(revealWinner, game.round)
	a.EqualError(game.participantChecks(p(1)), "it is not your turn")
	a.EqualError(game.participantChecks(p(2)), "it is not your turn")
	a.True(game.isGameOver())

	a.Equal([]*participant{p(1)}, game.winners)
	a.Equal(175, p(1).balance)
	a.Equal(-150, p(2).balance)
	a.Equal(-25, p(3).balance)

	a.False(p(1).hand[0].IsBitSet(faceUp))
	a.False(p(1).hand[1].IsBitSet(faceUp))
	a.True(p(1).hand[2].IsBitSet(faceUp))
	a.True(p(1).hand[3].IsBitSet(faceUp))
	a.True(p(1).hand[4].IsBitSet(faceUp))
	a.True(p(1).hand[5].IsBitSet(faceUp))
	a.False(p(1).hand[6].IsBitSet(faceUp))
}

func createParticipantGetter(game *Game) func(id int64) *participant {
	return func(id int64) *participant {
		return game.idToParticipant[id]
	}
}

func TestGame_endGame(t *testing.T) {
	a := assert.New(t)
	game, _ := NewGame("", []int64{1, 2, 3}, DefaultOptions())
	a.NoError(game.Start())
	p := createParticipantGetter(game)

	p(1).hand = deck.CardsFromString("2c,3c,4c,5c,6c,7c,8c")
	p(1).didFold = true

	p(2).hand = deck.CardsFromString("10c,10d,10h,9c,9d,8s,7s")

	p(3).hand = deck.CardsFromString("7d,7h,8d,8h,11c,12d,13h")
	game.endGame()

	a.Equal([]*participant{p(2)}, game.winners)
	a.Equal(-25, p(1).balance)
	a.Equal(50, p(2).balance)
	a.Equal(-25, p(3).balance)

	a.PanicsWithValue("endGame() already called", func() {
		game.endGame()
	})

	m := game.pendingLogs
	a.Equal(3, len(m))
	a.Equal("{} had a Full house and won ${50}", m[0].Message)
	a.Equal([]int64{2}, m[0].PlayerIDs)

	a.Equal("{} folded and lost ${25}", m[1].Message)
	a.Equal([]int64{1}, m[1].PlayerIDs)

	a.Equal("{} had a Two pair and lost ${25}", m[2].Message)
	a.Equal([]int64{3}, m[2].PlayerIDs)
}

func TestGame_endGame_withTie(t *testing.T) {
	a := assert.New(t)
	game, _ := NewGame("", []int64{1, 2, 3}, DefaultOptions())
	a.NoError(game.Start())
	p := createParticipantGetter(game)

	p(1).hand = deck.CardsFromString("2c,3c,4c,5c,6c,7c,8c")
	p(1).didFold = true

	p(2).hand = deck.CardsFromString("14c,13c,12c,11d,10d,2s,2h")

	p(3).hand = deck.CardsFromString("14d,13d,12d,11c,10c,3s,3h")
	game.endGame()

	a.Equal([]*participant{p(2), p(3)}, game.winners)
	a.Equal(-25, p(1).balance)
	a.Equal(13, p(2).balance)
	a.Equal(12, p(3).balance)
}

func TestGame_nextRound_panics(t *testing.T) {
	a := assert.New(t)
	game, _ := NewGame("", []int64{1, 2, 3}, DefaultOptions())

	// we didn't call Start(), so this will panic
	a.PanicsWithValue("round 1 is not implemented", func() {
		game.nextRound()
	})

	game, _ = NewGame("", []int64{1, 2, 3}, DefaultOptions())
	a.NoError(game.Start())
	game.deck.Cards = deck.Hand{}

	a.PanicsWithValue("could not deal cards: end of deck reached", func() {
		game.nextRound()
	})
}

func TestGame_determineFirstToAct(t *testing.T) {
	a := assert.New(t)
	game, _ := NewGame("", []int64{1, 2}, DefaultOptions())
	p := createParticipantGetter(game)

	p(1).hand = deck.CardsFromString("2c,3c,5c,5d")
	p(2).hand = deck.CardsFromString("2d,3d,6c,!7d")

	for _, p := range game.idToParticipant {
		for i := 2; i < 4; i++ {
			p.hand[i].SetBit(faceUp)
		}
	}

	game.determineFirstToAct()
	a.Equal(1, game.decisionStartIndex)

	p(2).hand[3].SetBit(privateWild)

	game.determineFirstToAct()
	a.Equal(0, game.decisionStartIndex, "does not use wild to calculate best hand")
}
