package app_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type deletionCall struct {
	scope app.DeletionScope
	id    uuid.UUID
	at    time.Time
	by    uuid.UUID
}
type deletionEnqueue func(context.Context, app.DeletionScope, uuid.UUID, time.Time, uuid.UUID) error

func (f deletionEnqueue) Enqueue(ctx context.Context, scope app.DeletionScope, id uuid.UUID, at time.Time, by uuid.UUID) error {
	return f(ctx, scope, id, at, by)
}

// A project's deletion enqueues the deletion of its views; a workspace's, of each project's it deleted, then of the
// workspace's own views, those without a project, even with no project deleted (P3b spec 2.7). Each in the
// publisher's context, its transaction, with the event's moment and actor: the views are deleted as of the request,
// not of the worker's run. The whole event is validated before the first insert, and a failed insert is the
// publisher's.
func TestDeleteRowsOnDeletion(t *testing.T) {
	ws, p, q, by := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	at := time.Date(2026, 11, 20, 9, 0, 0, 123456000, time.UTC)
	project := shared.Event{Type: "project.deleted", Entity: shared.EntityRef{Kind: "project", ID: p}, ProjectID: p, WorkspaceID: ws, OccurredAt: at, ActorID: &by}
	workspace := shared.Event{Type: "workspace.deleted", Entity: shared.EntityRef{Kind: "workspace", ID: ws}, WorkspaceID: ws, OccurredAt: at, ActorID: &by,
		Changes: []shared.Change{{Field: "project", OldID: &p}, {Field: "project", OldID: &q}}}
	empty := workspace
	empty.Changes = nil
	for _, tt := range []struct {
		name  string
		event shared.Event
		want  []deletionCall
	}{
		{"project", project, []deletionCall{{app.ProjectScope, p, at, by}}},
		{"workspace", workspace, []deletionCall{{app.ProjectScope, p, at, by}, {app.ProjectScope, q, at, by}, {app.WorkspaceScope, ws, at, by}}},
		{"workspace without projects", empty, []deletionCall{{app.WorkspaceScope, ws, at, by}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var calls []deletionCall
			sub := app.NewDeleteRowsOnDeletion(deletionEnqueue(func(ctx context.Context, s app.DeletionScope, id uuid.UUID, a time.Time, b uuid.UUID) error {
				if ctx != t.Context() {
					t.Error("context replaced")
				}
				calls = append(calls, deletionCall{s, id, a, b})
				return nil
			}))
			if sub.Name != "navigation_delete_rows_on_deletion" || !reflect.DeepEqual(sub.Types, []string{"project.deleted", "workspace.deleted"}) || sub.HandleAll != nil {
				t.Fatal(sub)
			}
			if err := sub.Handle(t.Context(), tt.event); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(calls, tt.want) {
				t.Fatalf("calls=%+v want=%+v", calls, tt.want)
			}
		})
	}
	for name, change := range map[string]func(*shared.Event){
		"actor":      func(e *shared.Event) { e.ActorID = nil },
		"zero actor": func(e *shared.Event) { v := uuid.UUID{}; e.ActorID = &v },
		"time":       func(e *shared.Event) { e.OccurredAt = time.Time{} },
		"entity":     func(e *shared.Event) { e.Entity.Kind = "project" },
		"other ws":   func(e *shared.Event) { e.WorkspaceID = uuid.NewV7() },
		"project id": func(e *shared.Event) { e.ProjectID = p },
		"new id":     func(e *shared.Event) { e.Changes = []shared.Change{{Field: "project", NewID: &p}} },
		"duplicate": func(e *shared.Event) {
			e.Changes = []shared.Change{{Field: "project", OldID: &p}, {Field: "project", OldID: &p}}
		},
		"bad second": func(e *shared.Event) {
			e.Changes = []shared.Change{{Field: "project", OldID: &p}, {Field: "other", OldID: &q}}
		},
		"text": func(e *shared.Event) { s := "bad"; e.Changes = []shared.Change{{Field: "project", OldID: &p, Old: &s}} },
		"type": func(e *shared.Event) { e.Type = "unknown" },
	} {
		t.Run(name, func(t *testing.T) {
			e := workspace
			change(&e)
			called := false
			sub := app.NewDeleteRowsOnDeletion(deletionEnqueue(func(context.Context, app.DeletionScope, uuid.UUID, time.Time, uuid.UUID) error {
				called = true
				return nil
			}))
			if err := sub.Handle(t.Context(), e); err == nil || called {
				t.Fatalf("invalid event accepted: err=%v called=%t", err, called)
			}
		})
	}
	for name, bad := range map[string]func(*shared.Event){
		"mismatched project": func(e *shared.Event) { e.Entity.ID = q },
		"changes":            func(e *shared.Event) { e.Changes = []shared.Change{{Field: "project", OldID: &q}} },
	} {
		e := project
		bad(&e)
		if err := app.NewDeleteRowsOnDeletion(deletionEnqueue(func(context.Context, app.DeletionScope, uuid.UUID, time.Time, uuid.UUID) error {
			t.Fatalf("%s: enqueued", name)
			return nil
		})).Handle(t.Context(), e); err == nil {
			t.Fatalf("%s accepted", name)
		}
	}
	failed := errors.New("insert failed")
	for _, e := range []shared.Event{project, empty} {
		if err := app.NewDeleteRowsOnDeletion(deletionEnqueue(func(context.Context, app.DeletionScope, uuid.UUID, time.Time, uuid.UUID) error {
			return failed
		})).Handle(t.Context(), e); !errors.Is(err, failed) {
			t.Fatalf("%s: enqueue failure = %v", e.Type, err)
		}
	}
}
