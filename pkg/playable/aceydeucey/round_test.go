package aceydeucey

import (
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/deck"
	"testing"
)

func TestRound_addCard_standard(t *testing.T) {
	a := assert.New(t)

	r := NewRound()
	response, err := r.addCard(deck.CardFromString("3s"))
	a.Equal(addCardResponseOK, response)
	a.NoError(err)

	response, err = r.addCard(deck.CardFromString("5s"))
	a.Equal(addCardResponseWaitingOnBet, response)
	a.NoError(err)

	a.EqualError(r.nextGame(), "game is not over")
	a.False(r.isRoundOver())
	response, err = r.addCard(deck.CardFromString("4s"))
	a.Equal(addCardResponseWon, response)
	a.NoError(err)

	a.True(r.isRoundOver())

	r.Games[0].MiddleCard = nil
	response, err = r.addCard(deck.CardFromString("2s"))
	a.Equal(addCardResponseLost, response)
	a.NoError(err)

	r.Games[0].MiddleCard = nil
	response, err = r.addCard(deck.CardFromString("6s"))
	a.Equal(addCardResponseLost, response)
	a.NoError(err)

	r.Games[0].MiddleCard = nil
	response, err = r.addCard(deck.CardFromString("3s"))
	a.Equal(addCardResponseLostPost, response)
	a.NoError(err)

	r.Games[0].MiddleCard = nil
	response, err = r.addCard(deck.CardFromString("5d"))
	a.Equal(addCardResponseLostPost, response)
	a.NoError(err)

	response, err = r.addCard(deck.CardFromString("6d"))
	a.Equal(addCardResponseFail, response)
	a.Equal(errorRoundIsOver, err)

	a.Equal(errorRoundIsOver, r.nextGame())
}

func TestRound_addCard_firstCardAce(t *testing.T) {
	a := assert.New(t)
	r := NewRound()

	resp, err := r.addCard(deck.CardFromString("14s"))
	a.NoError(err)
	a.Equal(resp, addCardResponseWaitingOnAce)

	resp, err = r.addCard(deck.CardFromString("12d"))
	a.EqualError(err, "ace has not been decided")
	a.Equal(addCardResponseFail, resp)

	a.NoError(r.setAce(false))
	a.EqualError(r.setAce(false), "ace has already been decided")
	a.True(r.Games[0].FirstCard.IsBitSet(aceStateLow))
	a.False(r.Games[0].FirstCard.IsBitSet(aceStateHigh))

	r.Games[0].FirstCard.UnsetBit(aceStateLow)
	a.NoError(r.setAce(true))
	a.EqualError(r.setAce(true), "ace has already been decided")
	a.False(r.Games[0].FirstCard.IsBitSet(aceStateLow))
	a.True(r.Games[0].FirstCard.IsBitSet(aceStateHigh))

	resp, err = r.addCard(deck.CardFromString("12d"))
	a.NoError(err)
	a.Equal(addCardResponseWaitingOnBet, resp)

	resp, err = r.addCard(deck.CardFromString("13d"))
	a.NoError(err)
	a.Equal(addCardResponseWon, resp)

	r = NewRound()
	resp, err = r.addCard(deck.CardFromString("14s"))
	a.Equal(addCardResponseWaitingOnAce, resp)
	a.NoError(err)
	a.NoError(r.setAce(false))
	resp, err = r.addCard(deck.CardFromString("12d"))
	a.Equal(addCardResponseWaitingOnBet, resp)
	a.NoError(err)

	resp, err = r.addCard(deck.CardFromString("13d"))
	a.NoError(err)
	a.Equal(addCardResponseLost, resp)
}

func TestRound_addCard_freeGame(t *testing.T) {
	r := NewRound()
	assertAddCard(t, r, deck.CardFromString("4s"), addCardResponseOK)
	assertAddCard(t, r, deck.CardFromString("5s"), addCardResponseFreeGame)

	r = NewRound()
	assertAddCard(t, r, deck.CardFromString("5s"), addCardResponseOK)
	assertAddCard(t, r, deck.CardFromString("4s"), addCardResponseFreeGame)

	r = NewRound()
	assertAddCard(t, r, deck.CardFromString("14s"), addCardResponseWaitingOnAce)
	assert.NoError(t, r.setAce(false))
	assertAddCard(t, r, deck.CardFromString("2s"), addCardResponseFreeGame)

	r = NewRound()
	assertAddCard(t, r, deck.CardFromString("14s"), addCardResponseWaitingOnAce)
	assert.NoError(t, r.setAce(true))
	assertAddCard(t, r, deck.CardFromString("2s"), addCardResponseWaitingOnBet)

	r = NewRound()
	assertAddCard(t, r, deck.CardFromString("14s"), addCardResponseWaitingOnAce)
	assert.NoError(t, r.setAce(false))
	assertAddCard(t, r, deck.CardFromString("13s"), addCardResponseWaitingOnBet)

	r = NewRound()
	assertAddCard(t, r, deck.CardFromString("14s"), addCardResponseWaitingOnAce)
	assert.NoError(t, r.setAce(true))
	assertAddCard(t, r, deck.CardFromString("13s"), addCardResponseFreeGame)

	r = NewRound()
	assertAddCard(t, r, deck.CardFromString("13s"), addCardResponseOK)
	assertAddCard(t, r, deck.CardFromString("14s"), addCardResponseFreeGame)
}

func TestRound_addCard_doubleGame(t *testing.T) {
	a := assert.New(t)

	r := NewRound()
	assertAddCard(t, r, deck.CardFromString("8s"), addCardResponseOK)
	assertAddCard(t, r, deck.CardFromString("8d"), addCardResponseDoubleGame)

	a.Equal(2, len(r.Games))
	a.Equal(0, r.activeGameIndex)
	a.Equal(deck.CardFromString("8s").String(), r.Games[0].FirstCard.String())
	a.Nil(r.Games[0].LastCard)
	a.Equal(deck.CardFromString("8d").String(), r.Games[1].FirstCard.String())
	a.Nil(r.Games[1].LastCard)

	assertAddCard(t, r, deck.CardFromString("8c"), addCardResponseDoubleGame)
	a.Equal(3, len(r.Games))
	a.Equal(0, r.activeGameIndex)
	a.Equal(deck.CardFromString("8s").String(), r.Games[0].FirstCard.String())
	a.Nil(r.Games[0].LastCard)
	a.Equal(deck.CardFromString("8d").String(), r.Games[1].FirstCard.String())
	a.Nil(r.Games[1].LastCard)
	a.Equal(deck.CardFromString("8c").String(), r.Games[2].FirstCard.String())
	a.Nil(r.Games[2].LastCard)

	assertAddCard(t, r, deck.CardFromString("9c"), addCardResponseFreeGame)
	assertAddCard(t, r, deck.CardFromString("10c"), addCardResponseFail, "game is over")
	a.False(r.isRoundOver())

	a.NoError(r.nextGame())
	assertAddCard(t, r, deck.CardFromString("10c"), addCardResponseWaitingOnBet)
	assertAddCard(t, r, deck.CardFromString("9c"), addCardResponseWon)
	a.False(r.isRoundOver())

	assertAddCard(t, r, deck.CardFromString("10c"), addCardResponseFail, "game is over")
	a.NoError(r.nextGame())
	assertAddCard(t, r, deck.CardFromString("10c"), addCardResponseWaitingOnBet)
	assertAddCard(t, r, deck.CardFromString("11c"), addCardResponseLost)

	a.True(r.isRoundOver())
}

func TestRound_setAce_failConditions(t *testing.T) {
	a := assert.New(t)
	r := NewRound()

	assertAddCard(t, r, deck.CardFromString("13s"), addCardResponseOK)
	a.EqualError(r.setAce(false), "first card is not an ace")

	r.Games[0].gameOver = true
	a.EqualError(r.setAce(false), "Round is over")
}

func TestRound_roundState(t *testing.T) {
	a := assert.New(t)
	r := NewRound()

	a.Equal(roundStateStart, r.getState())

	r.Games[0].FirstCard = deck.CardFromString("8s")
	a.Equal(roundStateFirstCard, r.getState())

	r.Games[0].FirstCard = deck.CardFromString("14s")
	a.Equal(roundStatePendingAceDecision, r.getState())
	r.Games[0].FirstCard.SetBit(aceStateHigh)
	a.Equal(roundStateFirstCard, r.getState())

	_, _ = r.addCard(deck.CardFromString("13s"))
	a.Equal(roundStateRoundOver, r.getState())

	r = NewRound()
	r.Games[0].FirstCard = deck.CardFromString("5s")
	r.Games[0].LastCard = deck.CardFromString("8s")
	a.Equal(roundStatePendingBet, r.getState())

	r = NewRound()
	assertAddCard(t, r, deck.CardFromString("5s"), addCardResponseOK)
	assertAddCard(t, r, deck.CardFromString("5d"), addCardResponseDoubleGame)
	assertAddCard(t, r, deck.CardFromString("6s"), addCardResponseFreeGame)
	a.Equal(roundStateGameOver, r.getState())
}

func TestRound_getCardsInActiveGame(t *testing.T) {
	a := assert.New(t)
	r := NewRound()

	a.Nil(r.getCardsInActiveGame())

	assertAddCard(t, r, deck.CardFromString("2s"), addCardResponseOK)
	a.Equal("2s", deck.CardsToString(r.getCardsInActiveGame()))

	assertAddCard(t, r, deck.CardFromString("4s"), addCardResponseWaitingOnBet)
	a.Equal("2s,4s", deck.CardsToString(r.getCardsInActiveGame()))

	assertAddCard(t, r, deck.CardFromString("3s"), addCardResponseWon)
	a.Nil(r.getCardsInActiveGame())

	r = NewRound()
	assertAddCard(t, r, deck.CardFromString("2s"), addCardResponseOK)
	assertAddCard(t, r, deck.CardFromString("2c"), addCardResponseDoubleGame)
	assertAddCard(t, r, deck.CardFromString("4d"), addCardResponseWaitingOnBet)
	assertAddCard(t, r, deck.CardFromString("3c"), addCardResponseWon)
	a.Equal("2c", deck.CardsToString(r.getCardsInActiveGame()))
}

func assertAddCard(t *testing.T, r *Round, card *deck.Card, expResp addCardResponse, expErr ...string) {
	t.Helper()

	resp, err := r.addCard(card)
	assert.Equal(t, expResp, resp)

	if len(expErr) == 0 {
		assert.NoError(t, err)
	} else {
		assert.EqualError(t, err, expErr[0])
	}
}
