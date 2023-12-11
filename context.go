package teamvite

import "context"

// contextKey represents an internal key for adding context fields.
// This is considered best practice as it prevents other packages from
// interfering with our context keys.
type contextKey int

// List of context keys.
// These are used to store request-scoped information.
const (
	// Stores the current logged in player in the context.
	playerContextKey = contextKey(iota + 1)

	// Stores the "flash" in the context. This is a term used in web development
	// for a message that is passed from one request to the next for informational
	// purposes. This could be moved into the "http" package as it is only HTTP
	// related but both the "http" and "http/html" packages use it so it is
	// easier to move it to the root.
	flashContextKey
)

// NewContextWithplayer returns a new context with the given player.
func NewContextWithplayer(ctx context.Context, player *Player) context.Context {
	return context.WithValue(ctx, playerContextKey, player)
}

// playerFromContext returns the current logged in player.
func PlayerFromContext(ctx context.Context) *Player {
	player, _ := ctx.Value(playerContextKey).(*Player)
	return player
}

// playerIDFromContext is a helper function that returns the ID of the current
// logged in player. Returns zero if no player is logged in.
func PlayerIDFromContext(ctx context.Context) uint64 {
	if player := PlayerFromContext(ctx); player != nil {
		return player.ID
	}
	return 0
}

// NewContextWithFlash returns a new context with the given flash value.
func NewContextWithFlash(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, flashContextKey, v)
}

// FlashFromContext returns the flash value for the current request.
func FlashFromContext(ctx context.Context) string {
	v, _ := ctx.Value(flashContextKey).(string)
	return v
}
