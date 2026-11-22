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

// forgetCall is a call of the recent visits' cascade: every account's visits of a work item in a workspace it
// deleted, at a moment by an account, nil for none.
type forgetCall struct {
	workspace, issue uuid.UUID
	at               time.Time
	by               *uuid.UUID
}

// forgetLog is a fake RecentVisitCascades: it keeps each call, made in want's context, and answers err.
type forgetLog struct {
	t     *testing.T
	want  context.Context
	calls []forgetCall
	err   error
}

func (f *forgetLog) DeleteIssueRecentVisits(ctx context.Context, workspace, issue uuid.UUID, at time.Time, by *uuid.UUID) error {
	if ctx != f.want {
		f.t.Error("the publisher's context replaced")
	}
	f.calls = append(f.calls, forgetCall{workspace, issue, at, by})
	return f.err
}

// issueDeletion is a work item's deletion as the issue module publishes it, one event for each work item of the
// deleted subtree (its delete core, P3d spec 2.6): of the work item, in its workspace and project, at the deletion's
// moment by its actor, and no change.
func issueDeletion() (e shared.Event, ws, item uuid.UUID, at time.Time, by uuid.UUID) {
	ws, item, by = uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	at = time.Date(2026, 11, 22, 9, 0, 0, 123456000, time.UTC)
	return shared.Event{ID: uuid.NewV7(), Type: "issue.deleted", OccurredAt: at, WorkspaceID: ws, ProjectID: uuid.NewV7(), ActorID: &by,
		Entity: shared.EntityRef{Kind: "issue", ID: item}}, ws, item, at, by
}

// One subscriber, beside the deletion of a deleted project's rows and the favorites' cascades, takes a work item's
// deletion alone (spec 2.6, R10): every account's visits of the work item in its workspace go, in the publisher's
// context, its transaction, at the event's moment and by its actor; by none when the deletion has no actor. The
// port's failure is the publisher's.
func TestForgettingADeletedWorkItemsVisits(t *testing.T) {
	e, ws, item, at, by := issueDeletion()
	log := &forgetLog{t: t, want: t.Context()}
	sub := app.NewForgetDeletedIssueVisits(log)
	if sub.Name != "navigation_forget_deleted_issue_visits" || sub.HandleAll != nil || !reflect.DeepEqual(sub.Types, []string{"issue.deleted"}) {
		t.Fatalf("subscriber %q of %v", sub.Name, sub.Types)
	}
	if err := sub.Handle(t.Context(), e); err != nil {
		t.Fatal(err)
	}
	if want := []forgetCall{{ws, item, at, &by}}; !reflect.DeepEqual(log.calls, want) {
		t.Fatalf("calls %+v; want %+v", log.calls, want)
	}

	nobody := e
	nobody.ActorID = nil
	log = &forgetLog{t: t, want: t.Context()}
	if err := app.NewForgetDeletedIssueVisits(log).Handle(t.Context(), nobody); err != nil {
		t.Fatal(err)
	}
	if want := []forgetCall{{ws, item, at, nil}}; !reflect.DeepEqual(log.calls, want) {
		t.Fatalf("a deletion without an actor: calls %+v; want %+v", log.calls, want)
	}

	failed := errors.New("the visits' lock timed out")
	log = &forgetLog{t: t, want: t.Context(), err: failed}
	if err := app.NewForgetDeletedIssueVisits(log).Handle(t.Context(), e); !errors.Is(err, failed) {
		t.Errorf("the port's failure = %v; want it, the publisher's", err)
	}
}

// The event is validated whole before any write (spec 2.6), as the issue module writes it: its type, a work item
// with an id, a moment, its workspace and its project, an actor that is none or an account, and no change. A
// malformed event is refused, and the publisher's write with it, and nothing is deleted.
func TestForgettingADeletedWorkItemsVisitsRefusesAMalformedEvent(t *testing.T) {
	name := "Fix the login"
	for defect, change := range map[string]func(*shared.Event){
		"another type":    func(e *shared.Event) { e.Type = "issue.archived" },
		"a project":       func(e *shared.Event) { e.Entity.Kind = "project" },
		"no work item":    func(e *shared.Event) { e.Entity.ID = uuid.UUID{} },
		"no moment":       func(e *shared.Event) { e.OccurredAt = time.Time{} },
		"no workspace":    func(e *shared.Event) { e.WorkspaceID = uuid.UUID{} },
		"no project":      func(e *shared.Event) { e.ProjectID = uuid.UUID{} },
		"a zero actor":    func(e *shared.Event) { e.ActorID = &uuid.UUID{} },
		"a change":        func(e *shared.Event) { e.Changes = []shared.Change{{Field: "name", Old: &name}} },
		"an empty change": func(e *shared.Event) { e.Changes = []shared.Change{{}} },
	} {
		t.Run(defect, func(t *testing.T) {
			e, _, _, _, _ := issueDeletion()
			change(&e)
			log := &forgetLog{t: t, want: t.Context()}
			if err := app.NewForgetDeletedIssueVisits(log).Handle(t.Context(), e); err == nil || len(log.calls) != 0 {
				t.Fatalf("a malformed event: err=%v, calls %+v; want it refused before any write", err, log.calls)
			}
		})
	}
}
