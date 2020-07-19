package sevencard

type round int

const (
	beforeDeal         round = iota // no cards
	firstBettingRound               // nolint 2D, 1U
	secondBettingRound              // nolint 2D, 2U
	thirdBettingRound               // nolint 2D, 3U
	fourthBettingRound              // nolint 2D, 4U
	finalBettingRound               // nolint 2D, 4U, 1D
)
