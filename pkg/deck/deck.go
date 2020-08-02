package deck

import (
	"crypto/sha1" // nolint:gosec
	"encoding/hex"
	"errors"
	"math/rand"
	"time"
)

// ErrEndOfDeck is an error when Draw() is attempted and there are no more cards
var ErrEndOfDeck = errors.New("end of deck reached")

// Deck represents a playing deck
type Deck struct {
	Cards []*Card `json:"cards"`
	seed  int64
	rng   *rand.Rand
}

// New returns a new deck of cards.
// Important! this deck is unshuffled. You must call the Shuffle() method to shuffle the cards
func New() *Deck {
	d := &Deck{
		seed: -1,
	}

	d.buildDeck()
	return d
}

// SetSeed will set the seed
// This should only be used by tests. Setting the seed is normally handled when you call Shuffle()
func (d *Deck) SetSeed(seed int64) {
	d.seed = seed
	d.rng = rand.New(rand.NewSource(seed))
}

func (d *Deck) buildDeck() {
	cards := make([]*Card, 0, 52)
	for _, suit := range []Suit{Clubs, Diamonds, Hearts, Spades} {
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
// You can manually specify the seed, or you can leave it as 0. This method returns the seed used.
func (d *Deck) Shuffle(seed int64) {
	if seed < 0 {
		panic("seed cannot be < 0")
	}

	// we always want to shuffle from an unshuffled deck.
	// this check here is to make sure we aren't double building the deck
	if len(d.Cards) != 52 || d.seed != -1 {
		d.buildDeck()
	}

	if seed == 0 {
		seed = time.Now().UnixNano()
	}

	d.SetSeed(seed)

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

// GetSeed returns the seed used to shuffle the deck
func (d *Deck) GetSeed() int64 {
	return d.seed
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

// CanDraw returns true if there are {want} cards left in the deck
func (d *Deck) CanDraw(want int) bool {
	return len(d.Cards) >= want
}

// CardsLeft returns the number of cards left in the deck
func (d *Deck) CardsLeft() int {
	return len(d.Cards)
}
