package aceydeucey

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
			Body:  fmt.Sprintf("Players take turns betting whether a third card will fall between two dealt cards. Ante: %s.", money.FormatCents(g.options.Ante)),
		},
		{
			Title: "Gameplay",
			Body:  "Two cards are dealt face-up. If the first card is an Ace, you choose high or low. Then you bet whether the third card's value falls between the first two cards.",
		},
		{
			Title: "Betting",
			Body:  "You may bet any amount up to the pot (half the pot if not everyone has had a turn yet). If the third card is between the two cards, you win your bet from the pot. If not, you pay your bet into the pot. If the third card matches either card, you pay double.",
		},
	}

	switch g.options.GameType {
	case GameTypeContinuousShoe:
		sections = append(sections, playable.RuleSection{
			Title: "Continuous Shoe",
			Body:  "The deck is reshuffled before each player's turn.",
		})
	case GameTypeChaos:
		sections = append(sections, playable.RuleSection{
			Title: "Chaos Mode",
			Body:  "All cards are dealt from a continuous shoe with chaotic rules.",
		})
	}

	if g.options.AllowPass {
		sections = append(sections, playable.RuleSection{
			Title: "Passing",
			Body:  "You may pass your turn without betting.",
		})
	}

	sections = append(sections, playable.RuleSection{
		Title: "End of Game",
		Body:  "The game ends when the pot is empty.",
	})

	return sections
}
