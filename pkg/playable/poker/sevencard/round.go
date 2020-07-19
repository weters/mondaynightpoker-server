package sevencard

type round int

const (
	beforeDeal         round = iota // no cards
	firstBettingRound               // 2D, 1U
	secondBettingRound              // 2D, 2U
	thirdBettingRound               // 2D, 3U
	fourthBettingRound              // 2D, 4U
	finalBettingRound               // 2D, 4U, 1D
	revealWinner
)
