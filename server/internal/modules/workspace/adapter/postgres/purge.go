package postgresadapter

import (
	"context"
	"fmt"
	"math"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
)

// PurgeWorkspaceUserProperties commits one bounded batch on the pool, independently of
// any request transaction carried in ctx.
func (s *Store) PurgeWorkspaceUserProperties(ctx context.Context, b app.PurgeBatch) ([]uuid.UUID, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return nil, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	ids, err := gen.New(s.pool).PurgeWorkspaceUserProperties(ctx, gen.PurgeWorkspaceUserPropertiesParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return nil, fmt.Errorf("purge workspace_user_properties: %w", err)
	}
	return ids, nil
}

// PurgeWorkspaceMemberInvites commits one bounded batch on the pool, independently of
// any request transaction carried in ctx.
func (s *Store) PurgeWorkspaceMemberInvites(ctx context.Context, b app.PurgeBatch) ([]uuid.UUID, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return nil, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	ids, err := gen.New(s.pool).PurgeWorkspaceMemberInvites(ctx, gen.PurgeWorkspaceMemberInvitesParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return nil, fmt.Errorf("purge workspace_member_invites: %w", err)
	}
	return ids, nil
}

// PurgeWorkspaceMembers commits one bounded batch on the pool, independently of
// any request transaction carried in ctx.
func (s *Store) PurgeWorkspaceMembers(ctx context.Context, b app.PurgeBatch) ([]uuid.UUID, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return nil, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	ids, err := gen.New(s.pool).PurgeWorkspaceMembers(ctx, gen.PurgeWorkspaceMembersParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return nil, fmt.Errorf("purge workspace_members: %w", err)
	}
	return ids, nil
}

// PurgeWorkspaces commits one bounded batch on the pool, independently of
// any request transaction carried in ctx.
func (s *Store) PurgeWorkspaces(ctx context.Context, b app.PurgeBatch) ([]uuid.UUID, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return nil, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	ids, err := gen.New(s.pool).PurgeWorkspaces(ctx, gen.PurgeWorkspacesParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return nil, fmt.Errorf("purge workspaces: %w", err)
	}
	return ids, nil
}
