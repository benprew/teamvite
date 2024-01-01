package teamvite

import "context"

type Division struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type DivisionService interface {
	FindDivisionByID(ctx context.Context, id uint64) (*Division, error)

	// Retrieves a list of Divisions based on a filter.
	FindDivisions(ctx context.Context, filter DivisionFilter) ([]*Division, int, error)
}

type DivisionFilter struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`

	// Restrict to subset of range.
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}
