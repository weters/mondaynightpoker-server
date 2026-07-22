package bourre

import (
	"fmt"
	"mondaynightpoker-server/pkg/money"
	"mondaynightpoker-server/pkg/playable"
)

// Rules returns the rules for the current game configuration
func (g *Game) Rules() []playable.RuleSection {
	sections := []playable.RuleSection{
		{
			Title: "Overview",
			Body:  "Each player is dealt 5 cards. A trump card is revealed. Players take turns playing cards in 5 trick rounds, trying to win the most tricks.",
		},
		{
			Title: "Ante",
			Body:  fmt.Sprintf("Each player antes %s to start.", money.FormatCents(g.options.Ante)),
		},
		{
			Title: "Trade-In Round",
			Body:  "Before tricks begin, each player may discard and draw new cards, or fold. The last remaining player cannot fold.",
		},
		{
			Title: "Playing Tricks",
			Body:  "Players must follow the lead suit if possible, and must play to win if they can. If they cannot follow suit, they must play a trump card if they have one.",
		},
		{
			Title: "Winning",
			Body:  "The player who wins the most tricks wins the pot. If there is a tie or if any player wins zero tricks, the game continues with a new pot funded by the losers.",
		},
	}

	if g.options.FiveSuit {
		sections = append(sections, playable.RuleSection{
			Title: "Five Suit",
			Body:  "This game uses a five-suit deck, allowing up to 10 players.",
		})
	}

	return sections
}
