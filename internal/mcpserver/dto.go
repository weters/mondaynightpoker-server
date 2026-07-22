package mcpserver

import (
	"time"

	"mondaynightpoker-server/pkg/model"
)

// PlayerDTO is a read-only representation of a player. It never contains any
// password material.
type PlayerDTO struct {
	ID          int64     `json:"id" jsonschema:"the player's unique identifier"`
	Email       string    `json:"email" jsonschema:"the player's email address"`
	DisplayName string    `json:"displayName" jsonschema:"the player's display name"`
	IsSiteAdmin bool      `json:"isSiteAdmin" jsonschema:"whether the player is a site administrator"`
	Status      string    `json:"status" jsonschema:"the player's account status"`
	Created     time.Time `json:"created" jsonschema:"when the player was created"`
	Updated     time.Time `json:"updated" jsonschema:"when the player was last updated"`
}

// fromPlayer maps a model.Player to a PlayerDTO.
func fromPlayer(p *model.Player) PlayerDTO {
	return PlayerDTO{
		ID:          p.ID,
		Email:       p.Email,
		DisplayName: p.DisplayName,
		IsSiteAdmin: p.IsSiteAdmin,
		Status:      string(p.Status),
		Created:     p.Created,
		Updated:     p.Updated,
	}
}

// fromPlayers maps a slice of model.Player to a slice of PlayerDTO.
func fromPlayers(players []*model.Player) []PlayerDTO {
	out := make([]PlayerDTO, 0, len(players))
	for _, p := range players {
		out = append(out, fromPlayer(p))
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
	Balance int `json:"balance" jsonschema:"the player's balance at the table"`
}

// fromTableWithBalance maps a model.TableWithBalance to a TableWithBalanceDTO.
func fromTableWithBalance(t *model.TableWithBalance) TableWithBalanceDTO {
	return TableWithBalanceDTO{
		TableDTO: fromTable(t.Table),
		Balance:  t.Balance,
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
	PlayerEmail string `json:"playerEmail" jsonschema:"the email of the player who created the table"`
}

// fromTablesWithEmail maps a slice of model.TableWithPlayerEmail to DTOs.
func fromTablesWithEmail(tables []*model.TableWithPlayerEmail) []TableWithEmailDTO {
	out := make([]TableWithEmailDTO, 0, len(tables))
	for _, t := range tables {
		out = append(out, TableWithEmailDTO{
			TableDTO:    fromTable(t.Table),
			PlayerEmail: t.Email,
		})
	}
	return out
}

// PlayerTableDTO is a single roster entry: a player and their table-specific
// permissions and balance.
type PlayerTableDTO struct {
	Player       PlayerDTO `json:"player" jsonschema:"the player"`
	PlayerID     int64     `json:"playerId" jsonschema:"the player's id"`
	TableUUID    string    `json:"tableUuid" jsonschema:"the table's uuid"`
	IsTableAdmin bool      `json:"isTableAdmin" jsonschema:"whether the player is a table administrator"`
	CanStart     bool      `json:"canStart" jsonschema:"whether the player can start a game"`
	CanRestart   bool      `json:"canRestart" jsonschema:"whether the player can restart a game"`
	CanTerminate bool      `json:"canTerminate" jsonschema:"whether the player can terminate a game"`
	Balance      int       `json:"balance" jsonschema:"the player's balance at the table"`
	TableStake   int       `json:"tableStake" jsonschema:"the player's table stake"`
	Active       bool      `json:"active" jsonschema:"whether the player is active at the table"`
	IsBlocked    bool      `json:"isBlocked" jsonschema:"whether the player is blocked at the table"`
}

// fromPlayerTable maps a model.PlayerTable to a PlayerTableDTO.
func fromPlayerTable(pt *model.PlayerTable) PlayerTableDTO {
	return PlayerTableDTO{
		Player:       fromPlayer(pt.Player),
		PlayerID:     pt.PlayerID,
		TableUUID:    pt.TableUUID,
		IsTableAdmin: pt.IsTableAdmin,
		CanStart:     pt.CanStart,
		CanRestart:   pt.CanRestart,
		CanTerminate: pt.CanTerminate,
		Balance:      pt.Balance,
		TableStake:   pt.TableStake,
		Active:       pt.Active,
		IsBlocked:    pt.IsBlocked,
	}
}

// fromPlayerTables maps a slice of model.PlayerTable to DTOs.
func fromPlayerTables(pts []*model.PlayerTable) []PlayerTableDTO {
	out := make([]PlayerTableDTO, 0, len(pts))
	for _, pt := range pts {
		out = append(out, fromPlayerTable(pt))
	}
	return out
}

// PlayerStatsDTO mirrors model.PlayerStats.
type PlayerStatsDTO struct {
	TablesJoined     int            `json:"tablesJoined" jsonschema:"the number of tables the player has joined"`
	GamesPlayed      int            `json:"gamesPlayed" jsonschema:"the number of games the player has played"`
	TotalWinnings    int            `json:"totalWinnings" jsonschema:"the player's total winnings"`
	WinningsByGame   map[string]int `json:"winningsByGame" jsonschema:"the player's winnings grouped by game type"`
	GamesCountByType map[string]int `json:"gamesCountByType" jsonschema:"the number of games played grouped by game type"`
}

// fromPlayerStats maps a model.PlayerStats to a PlayerStatsDTO.
func fromPlayerStats(s *model.PlayerStats) PlayerStatsDTO {
	return PlayerStatsDTO{
		TablesJoined:     s.TablesJoined,
		GamesPlayed:      s.GamesPlayed,
		TotalWinnings:    s.TotalWinnings,
		WinningsByGame:   s.WinningsByGame,
		GamesCountByType: s.GamesCountByType,
	}
}

// GraphPointDTO mirrors model.GraphPoint.
type GraphPointDTO struct {
	Created time.Time `json:"created" jsonschema:"when the data point was recorded"`
	Balance int       `json:"balance" jsonschema:"the player's balance at that point"`
}

// PlayerProfileDTO mirrors model.PlayerProfile.
type PlayerProfileDTO struct {
	Player    PlayerDTO             `json:"player" jsonschema:"the player"`
	Stats     PlayerStatsDTO        `json:"stats" jsonschema:"the player's aggregate stats"`
	Tables    []TableWithBalanceDTO `json:"tables" jsonschema:"the tables the player belongs to"`
	GraphData []GraphPointDTO       `json:"graphData" jsonschema:"balance data points for the profile graph"`
}

// fromPlayerProfile maps a model.PlayerProfile to a PlayerProfileDTO.
func fromPlayerProfile(p *model.PlayerProfile) PlayerProfileDTO {
	graph := make([]GraphPointDTO, 0, len(p.GraphData))
	for _, gp := range p.GraphData {
		graph = append(graph, GraphPointDTO{Created: gp.Created, Balance: gp.Balance})
	}

	return PlayerProfileDTO{
		Player:    fromPlayer(p.Player),
		Stats:     fromPlayerStats(p.Stats),
		Tables:    fromTablesWithBalance(p.Tables),
		GraphData: graph,
	}
}
