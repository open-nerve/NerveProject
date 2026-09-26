package app_test

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
)

// fakeAdmin is the ports of the server administrator's commands over one
// account, userID with the address email, and the writes of fakeCredentials,
// logging every call to the same log. A second account has taken.
type fakeAdmin struct {
	*fakeCredentials
	tokens      int   // RevokeAllAPITokens revokes this many
	usable      int   // CountUsableAPITokens counts this many
	activateErr error // ActivateUser fails with it
}

const takenEmail = "bob@corp.com"

func newFakeAdmin(log *callLog) *fakeAdmin {
	return &fakeAdmin{fakeCredentials: &fakeCredentials{log: log, email: "alice@corp.com"}, tokens: 3, usable: 2}
}

func (f *fakeAdmin) LockAccount(ctx context.Context, email string) (uuid.UUID, error) {
	f.log.add(ctx, "lock "+email)
	if email != f.email {
		return uuid.Nil(), app.ErrNotFound
	}
	return userID, nil
}

func (f *fakeAdmin) ChangeEmail(ctx context.Context, id uuid.UUID, email string, now time.Time) error {
	if email == takenEmail {
		return domain.ErrEmailTaken
	}
	f.log.add(ctx, "email "+id.String()+" "+email)
	f.writtenAt = append(f.writtenAt, now)
	return nil
}

func (f *fakeAdmin) ActivateUser(ctx context.Context, id uuid.UUID, now time.Time) error {
	if f.activateErr != nil {
		return f.activateErr
	}
	f.log.add(ctx, "activate "+id.String())
	f.writtenAt = append(f.writtenAt, now)
	return nil
}

func (f *fakeAdmin) RevokeAllAPITokens(ctx context.Context, userID uuid.UUID, now time.Time) (int, error) {
	f.log.add(ctx, "revoke the tokens of "+userID.String())
	f.writtenAt = append(f.writtenAt, now)
	return f.tokens, nil
}

func (f *fakeAdmin) CountUsableAPITokens(ctx context.Context, userID uuid.UUID, now time.Time) (int, error) {
	f.log.add(ctx, "count the tokens of "+userID.String())
	f.writtenAt = append(f.writtenAt, now)
	return f.usable, nil
}
