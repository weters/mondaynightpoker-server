package mcpserver

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mondaynightpoker-server/pkg/model"
)

// Every monetary field crosses the MCP boundary as integer cents. These tests pin
// the cents value, the preformatted dollar twin, and the JSON key names, so a field
// cannot silently start meaning dollars.

func TestFromTableWithBalance_Money(t *testing.T) {
	a := assert.New(t)

	dto := fromTableWithBalance(&model.TableWithBalance{
		Table:   &model.Table{UUID: "abc", Name: "Table"},
		Balance: 150,
	})

	a.Equal(150, dto.BalanceCents)
	a.Equal("$1.50", dto.BalanceDisplay)

	// negative balances are common in poker and must not render as "$-1.50"
	negative := fromTableWithBalance(&model.TableWithBalance{
		Table:   &model.Table{UUID: "abc"},
		Balance: -50,
	})
	a.Equal(-50, negative.BalanceCents)
	a.Equal("-$0.50", negative.BalanceDisplay)
}

func TestFromPlayerTable_Money(t *testing.T) {
	a := assert.New(t)

	dto := fromPlayerTable(&model.PlayerTable{
		Player:     &model.Player{ID: 1},
		Balance:    -2575,
		TableStake: 10000,
	}, playerCaller(1))

	a.Equal(-2575, dto.BalanceCents)
	a.Equal("-$25.75", dto.BalanceDisplay)
	a.Equal(10000, dto.TableStakeCents)
	a.Equal("$100", dto.TableStakeDisplay)
}

func TestFromPlayerStats_Money(t *testing.T) {
	a := assert.New(t)

	stats := &model.PlayerStats{
		TablesJoined:     2,
		GamesPlayed:      7,
		TotalWinnings:    12345,
		WinningsByGame:   map[string]int{"guts": 500},
		GamesCountByType: map[string]int{"guts": 3},
	}

	dto := fromPlayerStats(stats)

	a.Equal(12345, dto.TotalWinningsCents)
	a.Equal("$123.45", dto.TotalWinningsDisplay)
	a.Equal(map[string]int{"guts": 500}, dto.WinningsByGameCents)

	// the DTO must own its maps; mutating the model afterwards must not leak through
	stats.WinningsByGame["guts"] = 999
	stats.GamesCountByType["guts"] = 999
	a.Equal(500, dto.WinningsByGameCents["guts"])
	a.Equal(3, dto.GamesCountByType["guts"])
}

func TestFromPlayerProfile_GraphBalanceInCents(t *testing.T) {
	a := assert.New(t)

	dto := fromPlayerProfile(&model.PlayerProfile{
		Player:    &model.Player{ID: 1},
		Stats:     &model.PlayerStats{},
		Tables:    []*model.TableWithBalance{},
		GraphData: []*model.GraphPoint{{Balance: 250}},
	}, playerCaller(1))

	require.Len(t, dto.GraphData, 1)
	a.Equal(250, dto.GraphData[0].BalanceCents)
}

// TestMoneyJSONKeys locks the wire names. The "Cents" suffix is what tells the model
// the unit, so a rename back to a bare "balance" is a regression, not a refactor.
func TestMoneyJSONKeys(t *testing.T) {
	a := assert.New(t)

	roster, err := json.Marshal(fromPlayerTable(&model.PlayerTable{
		Player: &model.Player{ID: 1},
	}, playerCaller(1)))
	require.NoError(t, err)

	a.Contains(string(roster), `"balanceCents"`)
	a.Contains(string(roster), `"balanceDisplay"`)
	a.Contains(string(roster), `"tableStakeCents"`)
	a.Contains(string(roster), `"tableStakeDisplay"`)
	a.NotContains(string(roster), `"balance"`)
	a.NotContains(string(roster), `"tableStake"`)

	stats, err := json.Marshal(fromPlayerStats(&model.PlayerStats{}))
	require.NoError(t, err)

	a.Contains(string(stats), `"totalWinningsCents"`)
	a.Contains(string(stats), `"totalWinningsDisplay"`)
	a.Contains(string(stats), `"winningsByGameCents"`)
	a.NotContains(string(stats), `"totalWinnings"`)
}

// TestServerInstructionsDeclareCents guards the one piece of context that covers the
// cents-only fields, which have no preformatted twin to fall back on.
func TestServerInstructionsDeclareCents(t *testing.T) {
	a := assert.New(t)
	a.Contains(serverInstructions, "CENTS")
	a.Contains(serverInstructions, "balanceDisplay")
}
