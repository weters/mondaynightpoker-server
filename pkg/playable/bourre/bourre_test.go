package bourre

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"mondaynightpoker-server/pkg/deck"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestNewGame(t *testing.T) {
	g, err := NewGame("", []int64{10, 20}, Options{})
	assert.NoError(t, err)
	assert.NotNil(t, g)

	assert.Equal(t, int64(10), g.idToPlayer[10].PlayerID)
	assert.Equal(t, int64(20), g.idToPlayer[20].PlayerID)
	assert.Equal(t, 0, g.playerOrder[g.idToPlayer[10]])
	assert.Equal(t, 1, g.playerOrder[g.idToPlayer[20]])
}

func Test_newGame(t *testing.T) {
	g, err := newGame([]*Player{NewPlayer(1)}, nil, Options{})
	assert.Nil(t, g)
	assert.EqualError(t, err, "expected 2–8 players, got 1")

	players := make([]*Player, 0)
	for i := 0; i <= playersLimit; i++ {
		players = append(players, NewPlayer(1))
	}

	g, err = newGame(players, nil, Options{})
	assert.Nil(t, g)
	assert.EqualError(t, err, "expected 2–8 players, got 9")

	testPlayers := []*Player{
		NewPlayer(1),
		NewPlayer(1),
		NewPlayer(1),
		NewPlayer(1),
	}
	g, err = newGame(testPlayers, nil, Options{Ante: 50})
	assert.NotNil(t, g)
	assert.NoError(t, err)

	assert.NoError(t, g.Deal())
	assert.Equal(t, 4, len(g.playerOrder))
	for player := range g.playerOrder {
		assert.Equal(t, -50, player.balance)
		assert.Equal(t, 5, len(player.hand))
	}
}

// this will test that cards are played to win
func TestGameLogic(t *testing.T) {
	game, players := setupGame("14S", []string{
		"10H,11H",
		"9H,13S",
		"12H,7H",
		"14C,8S",
		"14H,9S",
		"10S,2S",
		"2H,2D",
	})

	game.roundNo++ // bypass trade-in round

	assert.NoError(t, game.playerDidPlayCard(players[0], players[0].hand[0]))                    // 10 of hearts
	assert.Equal(t, ErrIsNotPlayersTurn, game.playerDidPlayCard(players[0], players[0].hand[0])) // 10 of hearts
	assert.Equal(t, deck.Card{Rank: 10, Suit: deck.Hearts}, *game.winningCardPlayed.card)

	assert.Equal(t, ErrCardNotInPlayersHand, game.playerDidPlayCard(players[1], players[0].hand[0])) // not in hand
	assert.Equal(t, ErrPlayOnSuit, game.playerDidPlayCard(players[1], players[1].hand[1]))           // K of Spades
	assert.NoError(t, game.playerDidPlayCard(players[1], players[1].hand[0]))                        // 9 of Hearts (losing)
	assert.Equal(t, deck.Card{Rank: 10, Suit: deck.Hearts}, *game.winningCardPlayed.card)

	assert.Equal(t, ErrPlayToWinOnSuit, game.playerDidPlayCard(players[2], players[2].hand[1])) // 7 of Hearts
	assert.NoError(t, game.playerDidPlayCard(players[2], players[2].hand[0]))                   // Q of Hearts
	assert.Equal(t, deck.Card{Rank: 12, Suit: deck.Hearts}, *game.winningCardPlayed.card)

	assert.Equal(t, ErrPlayTrump, game.playerDidPlayCard(players[3], players[3].hand[0])) // A of Clubs
	assert.NoError(t, game.playerDidPlayCard(players[3], players[3].hand[1]))             // 8 of Spades (winning)
	assert.Equal(t, deck.Card{Rank: 8, Suit: deck.Spades}, *game.winningCardPlayed.card)

	assert.Equal(t, ErrPlayOnSuit, game.playerDidPlayCard(players[4], players[4].hand[1])) // 9 of Spades
	assert.NoError(t, game.playerDidPlayCard(players[4], players[4].hand[0]))              // A of Hearts
	assert.Equal(t, deck.Card{Rank: 8, Suit: deck.Spades}, *game.winningCardPlayed.card)

	assert.Equal(t, ErrPlayToWinOnTrump, game.playerDidPlayCard(players[5], players[5].hand[1])) // 2 of Spades
	assert.NoError(t, game.playerDidPlayCard(players[5], players[5].hand[0]))                    // 10 of Spades
	assert.Equal(t, deck.Card{Rank: 10, Suit: deck.Spades}, *game.winningCardPlayed.card)

	assert.Equal(t, ErrPlayOnSuit, game.playerDidPlayCard(players[6], players[6].hand[1])) // 2 of Diamonds
	assert.NoError(t, game.playerDidPlayCard(players[6], players[6].hand[0]))              // 2 of Hearts
	assert.Equal(t, deck.Card{Rank: 10, Suit: deck.Spades}, *game.winningCardPlayed.card)
}

func TestProperOrder(t *testing.T) {
	game, players := setupGame("14s", []string{
		"2c,3c,4c,5c,6c",
		"2d,3d,4d,5d,6d",
		"2h,3h,4h,5h,6h",
		"2s,3s,4s,5s,6s",
	})

	game.roundNo++ // bypass trade-in round

	err := game.playerDidPlayCard(players[1], players[1].hand[0])
	assert.Equal(t, ErrIsNotPlayersTurn, err)

	// this won't actually be done
	game.roundNo++
	err = game.playerDidPlayCard(players[1], players[1].hand[0])
	assert.NoError(t, err)

	err = game.playerDidPlayCard(players[1], players[1].hand[0])
	assert.Equal(t, ErrIsNotPlayersTurn, err, "ensure there's no double play")

	assert.NoError(t, game.playerDidPlayCard(players[2], players[2].hand[0]))
	assert.NoError(t, game.playerDidPlayCard(players[3], players[3].hand[0]))
	assert.Equal(t, ErrRoundNotOver, game.nextRound())

	assert.NoError(t, game.playerDidPlayCard(players[0], players[0].hand[0]))

	assert.Error(t, ErrRoundIsOver, game.playerDidPlayCard(players[1], players[1].hand[0]))

	assert.NoError(t, game.nextRound())
	assert.Equal(t, 1, players[3].winCount)
}

func TestGame_PlayerDiscards(t *testing.T) {
	game, players := setupGame("14s", []string{
		"2c,2c,2c,2c,2c", // max 3
		"2c,2c,2c,2c,2c", // max 3
		"2c,2c,2c,2c,2c", // max 3
		"2c,2c,2c,2c,2c", // max 3
		"2c,2c,2c,2c,2c", // max 3
		"2c,2c,2c,2c,2c", // max 4
		"2c,2c,2c,2c,2c", // max 4
	})

	h := func(playerIndex, nCards int) []*deck.Card {
		return players[playerIndex].hand[0:nCards]
	}

	game.roundNo++
	assert.Equal(t, ErrTradeInRoundIsOver, game.playerDidDiscard(players[0], h(0, 3)))

	game.roundNo = 0

	assert.Equal(t, ErrIsNotPlayersTurn, game.playerDidDiscard(players[1], []*deck.Card{}))
	assert.EqualError(t, game.playerDidDiscard(players[0], h(0, 4)), "you cannot draw 4 cards, the max for you is 3")
	assert.NoError(t, game.playerDidDiscard(players[0], h(0, 3)))
	assert.NoError(t, game.playerDidDiscard(players[1], h(1, 2)))
	assert.NoError(t, game.playerDidDiscard(players[2], nil))
	assert.NoError(t, game.playerDidDiscard(players[3], h(3, 3)))
	assert.EqualError(t, game.playerDidDiscard(players[4], h(4, 4)), "you cannot draw 4 cards, the max for you is 3")
	assert.NoError(t, game.playerDidDiscard(players[4], h(4, 3)))
	assert.EqualError(t, game.playerDidDiscard(players[5], h(5, 5)), "you cannot draw 5 cards, the max for you is 4")
	assert.NoError(t, game.playerDidDiscard(players[5], h(5, 4)))
	assert.EqualError(t, game.playerDidDiscard(players[6], h(6, 5)), "you cannot draw 5 cards, the max for you is 4")
	assert.NoError(t, game.playerDidDiscard(players[6], h(6, 4)))
	assert.Equal(t, ErrRoundIsOver, game.playerDidDiscard(players[0], h(0, 3)))

	assert.False(t, players[0].folded)
	assert.False(t, players[1].folded)
	assert.True(t, players[2].folded)
	assert.False(t, players[3].folded)
	assert.False(t, players[4].folded)
	assert.False(t, players[5].folded)
	assert.False(t, players[6].folded)
}

func TestGame_ReplaceDiscard(t *testing.T) {
	game, players := setupGame("14S", []string{
		"2c,3c,4c,5c,6c",
		"2d,3d,4d,5d,6d",
		"2h,3h,4h,5h,6h",
		"2s,3s,4s,5s,6s",
		"7c,8c,9c,10c,11c",
		"7d,8d,9d,10d,11d",
		"7h,8h,9h,10h,11h",
		"7s,8s,9s,10s,11s",
	})

	game.deck.Cards = cardsFromString("12c,13c,14c,12d,13d,14d,12h,13h,14h,12s,13s")

	rand.Seed(0) // consistent test
	assert.Equal(t, ErrRoundNotOver, game.replaceDiscards())
	assert.NoError(t, game.playerDidDiscard(players[0], cardsFromString("2c,3c,4c"))) // gets 12,13,14 of clubs
	assert.NoError(t, game.playerDidDiscard(players[1], cardsFromString("2d,3d,4d"))) // gets 12,13,14 of diamonds
	assert.NoError(t, game.playerDidDiscard(players[2], cardsFromString("2h,3h")))    // gets 12,13 of hearts
	assert.NoError(t, game.playerDidDiscard(players[3], cardsFromString("2s,3s")))    // gets 14 of hearts, 12 of spades
	assert.NoError(t, game.playerDidDiscard(players[4], cardsFromString("7c,8c,9c"))) // gets 12,13 of spades, random, random
	assert.NoError(t, game.playerDidDiscard(players[5], []*deck.Card{}))              // no trade
	assert.NoError(t, game.playerDidDiscard(players[6], cardsFromString("7h")))       // gets random
	assert.NoError(t, game.playerDidDiscard(players[7], nil))                         // bails
	assert.NoError(t, game.replaceDiscards())

	assert.Equal(t, "5c,6c,12c,13c,14c", cardsToString(players[0].hand))
	assert.Equal(t, "5d,6d,12d,13d,14d", cardsToString(players[1].hand))
	assert.Equal(t, "4h,5h,6h,12h,13h", cardsToString(players[2].hand))
	assert.Equal(t, "4s,5s,6s,14h,12s", cardsToString(players[3].hand))
	// now we hit randoms
	assert.Equal(t, "10c,11c,13s,3h,2d", cardsToString(players[4].hand))
	assert.Equal(t, "7d,8d,9d,10d,11d", cardsToString(players[5].hand))
	assert.Equal(t, "8h,9h,10h,11h,2s", cardsToString(players[6].hand))
}

func TestGame_IsGameOver(t *testing.T) {
	game := &Game{}
	game.roundNo = 5
	assert.False(t, game.canGameEnd())

	game.roundNo = 6
	assert.True(t, game.canGameEnd())
}

func TestGame_FullGame_TwoTwoOne(t *testing.T) {
	game, players := setupGame("14s", []string{
		"2c,3c,4c,5c,6c",
		"2d,3d,4d,5d,6d",
		"2h,3h,4h,5h,6h",
	})

	game.ante = 88
	game.pot = 99

	playCard := createPlayCardFunc(t, game, players)

	assert.Equal(t, ErrTradeInRoundInProgress, game.playerDidPlayCard(players[0], players[0].hand[0]))
	assert.NoError(t, game.playerDidDiscard(players[0], []*deck.Card{}))
	assert.NoError(t, game.playerDidDiscard(players[1], []*deck.Card{}))
	assert.NoError(t, game.playerDidDiscard(players[2], []*deck.Card{}))
	assert.NoError(t, game.replaceDiscards())

	// round 1
	playCard(0, 0)
	playCard(1, 0)
	playCard(2, 0)
	assert.NoError(t, game.nextRound())
	// round 2
	playCard(1, 0)
	playCard(2, 0)
	playCard(0, 0)
	assert.NoError(t, game.nextRound())
	// round 3
	playCard(2, 0)
	playCard(0, 0)
	playCard(1, 0)
	assert.NoError(t, game.nextRound())
	// round 4
	playCard(0, 0)
	playCard(1, 0)
	playCard(2, 0)
	assert.NoError(t, game.nextRound())
	// round 5
	playCard(1, 0)
	playCard(2, 0)
	playCard(0, 0)
	assert.Nil(t, game.result)
	assert.NoError(t, game.nextRound())

	res := game.result
	assert.NotNil(t, res)
	assert.Equal(t, 2, len(res.Winners))
	assert.Equal(t, 0, len(res.PaidAnte))
	assert.Equal(t, 0, len(res.PaidPot))
	assert.Equal(t, 0, len(res.Folded))
	assert.Equal(t, []*Player{players[2]}, res.Booted)
	assert.Equal(t, 99, res.OldPot)
	assert.Equal(t, 99, res.NewPot)
	assert.Equal(t, 88, res.Ante)
	assert.Equal(t, 0, res.WinningAmount)

	newGame, err := res.NewGame()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(newGame.playerOrder))
	assert.Equal(t, 1, newGame.playerOrder[players[0]])
	assert.Equal(t, 0, newGame.playerOrder[players[1]])
	assert.Equal(t, 1, len(newGame.foldedPlayers)) // booted
}

func TestGame_FullGame_WithWinner(t *testing.T) {
	game, players := setupGame("14s", []string{
		"2c,3c,4c,5c,13s", // 1 bourré card
		"2d,3d,4d,5d,6d",
		"2s,3s,4s,5s,6s", // 4 bourré cards
	})

	foldedPlayer := &Player{}
	game.foldedPlayers = map[*Player]bool{foldedPlayer: true}

	game.ante = 88
	game.pot = 99

	playCard := createPlayCardFunc(t, game, players)

	game.roundNo++ // bypass the trade-in round

	// round 1
	playCard(0, 0)
	playCard(1, 0)
	playCard(2, 0)
	assert.NoError(t, game.nextRound())
	// round 2
	playCard(1, 0)
	playCard(2, 0)
	playCard(0, 3) // player has to play their bourré card
	assert.NoError(t, game.nextRound())
	// round 3
	playCard(2, 0)
	playCard(0, 0)
	playCard(1, 0)
	assert.NoError(t, game.nextRound())
	// round 4
	playCard(0, 0)
	playCard(1, 0)
	playCard(2, 0)
	assert.NoError(t, game.nextRound())
	// round 5
	playCard(1, 0)
	playCard(2, 0)
	playCard(0, 0)
	assert.NoError(t, game.nextRound())

	res := game.result
	assert.NotNil(t, res)
	assert.Equal(t, []*Player{players[2]}, res.Winners)
	assert.Equal(t, []*Player{players[0]}, res.PaidAnte)
	assert.Equal(t, []*Player{players[1]}, res.PaidPot)
	assert.Equal(t, []*Player{foldedPlayer}, res.Folded)
	assert.Equal(t, 0, len(res.Booted))
	assert.Equal(t, 99, res.OldPot)
	assert.Equal(t, 99+88, res.NewPot)
	assert.Equal(t, 88, res.Ante)
	assert.Equal(t, 99, res.WinningAmount)

	newGame, err := res.NewGame()
	assert.NoError(t, err)
	assert.Equal(t, 3, len(newGame.playerOrder))
	assert.Equal(t, 2, newGame.playerOrder[players[0]])
	assert.Equal(t, 0, newGame.playerOrder[players[1]])
	assert.Equal(t, 1, newGame.playerOrder[players[2]])
	assert.Equal(t, -88, players[0].balance) // owes ante
	assert.Equal(t, -99, players[1].balance) // owes pot
	assert.Equal(t, 99, players[2].balance)
	assert.Equal(t, []*Player{foldedPlayer}, newGame.getFoldedPlayers())
}

func TestGame_FullGame_WithGameEnding(t *testing.T) {
	game, players := setupGame("14s", []string{
		"2c,3c,4c,5c,6c",
		"2d,3d,4d,5d,6d",
	})

	game.ante = 88
	game.pot = 99

	playCard := createPlayCardFunc(t, game, players)

	game.roundNo++ // bypass the trade-in round

	// round 1
	playCard(0, 0)
	playCard(1, 0)
	assert.NoError(t, game.nextRound())
	// round 2
	playCard(1, 0)
	playCard(0, 0)
	assert.NoError(t, game.nextRound())
	// round 3
	playCard(0, 0)
	playCard(1, 0)
	assert.NoError(t, game.nextRound())
	// round 4
	playCard(1, 0)
	playCard(0, 0)
	assert.NoError(t, game.nextRound())
	// round 5
	playCard(0, 0)
	playCard(1, 0)
	assert.NoError(t, game.nextRound())

	res := game.result
	assert.NotNil(t, res)
	assert.Equal(t, []*Player{players[0]}, res.Winners)
	assert.Equal(t, 0, len(res.PaidAnte))
	assert.Equal(t, 0, len(res.PaidPot))
	assert.Equal(t, 0, len(res.Folded))
	assert.Equal(t, []*Player{players[1]}, res.Booted)
	assert.Equal(t, 99, res.OldPot)
	assert.Equal(t, 0, res.NewPot)
	assert.Equal(t, 88, res.Ante)
	assert.Equal(t, 99, res.WinningAmount)
	assert.Equal(t, 99, players[0].balance)

	newGame, err := res.NewGame()
	assert.Equal(t, ErrCannotCreateGame, err)
	assert.Nil(t, newGame)
}

func TestGame_EndGame_TwoPlayer_CleanWipe(t *testing.T) {
	game, players := setupGame("2s", []string{
		"10s,11s,12s,13s,14s",
		"2h,3h,4h,5h,7h",
	})

	game.ante = 25
	game.pot = 50

	players[0].balance = -25
	players[1].balance = -25

	playCard := createPlayCardFunc(t, game, players)
	_ = game.playerDidDiscard(players[0], []*deck.Card{})
	_ = game.playerDidDiscard(players[1], []*deck.Card{})
	_ = game.replaceDiscards()

	playCard(0, 0)
	playCard(1, 0)
	_ = game.nextRound()

	playCard(1, 0)
	playCard(0, 0)
	_ = game.nextRound()

	playCard(0, 0)
	playCard(1, 0)
	_ = game.nextRound()

	playCard(1, 0)
	playCard(0, 0)
	_ = game.nextRound()

	playCard(0, 0)
	playCard(1, 0)
	_ = game.nextRound()

	res := game.result
	assert.NotNil(t, res)
	assert.Equal(t, 50, res.OldPot)
	assert.Equal(t, 50, res.NewPot)

	game2, err := res.NewGame()
	assert.NoError(t, err)
	assert.NotNil(t, game2)

	assert.Equal(t, 50, game2.pot)
	assert.Equal(t, 25, players[0].balance)
	assert.Equal(t, -75, players[1].balance)
}

func TestGame_EndGame_NeedsOne(t *testing.T) {
	game, players := setupGame("2s", []string{
		"10s,11s,12s,13s,14s",
		"2h,3h,4h,5h,7h",
	})

	assert.NoError(t, game.playerDidDiscard(players[0], nil))
	assert.Equal(t, ErrLastPlayerMustPlay, game.playerDidDiscard(players[1], nil))
	assert.NoError(t, game.playerDidDiscard(players[1], []*deck.Card{}))
	assert.NoError(t, game.replaceDiscards())
	assert.True(t, game.canGameEnd())
}

func TestGame_replaceDiscards(t *testing.T) {
	game, players := setupGame("14s", []string{
		"2c,3c,4c,5c,6c",
		"7c,8c,9c,10c,11c",
		"2d,3d,4d,5d,6d",
		"7d,8d,9d,10d,11d",
	})

	game.deck.Cards = cardsFromString("3s,4s,5s,6s,7s")
	assert.NoError(t, game.playerDidDiscard(players[0], cardsFromString("2c,3c")))
	assert.NoError(t, game.playerDidDiscard(players[1], cardsFromString("7c,8c")))
	assert.NoError(t, game.playerDidDiscard(players[2], cardsFromString("2d,3d")))
	assert.NoError(t, game.playerDidDiscard(players[3], cardsFromString("7d,8d")))

	rand.Seed(0)
	assert.NoError(t, game.replaceDiscards())

	assert.Equal(t, "4c,5c,6c,3s,4s", cardsToString(players[0].hand))
	assert.Equal(t, "9c,10c,11c,5s,6s", cardsToString(players[1].hand))
	assert.Equal(t, "4d,5d,6d,7s,7c", cardsToString(players[2].hand))
	// ensure the trump card is included in shuffle
	assert.Equal(t, "9d,10d,11d,14s,2c", cardsToString(players[3].hand))
}

func createPlayCardFunc(t *testing.T, game *Game, players []*Player) func(player, card int) {
	t.Helper()
	return func(player, card int) {
		assert.NoError(t, game.playerDidPlayCard(players[player], players[player].hand[card]))
	}
}

// -----

func cardsFromString(s string) []*deck.Card {
	parts := strings.Split(s, ",")
	cards := make([]*deck.Card, len(parts))
	for i, s := range parts {
		cards[i] = cardFromString(s)
	}

	return cards
}

//noinspection SpellCheckingInspection
var cardRx = regexp.MustCompile(`^(?i)(\d{1,2})([cdhs])$`)

func cardFromString(s string) *deck.Card {
	matches := cardRx.FindStringSubmatch(s)
	if len(matches) == 0 {
		panic(fmt.Sprintf("could not parse card from %s: does not match regex", s))
	}

	rank, err := strconv.Atoi(matches[1])
	if err != nil {
		panic(fmt.Sprintf("could not parse card from %s: %v", s, err))
	}

	var suit deck.Suit
	switch strings.ToUpper(matches[2]) {
	case "C":
		suit = deck.Clubs
	case "D":
		suit = deck.Diamonds
	case "H":
		suit = deck.Hearts
	case "S":
		suit = deck.Spades
	default:
		panic(fmt.Sprintf("could not parse card from %s: invalid suit", s))
	}

	return &deck.Card{
		Rank: rank,
		Suit: suit,
	}
}

func setupGame(trump string, playerHands []string) (*Game, []*Player) {
	trumpCard := cardFromString(trump)

	players := make([]*Player, len(playerHands))
	for i, handStr := range playerHands {
		parts := strings.Split(handStr, ",")
		hand := make([]*deck.Card, len(parts))
		for j, s := range parts {
			hand[j] = cardFromString(s)
		}

		players[i] = &Player{
			hand: hand,
		}
	}

	playerOrder := make(map[*Player]int)
	for i, p := range players {
		playerOrder[p] = i
	}

	return &Game{
		deck:           &deck.Deck{},
		playerOrder:    playerOrder,
		trumpCard:      trumpCard,
		playerDiscards: make(map[*Player][]*deck.Card),
		foldedPlayers:  make(map[*Player]bool),
	}, players
}

func TestGame_IsRoundOver(t *testing.T) {
	game, players := setupGame("14s", []string{"2c,3c,4c,5c,6c", "2d,3d,4d,5d,6d"})
	assert.False(t, game.isRoundOver())
	_ = game.playerDidDiscard(players[0], []*deck.Card{})
	_ = game.playerDidDiscard(players[1], []*deck.Card{})
	assert.True(t, game.isRoundOver())
	assert.NoError(t, game.replaceDiscards())
	assert.False(t, game.isRoundOver())

	assert.NoError(t, game.playerDidPlayCard(players[0], players[0].hand[0]))
	assert.NoError(t, game.playerDidPlayCard(players[1], players[1].hand[0]))
	assert.True(t, game.isRoundOver())
	assert.NoError(t, game.nextRound())
	assert.False(t, game.isRoundOver())
}

func TestGame_Name(t *testing.T) {
	assert.Equal(t, "bourre", (&Game{}).Name())
}

func TestGame_NextRound(t *testing.T) {
	game := &Game{}
	game.roundNo = 6
	assert.Equal(t, ErrRoundNotOver, game.nextRound())
}
