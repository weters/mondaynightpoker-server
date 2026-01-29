package guts

import "time"

// dealerAction is an action that "dealer" would take, such as progressing the game
type dealerAction int

const (
	dealerActionShowdown dealerAction = iota
	dealerActionRevealDeckCard
	dealerActionResolveBloodyGuts
	dealerActionNextRound
	dealerActionEndGame
)

type pendingDealerAction struct {
	Action       dealerAction
	ExecuteAfter time.Time
}
