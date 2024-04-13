package teamvite

import "context"

type Season struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// interface for the season service
type SeasonService interface {
	FindSeasons(ctx context.Context, filter SeasonFilter) ([]*Season, int, error)
}

// filter for the season service
type SeasonFilter struct {
	ID   *uint64
	Name string
}
