package app_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// newGet is getIssue over web's world (newCreate's), with web's work item
// 7 stored in it: Fix, in Backlog, high, assigned to bob, labelled Bug,
// made and last written by carol, so that alice, who reads it, is neither.
func newGet() (*app.GetIssue, *world, uuid.UUID) {
	_, w := newCreate()
	id := uuid.NewV7()
	w.issues.issue = &app.IssueRow{ID: id, WorkspaceID: acme, ProjectID: web, SequenceID: 7, Name: "Fix", StateID: backlog.ID,
		Priority: domain.PriorityHigh, SortOrder: 65535, By: carol, At: clockNow}
	w.issues.assignees, w.issues.labels = []uuid.UUID{bob}, []uuid.UUID{bug}
	return app.NewGetIssue(w.locks), w, id
}

// A read, outside any transaction and without a lock (M4 design 3.5): the
// work item, then the decision on issue.read in its project, of its
// workspace; the answer is the work item as read.
func TestGetIssueReadsTheWorkItem(t *testing.T) {
	uc, w, id := newGet()
	got, err := uc.Execute(as(alice), id)
	want := domain.Issue{ID: id, WorkspaceID: acme, ProjectID: web, SequenceID: 7, Name: "Fix", StateID: backlog.ID, Priority: domain.PriorityHigh,
		SortOrder: 65535, AssigneeIDs: []uuid.UUID{bob}, LabelIDs: []uuid.UUID{bug}, CreatedByID: &carol, UpdatedByID: &carol, CreatedAt: clockNow,
		UpdatedAt: clockNow}
	if err != nil || !reflect.DeepEqual(got, want) {
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
// caller, unauthenticated before any read.
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
			var se, wantSE *shared.Error
			if !errors.Is(err, tt.want) && (!errors.As(err, &se) || !errors.As(tt.want, &wantSE) || se.Code != wantSE.Code) ||
				got.ID != (uuid.UUID{}) {
				t.Errorf("Execute() = %+v, %v; want no work item and %v", got, err, tt.want)
			}
			calls := []string{line("GetIssue", id, "outside tx"), decided}[:tt.calls]
			if !slices.Equal(w.log.calls, calls) {
				t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
			}
		})
	}
}
