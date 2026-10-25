package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListIssueRelationsDeps supplies source authorization, ordinary project scope
// and one independent read snapshot for the whole relation list.
type ListIssueRelationsDeps struct {
	Locks     Locks
	Relations RelationLister
	Projects  VisibleProjects
	ReadTx    shared.TxManager
}

// ListIssueRelations lists a work item's relations in their creation order.
type ListIssueRelations struct{ d ListIssueRelationsDeps }

func NewListIssueRelations(d ListIssueRelationsDeps) *ListIssueRelations {
	return &ListIssueRelations{d: d}
}

// Execute authorizes the source explicitly, including an archived source
// project, then filters ordinary cross-project targets within the same snapshot.
// Each target retains seen's visible child count and hidden-parent projection.
func (u *ListIssueRelations) Execute(ctx context.Context, id uuid.UUID) ([]IssueRelation, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	var out []IssueRelation
	err = u.d.ReadTx.WithinTx(ctx, func(ctx context.Context) error {
		i, sourceGrant, err := u.d.Locks.see(ctx, actor, id, domain.ActionRead)
		if err != nil {
			return err
		}
		relations, err := u.d.Relations.Relations(ctx, i.ID)
		if err != nil || len(relations) == 0 {
			return err
		}
		ids := make([]uuid.UUID, len(relations))
		for n, r := range relations {
			ids[n] = r.Other(i.ID)
		}
		others, err := u.d.Relations.GetIssues(ctx, ids)
		if err != nil {
			return err
		}
		projects, err := u.d.Projects.Visible(ctx, i.WorkspaceID, actor.UserID)
		if err != nil {
			return err
		}
		g := make(grants, len(projects)+1)
		for _, p := range projects {
			g[p.ID] = shared.Grant{ProjectRole: p.Role, OwnOnly: p.OwnOnly}
		}
		// The source was authorized independently; ordinary scope can omit its
		// archived project and must never widen the explicit source grant.
		g[i.ProjectID] = sourceGrant
		var visible []domain.Issue
		for _, o := range others {
			grant, ok := g[o.ProjectID]
			if ok && o.ArchivedAt == nil && !o.HiddenFrom(actor.UserID, grant.OwnOnly) {
				visible = append(visible, o)
			}
		}
		seen, err := u.d.Locks.seen(ctx, actor.UserID, g, visible...)
		if err != nil {
			return err
		}
		byID := make(map[uuid.UUID]domain.Issue, len(seen))
		for _, o := range seen {
			byID[o.ID] = o
		}
		for _, r := range relations {
			if o, ok := byID[r.Other(i.ID)]; ok {
				out = append(out, IssueRelation{ID: r.ID, Type: r.TypeFrom(i.ID), Issue: o, CreatedAt: r.CreatedAt})
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
