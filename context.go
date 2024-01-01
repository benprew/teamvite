package teamvite

import "context"

// request context consisting of a player and a flash message.

// contextKey represents an internal key for adding context fields.
// This is considered best practice as it prevents other packages from
// interfering with our context keys.
type contextKey int

// List of context keys.
// These are used to store request-scoped information.
const (
	// Stores the current logged in player in the context.
	userContextKey = contextKey(iota + 1)

	// Stores the "flash" in the context. This is a term used in web development
	// for a message that is passed from one request to the next for informational
	// purposes. This could be moved into the "http" package as it is only HTTP
	// related but both the "http" and "http/html" packages use it so it is
	// easier to move it to the root.
	flashContextKey

	// Stores the template to be rendered in the context.
	templateKey

	// Store the domain models in the context.
	// The website is mostly built around CRUD-lite operations on these domain
	// models
	playerKey
	teamKey
	gameKey
)

// NewContextWithUser returns a new context with the given player.
func NewContextWithUser(ctx context.Context, player *Player) context.Context {
	return context.WithValue(ctx, userContextKey, player)
}

// playerFromContext returns the current logged in player.
func UserFromContext(ctx context.Context) *Player {
	player, _ := ctx.Value(userContextKey).(*Player)
	return player
}

// playerIDFromContext is a helper function that returns the ID of the current
// logged in player. Returns zero if no player is logged in.
func UserIDFromContext(ctx context.Context) uint64 {
	if player := UserFromContext(ctx); player != nil {
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

// Domain model contexts

func NewContextWithPlayer(ctx context.Context, template string, player *Player) context.Context {
	ctx = context.WithValue(ctx, playerKey, player)
	ctx = context.WithValue(ctx, templateKey, template)
	return ctx
}

func NewContextWithTeam(ctx context.Context, template string, team *Team) context.Context {
	ctx = context.WithValue(ctx, teamKey, team)
	ctx = context.WithValue(ctx, templateKey, template)
	return ctx
}

func NewContextWithGame(ctx context.Context, template string, game *Game) context.Context {
	ctx = context.WithValue(ctx, gameKey, game)
	ctx = context.WithValue(ctx, templateKey, template)
	return ctx
}

func TemplateFromContext(ctx context.Context) string {
	template, _ := ctx.Value(templateKey).(string)
	return template
}

func TeamFromContext(ctx context.Context) *Team {
	team, _ := ctx.Value(teamKey).(*Team)
	return team
}

func GameFromContext(ctx context.Context) *Game {
	game, _ := ctx.Value(gameKey).(*Game)
	return game
}

func PlayerFromContext(ctx context.Context) *Player {
	player, _ := ctx.Value(playerKey).(*Player)
	return player
}
