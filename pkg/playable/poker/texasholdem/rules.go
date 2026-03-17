package texasholdem

import (
	"fmt"
	"mondaynightpoker-server/pkg/playable"
)

// Rules returns the rules for the current game configuration
func (g *Game) Rules() []playable.RuleSection {
	sections := make([]playable.RuleSection, 0, 6)

	holeCards := 2
	overviewSuffix := ""
	switch g.options.Variant {
	case Pineapple:
		holeCards = 3
		overviewSuffix = " In Pineapple, you must discard one card before the flop."
	case LazyPineapple:
		holeCards = 3
		overviewSuffix = " In Lazy Pineapple, you keep all 3 cards until after the flop, then discard one."
	}

	sections = append(sections, playable.RuleSection{
		Title: "Overview",
		Body:  fmt.Sprintf("Each player is dealt %d hole cards. Five community cards are revealed over multiple rounds. Make the best 5-card hand using any combination of your hole cards and the community cards.%s", holeCards, overviewSuffix),
	})

	blindsBody := fmt.Sprintf("Ante: %s.", formatCents(g.options.Ante))
	if g.options.SmallBlind > 0 || g.options.BigBlind > 0 {
		blindsBody += fmt.Sprintf(" Small blind: %s. Big blind: %s.", formatCents(g.options.SmallBlind), formatCents(g.options.BigBlind))
	}
	sections = append(sections, playable.RuleSection{
		Title: "Blinds & Ante",
		Body:  blindsBody,
	})

	sections = append(sections, playable.RuleSection{
		Title: "Betting Rounds",
		Body:  "There are four betting rounds: pre-flop, after the flop (3 cards), after the turn (4th card), and after the river (5th card). You may check, bet, call, raise, or fold.",
	})

	sections = append(sections, playable.RuleSection{
		Title: "Showdown",
		Body:  "After the final betting round, remaining players reveal their hands. The best 5-card poker hand wins the pot.",
	})

	return sections
}

func formatCents(cents int) string {
	if cents%100 == 0 {
		return fmt.Sprintf("$%d", cents/100)
	}
	return fmt.Sprintf("$%.2f", float64(cents)/100)
}
