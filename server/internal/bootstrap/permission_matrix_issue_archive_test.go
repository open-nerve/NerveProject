package bootstrap

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// The rows of the permission matrix of archiving and unarchiving a work
// item (M4 design 3.8, 9.2), naming it by its id: its column's work item
// (issueOf), which each cell's prepare moves into Done, completed, or into
// Done and archived, on the cell's copy, for the cases that need it. A row
// of 9.2's table has the issue table's columns; a variant on an archived
// project, the archived project's.

var cellNotArchivable = cell{http.StatusConflict, "issue.not_archivable"}

func issueArchiveMatrixRows() []matrixRow {
	archive, unarchive := toWorkItem(http.MethodPost, "/archive", ""), toWorkItem(http.MethodPost, "/unarchive", "")
	return []matrixRow{
		// The project's admins and members, the member who made it too, not
		// its guests (M4 design 3.4, 9.2).
		{op: "archiveIssue", write: true, columns: issueColumns, prepare: inDone(false), request: archive,
			cells: onAWorkItem(cellOK, cellOK, cellOK, cellForbidden), check: asArchived(true, true)},
		// A work item in Todo is not archived (M4 design 3.8), after the
		// decision: who may not archive it learns nothing of its state.
		{op: "archiveIssue", variant: "in Todo", write: true, columns: issueColumns, request: archive,
			cells: onAWorkItem(cellNotArchivable, cellNotArchivable, cellNotArchivable, cellForbidden)},
		// An archived work item is answered as it is, after the decision.
		{op: "archiveIssue", variant: "archived work item", write: true, columns: issueColumns, prepare: inDone(true), request: archive,
			cells: onAWorkItem(cellOK, cellOK, cellOK, cellForbidden), check: asArchived(true, false)},
		// An archived project's work items are not archived, after the
		// decision, before their state is looked at.
		{op: "archiveIssue", variant: "archived", write: true, columns: archivedColumns, request: archive,
			cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)},
		// Nor are they answered as they are when archived: the project's 409
		// comes first.
		{op: "archiveIssue", variant: "archived, its work item archived", write: true, columns: archivedColumns, prepare: inDone(true),
			request: archive, cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)},
		// As archiveIssue (M4 design 3.4, 9.2), on an archived work item.
		{op: "unarchiveIssue", write: true, columns: issueColumns, prepare: inDone(true), request: unarchive,
			cells: onAWorkItem(cellOK, cellOK, cellOK, cellForbidden), check: asArchived(false, true)},
		// A work item that is not archived is answered as it is, after the
		// decision.
		{op: "unarchiveIssue", variant: "not archived", write: true, columns: issueColumns, request: unarchive,
			cells: onAWorkItem(cellOK, cellOK, cellOK, cellForbidden), check: readsTheIssue},
		// An archived project's work items are not unarchived, after the
		// decision, before the work item is looked at.
		{op: "unarchiveIssue", variant: "archived", write: true, columns: archivedColumns, request: unarchive,
			cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)},
		{op: "unarchiveIssue", variant: "archived, its work item archived", write: true, columns: archivedColumns, prepare: inDone(true),
			request: unarchive, cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)},
	}
}

// inDone is a cell's prepare: its column's work item moved into its
// project's Done, completed when it was last changed, and archived then
// too when archived, as archiveIssue leaves one, through SQL on the cell's
// copy at url, over a pool it closes before the cell's app starts.
func inDone(archived bool) func(t *testing.T, c caller, s seeded, url string) {
	return func(t *testing.T, c caller, s seeded, url string) {
		t.Helper()
		pool := openPool(t, url)
		defer pool.Close()
		tag, err := pool.Exec(pgtest.Soon(t), `UPDATE issues SET state_id = $2, completed_at = updated_at,
			archived_at = CASE WHEN $3 THEN updated_at END WHERE id = $1`, s.issue(projectOf(c), issueOf(c)), s.state(projectOf(c), "Done"), archived)
		if err != nil || tag.RowsAffected() != 1 {
			t.Fatalf("moving %s's work item into Done: %v, %d rows", c, err, tag.RowsAffected())
		}
	}
}

// asArchived is the check of a row whose prepare moved the column's work
// item into Done (inDone): the answer is the work item so moved, as
// seeded otherwise (seededIssue), completed, archived when archived;
// changed last by the caller when written, else by its maker.
func asArchived(archived, written bool) func(t *testing.T, c caller, s seeded, answer string) {
	return func(t *testing.T, c caller, s seeded, answer string) {
		var got struct {
			answeredIssue
			ArchivedAt *string `json:"archived_at"`
		}
		decodeAnswer(t, answer, &got)
		want := seededIssue(c, s)
		want.StateID = s.state(projectOf(c), "Done")
		if written {
			me := s.account(accountOf(c))
			want.UpdatedByID = &me
		}
		if (got.ArchivedAt != nil) != archived || got.CompletedAt == nil {
			t.Errorf("%s answers %s; want it completed, archived %v", c, answer, archived)
		}
		got.CompletedAt = nil
		if !reflect.DeepEqual(got.answeredIssue, want) {
			t.Errorf("%s answers %s; want %+v", c, answer, want)
		}
	}
}
