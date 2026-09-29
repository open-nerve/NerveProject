package postgresadapter

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CreateWorkspace inserts w and returns it as stored. A slug an undeleted
// workspace has is domain.ErrSlugTaken. The domain checked every value, so a
// CHECK violation is a bug: an internal error (500), not a domain error.
func (s *Store) CreateWorkspace(ctx context.Context, w app.WorkspaceRow) (domain.Workspace, error) {
	row, err := s.queries(ctx).CreateWorkspace(ctx, gen.CreateWorkspaceParams{
		ID: w.ID, Name: w.Name, Slug: w.Slug, OrganizationSize: w.OrganizationSize, Timezone: w.Timezone,
		CreatedBy: &w.CreatedBy, Now: w.Now,
	})
	switch {
	case uniqueViolation(err, "workspaces_slug_key"):
		return domain.Workspace{}, domain.ErrSlugTaken
	case err != nil:
		return domain.Workspace{}, fmt.Errorf("create workspace: %w", err)
	}
	return domain.Workspace{
		ID: row.ID, Name: row.Name, Slug: row.Slug, OrganizationSize: row.OrganizationSize, Timezone: row.Timezone,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}, nil
}

// CreateMember inserts m, active.
func (s *Store) CreateMember(ctx context.Context, m app.MemberRow) error {
	err := s.queries(ctx).CreateMember(ctx, gen.CreateMemberParams{
		ID: m.ID, WorkspaceID: m.WorkspaceID, MemberID: m.MemberID, Role: int16(m.Role), CreatedBy: &m.CreatedBy, Now: m.Now,
	})
	if err != nil {
		return fmt.Errorf("create workspace member: %w", err)
	}
	return nil
}

// ListWorkspaces returns the undeleted workspaces of which userID is an
// active member, with his role and the number of active members, by name,
// then id.
func (s *Store) ListWorkspaces(ctx context.Context, userID uuid.UUID) ([]domain.Workspace, error) {
	rows, err := s.queries(ctx).ListWorkspaces(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}
	out := make([]domain.Workspace, len(rows))
	for i, r := range rows {
		out[i] = domain.Workspace{
			ID: r.ID, Name: r.Name, Slug: r.Slug, OrganizationSize: r.OrganizationSize, Timezone: r.Timezone,
			Role: shared.Role(r.Role), TotalMembers: int(r.TotalMembers), CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		}
	}
	return out, nil
}

// WorkspaceBySlug returns the undeleted workspace with slug and its number
// of active members, without a role; app.ErrNotFound when there is none.
func (s *Store) WorkspaceBySlug(ctx context.Context, slug string) (domain.Workspace, error) {
	r, err := s.queries(ctx).WorkspaceBySlug(ctx, slug)
	if err != nil {
		return domain.Workspace{}, notFound(err)
	}
	return domain.Workspace{
		ID: r.ID, Name: r.Name, Slug: r.Slug, OrganizationSize: r.OrganizationSize, Timezone: r.Timezone,
		TotalMembers: int(r.TotalMembers), CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}, nil
}

// UpdateWorkspace applies p to the workspace id, by the account by at now,
// and returns it as stored with its number of active members, without a
// role. The caller holds the workspace's lock, so the row is there; its
// absence is an error, not app.ErrNotFound. The domain checked every value.
func (s *Store) UpdateWorkspace(ctx context.Context, id uuid.UUID, p domain.WorkspacePatch, by uuid.UUID, now time.Time) (domain.Workspace, error) {
	r, err := s.queries(ctx).UpdateWorkspace(ctx, gen.UpdateWorkspaceParams{
		SetName: p.Name != nil, Name: deref(p.Name),
		SetOrganizationSize: p.OrganizationSize != nil, OrganizationSize: deref(p.OrganizationSize),
		SetTimezone: p.Timezone != nil, Timezone: deref(p.Timezone),
		UpdatedBy: &by, Now: now, ID: id,
	})
	if err != nil {
		return domain.Workspace{}, fmt.Errorf("update workspace: %w", err)
	}
	return domain.Workspace{
		ID: r.ID, Name: r.Name, Slug: r.Slug, OrganizationSize: r.OrganizationSize, Timezone: r.Timezone,
		TotalMembers: int(r.TotalMembers), CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}, nil
}

// deref is the value p points at, or the zero value for nil.
func deref[T any](p *T) T {
	var zero T
	if p == nil {
		return zero
	}
	return *p
}

// SlugTaken reports whether an undeleted workspace has slug.
func (s *Store) SlugTaken(ctx context.Context, slug string) (bool, error) {
	taken, err := s.queries(ctx).SlugTaken(ctx, slug)
	if err != nil {
		return false, fmt.Errorf("check slug: %w", err)
	}
	return taken, nil
}
