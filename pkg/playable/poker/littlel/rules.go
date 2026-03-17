package littlel

import (
	"fmt"
	"mondaynightpoker-server/pkg/playable"
)

// Rules returns the rules for the current game configuration
func (g *Game) Rules() []playable.RuleSection {
	return []playable.RuleSection{
		{
			Title: "Overview",
			Body:  fmt.Sprintf("Each player is dealt %d cards plus 3 community cards are placed face-down. Ante: %s.", g.options.InitialDeal, formatCents(g.options.Ante)),
		},
		{
			Title: "Trade-In",
			Body:  fmt.Sprintf("Before betting begins, each player may trade cards. Allowed trade counts: %s.", g.tradeIns),
		},
		{
			Title: "Betting Rounds",
			Body:  "After the trade-in, there are 3 betting rounds. One community card is revealed before each round. Betting is pot-limit.",
		},
		{
			Title: "Community Cards",
			Body:  "The 3 community cards are arranged in an L shape. You may use either the left column pair or the bottom row pair. You cannot use all three cards, and you cannot use the diagonal pair.",
		},
		{
			Title: "Showdown",
			Body:  "After the final betting round, remaining players reveal their hands. The best 5-card hand using your cards and community cards wins.",
		},
	}
}

func formatCents(cents int) string {
	if cents%100 == 0 {
		return fmt.Sprintf("$%d", cents/100)
	}
	return fmt.Sprintf("$%.2f", float64(cents)/100)
}
