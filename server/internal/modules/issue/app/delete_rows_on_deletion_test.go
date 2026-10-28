package app_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type deletionCall struct {
	scope string
	id    uuid.UUID
	at    time.Time
	by    uuid.UUID
}
type deletionEnqueue func(context.Context, string, uuid.UUID, time.Time, uuid.UUID) error

func (f deletionEnqueue) Enqueue(ctx context.Context, s string, id uuid.UUID, at time.Time, by uuid.UUID) error {
	return f(ctx, s, id, at, by)
}
func TestDeleteRowsOnDeletion(t *testing.T) {
	ws, p, q, old, by := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	at := time.Now().Add(-61 * 24 * time.Hour)
	project := shared.Event{Type: "project.deleted", Entity: shared.EntityRef{Kind: "project", ID: p}, ProjectID: p, WorkspaceID: ws, OccurredAt: at, ActorID: &by}
	workspace := shared.Event{Type: "workspace.deleted", Entity: shared.EntityRef{Kind: "workspace", ID: ws}, WorkspaceID: ws, OccurredAt: at, ActorID: &by, Changes: []shared.Change{{Field: "project", OldID: &p}, {Field: "project", OldID: &q}}}
	for _, e := range []shared.Event{project, workspace, func() shared.Event { v := workspace; v.Changes = nil; return v }()} {
		var calls []deletionCall
		sub := app.NewDeleteRowsOnDeletion(deletionEnqueue(func(ctx context.Context, s string, id uuid.UUID, a time.Time, b uuid.UUID) error {
			if ctx != t.Context() {
				t.Error("context replaced")
			}
			calls = append(calls, deletionCall{s, id, a, b})
			return nil
		}))
		if sub.Name != "delete_rows_on_deletion" || !reflect.DeepEqual(sub.Types, []string{"project.deleted", "workspace.deleted"}) {
			t.Fatal(sub)
		}
		if err := sub.Handle(t.Context(), e); err != nil {
			t.Fatal(err)
		}
		want := []deletionCall{{"project", p, at, by}}
		if e.Type == "workspace.deleted" {
			want = nil
			for _, c := range e.Changes {
				want = append(want, deletionCall{"project", *c.OldID, at, by})
			}
			want = append(want, deletionCall{"workspace_drafts", ws, at, by})
		}
		if !reflect.DeepEqual(calls, want) {
			t.Fatalf("calls=%+v want=%+v", calls, want)
		}
		for _, call := range calls {
			if call.id == old {
				t.Fatal("older project included")
			}
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
			calls := 0
			sub := app.NewDeleteRowsOnDeletion(deletionEnqueue(func(context.Context, string, uuid.UUID, time.Time, uuid.UUID) error { calls++; return nil }))
			if err := sub.Handle(t.Context(), e); err == nil || calls != 0 {
				t.Fatalf("invalid shape accepted: %v calls=%d", err, calls)
			}
		})
	}
	calls := 0
	sub := app.NewDeleteRowsOnDeletion(deletionEnqueue(func(context.Context, string, uuid.UUID, time.Time, uuid.UUID) error { calls++; return errDisk }))
	if err := sub.Handle(t.Context(), workspace); !errors.Is(err, errDisk) || calls != 1 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}
}
