package app_test

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
)

// userUpdate is one UpdateUser call.
type userUpdate struct {
	id    uuid.UUID
	patch domain.UserPatch
	now   time.Time
}

// profileUpdate is one UpdateProfile call.
type profileUpdate struct {
	userID uuid.UUID
	patch  domain.ProfilePatch
	now    time.Time
}

// fakeAccount is one account and its profile, in memory: it finds them by
// the account's id only. err, when set, is every call's error.
type fakeAccount struct {
	user           domain.User
	profile        domain.Profile
	err            error
	userUpdates    []userUpdate
	profileReads   []uuid.UUID
	profileUpdates []profileUpdate
}

func (f *fakeAccount) find(id uuid.UUID) error {
	switch {
	case f.err != nil:
		return f.err
	case id != f.user.ID:
		return app.ErrNotFound
	}
	return nil
}

func (f *fakeAccount) UpdateUser(_ context.Context, id uuid.UUID, p domain.UserPatch, now time.Time) (domain.User, error) {
	f.userUpdates = append(f.userUpdates, userUpdate{id, p, now})
	if err := f.find(id); err != nil {
		return domain.User{}, err
	}
	return f.user, nil
}

func (f *fakeAccount) GetProfile(_ context.Context, userID uuid.UUID) (domain.Profile, error) {
	f.profileReads = append(f.profileReads, userID)
	if err := f.find(userID); err != nil {
		return domain.Profile{}, err
	}
	return f.profile, nil
}

func (f *fakeAccount) UpdateProfile(_ context.Context, userID uuid.UUID, p domain.ProfilePatch, now time.Time) (domain.Profile, error) {
	f.profileUpdates = append(f.profileUpdates, profileUpdate{userID, p, now})
	if err := f.find(userID); err != nil {
		return domain.Profile{}, err
	}
	return f.profile, nil
}
