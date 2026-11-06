// Package app holds the planning module's use cases and the ports they need
// (M6 design 6).
package app

import (
	"context"
	"errors"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// RowDeletions inserts a project's cycle deletion in the publishing transaction.
type RowDeletions interface {
	Enqueue(ctx context.Context, project uuid.UUID, at time.Time, by uuid.UUID) error
}

// NewDeleteRowsOnDeletion enqueues, within the deletion's transaction, the
// deletion of the cycles of each project the event deletes, at the event's
// moment and by its actor (M6 P1 spec 2.4).
func NewDeleteRowsOnDeletion(rows RowDeletions) shared.EventSubscriber {
	return shared.EventSubscriber{Name: "planning_delete_rows_on_deletion", Types: []string{"project.deleted", "workspace.deleted"}, Handle: func(ctx context.Context, e shared.Event) error {
		projects, err := deletedProjects(e)
		if err != nil {
			return err
		}
		for _, id := range projects {
			if err := rows.Enqueue(ctx, id, e.OccurredAt, *e.ActorID); err != nil {
				return err
			}
		}
		return nil
	}}
}

// deletedProjects validates the whole payload before the first insert: a
// project's deletion names its project, a workspace's lists each project it
// deleted once, as an old id and nothing else.
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
