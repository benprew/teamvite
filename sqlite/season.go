package sqlite

import (
	"context"
	"database/sql"
	"strings"

	"github.com/benprew/teamvite"
)

type SeasonService struct {
	db *sql.DB
}

// Ensure service implements interface.
var _ teamvite.SeasonService = (*SeasonService)(nil)

// NewSeasonService returns a new instance of SeasonService.
func NewSeasonService(db *sql.DB) *SeasonService {
	return &SeasonService{db: db}
}

func (s *SeasonService) FindSeasons(ctx context.Context, filter teamvite.SeasonFilter) ([]*teamvite.Season, int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, err
	}
	defer tx.Rollback()

	seasons, n, err := findSeasons(ctx, tx, filter)
	return seasons, n, err
}

func findSeasons(ctx context.Context, tx *sql.Tx, filter teamvite.SeasonFilter) (_ []*teamvite.Season, n int, err error) {
	var seasons []*teamvite.Season

	// Build WHERE clause.
	var where []string
	var args []interface{}
	if filter.ID != nil {
		where = append(where, "id = ?")
		args = append(args, *filter.ID)
	}
	if filter.Name != "" {
		where = append(where, "name like ?")
		args = append(args, "%"+filter.Name+"%")
	}

	// Build query.
	query := `
		select
			id,
			name
			COUNT(*) OVER()
		from seasons
	`
	if len(where) > 0 {
		query += "where " + strings.Join(where, " and ")
	}
	query += " order by name"

	// Execute query.
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	// Iterate over rows.
	for rows.Next() {
		var season teamvite.Season
		if err := rows.Scan(
			&season.ID,
			&season.Name,
			&n,
		); err != nil {
			return nil, 0, err
		}
		seasons = append(seasons, &season)
	}

	return seasons, n, err
}
