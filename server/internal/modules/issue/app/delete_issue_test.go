package app_test

import (
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// deleteOver is deleteIssue over w's locks and rows as they are when it is
// called, its clock at clockNow: after gated rebuilds w's locks, it is
// deleteIssue over the gated ones.
func deleteOver(w *world) *app.DeleteIssue {
	return app.NewDeleteIssue(app.DeleteIssueDeps{Locks: w.locks, Rows: w.issues, Clock: clockAt{at: clockNow, log: w.log}})
}

// newDelete is deleteIssue over newUpdate's world: web's work item 7, made
// by carol, assigned to bob and out, labelled Bug.
func newDelete() (*app.DeleteIssue, *world, uuid.UUID) {
	_, w, id := newUpdate()
	return deleteOver(w), w, id
}

// deletingRows are the calls of the deletion of the rows of the work items
// ids, a subtree, in the global order of M4 design 3.5: the work items,
// their assignees, their labels, their relations, their subscribers, their
// activity.
func deletingRows(ids []uuid.UUID) []string {
	return []string{line("DeleteIssueRows", ids), line("DeleteIssueAssigneeRows", ids), line("DeleteIssueLabelRows", ids),
		line("DeleteIssueRelationRows", ids), line("DeleteIssueSubscriberRows", ids), line("DeleteIssueCommentRows", ids), line("DeleteIssueReactionRows", ids), line("DeleteIssueActivityRows", ids)}
}

// A deletion, in the order of M4 design 3.5: Locks' calls, its decisions
// on issue.delete (lockedAs); the work item's subtree, read under the
// lock; the clock; the rows of each work item of the subtree, table by
// table in the global order (deletingRows), each alice's at the moment.
// The subtree holds two work items under the one deleted, which is not its
// first, so that a deletion of the work item asked for alone, or of the
// subtree's first alone, shows. An archived work item is deleted as any
// other (M4 design 3.8). So is the subtree of alice's own work item when
// she is a workspace admin who is only a guest of web (adminGuest): the
// guest rule hides carol's work item under it from her, and it goes with
// the subtree, as her own one does (3.6).
func TestDeleteIssueDeletesTheWorkItemsSubtreeAndItsRows(t *testing.T) {
	for what, prepare := range map[string]func(w *world, subtree []uuid.UUID){
		"a work item":           func(*world, []uuid.UUID) {},
		"an archived work item": func(w *world, _ []uuid.UUID) { w.issues.archivedAt = &earlier },
		"an admin who is a guest's own work item": func(w *world, subtree []uuid.UUID) {
			w.issues.issue.By, w.auth.grants = alice, []shared.Grant{adminGuest, adminGuest}
			w.issues.others = map[uuid.UUID]domain.Issue{subtree[0]: {ID: subtree[0], WorkspaceID: acme, ProjectID: web, CreatedByID: &carol},
				subtree[2]: {ID: subtree[2], WorkspaceID: acme, ProjectID: web, CreatedByID: &alice}}
		},
	} {
		t.Run(what, func(t *testing.T) {
			uc, w, id := newDelete()
			subtree := []uuid.UUID{uuid.NewV7(), id, uuid.NewV7()}
			prepare(w, subtree)
			w.issues.subtree = subtree
			if err := uc.Execute(as(alice), id); err != nil {
				t.Fatal(err)
			}
			calls := slices.Concat(lockedAs(domain.ActionDelete, id), []string{line("Subtree", id), "Now"}, deletingRows(subtree))
			if !slices.Equal(w.log.calls, calls) {
				t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
			}
			deletion := app.IssueDeletion{IssueIDs: subtree, By: alice, At: clockNow}
			if want := slices.Repeat([]app.IssueDeletion{deletion}, 8); !reflect.DeepEqual(w.issues.deletions, want) {
				t.Errorf("the deletions %+v, want %+v", w.issues.deletions, want)
			}
		})
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
	for _, port := range []string{"GetIssue", "Authorize", "Lock", "Subtree", "DeleteIssueRows", "DeleteIssueAssigneeRows",
		"DeleteIssueLabelRows", "DeleteIssueRelationRows", "DeleteIssueSubscriberRows", "DeleteIssueCommentRows", "DeleteIssueReactionRows", "DeleteIssueActivityRows"} {
		t.Run(port, func(t *testing.T) {
			uc, w, id := newDelete()
			w.log.fail = port
			wantStoppedAt(t, w, uc.Execute(as(alice), id), port)
		})
	}
}
