package app_test

import (
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// newDelete is deleteIssue over newUpdate's world: web's work item 7, made
// by carol, assigned to bob and out, labelled Bug.
func newDelete() (*app.DeleteIssue, *world, uuid.UUID) {
	_, w, id := newUpdate()
	return app.NewDeleteIssue(app.DeleteIssueDeps{Locks: w.locks, Rows: w.issues, Clock: clockAt{at: clockNow, log: w.log}}), w, id
}

// deletingRows are the calls of the deletion of the work item id's rows,
// in the global order of M4 design 3.5: the work item, its assignees, its
// labels, its subscribers, its activity.
func deletingRows(id uuid.UUID) []string {
	return []string{line("DeleteIssueRow", id), line("DeleteIssueAssigneeRows", id), line("DeleteIssueLabelRows", id),
		line("DeleteIssueSubscriberRows", id), line("DeleteIssueActivityRows", id)}
}

// A deletion, in the order of M4 design 3.5: Locks' calls, its decisions
// on issue.delete (lockedAs); the clock; the work item's rows, table by
// table in the global order (deletingRows), each alice's at the moment.
func TestDeleteIssueDeletesTheWorkItemAndItsRows(t *testing.T) {
	uc, w, id := newDelete()
	if err := uc.Execute(as(alice), id); err != nil {
		t.Fatal(err)
	}
	if calls := slices.Concat(lockedAs(domain.ActionDelete, id), []string{"Now"}, deletingRows(id)); !slices.Equal(w.log.calls, calls) {
		t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
	}
	deletion := app.IssueDeletion{IssueID: id, By: alice, At: clockNow}
	if want := slices.Repeat([]app.IssueDeletion{deletion}, 5); !slices.Equal(w.issues.deletions, want) {
		t.Errorf("the deletions %+v, want %+v", w.issues.deletions, want)
	}
}

// Every refusal of the decision is answered before any lock (M4 design
// 3.5), and the decision under the lock is final: a work item not there,
// or whose project the caller does not see, before or under the lock, is
// issue.not_found; a role that may not delete it forbidden; an archived
// project refuses it under the lock (409 project.archived). None deletes
// a row.
func TestDeleteIssueRefusals(t *testing.T) {
	for _, tt := range []struct {
		name  string
		set   func(w *world, id *uuid.UUID)
		want  error
		calls int // of lockedAs'
	}{
		{"no work item", func(_ *world, id *uuid.UUID) { *id = uuid.NewV7() }, domain.ErrIssueNotFound, 1},
		{"a project not seen", func(w *world, _ *uuid.UUID) { w.auth.answers = []error{shared.ErrNotVisible} }, domain.ErrIssueNotFound, 2},
		{"a role not allowed", func(w *world, _ *uuid.UUID) { w.auth.answers = []error{shared.Forbidden()} }, shared.Forbidden(), 2},
		{"the project deleted while the lock waited", func(w *world, _ *uuid.UUID) { w.project.locked = nil }, domain.ErrIssueNotFound, 4},
		{"the work item deleted meanwhile", func(w *world, _ *uuid.UUID) { w.project.waited = func() { w.issues.issue.ID = uuid.NewV7() } },
			domain.ErrIssueNotFound, 5},
		{"no longer seen", func(w *world, _ *uuid.UUID) { w.auth.answers = []error{nil, shared.ErrNotVisible} }, domain.ErrIssueNotFound, 6},
		{"no longer allowed", func(w *world, _ *uuid.UUID) { w.auth.answers = []error{nil, shared.Forbidden()} }, shared.Forbidden(), 6},
		{"archived", func(w *world, _ *uuid.UUID) { w.project.locked.Archived = true }, domain.ErrProjectArchived, 6},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, id := newDelete()
			tt.set(w, &id)
			err := uc.Execute(as(alice), id)
			wantRefusal(t, w, err, tt.want, lockedAs(domain.ActionDelete, id)[:tt.calls])
			if len(w.issues.deletions) != 0 {
				t.Errorf("the deletions %+v, want none", w.issues.deletions)
			}
		})
	}
}

// A port that fails ends the deletion with its failure, as itself, and no
// call after it runs: the transaction rolls back whatever was deleted.
func TestDeleteIssueStopsAtAFailingPort(t *testing.T) {
	for _, port := range []string{"GetIssue", "Authorize", "Lock", "DeleteIssueRow", "DeleteIssueAssigneeRows", "DeleteIssueLabelRows",
		"DeleteIssueSubscriberRows", "DeleteIssueActivityRows"} {
		t.Run(port, func(t *testing.T) {
			uc, w, id := newDelete()
			w.log.fail = port
			wantStoppedAt(t, w, uc.Execute(as(alice), id), port)
		})
	}
}
