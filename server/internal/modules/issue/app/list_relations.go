package app

import (
	"context"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListIssueRelations lists a work item's relations: GET
// /api/v0/issues/{issue_id}/relations (M4 design 3.10, 5.1).
type ListIssueRelations struct {
	locks     Locks
	relations RelationLister
	projects  ArchivedProjects
}

// NewListIssueRelations returns the use case.
func NewListIssueRelations(locks Locks, relations RelationLister, projects ArchivedProjects) *ListIssueRelations {
	return &ListIssueRelations{locks: locks, relations: relations, projects: projects}
}

// Execute reads the work item id and decides issue.read on it for the
// caller, as getIssue does, without a lock or a transaction (Locks.see),
// with its refusals; then its undeleted relations, in the order they were
// made, each as the work item reads it, whose other work item is
// undeleted, unarchived, of no archived project but the work item's own
// (M4 design 3.8), and seen by the caller, of any project of the
// workspace (Locks.seeEach); each other as he sees it (Locks.seen).
func (u *ListIssueRelations) Execute(ctx context.Context, id uuid.UUID) ([]IssueRelation, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	i, _, err := u.locks.see(ctx, actor, id, domain.ActionRead)
	if err != nil {
		return nil, err
	}
	relations, err := u.relations.Relations(ctx, i.ID)
	if err != nil || len(relations) == 0 {
		return nil, err
	}
	ids := make([]uuid.UUID, len(relations))
	for n, r := range relations {
		ids[n] = r.Other(i.ID)
	}
	others, err := u.relations.GetIssues(ctx, ids)
	if err != nil {
		return nil, err
	}
	archived, err := u.projects.Archived(ctx, projectsBut(i.ProjectID, others))
	if err != nil {
		return nil, err
	}
	others = slices.DeleteFunc(others, func(o domain.Issue) bool { return o.ArchivedAt != nil || archived[o.ProjectID] })
	visible, g, err := u.locks.seeEach(ctx, actor, others)
	if err != nil {
		return nil, err
	}
	seen, err := u.locks.seen(ctx, actor.UserID, g, visible...)
	if err != nil {
		return nil, err
	}
	byID := make(map[uuid.UUID]domain.Issue, len(seen))
	for _, o := range seen {
		byID[o.ID] = o
	}
	var out []IssueRelation
	for _, r := range relations {
		if o, ok := byID[r.Other(i.ID)]; ok {
			out = append(out, IssueRelation{ID: r.ID, Type: r.TypeFrom(i.ID), Issue: o, CreatedAt: r.CreatedAt})
		}
	}
	return out, nil
}

// projectsBut are the projects of items but project, each once, in
// ascending id order: those whose work items a list of the relations of
// project's work item leaves out when they are archived (M4 design 3.8).
func projectsBut(project uuid.UUID, items []domain.Issue) []uuid.UUID {
	var ids []uuid.UUID
	for _, i := range items {
		if i.ProjectID != project {
			ids = append(ids, i.ProjectID)
		}
	}
	return ascending(ids)
}
