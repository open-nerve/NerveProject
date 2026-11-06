package app_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type deletionCall struct {
	project uuid.UUID
	at      time.Time
	by      uuid.UUID
}
type deletionEnqueue func(context.Context, uuid.UUID, time.Time, uuid.UUID) error

func (f deletionEnqueue) Enqueue(ctx context.Context, project uuid.UUID, at time.Time, by uuid.UUID) error {
	return f(ctx, project, at, by)
}

// A project's deletion enqueues its cycles' deletion, a workspace's each of its projects', in the publisher's
// context, with the event's moment and actor: the rows are deleted as of the request, not of the worker's run.
func TestDeleteRowsOnDeletion(t *testing.T) {
	ws, p, q, by := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	at := time.Now().Add(-61 * 24 * time.Hour)
	project := shared.Event{Type: "project.deleted", Entity: shared.EntityRef{Kind: "project", ID: p}, ProjectID: p, WorkspaceID: ws, OccurredAt: at, ActorID: &by}
	workspace := shared.Event{Type: "workspace.deleted", Entity: shared.EntityRef{Kind: "workspace", ID: ws}, WorkspaceID: ws, OccurredAt: at, ActorID: &by,
		Changes: []shared.Change{{Field: "project", OldID: &p}, {Field: "project", OldID: &q}}}
	empty := workspace
	empty.Changes = nil
	for _, tt := range []struct {
		event shared.Event
		want  []deletionCall
	}{
		{project, []deletionCall{{p, at, by}}},
		{workspace, []deletionCall{{p, at, by}, {q, at, by}}},
		{empty, nil},
	} {
		var calls []deletionCall
		sub := app.NewDeleteRowsOnDeletion(deletionEnqueue(func(ctx context.Context, id uuid.UUID, a time.Time, b uuid.UUID) error {
			if ctx != t.Context() {
				t.Error("context replaced")
			}
			calls = append(calls, deletionCall{id, a, b})
			return nil
		}))
		if sub.Name != "planning_delete_rows_on_deletion" || !reflect.DeepEqual(sub.Types, []string{"project.deleted", "workspace.deleted"}) {
			t.Fatal(sub)
		}
		if err := sub.Handle(t.Context(), tt.event); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(calls, tt.want) {
			t.Fatalf("%s: calls=%+v want=%+v", tt.event.Type, calls, tt.want)
		}
	}
	for name, change := range map[string]func(*shared.Event){
		"actor":      func(e *shared.Event) { e.ActorID = nil },
		"zero actor": func(e *shared.Event) { v := uuid.UUID{}; e.ActorID = &v },
		"time":       func(e *shared.Event) { e.OccurredAt = time.Time{} },
		"entity":     func(e *shared.Event) { e.Entity.Kind = "project" },
		"workspace":  func(e *shared.Event) { e.WorkspaceID = uuid.NewV7() },
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
			sub := app.NewDeleteRowsOnDeletion(deletionEnqueue(func(context.Context, uuid.UUID, time.Time, uuid.UUID) error {
				called = true
				return nil
			}))
			if err := sub.Handle(t.Context(), e); err == nil || called {
				t.Fatalf("invalid event accepted: err=%v called=%t", err, called)
			}
		})
	}
	// A project event whose entity is not the project it names.
	bad := project
	bad.Entity.ID = q
	if err := app.NewDeleteRowsOnDeletion(deletionEnqueue(func(context.Context, uuid.UUID, time.Time, uuid.UUID) error {
		return nil
	})).Handle(t.Context(), bad); err == nil {
		t.Fatal("mismatched project accepted")
	}
	// An enqueue failure is the publisher's: it rolls the deletion back.
	failed := errors.New("insert failed")
	if err := app.NewDeleteRowsOnDeletion(deletionEnqueue(func(context.Context, uuid.UUID, time.Time, uuid.UUID) error {
		return failed
	})).Handle(t.Context(), project); !errors.Is(err, failed) {
		t.Fatalf("enqueue failure = %v", err)
	}
}
