package app

import (
	"context"
	"errors"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// UpdateMe changes the caller's names and time zone: PATCH /api/v0/me. It is
// one statement, no transaction (M2 design 6.4).
type UpdateMe struct {
	users UserUpdater
	clock Clock
}

// NewUpdateMe returns the use case.
func NewUpdateMe(users UserUpdater, clock Clock) *UpdateMe {
	return &UpdateMe{users: users, clock: clock}
}

// Execute checks p and applies it to the caller's account at the clock's
// now; the fields p leaves nil stay as they are.
func (u *UpdateMe) Execute(ctx context.Context, p domain.UserPatch) (domain.User, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.User{}, err
	}
	if err := domain.CheckUserPatch(p); err != nil {
		return domain.User{}, err
	}
	user, err := u.users.UpdateUser(ctx, actor.UserID, p, u.clock.Now())
	if errors.Is(err, ErrNotFound) {
		// Authentication found the account a moment ago; it is gone now.
		return domain.User{}, shared.Unauthenticated()
	}
	return user, err
}
