package sqlite

import (
	"context"
	"database/sql"

	"github.com/benprew/teamvite"
	"golang.org/x/crypto/bcrypt"
)

type PlayerService struct {
	db *sql.DB
}

// Ensure service implements interface.
var _ teamvite.PlayerService = (*PlayerService)(nil)

func NewPlayerService(db *sql.DB) *PlayerService {
	return &PlayerService{db: db}
}

// Holds all the functions that involve the player service
func (ps *PlayerService) FindPlayerByID(ctx context.Context, id uint64) (*teamvite.Player, error) {
	tx, err := ps.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	players, _, err := findPlayers(ctx, tx, teamvite.PlayerFilter{ID: &id})
	if err != nil {
		return nil, err
	}
	if len(players) == 0 {
		return nil, teamvite.Errorf(
			teamvite.ENOTFOUND,
			"player not found: %v", id,
		)
	}
	return players[0], nil

}

func (ps *PlayerService) FindPlayers(ctx context.Context, filter teamvite.PlayerFilter) ([]*teamvite.Player, int, error) {
	tx, err := ps.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, err
	}
	defer tx.Rollback()

	players, n, err := findPlayers(ctx, tx, filter)
	return players, n, err
}

func (ps *PlayerService) Teams(ctx context.Context, teamIDs ...uint64) ([]teamvite.PlayerTeam, error) {
	var playerTeams []teamvite.PlayerTeam
	var args []interface{}

	args = append(args, teamvite.PlayerFromContext(ctx).ID)

	query := `
		SELECT
			t.*,
			pt.remind_email,
			pt.remind_sms
		FROM teams t
			JOIN players_teams pt
			ON t.id = pt.team_id
		WHERE player_id = ?`

	if len(teamIDs) > 0 {
		query += " and t.id in ?"
		args = append(args, teamIDs)
	}

	rows, err := ps.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var pt teamvite.PlayerTeam
		err := rows.Scan(&pt.Team.ID, &pt.Team.Name, &pt.Team.DivisionID, &pt.RemindEmail, &pt.RemindSMS)
		if err != nil {
			return nil, err
		}
		playerTeams = append(playerTeams, pt)
	}

	return playerTeams, err
}

func (ps *PlayerService) NextRemindedGame(ctx context.Context, playerID uint64) (teamvite.Game, error) {
	var g teamvite.Game
	err := ps.db.QueryRow(`
		SELECT games.*
		FROM games
		INNER JOIN players_games pg ON games.id = pg.game_id
		INNER JOIN players_teams pt on pt.player_id = pg.player_id and pt.team_id = games.team_id
		WHERE
			pg.player_id = 117
			AND games.time > datetime('now')
			AND pg.reminder_sent = true
			AND pt.remind_sms = true
		ORDER BY games.time ASC
		LIMIT 1;`,
		playerID).Scan(&g.ID, &g.TeamID, &g.SeasonID, &g.Time, &g.Description)
	return g, err
}

func (ps *PlayerService) CreatePlayer(ctx context.Context, player *teamvite.Player) error {
	tx, err := ps.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	result, err := tx.Exec(
		"insert into players (name, email, phone, password) values (?, ?, ?, ?)",
		player.Name, player.Email, player.Phone, player.Password)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	player.ID = uint64(id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (ps *PlayerService) UpdatePlayer(ctx context.Context) error {
	player := teamvite.PlayerFromContext(ctx)
	_, err := ps.db.Exec(
		"update players set name=?, email=?, phone=?, password=? where id = ?",
		player.Name, player.Email, player.Phone, player.Password, player.ID)
	return err
}

func (ps *PlayerService) UpdatePlayerTeam(ctx context.Context, playerTeam *teamvite.PlayerTeam) error {
	playerID := teamvite.UserIDFromContext(ctx)
	_, err := ps.db.Exec(
		"update players_teams set remind_email = ?, remind_sms = ? where player_id = ? and team_id = ?",
		playerTeam.RemindEmail, playerTeam.RemindSMS, playerID, playerTeam.Team.ID)
	return err
}

func (ps *PlayerService) ResetPassword(ctx context.Context, password string) error {
	player := teamvite.PlayerFromContext(ctx)
	pHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	player.Password = sql.NullString{String: string(pHash[:]), Valid: true}

	_, err = ps.db.Exec("update players set password = ? where id = ?", player.Password, player.ID)
	return err
}

func findPlayers(ctx context.Context, tx *sql.Tx, filter teamvite.PlayerFilter) (_ []*teamvite.Player, n int, err error) {
	var players []*teamvite.Player
	var query string
	var args []interface{}

	query = `
		select
			p.id, p.name, p.email, p.phone, p.password
		from players p
		where 1 = 1
	`

	if filter.ID != nil {
		query += " and p.id = ?"
		args = append(args, *filter.ID)
	}

	if filter.Name != nil {
		query += " and p.name like ?"
		args = append(args, *filter.Name)
	}

	if filter.TeamID != nil {
		query += " and id in (select player_id from players_teams where team_id = ?)"
		args = append(args, *filter.TeamID)
	}

	if filter.Email != "" {
		query += " and p.email = ?"
		args = append(args, filter.Email)
	}

	if filter.Phone != 0 {
		query += " and p.phone = ?"
		args = append(args, filter.Phone)
	}

	rows, err := tx.QueryContext(ctx, query+FormatLimitOffset(filter.Limit, filter.Offset), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var p teamvite.Player
		err := rows.Scan(&p.ID, &p.Name, &p.Email, &p.Phone, &p.Password)
		if err != nil {
			return nil, 0, err
		}
		players = append(players, &p)
	}

	return players, len(players), err
}
