package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListRecentVisitsDeps are what the list of one's recent visits needs: the
// workspace module's directory, the Authorizer, the visits, the reads of
// their targets and the read-only snapshot transactions.
type ListRecentVisitsDeps struct {
	Workspaces Workspaces
	Authorizer shared.Authorizer
	Visits     RecentVisitReader
	Targets    RecentTargets
	Tx         shared.TxManager
}

// ListRecentVisits lists one's recent visits in a workspace
// (listRecentVisits).
type ListRecentVisits struct{ d ListRecentVisitsDeps }

// NewListRecentVisits returns the use case over d.
func NewListRecentVisits(d ListRecentVisitsDeps) *ListRecentVisits { return &ListRecentVisits{d: d} }

// Execute lists the caller's recent visits in the workspace slug names, of
// kind alone when kind is not nil (P3d spec 2.4, 2.5, 2.7, R8). Its
// refusals, in order: a kind not one of the two, a 422 before any read
// (domain.CheckRecentVisitKind); a slug of none and a workspace he is not
// an active member of, workspace.not_found. Then, in one read-only
// snapshot transaction and without a lock, his undeleted visits there, by
// visit, the newest first, then by id descending, and what he sees of their
// targets (RecentTargets.sight); it answers the first
// domain.RecentVisitCap of those the list lists (recentSight.visible), each
// with its target's project and work item now. It writes nothing.
func (u *ListRecentVisits) Execute(ctx context.Context, slug string, kind *domain.RecentVisitKind) ([]ShownRecentVisit, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	if kind != nil {
		if err := domain.CheckRecentVisitKind(*kind); err != nil {
			return nil, err
		}
	}
	var out []ShownRecentVisit
	err = u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		workspaceID, err := decideWorkspace(ctx, u.d.Workspaces, u.d.Authorizer, actor, domain.ActionRecentVisitRead, slug)
		if err != nil {
			return err
		}
		visits, err := u.d.Visits.RecentVisits(ctx, actor.UserID, workspaceID, kind)
		if err != nil || len(visits) == 0 {
			return err
		}
		var issues, projects []uuid.UUID
		for _, v := range visits {
			projects = append(projects, v.ProjectID)
			if v.Kind == domain.RecentVisitIssue {
				issues = append(issues, v.EntityID)
			}
		}
		s, err := u.d.Targets.sight(ctx, actor, workspaceID, issues, projects)
		if err != nil {
			return err
		}
		out = s.visible(visits)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
