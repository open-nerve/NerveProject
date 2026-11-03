package app

import (
	"context"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type ScopeJobs interface {
	Enqueue(context.Context, domain.ScopeKind, uuid.UUID, DeletionStamp) error
}

func DeletionSubscribers(rows ScopeDeletions, jobs ScopeJobs) []shared.EventSubscriber {
	return []shared.EventSubscriber{
		{Name: "asset_rows_on_deletion", Types: []string{"issue.deleted", "issue.comment_deleted"}, Handle: func(ctx context.Context, e shared.Event) error {
			kind, err := assetDeletionEvent(e)
			if err != nil {
				return err
			}
			return rows.DeleteScopeRows(ctx, kind, e.Entity.ID, DeletionStamp{At: e.OccurredAt, By: e.ActorID})
		}},
		{Name: "asset_scope_on_deletion", Types: []string{"project.deleted", "workspace.deleted"}, Handle: func(ctx context.Context, e shared.Event) error {
			kind, err := assetDeletionEvent(e)
			if err != nil {
				return err
			}
			return jobs.Enqueue(ctx, kind, e.Entity.ID, DeletionStamp{At: e.OccurredAt, By: e.ActorID})
		}},
	}
}
func assetDeletionEvent(e shared.Event) (domain.ScopeKind, error) {
	invalid := errors.New("invalid asset parent deletion event")
	if e.Entity.ID == uuid.Nil() || e.OccurredAt.IsZero() || e.WorkspaceID == uuid.Nil() || (e.ActorID != nil && *e.ActorID == uuid.Nil()) {
		return "", invalid
	}
	switch e.Type {
	case "issue.deleted":
		if e.Entity.Kind == "issue" && e.ProjectID != uuid.Nil() {
			return domain.ScopeIssue, nil
		}
	case "issue.comment_deleted":
		if e.Entity.Kind == "issue_comment" && e.ProjectID != uuid.Nil() {
			return domain.ScopeComment, nil
		}
	case "project.deleted":
		if e.Entity.Kind == "project" && e.Entity.ID == e.ProjectID {
			return domain.ScopeProject, nil
		}
	case "workspace.deleted":
		if e.Entity.Kind == "workspace" && e.Entity.ID == e.WorkspaceID && e.ProjectID == uuid.Nil() {
			return domain.ScopeWorkspace, nil
		}
	}
	return "", invalid
}
