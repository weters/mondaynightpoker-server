package mcpserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"mondaynightpoker-server/internal/oauth"
	"mondaynightpoker-server/pkg/model"
	"mondaynightpoker-server/pkg/playable/gamelog"
	"mondaynightpoker-server/pkg/room/gamefactory"
)

// maxAnalyzedGameLogs bounds how many game logs a single tendencies call parses.
//
// Every log is decoded in memory, so an unbounded range on a long-running table
// would be paced by the jsonb rather than by the caller. When the range holds more
// games than this, the most recent ones are analyzed and the response says so
// rather than quietly presenting a partial answer as complete.
const maxAnalyzedGameLogs = 500

// getHandHistoryInput is the input for the get_hand_history tool.
type getHandHistoryInput struct {
	UUID string `json:"uuid" jsonschema:"the table's uuid"`
	ID   int64  `json:"id" jsonschema:"the game's numeric id"`
}

// getHandHistoryOutput is the output for the get_hand_history tool.
type getHandHistoryOutput struct {
	Hand HandDTO `json:"hand" jsonschema:"the normalized hand"`
}

// getHandHistory returns a single game's log decoded into the normalized hand
// shape, rather than the raw game-specific payload get_game exposes.
//
// Authorization matches get_game exactly: the table uuid is the capability, and a
// game is only returned when it belongs to that table. A missing game, a game at
// another table, and a game whose log cannot be read are reported identically so a
// caller cannot probe which part of the request was wrong.
func (s *server) getHandHistory(ctx context.Context, _ *mcp.CallToolRequest, _ oauth.Caller, in getHandHistoryInput) (getHandHistoryOutput, error) {
	game, err := s.gameAtTable(ctx, in.UUID, in.ID, true)
	if err != nil {
		return getHandHistoryOutput{}, err
	}

	hand, err := gamefactory.ParseStoredGameLog(game.GameType, game.RawData())
	if err != nil {
		return getHandHistoryOutput{}, err
	}
	if hand == nil {
		// The game row exists but never received a log, which is what an
		// in-progress or terminated game looks like.
		return getHandHistoryOutput{}, errNotFound("hand history")
	}

	adjustments, err := s.repos.Games.GetGameAdjustments(ctx, []int64{game.ID})
	if err != nil {
		return getHandHistoryOutput{}, err
	}

	names, nets := adjustmentLookups(adjustments[game.ID])

	return getHandHistoryOutput{Hand: fromHand(hand, game, names, nets)}, nil
}

// adjustmentLookups turns a game's ledger rows into display-name and net lookups
// keyed by player id.
func adjustmentLookups(adjustments []*model.GameAdjustment) (names map[int64]string, nets map[int64]int) {
	names = make(map[int64]string, len(adjustments))
	nets = make(map[int64]int, len(adjustments))
	for _, a := range adjustments {
		names[a.PlayerID] = a.DisplayName
		nets[a.PlayerID] = a.Adjustment
	}

	return names, nets
}

// getPlayerTendenciesInput is the input for the get_player_tendencies tool.
type getPlayerTendenciesInput struct {
	ID       int64   `json:"id" jsonschema:"the player's numeric id"`
	From     *string `json:"from,omitempty" jsonschema:"optional start of the date range (RFC3339); defaults to the epoch"`
	To       *string `json:"to,omitempty" jsonschema:"optional end of the date range (RFC3339); defaults to now"`
	GameType *string `json:"gameType,omitempty" jsonschema:"optional game-type identifier to restrict the analysis to, as returned by list_game_types (e.g. texas-hold-em); when omitted every game type is included"`
}

func (in getPlayerTendenciesInput) targetPlayerID() int64 { return in.ID }

// getPlayerTendenciesOutput is the output for the get_player_tendencies tool.
type getPlayerTendenciesOutput struct {
	Player     PlayerDTO     `json:"player" jsonschema:"the player"`
	Tendencies TendenciesDTO `json:"tendencies" jsonschema:"the player's aggregate behavioral profile across every analyzed hand"`

	ByGameType map[string]TendenciesDTO `json:"byGameType" jsonschema:"the same profile broken out per game-type identifier; the numbers are only comparable across game types for the poker variants, since games without a betting round have no true equivalent of a call or a raise"`

	GamesAnalyzed int   `json:"gamesAnalyzed" jsonschema:"the number of game logs successfully parsed"`
	GamesInRange  int64 `json:"gamesInRange" jsonschema:"the total number of the player's completed games in the range, ignoring the analysis cap. It counts every game type even when gameType narrows the analysis, so with a filter set it will normally exceed gamesAnalyzed"`
	Truncated     bool  `json:"truncated" jsonschema:"whether the range held more games than could be examined; when true only the most recent games are reflected and the rates describe that subset. With gameType set this is computed before the filter, so a true value means the filtered profile may also be missing older matching games"`
	GamesSkipped  int   `json:"gamesSkipped" jsonschema:"game logs left out of the analysis, either because the stored log could not be parsed or because it parsed fine but did not name this player (a ledger/log disagreement); a non-zero value means those games are missing from every figure here"`
}

// getPlayerTendencies aggregates a player's decisions across their game logs.
//
// This is the behavioral counterpart to get_player_stats: the ledger already says
// how much someone won, and this says how they played. Every figure is derived
// from the persisted logs rather than the ledger, so games that never finished
// contribute nothing.
//
// A log that fails to parse is counted in GamesSkipped rather than failing the
// call. One malformed or newer-format payload should not make a player's whole
// history unreadable, but the count is surfaced so a caller can tell the
// difference between a complete answer and a partial one.
func (s *server) getPlayerTendencies(ctx context.Context, _ *mcp.CallToolRequest, caller oauth.Caller, in getPlayerTendenciesInput) (getPlayerTendenciesOutput, error) {
	from, to, err := parseDateRange(in.From, in.To)
	if err != nil {
		return getPlayerTendenciesOutput{}, err
	}

	player, err := s.repos.Players.GetPlayerByID(ctx, in.ID)
	if err != nil {
		return getPlayerTendenciesOutput{}, notFound(err, "player")
	}

	// Reject an unknown game type outright rather than silently returning an empty
	// profile, which would read as "this player has never played it".
	var gameTypeFilter string
	if in.GameType != nil && *in.GameType != "" {
		if _, err := gamefactory.Get(*in.GameType); err != nil {
			return getPlayerTendenciesOutput{}, err
		}

		gameTypeFilter = *in.GameType
	}

	records, total, err := s.repos.Players.GetPlayerGameLogs(ctx, in.ID, from, to, maxAnalyzedGameLogs)
	if err != nil {
		return getPlayerTendenciesOutput{}, err
	}

	var overall gamelog.Tendencies
	byGameType := make(map[string]*gamelog.Tendencies)
	analyzed := 0
	skipped := 0

	for _, rec := range records {
		// Resolve the stored display name before decoding: the filter is decidable
		// from it alone, so a filtered request skips the jsonb parse for every game
		// of another type instead of decoding it and throwing the result away.
		if gameTypeFilter != "" {
			name, err := gamefactory.NameForStoredGameType(rec.GameType)
			if err != nil || name != gameTypeFilter {
				continue
			}
		}

		hand, err := gamefactory.ParseStoredGameLog(rec.GameType, rec.Data)
		if err != nil || hand == nil {
			skipped++
			continue
		}

		participation := hand.FindParticipant(in.ID)
		if participation == nil {
			// The ledger says the player was in this game but the log does not
			// name them, so there is nothing to attribute.
			skipped++
			continue
		}

		analyzed++
		overall.Add(participation)

		perType, ok := byGameType[hand.GameType]
		if !ok {
			perType = &gamelog.Tendencies{}
			byGameType[hand.GameType] = perType
		}
		perType.Add(participation)
	}

	out := getPlayerTendenciesOutput{
		Player:        fromPlayer(player, caller),
		Tendencies:    fromTendencies(&overall),
		ByGameType:    make(map[string]TendenciesDTO, len(byGameType)),
		GamesAnalyzed: analyzed,
		GamesInRange:  total,
		Truncated:     total > int64(len(records)),
		GamesSkipped:  skipped,
	}

	for gameType, t := range byGameType {
		out.ByGameType[gameType] = fromTendencies(t)
	}

	return out, nil
}

// getPlayerVarianceInput is the input for the get_player_variance tool.
type getPlayerVarianceInput struct {
	ID   int64   `json:"id" jsonschema:"the player's numeric id"`
	From *string `json:"from,omitempty" jsonschema:"optional start of the date range (RFC3339); defaults to the epoch"`
	To   *string `json:"to,omitempty" jsonschema:"optional end of the date range (RFC3339); defaults to now"`
}

func (in getPlayerVarianceInput) targetPlayerID() int64 { return in.ID }

// getPlayerVarianceOutput is the output for the get_player_variance tool.
type getPlayerVarianceOutput struct {
	Player   PlayerDTO         `json:"player" jsonschema:"the player"`
	Variance PlayerVarianceDTO `json:"variance" jsonschema:"the player's spread and streaks"`
}

// getPlayerVariance reports how consistent a player's results are, at both game
// and night granularity.
//
// get_player_stats answers how much someone is up; this answers whether that
// number means anything. Two players up the same amount look identical until you
// see that one ground it out and the other swung wildly in both directions, and
// the standard deviation is what separates them.
func (s *server) getPlayerVariance(ctx context.Context, _ *mcp.CallToolRequest, caller oauth.Caller, in getPlayerVarianceInput) (getPlayerVarianceOutput, error) {
	from, to, err := parseDateRange(in.From, in.To)
	if err != nil {
		return getPlayerVarianceOutput{}, err
	}

	player, err := s.repos.Players.GetPlayerByID(ctx, in.ID)
	if err != nil {
		return getPlayerVarianceOutput{}, notFound(err, "player")
	}

	variance, err := s.repos.Players.GetPlayerVariance(ctx, in.ID, from, to)
	if err != nil {
		return getPlayerVarianceOutput{}, err
	}

	return getPlayerVarianceOutput{
		Player:   fromPlayer(player, caller),
		Variance: fromPlayerVariance(variance),
	}, nil
}
