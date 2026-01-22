package sevencard

// bitmask for deck.Card
const (
	faceUp = 1 << iota
	privateWild
	isMushroom   // card is 4♣ (mushroom)
	isAntidote   // card is 4♠/4♦/4♥ (antidote)
	wasDiscarded // card was discarded (antidote used or mushroom flipped)
)
