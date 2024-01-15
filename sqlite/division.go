package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/benprew/teamvite"
)

type DivisionService struct {
	db *sql.DB
}

// Ensure service implements interface.
var _ teamvite.DivisionService = (*DivisionService)(nil)

// NewDivisionService returns a new instance of DivisionService.
func NewDivisionService(db *sql.DB) *DivisionService {
	return &DivisionService{db: db}
}

func (s *DivisionService) FindDivisionByID(ctx context.Context, id uint64) (*teamvite.Division, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	Divisions, _, err := findDivisions(ctx, tx, teamvite.DivisionFilter{ID: id})
	if err != nil {
		return nil, err
	}
	if len(Divisions) == 0 {
		return nil, &teamvite.Error{
			Code:    teamvite.ENOTFOUND,
			Message: fmt.Sprintf("Division not found: %v", id),
		}
	}
	return Divisions[0], nil
}

// FindDivisions retrieves a list of Divisions based on a filter.
//
// Also returns a count of total matching Divisions which may different from the
// number of returned Divisions if the  "Limit" field is set.
func (s *DivisionService) FindDivisions(ctx context.Context, filter teamvite.DivisionFilter) ([]*teamvite.Division, int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, err
	}
	defer tx.Rollback()

	// Fetch list of matching Division objects.
	Divisions, n, err := findDivisions(ctx, tx, filter)
	return Divisions, n, err
}

func findDivisions(ctx context.Context, tx *sql.Tx, filter teamvite.DivisionFilter) (_ []*teamvite.Division, n int, err error) {
	divisions := make([]*teamvite.Division, 0)
	var query string
	var args []interface{}

	query = `
		select
			id, name
		from divisions
		where 1 = 1
	`

	if filter.ID != 0 {
		query += " and id = ?"
		args = append(args, filter.ID)
	}

	if filter.Name != "" {
		query += " and name = ?"
		args = append(args, filter.Name)
	}

	query += " order by name"

	rows, err := tx.QueryContext(ctx, query+FormatLimitOffset(filter.Limit, filter.Offset), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var d teamvite.Division
		err := rows.Scan(&d.ID, &d.Name)
		if err != nil {
			return nil, 0, err
		}
		divisions = append(divisions, &d)
	}

	return divisions, len(divisions), nil
}
