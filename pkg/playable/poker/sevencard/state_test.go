package sevencard

import (
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/deck"
	"testing"
)

func TestGame_getPlayerStateByPlayerID(t *testing.T) {
	a := assert.New(t)
	game, p := createTestGame()
	_ = game.participantBets(p(1), 25)
	playerState := game.getPlayerStateByPlayerID(99)

	a.Equal("Seven-Card Stud", playerState.GameState.Name)
	a.Nil(playerState.Actions)
	a.Nil(playerState.Participant)

	playerState = game.getPlayerStateByPlayerID(1)
	a.Equal([]Action{}, playerState.Actions)
	a.NotNil(playerState.Participant)

	playerState = game.getPlayerStateByPlayerID(2)
	a.Equal([]Action{ActionFold, ActionCall, ActionRaise}, playerState.Actions)
	a.Equal(0, playerState.Participant.CurrentBet)
	a.False(playerState.Participant.DidFold)
	a.Equal("Three of a kind", playerState.Participant.HandRank)
	a.Equal(int64(2), playerState.Participant.PlayerID)
	a.Equal("13c,13d,13h", playerState.Participant.Hand.String())

	a.Equal(25, playerState.GameState.Participants[0].CurrentBet)
	a.Equal(-50, playerState.GameState.Participants[0].Balance)
	a.Equal(int64(1), playerState.GameState.Participants[0].PlayerID)
	a.Equal(",,14h", playerState.GameState.Participants[0].Hand.String())
	a.Equal("", playerState.GameState.Participants[0].HandRank)
	a.False(playerState.GameState.Participants[0].DidFold)

	a.Equal(0, playerState.GameState.Participants[1].CurrentBet)
	a.Equal(-25, playerState.GameState.Participants[1].Balance)
	a.Equal(int64(2), playerState.GameState.Participants[1].PlayerID)
	a.Equal(",,13h", playerState.GameState.Participants[1].Hand.String())
	a.Equal("", playerState.GameState.Participants[1].HandRank)
	a.False(playerState.GameState.Participants[1].DidFold)

	a.Equal(int64(2), playerState.GameState.CurrentTurn)
	a.Equal(1, int(playerState.GameState.Round))
	a.Equal(100, playerState.GameState.Pot)
	a.Equal(25, playerState.GameState.Ante)
	a.Equal(25, playerState.GameState.CurrentBet)
	a.Equal(125, playerState.GameState.MaxBet)
	a.Nil(playerState.GameState.Winners)

	a.NoError(game.participantFolds(p(2)))
	a.NoError(game.participantFolds(p(3)))
	a.True(game.isGameOver())

	playerState = game.getPlayerStateByPlayerID(2)
	a.Equal([]int64{1}, playerState.GameState.Winners)
	a.Equal("14c,14d,14h", playerState.GameState.Participants[0].Hand.String())
	a.Equal("", playerState.GameState.Participants[1].Hand.String())
	a.Equal("", playerState.GameState.Participants[2].Hand.String())
}

func TestGame_getGameState_withPrivateWilds(t *testing.T) {
	a := assert.New(t)

	game, _ := NewGame(logrus.StandardLogger(), []int64{1, 2}, DefaultOptions())
	a.NoError(game.Start())

	game.round = finalBettingRound
	p := createParticipantGetter(game)
	p(1).hand = deck.CardsFromString("!2c,3c,!2d,!5c,6d,8h,9s")
	for i := 2; i < 6; i++ {
		p(1).hand[i].SetBit(faceUp)
	}
	p(1).hand[2].SetBit(privateWild)

	gs := game.getGameState()
	a.Equal(",,2d,!5c,6d,8h,", gs.Participants[0].Hand.String())

	game.nextRound()
	gs = game.getGameState()
	a.Equal("!2c,3c,!2d,!5c,6d,8h,9s", gs.Participants[0].Hand.String())
}
