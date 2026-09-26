package postgresadapter

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
)

// CreateAPIToken inserts t. The domain validated every value, so a CHECK
// violation is a bug: an internal error (500).
func (s *Store) CreateAPIToken(ctx context.Context, t app.NewAPIToken) error {
	err := s.queries(ctx).CreateAPIToken(ctx, gen.CreateAPITokenParams{
		ID: t.ID, UserID: t.UserID, TokenHash: t.TokenHash, Label: t.Label, Description: t.Description, ExpiredAt: t.ExpiredAt, Now: t.Now,
	})
	if err != nil {
		return fmt.Errorf("create API token: %w", err)
	}
	return nil
}

// ListAPITokens returns up to limit of userID's unrevoked tokens, newest
// first and then by id, after the row of after when it is not nil.
func (s *Store) ListAPITokens(ctx context.Context, userID uuid.UUID, after *domain.APITokenCursor, limit int) ([]domain.APIToken, error) {
	p := gen.ListAPITokensParams{UserID: userID, RowLimit: int32(limit)} // at most MaxPageSize + 1
	if after != nil {
		p.CursorCreatedAt, p.CursorID = &after.CreatedAt, &after.ID
	}
	rows, err := s.queries(ctx).ListAPITokens(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("list API tokens: %w", err)
	}
	tokens := make([]domain.APIToken, len(rows))
	for i, r := range rows {
		tokens[i] = domain.APIToken{
			ID: r.ID, Label: r.Label, Description: r.Description, ExpiredAt: r.ExpiredAt, LastUsed: r.LastUsed, CreatedAt: r.CreatedAt,
		}
	}
	return tokens, nil
}

// RevokeAPIToken soft-deletes token id of userID; false when userID has no
// such unrevoked token.
func (s *Store) RevokeAPIToken(ctx context.Context, id, userID uuid.UUID, now time.Time) (bool, error) {
	n, err := s.queries(ctx).RevokeAPIToken(ctx, gen.RevokeAPITokenParams{Now: now, UserID: userID, ID: id})
	if err != nil {
		return false, fmt.Errorf("revoke API token: %w", err)
	}
	return n == 1, nil
}

// APITokenByHash reads what authentication checks of the token with hash;
// app.ErrNotFound when there is none.
func (s *Store) APITokenByHash(ctx context.Context, hash []byte) (app.APITokenCredential, error) {
	r, err := s.queries(ctx).GetAPITokenByHash(ctx, hash)
	if err != nil {
		return app.APITokenCredential{}, notFound(err)
	}
	return app.APITokenCredential{
		ID: r.ID, UserID: r.UserID, ExpiredAt: r.ExpiredAt, LastUsed: r.LastUsed, Revoked: r.DeletedAt != nil, UserActive: r.UserActive,
	}, nil
}

// APITokenByID reads the same of token id; app.ErrNotFound when there is
// none.
func (s *Store) APITokenByID(ctx context.Context, id uuid.UUID) (app.APITokenCredential, error) {
	r, err := s.queries(ctx).GetAPITokenByID(ctx, id)
	if err != nil {
		return app.APITokenCredential{}, notFound(err)
	}
	return app.APITokenCredential{
		ID: r.ID, UserID: r.UserID, ExpiredAt: r.ExpiredAt, LastUsed: r.LastUsed, Revoked: r.DeletedAt != nil, UserActive: r.UserActive,
	}, nil
}

// TouchAPIToken sets token id's last_used to now when it is unset or older
// than staleBefore.
func (s *Store) TouchAPIToken(ctx context.Context, id uuid.UUID, now, staleBefore time.Time) error {
	if err := s.queries(ctx).TouchAPIToken(ctx, gen.TouchAPITokenParams{Now: now, ID: id, StaleBefore: staleBefore}); err != nil {
		return fmt.Errorf("touch API token: %w", err)
	}
	return nil
}
