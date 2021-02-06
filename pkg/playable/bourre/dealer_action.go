package bourre

import "time"

// dealerAction is an action that "dealer" would take, such as progressing the game to the next round
type dealerAction int

const (
	dealerActionNextRound dealerAction = iota
	dealerActionReplaceDiscards
	dealerActionClearGame
)

type pendingDealerAction struct {
	Action       dealerAction
	ExecuteAfter time.Time
}
