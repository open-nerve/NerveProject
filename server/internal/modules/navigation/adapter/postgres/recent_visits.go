package postgresadapter

import (
	"context"
	"fmt"
	"math"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
)

// RecentVisits lists user's undeleted recent visits in workspace, of kind
// alone when kind is not nil, by visit, the newest first, then by id
// descending (recent_visits.sql; P3d spec 2.5).
func (s *Store) RecentVisits(ctx context.Context, user, workspace uuid.UUID, kind *domain.RecentVisitKind) ([]domain.RecentVisit, error) {
	var of *string
	if kind != nil {
		of = (*string)(kind)
	}
	rows, err := s.queries(ctx).ListRecentVisits(ctx, gen.ListRecentVisitsParams{UserID: user, WorkspaceID: workspace, EntityType: of})
	if err != nil {
		return nil, fmt.Errorf("list the recent visits of user %s in workspace %s: %w", user, workspace, err)
	}
	out := make([]domain.RecentVisit, len(rows))
	for i, r := range rows {
		out[i] = domain.RecentVisit{ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, UserID: r.UserID,
			Kind: domain.RecentVisitKind(r.EntityType), EntityID: r.EntityIdentifier, VisitedAt: r.VisitedAt}
	}
	return out, nil
}

// LockRecentVisits takes the transaction-level advisory lock of user's
// visits in workspace, held until the transaction ctx carries ends: a
// recording's last lock (spec 2.8). Its key is the statement's.
func (s *Store) LockRecentVisits(ctx context.Context, user, workspace uuid.UUID) error {
	if err := s.queries(ctx).LockRecentVisits(ctx, gen.LockRecentVisitsParams{UserID: user, WorkspaceID: workspace}); err != nil {
		return fmt.Errorf("lock the recent visits of user %s in workspace %s: %w", user, workspace, err)
	}
	return nil
}

// RecordRecentVisit stores v, made by by at its VisitedAt, or revisits its
// owner's undeleted visit of its target in its workspace: its visit and its
// last change stamped by by at v's VisitedAt, its id and creation kept
// (spec 2.3, 2.5). The use case checked every value, so a CHECK's or a
// foreign key's violation is a bug: an internal error.
func (s *Store) RecordRecentVisit(ctx context.Context, v domain.RecentVisit, by uuid.UUID) error {
	err := s.queries(ctx).RecordRecentVisit(ctx, gen.RecordRecentVisitParams{ID: v.ID, WorkspaceID: v.WorkspaceID, ProjectID: v.ProjectID, UserID: v.UserID,
		EntityType: string(v.Kind), EntityIdentifier: v.EntityID, At: v.VisitedAt, By: by})
	if err != nil {
		return fmt.Errorf("record the visit of %s %s by user %s: %w", v.Kind, v.EntityID, v.UserID, err)
	}
	return nil
}

// TrimRecentVisits deletes, for good, user's undeleted visits in workspace
// past the keep newest by visit, passing over one another transaction holds
// (spec 2.5, 2.8). keep is the cap (domain.RecentVisitCap), from 1 to
// 2147483647.
func (s *Store) TrimRecentVisits(ctx context.Context, user, workspace uuid.UUID, keep int) error {
	if keep < 1 || keep > math.MaxInt32 {
		return fmt.Errorf("the recent visits kept must be from 1 to 2147483647, got %d", keep)
	}
	err := s.queries(ctx).TrimRecentVisits(ctx, gen.TrimRecentVisitsParams{UserID: user, WorkspaceID: workspace, Keep: int32(keep)})
	if err != nil {
		return fmt.Errorf("trim the recent visits of user %s in workspace %s: %w", user, workspace, err)
	}
	return nil
}

// DeleteIssueRecentVisits soft-deletes, in the transaction ctx carries,
// every account's undeleted visits of the work item issue in workspace,
// locked in id order, deleted and last changed by by, nil for none, at at:
// the cascade of the work item's deletion (spec 2.6).
func (s *Store) DeleteIssueRecentVisits(ctx context.Context, workspace, issue uuid.UUID, at time.Time, by *uuid.UUID) error {
	err := s.queries(ctx).DeleteIssueRecentVisits(ctx, gen.DeleteIssueRecentVisitsParams{At: at, By: by, WorkspaceID: workspace, IssueID: issue})
	if err != nil {
		return fmt.Errorf("delete the recent visits of work item %s: %w", issue, err)
	}
	return nil
}
