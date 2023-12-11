package teamvite

import (
	"context"
	"time"
)

type Game struct {
	ID          uint64     `db:"id,primarykey,autoincrement"`
	TeamID      uint64     `db:"team_id" json:"team_id"`
	SeasonID    uint64     `db:"season_id" json:"season_id"`
	Time        *time.Time `db:"time"`
	Description string     `db:"description"`
}

type PlayerGame struct {
	PlayerID     uint64 `db:"player_id"`
	GameID       uint64 `db:"game_id"`
	Status       string `db:"status"`
	ReminderSent bool   `db:"reminder_sent"`
}

func (g *Game) itemID() uint64 {
	return g.ID
}

func (g *Game) itemType() string {
	return "game"
}

type GameService interface {
	// Retrieves a single Game by ID along with associated memberships. Only the
	// Team manager & members can see a Game. Returns ENOTFOUND if Game does not
	// exist or user does not have permission to view it.
	FindGameByID(ctx context.Context, id uint64) (*Game, error)

	// Retrieves a list of Games based on a filter. Only returns Games for teams
	// the user is on. Also returns a count of total matching Games which may
	// different from the number of returned Games if the "Limit" field is set.
	FindGames(ctx context.Context, filter GameFilter) ([]*Game, int, error)

	// Creates a new Game and assigns the current user as the owner.  The owner
	// will automatically be added as a member of the new Game.
	CreateGame(ctx context.Context, game *Game) error

	// game status can be set on the game page, and it will upsert/find_or_create status
	// for emails, pass a session_id param and use that to get the player_id
	// otherwise just use the current session player_id
	SetStatus(ctx context.Context, game *Game, status string) error

	// Return the players bucketd by reply status for a game
	ResponsesForGame(ctx context.Context, game *Game) (_ []*GameResponse, err error)
}

// GameFilter represents a filter used by FindGames().
type GameFilter struct {
	// Filtering fields.
	ID       *uint64 `json:"id"`
	TeamID   *uint64 `json:"team_id"`
	PlayerID *uint64 `json:"player_id"`
	Time     int64   `json:"time"` // unix epoch seconds

	// Restrict to subset of range.
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

const GameLength = 60
