package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/benprew/teamvite"
)

type TeamService struct {
	db *sql.DB
}

// Ensure service implements interface.
var _ teamvite.TeamService = (*TeamService)(nil)

func NewTeamService(db *sql.DB) *TeamService {
	return &TeamService{db: db}
}

func (s *TeamService) FindTeamByID(ctx context.Context, id uint64) (*teamvite.Team, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	teams, _, err := findTeams(ctx, tx, teamvite.TeamFilter{ID: id})
	if err != nil {
		return nil, err
	}
	if len(teams) == 0 {
		return nil, teamvite.Errorf(
			teamvite.ENOTFOUND,
			fmt.Sprintf("team not found: %v", id),
		)
	}
	return teams[0], nil
}

func (s *TeamService) FindTeams(ctx context.Context, filter teamvite.TeamFilter) ([]*teamvite.Team, int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, err
	}
	defer tx.Rollback()

	// Fetch list of matching Team objects.
	teams, n, err := findTeams(ctx, tx, filter)
	return teams, n, err
}

func (s *TeamService) CreateTeam(ctx context.Context, team *teamvite.Team) error {
	tx, err := s.db.BeginTx(ctx, nil)

	result, err := tx.ExecContext(ctx, `
			insert into teams (name, division_id) values (?, ?)
		`,
		team.Name, team.DivisionID)
	if err != nil {
		return FormatError(err)
	}

	// Read back new Game ID into caller argument.
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	team.ID = uint64(id)
	return nil
}

func (s *TeamService) IsManagedBy(ctx context.Context, team *teamvite.Team) bool {
	var isMgr bool
	result, _ := s.db.Query(
		"select is_manager from players_teams where player_id = ? and team_id = ?",
		teamvite.UserIDFromContext(ctx),
		team.ID,
	)
	result.Scan(&isMgr)

	return isMgr
}

func (s *TeamService) AddPlayer(ctx context.Context, team *teamvite.Team) error {
	_, err := s.db.Exec(
		"insert into players_teams (player_id, team_id) values (?, ?)",
		teamvite.UserIDFromContext(ctx),
		team.ID)

	return FormatError(err)
}

func (s *TeamService) RemovePlayer(ctx context.Context, team *teamvite.Team) error {
	_, err := s.db.Exec(
		"delete from players_teams where player_id = ? and team_id = ?",
		teamvite.UserIDFromContext(ctx),
		team.ID)
	return FormatError(err)
}

func findTeams(ctx context.Context, tx *sql.Tx, filter teamvite.TeamFilter) (_ []*teamvite.Team, n int, err error) {
	var teams []*teamvite.Team
	var query string
	var args []interface{}

	query = `
		select
			t.id, t.name, t.division_id
		from teams t
		where 1 = 1
	`

	if filter.ID != 0 {
		query += " and t.id = ?"
		args = append(args, filter.ID)
	}

	if filter.Name != nil {
		query += " and t.name like ?"
		args = append(args, *filter.Name)
	}

	if filter.DivisionID != 0 {
		query += " and t.division_id = ?"
		args = append(args, filter.DivisionID)
	}

	rows, err := tx.QueryContext(ctx, query+FormatLimitOffset(filter.Limit, filter.Offset), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var t teamvite.Team
		err := rows.Scan(&t.ID, &t.Name, &t.DivisionID)
		if err != nil {
			return nil, 0, err
		}
		teams = append(teams, &t)
	}

	return teams, len(teams), nil
}
