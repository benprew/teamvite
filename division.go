package teamvite

import "context"

type Division struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
}

func (d *Division) ItemID() uint64 {
	return d.ID
}

func (p *Division) ItemType() string {
	return "division"
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
