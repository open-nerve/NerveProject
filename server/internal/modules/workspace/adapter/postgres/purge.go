package postgresadapter

import (
	"context"
	"fmt"
	"math"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
)

// PurgeWorkspaceUserProperties commits one bounded batch on the pool, independently of
// any request transaction carried in ctx.
func (s *Store) PurgeWorkspaceUserProperties(ctx context.Context, b app.PurgeBatch) (app.PurgePage, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	row, err := gen.New(s.pool).PurgeWorkspaceUserProperties(ctx, gen.PurgeWorkspaceUserPropertiesParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge workspace_user_properties: %w", err)
	}
	return app.PurgePage{Scanned: int(row.Scanned), Deleted: int(row.Deleted), After: row.AfterID}, nil
}

// PurgeWorkspaceMemberInvites commits one bounded batch on the pool, independently of
// any request transaction carried in ctx.
func (s *Store) PurgeWorkspaceMemberInvites(ctx context.Context, b app.PurgeBatch) (app.PurgePage, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	row, err := gen.New(s.pool).PurgeWorkspaceMemberInvites(ctx, gen.PurgeWorkspaceMemberInvitesParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge workspace_member_invites: %w", err)
	}
	return app.PurgePage{Scanned: int(row.Scanned), Deleted: int(row.Deleted), After: row.AfterID}, nil
}

// PurgeWorkspaceMembers commits one bounded batch on the pool, independently of
// any request transaction carried in ctx.
func (s *Store) PurgeWorkspaceMembers(ctx context.Context, b app.PurgeBatch) (app.PurgePage, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	row, err := gen.New(s.pool).PurgeWorkspaceMembers(ctx, gen.PurgeWorkspaceMembersParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge workspace_members: %w", err)
	}
	return app.PurgePage{Scanned: int(row.Scanned), Deleted: int(row.Deleted), After: row.AfterID}, nil
}

// PurgeWorkspaces scans candidates before deleting, retaining blocked progress.
func (s *Store) PurgeWorkspaces(ctx context.Context, b app.PurgeBatch) (app.PurgePage, error) {
	q := gen.New(s.pool)
	page, err := postgres.PurgeProtected(ctx, protectedPurgeQueries{
		candidates: func(ctx context.Context, cutoff time.Time, after uuid.UUID, limit int) ([]uuid.UUID, error) {
			return q.PurgeWorkspacesCandidates(ctx, gen.PurgeWorkspacesCandidatesParams{Cutoff: &cutoff, AfterID: after, BatchLimit: int32(limit)})
		},
		remove: func(ctx context.Context, cutoff time.Time, ids []uuid.UUID) ([]uuid.UUID, error) {
			return q.DeletePurgeWorkspaces(ctx, gen.DeletePurgeWorkspacesParams{Cutoff: &cutoff, Ids: ids})
		},
	}, s.protected, b.Cutoff, b.After, b.Limit)
	if err != nil {
		return app.PurgePage(page), fmt.Errorf("purge workspaces: %w", err)
	}
	return app.PurgePage(page), nil
}
