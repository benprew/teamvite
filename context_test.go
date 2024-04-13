package teamvite

import (
	"context"
	"reflect"
	"testing"
)

func TestUserFromContext(t *testing.T) {
	// Create a Player
	expectedPlayer := Player{Name: "Test Player"}

	// Create a context with the Player
	ctx := NewContextWithUser(context.Background(), &expectedPlayer)

	// Call UserFromContext
	player := UserFromContext(ctx)

	// Check if the returned *Player is as expected
	if !reflect.DeepEqual(player, &expectedPlayer) {
		t.Errorf("UserFromContext = %v; want %v", player, expectedPlayer)
	}
}

func TestUserFromContextWithNilUser(t *testing.T) {
	// Create a Player
	var expectedPlayer *Player

	// Create a context with the Player
	ctx := NewContextWithUser(context.Background(), expectedPlayer)

	// Call UserFromContext
	player := UserFromContext(ctx)

	// Check if the returned *Player is as expected
	if !reflect.DeepEqual(player, expectedPlayer) {
		t.Errorf("UserFromContext = %v; want %v", player, expectedPlayer)
	}
}
