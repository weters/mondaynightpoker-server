package mcpserver

import (
	"maps"
	"time"

	"mondaynightpoker-server/internal/oauth"
	"mondaynightpoker-server/pkg/model"
	"mondaynightpoker-server/pkg/money"
)

// visibleEmail is the single decision point for surfacing a player's email. It returns a
// non-nil pointer to email only when the caller is a site admin or is the owner of the
// record (ownerPlayerID == caller.PlayerID); otherwise it returns nil so the field is
// omitted from the JSON entirely (structural absence, never an empty string). Every mapper
// that emits an email routes it through this helper, so a new tool cannot leak another
// player's email by accident.
func visibleEmail(caller oauth.Caller, ownerPlayerID int64, email string) *string {
	if caller.IsSiteAdmin || ownerPlayerID == caller.PlayerID {
		return &email
	}
	return nil
}

// PlayerDTO is a read-only representation of a player. It never contains any
// password material.
type PlayerDTO struct {
	ID          int64     `json:"id" jsonschema:"the player's unique identifier"`
	Email       *string   `json:"email,omitempty" jsonschema:"the player's email address; present only for site admins or when the caller is this player, omitted otherwise"`
	DisplayName string    `json:"displayName" jsonschema:"the player's display name"`
	IsSiteAdmin bool      `json:"isSiteAdmin" jsonschema:"whether the player is a site administrator"`
	Status      string    `json:"status" jsonschema:"the player's account status"`
	Created     time.Time `json:"created" jsonschema:"when the player was created"`
	Updated     time.Time `json:"updated" jsonschema:"when the player was last updated"`
}

// fromPlayer maps a model.Player to a PlayerDTO, surfacing the email only when the caller
// is permitted to see it.
func fromPlayer(p *model.Player, caller oauth.Caller) PlayerDTO {
	return PlayerDTO{
		ID:          p.ID,
		Email:       visibleEmail(caller, p.ID, p.Email),
		DisplayName: p.DisplayName,
		IsSiteAdmin: p.IsSiteAdmin,
		Status:      string(p.Status),
		Created:     p.Created,
		Updated:     p.Updated,
	}
}

// fromPlayers maps a slice of model.Player to a slice of PlayerDTO.
func fromPlayers(players []*model.Player, caller oauth.Caller) []PlayerDTO {
	out := make([]PlayerDTO, 0, len(players))
	for _, p := range players {
		out = append(out, fromPlayer(p, caller))
	}
	return out
}

// TableDTO is a read-only representation of a poker table.
type TableDTO struct {
	UUID     string    `json:"uuid" jsonschema:"the table's unique identifier"`
	Name     string    `json:"name" jsonschema:"the table's name"`
	PlayerID int64     `json:"playerId" jsonschema:"the id of the player who created the table"`
	Created  time.Time `json:"created" jsonschema:"when the table was created"`
	Modified time.Time `json:"modified" jsonschema:"when the table was last modified"`
	Deleted  bool      `json:"deleted" jsonschema:"whether the table has been deleted"`
}

// fromTable maps a model.Table to a TableDTO.
func fromTable(t *model.Table) TableDTO {
	return TableDTO{
		UUID:     t.UUID,
		Name:     t.Name,
		PlayerID: t.PlayerID,
		Created:  t.Created,
		Modified: t.Modified,
		Deleted:  t.Deleted,
	}
}

// TableWithBalanceDTO is a TableDTO extended with a player's balance at the table.
type TableWithBalanceDTO struct {
	TableDTO
	BalanceCents   int    `json:"balanceCents" jsonschema:"the player's balance at the table, in cents (e.g. 150 means one dollar and fifty cents)"`
	BalanceDisplay string `json:"balanceDisplay" jsonschema:"the player's balance at the table, preformatted for display in dollars; show this to the user rather than converting balanceCents yourself"`
}

// fromTableWithBalance maps a model.TableWithBalance to a TableWithBalanceDTO.
func fromTableWithBalance(t *model.TableWithBalance) TableWithBalanceDTO {
	return TableWithBalanceDTO{
		TableDTO:       fromTable(t.Table),
		BalanceCents:   t.Balance,
		BalanceDisplay: money.FormatCents(t.Balance),
	}
}

// fromTablesWithBalance maps a slice of model.TableWithBalance to DTOs.
func fromTablesWithBalance(tables []*model.TableWithBalance) []TableWithBalanceDTO {
	out := make([]TableWithBalanceDTO, 0, len(tables))
	for _, t := range tables {
		out = append(out, fromTableWithBalance(t))
	}
	return out
}

// TableWithEmailDTO is a TableDTO extended with the email of the creating player.
type TableWithEmailDTO struct {
	TableDTO
	PlayerEmail *string `json:"playerEmail,omitempty" jsonschema:"the email of the player who created the table; present only for site admins or when the caller created the table, omitted otherwise"`
}

// fromTablesWithEmail maps a slice of model.TableWithPlayerEmail to DTOs, surfacing the
// creator's email only when the caller is permitted to see it.
func fromTablesWithEmail(tables []*model.TableWithPlayerEmail, caller oauth.Caller) []TableWithEmailDTO {
	out := make([]TableWithEmailDTO, 0, len(tables))
	for _, t := range tables {
		out = append(out, TableWithEmailDTO{
			TableDTO:    fromTable(t.Table),
			PlayerEmail: visibleEmail(caller, t.Table.PlayerID, t.Email),
		})
	}
	return out
}

// fromTablesWithBalanceAsEmail maps membership tables (which carry no creator email) into
// the TableWithEmailDTO shape, leaving PlayerEmail empty. It backs the non-admin
// list_tables path so its output matches the admin schema without leaking creator emails.
func fromTablesWithBalanceAsEmail(tables []*model.TableWithBalance) []TableWithEmailDTO {
	out := make([]TableWithEmailDTO, 0, len(tables))
	for _, t := range tables {
		out = append(out, TableWithEmailDTO{TableDTO: fromTable(t.Table)})
	}
	return out
}

// PlayerTableDTO is a single roster entry: a player and their table-specific
// permissions and balance.
type PlayerTableDTO struct {
	Player            PlayerDTO `json:"player" jsonschema:"the player"`
	PlayerID          int64     `json:"playerId" jsonschema:"the player's id"`
	TableUUID         string    `json:"tableUuid" jsonschema:"the table's uuid"`
	IsTableAdmin      bool      `json:"isTableAdmin" jsonschema:"whether the player is a table administrator"`
	CanStart          bool      `json:"canStart" jsonschema:"whether the player can start a game"`
	CanRestart        bool      `json:"canRestart" jsonschema:"whether the player can restart a game"`
	CanTerminate      bool      `json:"canTerminate" jsonschema:"whether the player can terminate a game"`
	BalanceCents      int       `json:"balanceCents" jsonschema:"the player's balance at the table, in cents (e.g. 150 means one dollar and fifty cents)"`
	BalanceDisplay    string    `json:"balanceDisplay" jsonschema:"the player's balance at the table, preformatted for display in dollars; show this to the user rather than converting balanceCents yourself"`
	TableStakeCents   int       `json:"tableStakeCents" jsonschema:"the player's table stake, in cents"`
	TableStakeDisplay string    `json:"tableStakeDisplay" jsonschema:"the player's table stake, preformatted for display in dollars; show this to the user rather than converting tableStakeCents yourself"`
	Active            bool      `json:"active" jsonschema:"whether the player is active at the table"`
	IsBlocked         bool      `json:"isBlocked" jsonschema:"whether the player is blocked at the table"`
}

// fromPlayerTable maps a model.PlayerTable to a PlayerTableDTO.
func fromPlayerTable(pt *model.PlayerTable, caller oauth.Caller) PlayerTableDTO {
	return PlayerTableDTO{
		Player:            fromPlayer(pt.Player, caller),
		PlayerID:          pt.PlayerID,
		TableUUID:         pt.TableUUID,
		IsTableAdmin:      pt.IsTableAdmin,
		CanStart:          pt.CanStart,
		CanRestart:        pt.CanRestart,
		CanTerminate:      pt.CanTerminate,
		BalanceCents:      pt.Balance,
		BalanceDisplay:    money.FormatCents(pt.Balance),
		TableStakeCents:   pt.TableStake,
		TableStakeDisplay: money.FormatCents(pt.TableStake),
		Active:            pt.Active,
		IsBlocked:         pt.IsBlocked,
	}
}

// fromPlayerTables maps a slice of model.PlayerTable to DTOs.
func fromPlayerTables(pts []*model.PlayerTable, caller oauth.Caller) []PlayerTableDTO {
	out := make([]PlayerTableDTO, 0, len(pts))
	for _, pt := range pts {
		out = append(out, fromPlayerTable(pt, caller))
	}
	return out
}

// PlayerStatsDTO mirrors model.PlayerStats.
type PlayerStatsDTO struct {
	TablesJoined         int            `json:"tablesJoined" jsonschema:"the number of tables the player has joined"`
	GamesPlayed          int            `json:"gamesPlayed" jsonschema:"the number of games the player has played"`
	TotalWinningsCents   int            `json:"totalWinningsCents" jsonschema:"the player's total winnings, in cents (e.g. 150 means one dollar and fifty cents)"`
	TotalWinningsDisplay string         `json:"totalWinningsDisplay" jsonschema:"the player's total winnings, preformatted for display in dollars; show this to the user rather than converting totalWinningsCents yourself"`
	WinningsByGameCents  map[string]int `json:"winningsByGameCents" jsonschema:"the player's winnings grouped by game type, in cents; convert to dollars before displaying"`
	GamesCountByType     map[string]int `json:"gamesCountByType" jsonschema:"the number of games played grouped by game type"`
}

// fromPlayerStats maps a model.PlayerStats to a PlayerStatsDTO. The maps are copied
// rather than aliased so the DTO never shares mutable state with the model.
func fromPlayerStats(s *model.PlayerStats) PlayerStatsDTO {
	return PlayerStatsDTO{
		TablesJoined:         s.TablesJoined,
		GamesPlayed:          s.GamesPlayed,
		TotalWinningsCents:   s.TotalWinnings,
		TotalWinningsDisplay: money.FormatCents(s.TotalWinnings),
		WinningsByGameCents:  maps.Clone(s.WinningsByGame),
		GamesCountByType:     maps.Clone(s.GamesCountByType),
	}
}

// GraphPointDTO mirrors model.GraphPoint.
type GraphPointDTO struct {
	Created      time.Time `json:"created" jsonschema:"when the data point was recorded"`
	BalanceCents int       `json:"balanceCents" jsonschema:"the player's balance at that point, in cents; convert to dollars before displaying"`
}

// PlayerProfileDTO mirrors model.PlayerProfile.
type PlayerProfileDTO struct {
	Player    PlayerDTO             `json:"player" jsonschema:"the player"`
	Stats     PlayerStatsDTO        `json:"stats" jsonschema:"the player's aggregate stats"`
	Tables    []TableWithBalanceDTO `json:"tables" jsonschema:"the tables the player belongs to"`
	GraphData []GraphPointDTO       `json:"graphData" jsonschema:"balance data points for the profile graph"`
}

// fromPlayerProfile maps a model.PlayerProfile to a PlayerProfileDTO.
func fromPlayerProfile(p *model.PlayerProfile, caller oauth.Caller) PlayerProfileDTO {
	graph := make([]GraphPointDTO, 0, len(p.GraphData))
	for _, gp := range p.GraphData {
		graph = append(graph, GraphPointDTO{Created: gp.Created, BalanceCents: gp.Balance})
	}

	return PlayerProfileDTO{
		Player:    fromPlayer(p.Player, caller),
		Stats:     fromPlayerStats(p.Stats),
		Tables:    fromTablesWithBalance(p.Tables),
		GraphData: graph,
	}
}
