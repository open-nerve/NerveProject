package app

import (
	"context"
	"errors"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// RowDeletions inserts work in the publishing transaction.
type RowDeletions interface {
	Enqueue(context.Context, string, uuid.UUID, time.Time, uuid.UUID) error
}

// NewDeleteRowsOnDeletion consumes only the projects in this deletion event.
func NewDeleteRowsOnDeletion(rows RowDeletions) shared.EventSubscriber {
	return shared.EventSubscriber{Name: "delete_rows_on_deletion", Types: []string{"project.deleted", "workspace.deleted"}, Handle: func(ctx context.Context, e shared.Event) error {
		projects, err := deletedProjects(e)
		if err != nil {
			return err
		}
		for _, id := range projects {
			if err := rows.Enqueue(ctx, "project", id, e.OccurredAt, *e.ActorID); err != nil {
				return err
			}
		}
		if e.Type == "workspace.deleted" {
			return rows.Enqueue(ctx, "workspace_drafts", e.WorkspaceID, e.OccurredAt, *e.ActorID)
		}
		return nil
	}}
}

// Validate the complete payload before the first insert. Publication still owns
// rollback if an insert or a later subscriber fails.
func deletedProjects(e shared.Event) ([]uuid.UUID, error) {
	invalid := errors.New("invalid deletion event shape")
	if e.ActorID == nil || *e.ActorID == (uuid.UUID{}) || e.OccurredAt.IsZero() || e.WorkspaceID == (uuid.UUID{}) || e.Entity.ID == (uuid.UUID{}) {
		return nil, invalid
	}
	switch e.Type {
	case "project.deleted":
		if e.Entity.Kind != "project" || e.Entity.ID != e.ProjectID || len(e.Changes) != 0 {
			return nil, invalid
		}
		return []uuid.UUID{e.ProjectID}, nil
	case "workspace.deleted":
		if e.Entity.Kind != "workspace" || e.Entity.ID != e.WorkspaceID || e.ProjectID != (uuid.UUID{}) {
			return nil, invalid
		}
		ids := make([]uuid.UUID, 0, len(e.Changes))
		seen := map[uuid.UUID]bool{}
		for _, c := range e.Changes {
			if c.Field != "project" || c.OldID == nil || *c.OldID == (uuid.UUID{}) || c.NewID != nil || c.Old != nil || c.New != nil || seen[*c.OldID] {
				return nil, invalid
			}
			seen[*c.OldID] = true
			ids = append(ids, *c.OldID)
		}
		return ids, nil
	default:
		return nil, invalid
	}
}
