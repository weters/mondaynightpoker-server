package mcpserver

import (
	"fmt"
	"time"

	"mondaynightpoker-server/pkg/deck"
	"mondaynightpoker-server/pkg/model"
	"mondaynightpoker-server/pkg/money"
	"mondaynightpoker-server/pkg/playable/gamelog"
)

// formatRate renders a 0..1 ratio as a percentage string for display, matching the
// discipline the money fields follow: emit a preformatted string alongside the raw
// value so a caller never has to decide how to render it.
func formatRate(rate float64) string {
	return fmt.Sprintf("%.1f%%", rate*100)
}

// RateDTO is a computed ratio paired with a display string.
//
// It is a pointer field everywhere it appears so an undefined rate is absent from
// the JSON rather than reported as zero. A player who has never reached a showdown
// has no showdown win rate, and "0.0%" would read as though they always lose.
type RateDTO struct {
	Value   float64 `json:"value" jsonschema:"the ratio, between 0 and 1"`
	Display string  `json:"display" jsonschema:"the ratio preformatted as a percentage; show this to the user rather than converting value yourself"`
}

// newRate builds a RateDTO, returning nil when the underlying statistic is
// undefined so the field is omitted.
func newRate(value float64, ok bool) *RateDTO {
	if !ok {
		return nil
	}

	return &RateDTO{Value: value, Display: formatRate(value)}
}

// CardDTO is a single card.
type CardDTO struct {
	Rank    int    `json:"rank" jsonschema:"the card's rank, 2-14 where 11-14 are jack, queen, king, ace"`
	Suit    string `json:"suit" jsonschema:"the card's suit"`
	IsWild  bool   `json:"isWild,omitempty" jsonschema:"whether the card is wild"`
	Display string `json:"display,omitempty" jsonschema:"the card preformatted for display; show this to the user. Omitted for a card whose suit is not recognized, which a hidden or placeholder card in an older log can be"`
}

// knownSuits are the suits deck.Card.String can render.
var knownSuits = map[deck.Suit]bool{
	deck.Hearts:   true,
	deck.Clubs:    true,
	deck.Diamonds: true,
	deck.Spades:   true,
	deck.Stars:    true,
}

// cardDisplay renders a card for display, returning an empty string rather than
// letting deck.Card.String panic.
//
// These cards come out of persisted jsonb rather than a live deck, so nothing
// guarantees the suit is one the renderer knows: a hidden hand, a placeholder, or
// a log written by an older version can all decode into a card String would panic
// on. Losing one display string is a fair trade for a tool that cannot be crashed
// by a row in the database.
func cardDisplay(c *deck.Card) string {
	if !knownSuits[c.Suit] {
		return ""
	}

	return c.String()
}

// fromCards maps deck cards to DTOs, skipping nils. A nil card is how the game
// packages represent a card that was never dealt or is not visible.
func fromCards(cards []*deck.Card) []CardDTO {
	out := make([]CardDTO, 0, len(cards))
	for _, c := range cards {
		if c == nil {
			continue
		}

		out = append(out, CardDTO{
			Rank:    c.Rank,
			Suit:    string(c.Suit),
			IsWild:  c.IsWild,
			Display: cardDisplay(c),
		})
	}

	return out
}

// HandActionDTO is one normalized action within a hand.
type HandActionDTO struct {
	Sequence      int       `json:"sequence" jsonschema:"the action's position in the hand, starting at 0"`
	Street        string    `json:"street,omitempty" jsonschema:"the phase of the hand the action occurred in; the vocabulary is game-specific and not comparable across game types"`
	PlayerID      int64     `json:"playerId" jsonschema:"the acting player's id"`
	DisplayName   string    `json:"displayName,omitempty" jsonschema:"the acting player's display name"`
	Kind          string    `json:"kind" jsonschema:"the normalized action: fold, check, call, bet, raise, discard, trade, stay-in, drop-out, play-card, or pass"`
	AmountCents   int       `json:"amountCents,omitempty" jsonschema:"the amount wagered by this action, in cents"`
	AmountDisplay string    `json:"amountDisplay,omitempty" jsonschema:"the wagered amount preformatted in dollars; show this rather than converting amountCents yourself"`
	Cards         []CardDTO `json:"cards,omitempty" jsonschema:"cards involved in the action"`
	AllIn         bool      `json:"allIn,omitempty" jsonschema:"whether the action put the player all in"`
}

// HandParticipantDTO is one player's involvement in a hand.
type HandParticipantDTO struct {
	PlayerID    int64  `json:"playerId" jsonschema:"the player's id"`
	DisplayName string `json:"displayName,omitempty" jsonschema:"the player's display name"`

	StartingCards []CardDTO `json:"startingCards,omitempty" jsonschema:"the cards the player started the hand with, when the game records them"`
	FinalCards    []CardDTO `json:"finalCards,omitempty" jsonschema:"the cards the player finished the hand with, when the game records them"`

	VoluntarilyPlayed bool `json:"voluntarilyPlayed" jsonschema:"whether the player chose to commit beyond any forced ante or blind; this is the cross-game equivalent of VPIP"`
	Folded            bool `json:"folded" jsonschema:"whether the player folded or dropped out"`
	WentToShowdown    bool `json:"wentToShowdown" jsonschema:"whether the player was still contesting the pot when the hand resolved"`
	Won               bool `json:"won" jsonschema:"whether the player won the hand, including hands won because everyone else folded"`

	AmountWageredCents   int    `json:"amountWageredCents" jsonschema:"what the player put in through their own actions, in cents, excluding the forced ante"`
	AmountWageredDisplay string `json:"amountWageredDisplay" jsonschema:"the wagered total preformatted in dollars; show this rather than converting amountWageredCents yourself"`

	NetCents   *int    `json:"netCents,omitempty" jsonschema:"the player's net result for the hand from the ledger, in cents; absent when the hand has no ledger entry for this player"`
	NetDisplay *string `json:"netDisplay,omitempty" jsonschema:"the net result preformatted in dollars; show this rather than converting netCents yourself"`
}

// HandDTO is a normalized, replayable summary of one completed game.
type HandDTO struct {
	GameID         int64     `json:"gameId" jsonschema:"the game's numeric id"`
	TableUUID      string    `json:"tableUuid" jsonschema:"the uuid of the table the game was played at"`
	GameType       string    `json:"gameType" jsonschema:"the game-type identifier the log was parsed as"`
	StoredGameType string    `json:"storedGameType" jsonschema:"the game's full display name as recorded on the game row, including the options it was created with"`
	Variant        string    `json:"variant,omitempty" jsonschema:"the sub-variant within the game type, when the game has one"`
	Created        time.Time `json:"created" jsonschema:"when the game was created"`

	AnteCents   int    `json:"anteCents" jsonschema:"the ante, in cents; zero when the game's log does not record one"`
	AnteDisplay string `json:"anteDisplay" jsonschema:"the ante preformatted in dollars; show this rather than converting anteCents yourself"`
	PotCents    int    `json:"potCents" jsonschema:"the pot, in cents"`
	PotDisplay  string `json:"potDisplay" jsonschema:"the pot preformatted in dollars; show this rather than converting potCents yourself"`

	Rounds int       `json:"rounds" jsonschema:"the number of rounds or hands the game ran; games that resolve in a single pass report 1"`
	Board  []CardDTO `json:"board,omitempty" jsonschema:"shared cards, when the game has any: community cards in the poker variants, the trump card in Bourre"`

	Participants []HandParticipantDTO `json:"participants" jsonschema:"each player's involvement in the hand"`
	Actions      []HandActionDTO      `json:"actions" jsonschema:"every action taken during the hand, in order"`
}

// fromHand maps a normalized hand to a DTO, enriching it with display names and
// the per-player net results from the ledger. names and nets may be nil.
func fromHand(hand *gamelog.Hand, game *model.Game, names map[int64]string, nets map[int64]int) HandDTO {
	dto := HandDTO{
		GameID:         game.ID,
		TableUUID:      game.TableUUID,
		GameType:       hand.GameType,
		StoredGameType: game.GameType,
		Variant:        hand.Variant,
		Created:        game.Created,
		AnteCents:      hand.AnteCents,
		AnteDisplay:    money.FormatCents(hand.AnteCents),
		PotCents:       hand.PotCents,
		PotDisplay:     money.FormatCents(hand.PotCents),
		Rounds:         hand.Rounds,
		Board:          fromCards(hand.Board),
		Participants:   make([]HandParticipantDTO, 0, len(hand.Participants)),
		Actions:        make([]HandActionDTO, 0, len(hand.Actions)),
	}

	for _, p := range hand.Participants {
		participant := HandParticipantDTO{
			PlayerID:             p.PlayerID,
			DisplayName:          names[p.PlayerID],
			StartingCards:        fromCards(p.StartingCards),
			FinalCards:           fromCards(p.FinalCards),
			VoluntarilyPlayed:    p.VoluntarilyPlayed,
			Folded:               p.Folded,
			WentToShowdown:       p.WentToShowdown,
			Won:                  p.Won,
			AmountWageredCents:   p.AmountWageredCents,
			AmountWageredDisplay: money.FormatCents(p.AmountWageredCents),
		}

		if net, ok := nets[p.PlayerID]; ok {
			display := money.FormatCents(net)
			participant.NetCents = &net
			participant.NetDisplay = &display
		}

		dto.Participants = append(dto.Participants, participant)
	}

	for _, a := range hand.Actions {
		action := HandActionDTO{
			Sequence:    a.Sequence,
			Street:      a.Street,
			PlayerID:    a.PlayerID,
			DisplayName: names[a.PlayerID],
			Kind:        string(a.Kind),
			AmountCents: a.AmountCents,
			Cards:       fromCards(a.Cards),
			AllIn:       a.AllIn,
		}
		if a.AmountCents != 0 {
			action.AmountDisplay = money.FormatCents(a.AmountCents)
		}

		dto.Actions = append(dto.Actions, action)
	}

	return dto
}

// TendenciesDTO is a player's behavioral profile aggregated over many hands.
//
// The counts are reported alongside the rates on purpose: a 100% fold rate over
// three hands and over three hundred are very different claims, and only the counts
// distinguish them.
type TendenciesDTO struct {
	HandsAnalyzed int `json:"handsAnalyzed" jsonschema:"the number of hands the player appeared in across the analyzed logs"`

	HandsVoluntarilyPlayed int      `json:"handsVoluntarilyPlayed" jsonschema:"hands where the player committed beyond any forced ante or blind"`
	VoluntaryPlayRate      *RateDTO `json:"voluntaryPlayRate,omitempty" jsonschema:"share of hands voluntarily played, the cross-game equivalent of VPIP; absent when no hands were analyzed"`

	HandsFolded int      `json:"handsFolded" jsonschema:"hands the player folded or dropped out of"`
	FoldRate    *RateDTO `json:"foldRate,omitempty" jsonschema:"share of hands folded; absent when no hands were analyzed"`

	HandsToShowdown int      `json:"handsToShowdown" jsonschema:"hands the player was still contesting when the hand resolved"`
	ShowdownRate    *RateDTO `json:"showdownRate,omitempty" jsonschema:"share of hands taken to a showdown; absent when no hands were analyzed"`

	HandsWonAtShowdown int      `json:"handsWonAtShowdown" jsonschema:"contested showdowns the player won"`
	ShowdownWinRate    *RateDTO `json:"showdownWinRate,omitempty" jsonschema:"share of contested showdowns won; absent when the player never reached one. The denominator is showdowns reached, not hands played, so folding more does not improve it"`

	HandsWon int      `json:"handsWon" jsonschema:"hands the player won, including those won because everyone else folded"`
	WinRate  *RateDTO `json:"winRate,omitempty" jsonschema:"share of hands won; absent when no hands were analyzed"`

	Bets     int `json:"bets" jsonschema:"total bets"`
	Raises   int `json:"raises" jsonschema:"total raises"`
	Calls    int `json:"calls" jsonschema:"total calls"`
	Checks   int `json:"checks" jsonschema:"total checks"`
	Folds    int `json:"folds" jsonschema:"total folds and drop-outs"`
	Discards int `json:"discards" jsonschema:"total discards"`
	Trades   int `json:"trades" jsonschema:"total trades"`

	AggressionFactor *float64 `json:"aggressionFactor,omitempty" jsonschema:"ratio of aggressive actions (bets plus raises) to calls; absent when the player has never called, where the ratio is undefined rather than infinite"`

	AmountWageredCents   int    `json:"amountWageredCents" jsonschema:"total the player put into pots through their own actions, in cents, excluding forced antes"`
	AmountWageredDisplay string `json:"amountWageredDisplay" jsonschema:"the wagered total preformatted in dollars; show this rather than converting amountWageredCents yourself"`
}

// fromTendencies maps aggregated tendencies to a DTO.
func fromTendencies(t *gamelog.Tendencies) TendenciesDTO {
	dto := TendenciesDTO{
		HandsAnalyzed:          t.HandsPlayed,
		HandsVoluntarilyPlayed: t.HandsVoluntarilyPlayed,
		HandsFolded:            t.HandsFolded,
		HandsToShowdown:        t.HandsToShowdown,
		HandsWonAtShowdown:     t.HandsWonAtShowdown,
		HandsWon:               t.HandsWon,
		Bets:                   t.Bets,
		Raises:                 t.Raises,
		Calls:                  t.Calls,
		Checks:                 t.Checks,
		Folds:                  t.Folds,
		Discards:               t.Discards,
		Trades:                 t.Trades,
		AmountWageredCents:     t.AmountWageredCents,
		AmountWageredDisplay:   money.FormatCents(t.AmountWageredCents),
	}

	dto.VoluntaryPlayRate = newRate(t.Rate(t.HandsVoluntarilyPlayed))
	dto.FoldRate = newRate(t.Rate(t.HandsFolded))
	dto.ShowdownRate = newRate(t.Rate(t.HandsToShowdown))
	dto.WinRate = newRate(t.Rate(t.HandsWon))
	dto.ShowdownWinRate = newRate(t.ShowdownWinRate())

	if factor, ok := t.AggressionFactor(); ok {
		dto.AggressionFactor = &factor
	}

	return dto
}

// SpreadDTO describes the distribution of a series of results.
type SpreadDTO struct {
	Count int `json:"count" jsonschema:"the number of results in the series"`

	TotalCents   int    `json:"totalCents" jsonschema:"the sum of the results, in cents"`
	TotalDisplay string `json:"totalDisplay" jsonschema:"the total preformatted in dollars; show this rather than converting totalCents yourself"`

	MeanCents   float64 `json:"meanCents" jsonschema:"the average result, in cents"`
	MeanDisplay string  `json:"meanDisplay" jsonschema:"the average preformatted in dollars; show this rather than converting meanCents yourself"`

	MedianCents   float64 `json:"medianCents" jsonschema:"the median result, in cents"`
	MedianDisplay string  `json:"medianDisplay" jsonschema:"the median preformatted in dollars; show this rather than converting medianCents yourself"`

	StdDevCents   float64 `json:"stdDevCents" jsonschema:"the sample standard deviation of the results, in cents; this is the swing measure. It is zero for fewer than two results, where it is undefined rather than small"`
	StdDevDisplay string  `json:"stdDevDisplay" jsonschema:"the standard deviation preformatted in dollars; show this rather than converting stdDevCents yourself"`

	BestCents    int    `json:"bestCents" jsonschema:"the best single result, in cents"`
	BestDisplay  string `json:"bestDisplay" jsonschema:"the best result preformatted in dollars; show this rather than converting bestCents yourself"`
	WorstCents   int    `json:"worstCents" jsonschema:"the worst single result, in cents"`
	WorstDisplay string `json:"worstDisplay" jsonschema:"the worst result preformatted in dollars; show this rather than converting worstCents yourself"`
}

// fromSpread maps a model.Spread to a DTO. Fractional cents are rounded for
// display only; the raw float is preserved for arithmetic.
func fromSpread(s model.Spread) SpreadDTO {
	return SpreadDTO{
		Count:         s.Count,
		TotalCents:    s.TotalCents,
		TotalDisplay:  money.FormatCents(s.TotalCents),
		MeanCents:     s.MeanCents,
		MeanDisplay:   money.FormatCents(roundCents(s.MeanCents)),
		MedianCents:   s.MedianCents,
		MedianDisplay: money.FormatCents(roundCents(s.MedianCents)),
		StdDevCents:   s.StdDevCents,
		StdDevDisplay: money.FormatCents(roundCents(s.StdDevCents)),
		BestCents:     s.BestCents,
		BestDisplay:   money.FormatCents(s.BestCents),
		WorstCents:    s.WorstCents,
		WorstDisplay:  money.FormatCents(s.WorstCents),
	}
}

// roundCents rounds a fractional cent amount to the nearest whole cent, halves
// away from zero, for rendering through the shared money formatter.
func roundCents(cents float64) int {
	if cents < 0 {
		return -int(-cents + 0.5)
	}

	return int(cents + 0.5)
}

// StreakDTO is an unbroken run of results with the same outcome.
type StreakDTO struct {
	Outcome string `json:"outcome" jsonschema:"winning or losing"`
	Length  int    `json:"length" jsonschema:"the number of consecutive results in the run"`

	NetCents   int    `json:"netCents" jsonschema:"the run's combined net result, in cents"`
	NetDisplay string `json:"netDisplay" jsonschema:"the run's net preformatted in dollars; show this rather than converting netCents yourself"`

	StartedAt time.Time `json:"startedAt" jsonschema:"when the first result in the run occurred"`
	EndedAt   time.Time `json:"endedAt" jsonschema:"when the last result in the run occurred"`
}

// newStreak maps a model.Streak to a DTO, returning nil when there is no such run
// so the field is omitted rather than reported as a zero-length streak.
func newStreak(s model.Streak) *StreakDTO {
	if s.Outcome == model.StreakNone || s.Length == 0 {
		return nil
	}

	return &StreakDTO{
		Outcome:    string(s.Outcome),
		Length:     s.Length,
		NetCents:   s.NetCents,
		NetDisplay: money.FormatCents(s.NetCents),
		StartedAt:  s.StartedAt,
		EndedAt:    s.EndedAt,
	}
}

// StreaksDTO summarizes the runs within a chronological series of results.
type StreaksDTO struct {
	LongestWinning *StreakDTO `json:"longestWinning,omitempty" jsonschema:"the longest run of consecutive winning results; absent when there was none"`
	LongestLosing  *StreakDTO `json:"longestLosing,omitempty" jsonschema:"the longest run of consecutive losing results; absent when there was none"`
	Current        *StreakDTO `json:"current,omitempty" jsonschema:"the run still in progress as of the most recent result; absent when the latest result broke even, which ends any run"`
}

// fromStreaks maps model.Streaks to a DTO.
func fromStreaks(s model.Streaks) StreaksDTO {
	return StreaksDTO{
		LongestWinning: newStreak(s.LongestWinning),
		LongestLosing:  newStreak(s.LongestLosing),
		Current:        newStreak(s.Current),
	}
}

// PlayerVarianceDTO is a player's consistency profile.
type PlayerVarianceDTO struct {
	ByGame    SpreadDTO `json:"byGame" jsonschema:"the distribution of the player's per-game results; this is the honest swing measure, having enough data points to mean something"`
	BySession SpreadDTO `json:"bySession" jsonschema:"the distribution of the player's per-night results, a night being one table"`

	GameStreaks    StreaksDTO `json:"gameStreaks" jsonschema:"runs of consecutive winning or losing games"`
	SessionStreaks StreaksDTO `json:"sessionStreaks" jsonschema:"runs of consecutive winning or losing nights; this is the streak a player would talk about. A break-even night belongs to neither run and ends whichever was in progress"`
}

// fromPlayerVariance maps a model.PlayerVariance to a DTO.
func fromPlayerVariance(v *model.PlayerVariance) PlayerVarianceDTO {
	return PlayerVarianceDTO{
		ByGame:         fromSpread(v.ByGame),
		BySession:      fromSpread(v.BySession),
		GameStreaks:    fromStreaks(v.GameStreaks),
		SessionStreaks: fromStreaks(v.SessionStreaks),
	}
}
