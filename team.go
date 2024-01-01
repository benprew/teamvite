package teamvite

import (
	"context"
)

type Team struct {
	ID           uint64 `db:"id,primarykey,autoincrement" json:"id"`
	Name         string `db:"name,size:128" json:"name"`
	DivisionID   uint64 `db:"division_id" json:"division_id"`
	DivisionName string `db:"division_name" json:"division_name"`
}

func (t *Team) ItemID() uint64 {
	return t.ID
}

func (t *Team) ItemType() string {
	return "team"
}

type TeamService interface {
	// Retrieves a single Team by ID along with associated memberships. Only the
	// Team manager & members can see a Team. Returns ENOTFOUND if Team does not
	// exist or user does not have permission to view it.
	FindTeamByID(ctx context.Context, id uint64) (*Team, error)

	// Retrieves a list of Teams based on a filter. Only returns Teams
	// the user is on. Also returns a count of total matching Teams which may
	// differ from the number of returned Teams if the "Limit" field is set.
	FindTeams(ctx context.Context, filter TeamFilter) ([]*Team, int, error)

	// Creates a new Team and assigns the current user as the owner.  The owner
	// will automatically be added as a member of the new Team.
	CreateTeam(ctx context.Context, team *Team) error

	// Updates an existing Team by ID. Only the Team manager or admin can update a
	// Team.  Returns the new Team state even if there was an error during update.
	//
	// Returns ENOTFOUND if Team does not exist. Returns EUNAUTHORIZED if user is
	// not the Team manager.
	// UpdateTeam(ctx context.Context, id int, upd TeamUpdate) (*Team, error)

	// Returns true if the current user is a manager of the Team.
	IsManagedBy(ctx context.Context, team *Team) bool

	// Adds the user in the context to the team
	AddPlayer(ctx context.Context, team *Team) error

	// Removes the user in the context from the team
	RemovePlayer(ctx context.Context, team *Team) error
}

type TeamFilter struct {
	// Filtering fields.
	ID           uint64  `json:"id"`
	Name         *string `json:"name"`
	DivisionID   uint64  `json:"division_id"`
	DivisionName string  `json:"division_name"`

	// Restrict to subset of range.
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}
