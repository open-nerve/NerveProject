package app_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/notification/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type issueDeletion struct {
	workspace, issue uuid.UUID
	stamp            app.DeletionStamp
}
type deleteIssue func(context.Context, uuid.UUID, uuid.UUID, app.DeletionStamp) error

func (f deleteIssue) DeleteIssueNotifications(ctx context.Context, workspace, issue uuid.UUID, stamp app.DeletionStamp) error {
	return f(ctx, workspace, issue, stamp)
}

// A work item's deletion soft-deletes its notifications in the publisher's context, its transaction, at the event's
// moment and by its actor; nerve's own deletion, with no actor, by no one (M7 P1b spec 2.8). A failure is the
// publisher's: it rolls the deletion back.
func TestForgetDeletedIssue(t *testing.T) {
	ws, p, item, by := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	at := time.Date(2026, 11, 17, 9, 0, 0, 123456000, time.UTC)
	deleted := shared.Event{ID: uuid.NewV7(), Type: "issue.deleted", Entity: shared.EntityRef{Kind: "issue", ID: item}, WorkspaceID: ws, ProjectID: p, OccurredAt: at, ActorID: &by}
	auto := deleted
	auto.ActorID = nil
	for _, tt := range []struct {
		event shared.Event
		want  issueDeletion
	}{
		{deleted, issueDeletion{ws, item, app.DeletionStamp{At: at, By: &by}}},
		{auto, issueDeletion{ws, item, app.DeletionStamp{At: at}}},
	} {
		var calls []issueDeletion
		sub := app.NewForgetDeletedIssue(deleteIssue(func(ctx context.Context, w, i uuid.UUID, s app.DeletionStamp) error {
			if ctx != t.Context() {
				t.Error("context replaced")
			}
			calls = append(calls, issueDeletion{w, i, s})
			return nil
		}))
		if sub.Name != "forget_deleted_issue_notifications" || !reflect.DeepEqual(sub.Types, []string{"issue.deleted"}) {
			t.Fatal(sub)
		}
		if err := sub.Handle(t.Context(), tt.event); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(calls, []issueDeletion{tt.want}) {
			t.Fatalf("calls=%+v want %+v", calls, tt.want)
		}
	}
	for name, change := range map[string]func(*shared.Event){
		"type":       func(e *shared.Event) { e.Type = "issue.updated" },
		"kind":       func(e *shared.Event) { e.Entity.Kind = "issue_comment" },
		"entity":     func(e *shared.Event) { e.Entity.ID = uuid.UUID{} },
		"time":       func(e *shared.Event) { e.OccurredAt = time.Time{} },
		"workspace":  func(e *shared.Event) { e.WorkspaceID = uuid.UUID{} },
		"project":    func(e *shared.Event) { e.ProjectID = uuid.UUID{} },
		"zero actor": func(e *shared.Event) { v := uuid.UUID{}; e.ActorID = &v },
	} {
		t.Run(name, func(t *testing.T) {
			e := deleted
			change(&e)
			called := false
			sub := app.NewForgetDeletedIssue(deleteIssue(func(context.Context, uuid.UUID, uuid.UUID, app.DeletionStamp) error {
				called = true
				return nil
			}))
			if err := sub.Handle(t.Context(), e); err == nil || called {
				t.Fatalf("invalid event accepted: err=%v called=%t", err, called)
			}
		})
	}
	failed := errors.New("update failed")
	if err := app.NewForgetDeletedIssue(deleteIssue(func(context.Context, uuid.UUID, uuid.UUID, app.DeletionStamp) error {
		return failed
	})).Handle(t.Context(), deleted); !errors.Is(err, failed) {
		t.Fatalf("deletion failure = %v", err)
	}
}

type deletionCall struct {
	project uuid.UUID
	at      time.Time
	by      uuid.UUID
}
type deletionEnqueue func(context.Context, uuid.UUID, time.Time, uuid.UUID) error

func (f deletionEnqueue) Enqueue(ctx context.Context, project uuid.UUID, at time.Time, by uuid.UUID) error {
	return f(ctx, project, at, by)
}

// A project's deletion enqueues its notifications' deletion, a workspace's each of its projects', in the publisher's
// context, with the event's moment and actor: the rows are deleted as of the request, not of the worker's run. The
// whole event is validated before the first insert.
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
		if sub.Name != "notification_delete_rows_on_deletion" || !reflect.DeepEqual(sub.Types, []string{"project.deleted", "workspace.deleted"}) {
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
	bad := project
	bad.Entity.ID = q
	if err := app.NewDeleteRowsOnDeletion(deletionEnqueue(func(context.Context, uuid.UUID, time.Time, uuid.UUID) error {
		return nil
	})).Handle(t.Context(), bad); err == nil {
		t.Fatal("mismatched project accepted")
	}
	failed := errors.New("insert failed")
	if err := app.NewDeleteRowsOnDeletion(deletionEnqueue(func(context.Context, uuid.UUID, time.Time, uuid.UUID) error {
		return failed
	})).Handle(t.Context(), project); !errors.Is(err, failed) {
		t.Fatalf("enqueue failure = %v", err)
	}
}
