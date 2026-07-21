package guts

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
			Body:  fmt.Sprintf("Each player is dealt %d cards and simultaneously declares In or Out. Players who go In compete for the pot.", g.options.CardCount),
		},
		{
			Title: "Ante & Pot",
			Body:  fmt.Sprintf("Each player antes %s to start. The maximum penalty is capped at %s.", money.FormatCents(g.options.Ante), money.FormatCents(g.options.MaxOwed)),
		},
		{
			Title: "Declaration",
			Body:  "All players secretly decide In or Out. Decisions are revealed simultaneously after everyone has chosen.",
		},
	}

	if g.options.AllowTrades {
		sections = append(sections, playable.RuleSection{
			Title: "Trading",
			Body:  fmt.Sprintf("Players who went In may trade up to %d cards before the showdown.", g.options.CardCount),
		})
	}

	showdownBody := "If multiple players go In, the best hand wins the pot. Losers pay a penalty equal to the pot (capped at %s) into the next round's pot."
	if g.options.BloodyGuts {
		showdownBody += " If only one player goes In, they must beat a hand drawn from the deck. The deck wins ties."
	} else {
		showdownBody += " If only one player goes In, they win the pot automatically."
	}
	sections = append(sections, playable.RuleSection{
		Title: "Showdown",
		Body:  fmt.Sprintf(showdownBody, money.FormatCents(g.options.MaxOwed)),
	})

	sections = append(sections, playable.RuleSection{
		Title: "End of Game",
		Body:  "The game continues with new rounds until a player wins the pot with no penalties owed. If no one goes In, everyone re-antes.",
	})

	return sections
}
