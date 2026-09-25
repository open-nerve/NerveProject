package app

import (
	"context"
	"uuid"
)

// accounts creates an account with its default profile: the step that
// registration and, from M2/P3, `nerve users create` share (M2 design 6.2).
// Run it inside the caller's transaction.
type accounts struct {
	users    UserCreator
	profiles ProfileCreator
}

func (a accounts) create(ctx context.Context, u NewUser) error {
	if err := a.users.CreateUser(ctx, u); err != nil {
		return err
	}
	return a.profiles.CreateDefaultProfile(ctx, uuid.NewV7(), u.ID, u.Now)
}
