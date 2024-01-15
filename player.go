package teamvite

import (
	"context"
	"database/sql"
	"fmt"
)

type Player struct {
	ID       uint64         `db:"id,primarykey,autoincrement"`
	Name     string         `db:"name,size:64"`
	Email    string         `db:"email,size:128"`
	Password sql.NullString `db:"password,size:256,default:''"`
	Phone    int            `db:"phone"`
}

// A team with additional player info from players_teams
type PlayerTeam struct {
	Team        Team
	RemindSMS   bool
	RemindEmail bool
}

func (p *Player) ItemID() uint64 {
	return p.ID
}

func (p *Player) ItemType() string {
	return "player"
}

type PlayerService interface {
	FindPlayerByID(ctx context.Context, id uint64) (*Player, error)

	// Retrieves a list of Players based on a filter. Also returns a count of total matching Teams which may
	// differ from the number of returned Teams if the "Limit" field is set.
	FindPlayers(ctx context.Context, filter PlayerFilter) ([]*Player, int, error)

	// Returns a list of teams that the player from the context plays on,
	// Includes extra info such as ismanager and reminder settings
	Teams(ctx context.Context, teamIDsFilter ...uint64) ([]PlayerTeam, error)

	NextRemindedGame(ctx context.Context, playerID uint64) (Game, error)

	CreatePlayer(ctx context.Context, player *Player) error

	// Updates a player's info
	UpdatePlayer(ctx context.Context) error

	// Update team status (reminder, is_manager, etc)
	UpdatePlayerTeam(ctx context.Context, playerTeam *PlayerTeam) error

	ResetPassword(ctx context.Context, password string) error
}

type PlayerFilter struct {
	// Filtering fields.
	ID     *uint64 `json:"id"`
	Name   *string `json:"name"`
	TeamID *uint64 `json:"team_id"`
	Email  string  `json:"email"`
	Phone  int     `json:"phone"`

	// Restrict to subset of range.
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

func ReminderID(teamID uint64) string {
	return fmt.Sprintf("reminders_%d", teamID)
}
