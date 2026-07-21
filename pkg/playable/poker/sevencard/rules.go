package sevencard

import (
	"fmt"
	"mondaynightpoker-server/pkg/money"
	"mondaynightpoker-server/pkg/playable"
)

// Rules returns the rules for the current game configuration
func (g *Game) Rules() []playable.RuleSection {
	sections := make([]playable.RuleSection, 0, 4)
	sections = append(sections, []playable.RuleSection{
		{
			Title: "Overview",
			Body:  fmt.Sprintf("Each player is dealt 7 cards (2 face-down, 4 face-up, 1 face-down). Ante: %s. Make the best 5-card poker hand.", money.FormatCents(g.options.Ante)),
		},
		{
			Title: "Betting Rounds",
			Body:  "There are 5 betting rounds. After the initial deal (3 cards), one card is dealt each round. The player with the best visible hand acts first.",
		},
	}...)

	sections = append(sections, g.variantRules()...)

	sections = append(sections, playable.RuleSection{
		Title: "Showdown",
		Body:  "After the final betting round, remaining players reveal their hands. The best 5-card poker hand wins the pot.",
	})

	return sections
}

func (g *Game) variantRules() []playable.RuleSection {
	switch g.options.Variant.(type) {
	case *Baseball:
		return []playable.RuleSection{
			{
				Title: "Baseball Rules",
				Body:  "3s and 9s are wild. A face-up 4 earns an extra card.",
			},
		}
	case *FollowTheQueen:
		return []playable.RuleSection{
			{
				Title: "Follow the Queen Rules",
				Body:  "Queens are always wild. When a Queen is dealt face-up, the next face-up card's rank also becomes wild. A new Queen resets the wild rank.",
			},
		}
	case *LowCardWild:
		return []playable.RuleSection{
			{
				Title: "Low Card Wild Rules",
				Body:  "Your lowest face-down card (and all cards of that rank in your hand) are wild. Your wild rank can change as new cards are dealt.",
			},
		}
	case *HighChicago:
		return []playable.RuleSection{
			{
				Title: "High Chicago Rules",
				Body:  "The pot is split in half. One half goes to the best poker hand, the other half goes to the player with the highest spade in the hole (face-down).",
			},
		}
	case *CouponsAndClippings:
		return []playable.RuleSection{
			{
				Title: "Coupons and Clippings Rules",
				Body:  "BOGO: When a second face-up card of the same rank appears, all cards of that rank become wild (replacing any previous wild rank). Nail Clipping: A face-up 10 refunds your ante from the pot.",
			},
		}
	case *Chiggs:
		return []playable.RuleSection{
			{
				Title: "Chiggs Rules",
				Body:  "All 4s are wild. The 4 of Clubs is the Mushroom: when revealed, your neighbors must play an antidote (another 4) or fold. A face-down Mushroom can be flipped voluntarily before anyone bets.",
			},
		}
	default:
		return nil
	}
}
