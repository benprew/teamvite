package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/benprew/teamvite"
)

type GameService struct {
	db *sql.DB
}

// Ensure service implements interface.
var _ teamvite.GameService = (*GameService)(nil)

// NewGameService returns a new instance of GameService.
func NewGameService(db *sql.DB) *GameService {
	return &GameService{db: db}
}

func (s *GameService) FindGameByID(ctx context.Context, id uint64) (*teamvite.Game, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	games, _, err := findGames(ctx, tx, teamvite.GameFilter{ID: id})
	if err != nil {
		return nil, err
	}
	if len(games) == 0 {
		return nil, &teamvite.Error{
			Code:    teamvite.ENOTFOUND,
			Message: fmt.Sprintf("game not found: %v", id),
		}
	}
	return games[0], nil
}

// FindGames retrieves a list of games based on a filter.
//
// Also returns a count of total matching Games which may different from the
// number of returned Games if the  "Limit" field is set.
func (s *GameService) FindGames(ctx context.Context, filter teamvite.GameFilter) ([]*teamvite.Game, int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, err
	}
	defer tx.Rollback()

	// Fetch list of matching Game objects.
	Games, n, err := findGames(ctx, tx, filter)
	return Games, n, err
}

// Inserts the given game into the db, returning the newly-inserted game
func (s *GameService) CreateGame(ctx context.Context, g *teamvite.Game) error {
	tx, err := s.db.BeginTx(ctx, nil)

	if g.Time == nil {
		return fmt.Errorf("game time is required")
	}
	if g.TeamID == 0 {
		return fmt.Errorf("team_id is required")
	}
	if g.SeasonID == 0 {
		return fmt.Errorf("season_id is required")
	}
	if g.Time.Before(time.Now().Add(-time.Hour * 24 * 30)) {
		return fmt.Errorf("game time too far in the past: %v", g.Time)
	}
	result, err := tx.ExecContext(ctx, `
			INSERT INTO games (team_id, season_id, time, description)
			VALUES (?, ?, ?, ?)
		`,
		g.TeamID, g.SeasonID, g.Time.Unix(), g.Description)
	if err != nil {
		return FormatError(err)
	}

	// Read back new Game ID into caller argument.
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	g.ID = uint64(id)
	return nil
}

func (s *GameService) UpdateStatus(ctx context.Context, game *teamvite.Game, status string) error {
	player := teamvite.UserFromContext(ctx)
	if status != "" && player.ID == 0 {
		return teamvite.Errorf(teamvite.EUNAUTHORIZED, "Can't set status without a player")
	}
	if status == "" || player.ID == 0 || game.ID == 0 {
		msg := fmt.Sprintf("[ERROR] Can't set status without player, status, and game. [status:%s,player:%v,game:%v]", status, player, game)
		log.Println(msg)
		return teamvite.Errorf(teamvite.EINVALID, msg)
	}

	fmt.Printf("setting game: %d and player: %d status to: %s\n", game.ID, player.ID, status)

	_, err := s.db.Exec(
		`INSERT INTO players_games
			(game_id, player_id, status)
			VALUES (?, ?, ?)
			ON CONFLICT (player_id, game_id) DO UPDATE SET status = ?;`,
		game.ID, player.ID, status, status,
	)
	return err
}

func (s *GameService) ResponsesForGame(ctx context.Context, game *teamvite.Game) (_ []*teamvite.GameResponse, err error) {
	//var r []*teamvite.GameResponse
	type Response int
	const (
		NoReply Response = iota
		Yes
		No
		Maybe
	)
	r := make([]*teamvite.GameResponse, 4)
	r[NoReply] = &teamvite.GameResponse{Name: "No Reply"}
	r[Yes] = &teamvite.GameResponse{Name: "Yes"}
	r[No] = &teamvite.GameResponse{Name: "No"}
	r[Maybe] = &teamvite.GameResponse{Name: "Maybe"}

	rows, err := s.db.Query(
		`
		SELECT
		CASE
			WHEN upper(pg.status) like 'Y%' then 1 -- 'Yes'
			WHEN upper(pg.status) like 'N%' then 2 -- 'No'
			WHEN upper(pg.status) like 'M%' then 3 -- 'Maybe'
			ELSE 0                      -- 'NoReply'
		END AS status,
		name
		FROM games g
		JOIN players_teams pt USING(team_id)
		JOIN players p ON pt.player_id = p.id
		LEFT JOIN players_games pg ON pg.game_id = g.id AND pg.player_id = p.id
		WHERE g.id = ?
		ORDER BY status desc, name`,
		game.ID,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	for rows.Next() {
		var statusInt int
		var name string
		rows.Scan(&statusInt, &name)
		status := Response(statusInt)
		r[status].Players = append(r[status].Players, name)
	}
	return r, nil
}

func findGames(ctx context.Context, tx *sql.Tx, filter teamvite.GameFilter) (_ []*teamvite.Game, n int, err error) {
	// Build WHERE clause. Each part of the WHERE clause is AND-ed together.
	// Values are appended to an arg list to avoid SQL injection.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := filter.ID; v != 0 {
		where, args = append(where, "id = ?"), append(args, v)
	}

	if v := filter.TeamID; v != 0 {
		where, args = append(where, "team_id = ?"), append(args, v)
	}

	if v := filter.PlayerID; v != 0 {
		where = append(where, `(
			id IN (SELECT games.id FROM games
				JOIN teams on games.team_id = teams.id
				JOIN players_teams USING(team_id)
				WHERE player_id = ?)
		)`)
		args = append(args, v)
	}

	if v := filter.Time; v != 0 {
		where = append(where, "time >= ?")
		args = append(args, v)
	}

	// Execue query with limiting WHERE clause and LIMIT/OFFSET injected.
	rows, err := tx.QueryContext(ctx, `
		SELECT
		    id,
		    team_id,
			season_id,
			time,
			description,
		    COUNT(*) OVER()
		FROM games
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY id ASC
		`+FormatLimitOffset(filter.Limit, filter.Offset),
		args...,
	)
	if err != nil {
		return nil, n, FormatError(err)
	}
	defer rows.Close()

	// Iterate over rows and deserialize into Game objects.
	games := make([]*teamvite.Game, 0)
	for rows.Next() {
		var game teamvite.Game
		if err := rows.Scan(
			&game.ID,
			&game.TeamID,
			&game.SeasonID,
			&game.Time,
			&game.Description,
			&n,
		); err != nil {
			return nil, 0, err
		}
		games = append(games, &game)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return games, n, nil
}
