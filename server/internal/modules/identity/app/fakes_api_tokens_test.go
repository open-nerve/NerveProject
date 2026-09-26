package app_test

import (
	"bytes"
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
)

// touch is one TouchAPIToken call.
type touch struct {
	id               uuid.UUID
	now, staleBefore time.Time
}

// revocation is one RevokeAPIToken call.
type revocation struct {
	id, userID uuid.UUID
	now        time.Time
}

// fakeAPITokens is the token ports, in memory. It finds a credential by the
// hash and by the id it holds, so a lookup by any other key fails; it lists
// at most limit of its rows.
type fakeAPITokens struct {
	log        *callLog
	credential app.APITokenCredential
	hash       []byte // the hash that finds credential
	readErr    error
	touches    []touch
	touchErr   error
	created    []app.NewAPIToken
	rows       []domain.APIToken
	listed     []listCall
	revoked    []revocation
	revokeOK   bool
	revokeErr  error
}

// listCall is one ListAPITokens call.
type listCall struct {
	userID uuid.UUID
	after  *domain.APITokenCursor
	limit  int
}

func (f *fakeAPITokens) APITokenByHash(ctx context.Context, hash []byte) (app.APITokenCredential, error) {
	f.log.add(ctx, "token by hash")
	if f.readErr != nil {
		return app.APITokenCredential{}, f.readErr
	}
	if !bytes.Equal(hash, f.hash) {
		return app.APITokenCredential{}, app.ErrNotFound
	}
	return f.credential, nil
}

func (f *fakeAPITokens) APITokenByID(ctx context.Context, id uuid.UUID) (app.APITokenCredential, error) {
	f.log.add(ctx, "token "+id.String())
	if f.readErr != nil {
		return app.APITokenCredential{}, f.readErr
	}
	if id != f.credential.ID {
		return app.APITokenCredential{}, app.ErrNotFound
	}
	return f.credential, nil
}

func (f *fakeAPITokens) TouchAPIToken(_ context.Context, id uuid.UUID, now, staleBefore time.Time) error {
	f.touches = append(f.touches, touch{id, now, staleBefore})
	return f.touchErr
}

func (f *fakeAPITokens) CreateAPIToken(ctx context.Context, n app.NewAPIToken) error {
	f.log.add(ctx, "insert "+n.ID.String())
	f.created = append(f.created, n)
	return nil
}

func (f *fakeAPITokens) ListAPITokens(_ context.Context, userID uuid.UUID, after *domain.APITokenCursor, limit int) ([]domain.APIToken, error) {
	f.listed = append(f.listed, listCall{userID, after, limit})
	return f.rows[:min(limit, len(f.rows))], nil
}

func (f *fakeAPITokens) RevokeAPIToken(_ context.Context, id, userID uuid.UUID, now time.Time) (bool, error) {
	f.revoked = append(f.revoked, revocation{id, userID, now})
	return f.revokeOK, f.revokeErr
}
