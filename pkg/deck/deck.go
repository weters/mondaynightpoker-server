package deck

import (
	"crypto/sha1" // nolint:gosec
	"encoding/hex"
	"errors"
	"flag"
	"github.com/sirupsen/logrus"
	"math/rand"
	"mondaynightpoker-server/internal/rng"
	"runtime/debug"
)

type deckType int

const (
	deckTypeStandard deckType = iota
	deckTypeFiveSuit
)

// ErrEndOfDeck is an error when Draw() is attempted and there are no more cards
var ErrEndOfDeck = errors.New("end of deck reached")

// Deck represents a playing deck
type Deck struct {
	Cards    []*Card `json:"cards"`
	rng      rng.Generator
	deckType deckType
}

// New returns a new deck of cards.
// Important! this deck is unshuffled. You must call the Shuffle() method to shuffle the cards
func New() *Deck {
	return newOfDeckType(deckTypeStandard)
}

// NewFiveSuit returns a deck with a fifth suit
func NewFiveSuit() *Deck {
	return newOfDeckType(deckTypeFiveSuit)
}

func newOfDeckType(deckType deckType) *Deck {
	d := &Deck{
		rng:      rng.Crypto{},
		deckType: deckType,
	}

	d.buildDeck()
	return d
}

func (d *Deck) buildDeck() {
	suits := []Suit{Clubs, Diamonds, Hearts, Spades}
	if d.deckType != deckTypeStandard {
		suits = []Suit{Clubs, Diamonds, Hearts, Spades, Stars}
	}

	cards := make([]*Card, 0, 52)
	for _, suit := range suits {
		for rank := 2; rank <= 14; rank++ {
			cards = append(cards, &Card{
				Rank: rank,
				Suit: suit,
			})
		}
	}

	d.Cards = cards
}

// Shuffle will shuffle the deck of cards
func (d *Deck) Shuffle() {
	// we always want to shuffle from an unshuffled deck.
	// this check here is to make sure we aren't double building the deck
	if len(d.Cards) != 52 {
		d.buildDeck()
	}

	for j := len(d.Cards) - 1; j > 0; j-- {
		i := d.rng.Intn(j + 1)

		d.Cards[i], d.Cards[j] = d.Cards[j], d.Cards[i]
	}
}

// ShuffleDiscards will replace the existing deck with the cards specified
func (d *Deck) ShuffleDiscards(discards []*Card) {
	cards := make([]*Card, len(discards))
	copy(cards, discards)

	for j := len(cards) - 1; j > 0; j-- {
		i := d.rng.Intn(j + 1)

		cards[i], cards[j] = cards[j], cards[i]
	}

	d.Cards = cards
}

// HashCode returns a SHA1 hash code of the deck.
func (d *Deck) HashCode() string {
	hash := sha1.New() // nolint:gosec
	for _, card := range d.Cards {
		_, _ = hash.Write([]byte(card.String()))
	}

	return hex.EncodeToString(hash.Sum(nil)[:])
}

// Draw will draw the next card
// If there are no more cards, an ErrEndOfDeck is returned along with a nil card.
func (d *Deck) Draw() (*Card, error) {
	if len(d.Cards) <= 0 {
		return nil, ErrEndOfDeck
	}

	card := d.Cards[0]
	d.Cards = d.Cards[1:]

	return card, nil
}

// UndoDraw will put the card back in the beginning of the deck
func (d *Deck) UndoDraw(card *Card) {
	cards := make([]*Card, len(d.Cards)+1)
	cards[0] = card
	copy(cards[1:], d.Cards)
	d.Cards = cards
}

// CanDraw returns true if there are {want} cards left in the deck
func (d *Deck) CanDraw(want int) bool {
	return len(d.Cards) >= want
}

// CardsLeft returns the number of cards left in the deck
func (d *Deck) CardsLeft() int {
	return len(d.Cards)
}

// RemoveCard will remove the specified card from the deck
// Returns true if the card was removed, returns false if the card could not be found
func (d *Deck) RemoveCard(targetCard *Card) bool {
	newDeck := make([]*Card, 0, len(d.Cards))
	for _, card := range d.Cards {
		if !card.Equal(targetCard) {
			newDeck = append(newDeck, card)
		}
	}

	cardWasRemoved := len(newDeck) != len(d.Cards)

	d.Cards = newDeck
	return cardWasRemoved
}

// SetSeed is a TESTING method for setting pseudo random number generator with a seed
// If seed < 0, the default crypto rng will be used
func (d *Deck) SetSeed(seed int64) {
	if seed < 0 {
		d.rng = rng.Crypto{}
		return
	}

	// if this isn't a test, ignore request
	if flag.Lookup("test.v") == nil {
		stack := debug.Stack()
		logrus.WithField("stack", string(stack)).Error("attempted to call SetSeed() from non-test context")

		d.rng = rng.Crypto{}
		return
	}

	d.rng = rand.New(rand.NewSource(seed)) // nolint:gosec
}
