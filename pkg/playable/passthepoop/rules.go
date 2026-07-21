package passthepoop

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
			Body:  fmt.Sprintf("Each player is dealt 1 card. You start with %d lives. Ante: %s. The player with the lowest card loses a life each round.", g.options.Lives, money.FormatCents(g.options.Ante)),
		},
		{
			Title: "Your Turn",
			Body:  "On your turn, you may Stay with your card or Trade with the next player. The last player (dealer) may Stay or draw from the deck instead.",
		},
		{
			Title: "Kings",
			Body:  "If you have a King, you cannot trade it away. If someone tries to trade with you and you have a King, you flip it to block the trade.",
		},
	}

	if g.options.AllowBlocks {
		sections = append(sections, playable.RuleSection{
			Title: "Blocks",
			Body:  "Each player has one block they can use to reject an incoming trade, even without a King.",
		})
	}

	editionName := g.options.Edition.Name()
	switch g.options.Edition.(type) {
	case *PairsEdition:
		sections = append(sections, playable.RuleSection{
			Title: fmt.Sprintf("%s Edition", editionName),
			Body:  "If trips or better appear on the board at the end of a round, everyone else loses all their lives.",
		})
	case *DiarrheaEdition:
		sections = append(sections, playable.RuleSection{
			Title: fmt.Sprintf("%s Edition", editionName),
			Body:  "If a player receives a card of a rank they have already held this game, they lose an extra life. If multiple players tie for lowest, they all lose all their lives, and the remaining players continue playing for the next lowest card.",
		})
	}

	sections = append(sections, playable.RuleSection{
		Title: "End of Game",
		Body:  "The last player with lives remaining wins the pot.",
	})

	return sections
}
