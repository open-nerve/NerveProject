package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Directory is what workspace.Provide offers the project module (M3 design
// 6.5): WorkspaceDirectory, the undeleted workspace a slug names, and
// WorkspaceMembers, the memberships a project write makes members from,
// locked (3.6 convention 3). It reads the store's tables under names of its
// own: the store's WorkspaceBySlug and ShareWorkspaceBySlug answer the
// workspace module's use cases.
type Directory struct {
	store *Store
}

// NewDirectory returns the directory over pool.
func NewDirectory(pool *pgxpool.Pool) *Directory {
	return &Directory{store: New(pool)}
}

// WorkspaceBySlug returns the undeleted workspace with slug, read without a
// lock; found is false when there is none.
func (d *Directory) WorkspaceBySlug(ctx context.Context, slug string) (w app.DirectoryEntry, found bool, err error) {
	r, err := d.store.queries(ctx).DirectoryWorkspace(ctx, slug)
	return directoryEntry(r.ID, r.Timezone, err)
}

// ShareWorkspaceBySlug returns the undeleted workspace with slug and locks
// its row FOR SHARE until the transaction ctx carries ends: the parent lock
// of a write that adds a project under it (M3 design 3.6 convention 2).
// found is false when there is none, also when it was deleted while the
// lock waited.
func (d *Directory) ShareWorkspaceBySlug(ctx context.Context, slug string) (w app.DirectoryEntry, found bool, err error) {
	r, err := d.store.queries(ctx).ShareDirectoryWorkspace(ctx, slug)
	return directoryEntry(r.ID, r.Timezone, err)
}

func directoryEntry(id uuid.UUID, timezone string, err error) (app.DirectoryEntry, bool, error) {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return app.DirectoryEntry{}, false, nil
	case err != nil:
		return app.DirectoryEntry{}, false, fmt.Errorf("find the workspace: %w", err)
	}
	return app.DirectoryEntry{ID: id, Timezone: timezone}, true, nil
}

// ShareMembers locks the undeleted memberships of userIDs in workspaceID,
// active or not, FOR SHARE in id order until the transaction ctx carries
// ends, and returns the roles of the active ones by account; an account
// without an active membership is not in the map (M3 design 3.6 convention
// 3).
func (d *Directory) ShareMembers(ctx context.Context, workspaceID uuid.UUID, userIDs []uuid.UUID) (map[uuid.UUID]shared.Role, error) {
	rows, err := d.store.queries(ctx).ShareMembers(ctx, gen.ShareMembersParams{WorkspaceID: workspaceID, UserIds: userIDs})
	if err != nil {
		return nil, fmt.Errorf("lock the workspace members: %w", err)
	}
	roles := map[uuid.UUID]shared.Role{}
	for _, r := range rows {
		if r.IsActive {
			roles[r.MemberID] = shared.Role(r.Role)
		}
	}
	return roles, nil
}
