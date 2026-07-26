package mcpserver

import (
	"context"
	"encoding/json"

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
	table, err := s.activeTable(ctx, in.UUID)
	if err != nil {
		return getHandHistoryOutput{}, errNotFound("game")
	}

	game, err := s.repos.Games.GetGameByID(ctx, in.ID)
	if err != nil {
		return getHandHistoryOutput{}, errNotFound("game")
	}

	if game.TableUUID != table.UUID {
		return getHandHistoryOutput{}, errNotFound("game")
	}

	raw, err := rawGameData(game)
	if err != nil {
		return getHandHistoryOutput{}, err
	}

	hand, err := gamefactory.ParseStoredGameLog(game.GameType, raw)
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

// rawGameData re-encodes a game's decoded jsonb payload back into JSON for the
// parsers. model.Game unmarshals the column into an interface{}, so this is a
// round trip rather than the original bytes; it is faithful because the value came
// from encoding/json in the first place.
func rawGameData(game *model.Game) (json.RawMessage, error) {
	data := game.Data()
	if data == nil {
		return nil, nil
	}

	return json.Marshal(data)
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
	GamesInRange  int64 `json:"gamesInRange" jsonschema:"the total number of the player's completed games in the range, ignoring the analysis cap"`
	Truncated     bool  `json:"truncated" jsonschema:"whether the range held more games than were analyzed; when true only the most recent games are reflected and the rates describe that subset"`
	GamesSkipped  int   `json:"gamesSkipped" jsonschema:"game logs that could not be parsed and were left out; a non-zero value means those games are missing from every figure here"`
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

	records, err := s.repos.Players.GetPlayerGameLogs(ctx, in.ID, from, to, maxAnalyzedGameLogs)
	if err != nil {
		return getPlayerTendenciesOutput{}, err
	}

	total, err := s.repos.Players.GetPlayerGameLogsCount(ctx, in.ID, from, to)
	if err != nil {
		return getPlayerTendenciesOutput{}, err
	}

	var overall gamelog.Tendencies
	byGameType := make(map[string]*gamelog.Tendencies)
	analyzed := 0
	skipped := 0

	for _, rec := range records {
		hand, err := gamefactory.ParseStoredGameLog(rec.GameType, rec.Data)
		if err != nil || hand == nil {
			skipped++
			continue
		}

		if gameTypeFilter != "" && hand.GameType != gameTypeFilter {
			continue
		}

		participation := findParticipation(hand, in.ID)
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

// findParticipation returns the player's participation record within the hand, or
// nil when the log does not name them.
func findParticipation(hand *gamelog.Hand, playerID int64) *gamelog.Participation {
	for _, p := range hand.Participants {
		if p.PlayerID == playerID {
			return p
		}
	}

	return nil
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
