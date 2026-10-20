package app_test

import (
	"context"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// newGet is getIssue over web's world (newCreate's), its write gate one
// that logs, with web's work item 7 stored in it: Fix, in Backlog, high,
// assigned to bob, labelled Bug, made and last written by carol, so that
// alice, who reads it, is neither.
func newGet() (*app.GetIssue, *world, uuid.UUID) {
	_, w := newCreate()
	w.locks = app.NewLocks(w.project, w.issues, w.auth, fakeGate{log: w.log}, fakeTx{log: w.log})
	id := uuid.NewV7()
	w.issues.issue = &app.IssueRow{ID: id, WorkspaceID: acme, ProjectID: web, SequenceID: 7, Name: "Fix", StateID: backlog.ID,
		Priority: domain.PriorityHigh, SortOrder: 65535, By: carol, At: clockNow}
	w.issues.assignees, w.issues.labels = []uuid.UUID{bob}, []uuid.UUID{bug}
	return app.NewGetIssue(w.locks), w, id
}

// readIssue is newGet's work item id as a read answers it.
func readIssue(id uuid.UUID) domain.Issue {
	return domain.Issue{ID: id, WorkspaceID: acme, ProjectID: web, SequenceID: 7, Name: "Fix", StateID: backlog.ID, Priority: domain.PriorityHigh,
		SortOrder: 65535, AssigneeIDs: []uuid.UUID{bob}, LabelIDs: []uuid.UUID{bug}, CreatedByID: &carol, UpdatedByID: &carol, CreatedAt: clockNow,
		UpdatedAt: clockNow}
}

// A read, outside any transaction, without a lock and outside the write
// gate (M4 design 3.5, 11.11): the work item, then the decision on
// issue.read in its project, of its workspace; the answer is the work item
// as read.
func TestGetIssueReadsTheWorkItem(t *testing.T) {
	uc, w, id := newGet()
	got, err := uc.Execute(as(alice), id)
	if want := readIssue(id); err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("Execute() = %+v, %v; want %+v", got, err, want)
	}
	if calls := []string{line("GetIssue", id, "outside tx"), line("Authorize", alice, domain.ActionRead, acme, web, "outside tx")}; !slices.Equal(
		w.log.calls, calls) {
		t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
	}
}

// A work item that is not there, or whose project the caller does not
// see, is issue.not_found; a role that may not read it, forbidden; a read
// or a decision that fails, its failure as itself; a request without a
// caller, unauthenticated before any read. Each answer is compared whole,
// not by its code alone.
func TestGetIssueRefusals(t *testing.T) {
	decided := line("Authorize", alice, domain.ActionRead, acme, web, "outside tx")
	for _, tt := range []struct {
		name  string
		set   func(w *world, id *uuid.UUID)
		ctx   context.Context
		want  error
		calls int // of read, decided
	}{
		{"no work item", func(_ *world, id *uuid.UUID) { *id = uuid.NewV7() }, as(alice), domain.ErrIssueNotFound, 1},
		{"a project not seen", func(w *world, _ *uuid.UUID) { w.auth.answers = []error{shared.ErrNotVisible} }, as(alice),
			domain.ErrIssueNotFound, 2},
		{"a role not allowed", func(w *world, _ *uuid.UUID) { w.auth.answers = []error{shared.Forbidden()} }, as(alice), shared.Forbidden(), 2},
		{"the read fails", func(w *world, _ *uuid.UUID) { w.log.fail = "GetIssue" }, as(alice), errDisk, 1},
		{"the decision fails", func(w *world, _ *uuid.UUID) { w.log.fail = "Authorize" }, as(alice), errDisk, 2},
		{"no caller", func(*world, *uuid.UUID) {}, context.Background(), shared.Unauthenticated(), 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, id := newGet()
			tt.set(w, &id)
			got, err := uc.Execute(tt.ctx, id)
			if !reflect.DeepEqual(err, tt.want) || got.ID != (uuid.UUID{}) {
				t.Errorf("Execute() = %+v, %v; want no work item and %v", got, err, tt.want)
			}
			calls := []string{line("GetIssue", id, "outside tx"), decided}[:tt.calls]
			if !slices.Equal(w.log.calls, calls) {
				t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
			}
		})
	}
}
