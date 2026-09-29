package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
)

// The parent locks of the writes on a workspace (M3 design 3.6 convention
// 2). Each locks the undeleted workspace row until the transaction ctx
// carries ends and returns the workspace's id; app.ErrNotFound when there is
// none, also when it was deleted while the lock waited. Outside a
// transaction the lock would end with its statement: call them inside one.

// LockWorkspaceBySlug locks the workspace with slug FOR NO KEY UPDATE: for
// a write of the workspace row itself or of a membership.
func (s *Store) LockWorkspaceBySlug(ctx context.Context, slug string) (uuid.UUID, error) {
	id, err := s.queries(ctx).LockWorkspaceBySlug(ctx, slug)
	return lockedWorkspace(id, err)
}

// ShareWorkspaceBySlug locks the workspace with slug FOR SHARE: for a write
// that adds or changes a row under the workspace.
func (s *Store) ShareWorkspaceBySlug(ctx context.Context, slug string) (uuid.UUID, error) {
	id, err := s.queries(ctx).ShareWorkspaceBySlug(ctx, slug)
	return lockedWorkspace(id, err)
}

func lockedWorkspace(id uuid.UUID, err error) (uuid.UUID, error) {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return uuid.UUID{}, app.ErrNotFound
	case err != nil:
		return uuid.UUID{}, fmt.Errorf("lock the workspace row: %w", err)
	}
	return id, nil
}
