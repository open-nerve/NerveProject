package bootstrap

import (
	"context"
	"net/http"
	"reflect"
	"testing"
	"uuid"

	projectdomain "github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// The rows of the permission matrix of a project's intake (M7 P2a spec
// 2.4, 2.5, 2.6): submitting a work item to it, under the project of each
// column, which prepareMatrix turns the intake of on (intakeOn); triaging
// one, naming it by its id (/intake-issues/{issue_id}): its column's work
// item (issueOf), which each cell's prepare puts into the intake on its
// copy (inTheIntake), or leaves where it is, in no intake; and the six
// writes a work item in the intake refuses (409 issue.in_triage, spec
// 2.6), each on its column's work item put into the intake so. Each cell
// writes on a copy of its own. A row of the issue table has its columns;
// a variant, the archived project's (archivedColumns), whose intake is
// off: three cells, its admin's after the 404 of X and the 403 of the
// workspace's member, who sees the project and is none of its members.

var (
	cellInTriage       = cell{http.StatusConflict, "issue.in_triage"}
	cellIntakeNotFound = cell{http.StatusNotFound, "issue.intake_not_found"}
)

// intakeRows are the work items the rows triage by id, under
// /intake-issues: a column that does not see its project answers
// issue.intake_not_found, the one 404 a triage answers (spec 2.5).
var intakeRows = rowsByID{path: "intake-issues", notFound: cellIntakeNotFound, find: seeded.issue}

func intakeMatrixRows() []matrixRow {
	declined := toTriage(`{"status":"declined"}`)
	// A write a work item in the intake refuses, on the archived project's,
	// by its admin: refused so before the project's 409 (DP-9), which every
	// other refusal of the write's own comes after; each write's own
	// refusals of one not in the intake, after the decision too, are in the
	// write's rows.
	inTriage := func(op string, request func(caller, seeded) (string, string, string)) matrixRow {
		return matrixRow{op: op, variant: "archived, in the intake", write: true, columns: archivedColumns, prepare: inTheIntake,
			request: request, cells: issueRows.ofArchived(cellInTriage, cellForbidden)}
	}
	return []matrixRow{
		// Every active member of the project, its guests too (M7 design
		// 3.10): createIssue's cells, but its guests'.
		{op: "submitIntakeIssue", write: true, columns: issueColumns, request: toProject(http.MethodPost, "/intake-issues", newIssue),
			cells: onTheIssues(cellCreated, cellCreated, cellCreated), check: submitsTheIssue},
		// An archived project takes no submission, after the decision, before
		// its intake, which is off, is looked at.
		{op: "submitIntakeIssue", variant: "archived", write: true, columns: archivedColumns,
			request: toProject(http.MethodPost, "/intake-issues", newIssue), cells: ofArchived(cellProjectArchived, cellForbidden)},
		// The project's admins, and the workspace's admins who are its
		// active members (M7 design 15 B): PA, PM+WA, and the workspace admin
		// who is a guest of the project on his own; not its members, nor
		// the guests who see it; a guest of a project that does not show its
		// guests every work item, the workspace admin who is only its guest
		// too, finds none another made (spec 2.5).
		{op: "triageIntakeIssue", write: true, columns: issueColumns, prepare: inTheIntake, request: declined,
			cells: onAnIntakeItem(cellOK, cellForbidden, cellForbidden, cellForbidden), check: declinesIt},
		// An archived project's intake is not triaged, after the work item's
		// intake row is found, before its intake, which is off, is looked at.
		{op: "triageIntakeIssue", variant: "archived", write: true, columns: archivedColumns, prepare: inTheIntake, request: declined,
			cells: intakeRows.ofArchived(cellProjectArchived, cellForbidden)},
		// A work item in no intake is not found, after the decision, before
		// its project's 409 (spec 2.5, I9): who may not triage learns nothing
		// of where it is, and who may finds none, as one not there.
		{op: "triageIntakeIssue", variant: "archived, in no intake", write: true, columns: archivedColumns, request: declined,
			cells: intakeRows.ofArchived(cellIntakeNotFound, cellForbidden)},
		inTriage("updateIssue", toWorkItem(http.MethodPatch, "", `{"name":"Mend"}`)),
		inTriage("archiveIssue", toWorkItem(http.MethodPost, "/archive", "")),
		inTriage("unarchiveIssue", toWorkItem(http.MethodPost, "/unarchive", "")),
		inTriage("deleteIssue", toWorkItem(http.MethodDelete, "", "")),
		// Its parent's end, as the work item named (spec 2.6).
		inTriage("addSubIssues", addsSubItem(otherItemOf)),
		// Its source's end.
		inTriage("createIssueRelations", relatesTo(otherItemOf)),
	}
}

// intakeOn turns the intake of acme's public and private projects on
// through the project store, by acme's admin (M7 P2a spec 2.4: it is off by
// default): there the rows' cells that pass the decision submit and
// triage. The other projects' stays off, as a project starts: the archived
// one's, whose rows' cells are refused as archived before the toggle is
// looked at; no cell gets so far in gone's or other's.
func (s projectSeed) intakeOn() {
	s.t.Helper()
	on := true
	for _, key := range []string{"acme/public", "acme/private"} {
		if err := s.store.UpdateProject(context.Background(), s.projects[key], projectdomain.ProjectPatch{IntakeView: &on},
			s.ids[callerAdmin], s.now); err != nil {
			s.t.Fatal(err)
		}
	}
}

// toTriage is the request of a row whose callers each triage their
// column's work item (issueOf) with body.
func toTriage(body string) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		return intakeRows.to(http.MethodPost, "/triage", issueOf(c), body)(c, s)
	}
}

// onAnIntakeItem are the cells of a row of the issue table that triages a
// work item: onAWorkItem's, each issue.not_found an issue.intake_not_found.
func onAnIntakeItem(admin, member, creator, guest cell) map[caller]cell {
	cells := onAWorkItem(admin, member, creator, guest)
	for c, answer := range cells {
		if answer == cellIssueNotFound {
			cells[c] = cellIntakeNotFound
		}
	}
	return cells
}

// inTheIntake is a cell's prepare: its column's work item put into its
// project's intake, as its submission would have left it (spec 2.3, 2.4),
// through SQL on the cell's copy at url, over a pool it closes before the
// cell's app starts: in its project's triage state, with an intake row,
// pending, by its maker; and in no relation, as no work item in the
// intake is, each of its relations deleted. The seed puts no work item in
// a cycle or a module, nor under another. gone's, deleted with gone, has
// its intake row deleted with it.
func inTheIntake(t *testing.T, c caller, s seeded, url string) {
	t.Helper()
	pool := openPool(t, url)
	defer pool.Close()
	id := s.issue(projectOf(c), issueOf(c))
	for _, step := range []struct {
		sql  string
		args []any
	}{
		{"UPDATE issues SET state_id = $2 WHERE id = $1", []any{id, s.state(projectOf(c), "Triage")}},
		{`INSERT INTO intake_issues (id, workspace_id, project_id, issue_id, status, created_by_id, updated_by_id, deleted_at)
			SELECT $2, workspace_id, project_id, id, 'pending', created_by_id, created_by_id, deleted_at FROM issues WHERE id = $1`,
			[]any{id, uuid.NewV7()}},
	} {
		if tag, err := pool.Exec(pgtest.Soon(t), step.sql, step.args...); err != nil || tag.RowsAffected() != 1 {
			t.Fatalf("putting %s's work item into the intake: %v, %d rows", c, err, tag.RowsAffected())
		}
	}
	if _, err := pool.Exec(pgtest.Soon(t), `UPDATE issue_relations SET deleted_at = now()
		WHERE $1 IN (issue_id, related_issue_id) AND deleted_at IS NULL`, id); err != nil {
		t.Fatalf("deleting the relations of %s's work item: %v", c, err)
	}
}

// submitsTheIssue: the column's project's next work item (nextNumber), Fix
// the login, in its triage state, of no priority, at a state's first sort
// order, made and last changed by the caller; its intake row, which the
// answer is read back with, pending (spec 2.4, 2.8).
func submitsTheIssue(t *testing.T, c caller, s seeded, answer string) {
	var i struct {
		ProjectID   uuid.UUID  `json:"project_id"`
		SequenceID  int        `json:"sequence_id"`
		Name        string     `json:"name"`
		StateID     uuid.UUID  `json:"state_id"`
		Priority    string     `json:"priority"`
		SortOrder   float64    `json:"sort_order"`
		CreatedByID *uuid.UUID `json:"created_by_id"`
		UpdatedByID *uuid.UUID `json:"updated_by_id"`
	}
	decodeAnswer(t, answer, &i)
	me, next := s.account(accountOf(c)), nextNumber(projectOf(c))
	if i.ProjectID != s.project(projectOf(c)) || i.SequenceID != next || i.Name != "Fix the login" ||
		i.StateID != s.state(projectOf(c), "Triage") || i.Priority != "none" || i.SortOrder != 65535 || i.CreatedByID == nil ||
		*i.CreatedByID != me || i.UpdatedByID == nil || *i.UpdatedByID != me {
		t.Errorf("%s submits %s; want %s's work item %d, Fix the login, in Triage, none, at 65535, by %s", c, answer, projectOf(c), next, me)
	}
	if got, want := intakeIn(t, answer), (&answeredIntake{Status: "pending"}); !reflect.DeepEqual(got, want) {
		t.Errorf("%s submits %s; want its intake %+v", c, answer, *want)
	}
}

// declinesIt: the column's work item as seeded, but in its triage state,
// where its prepare put it (inTheIntake), and changed last by its maker
// still: a triage but an accept changes no work item (spec 2.5); its intake
// row declined.
func declinesIt(t *testing.T, c caller, s seeded, answer string) {
	answered(t, c, s, answer, false, func(i *answeredIssue) {
		i.StateID, i.UpdatedByID = s.state(projectOf(c), "Triage"), i.CreatedByID
	})
	if got, want := intakeIn(t, answer), (&answeredIntake{Status: "declined"}); !reflect.DeepEqual(got, want) {
		t.Errorf("%s triages %s; want its intake %+v", c, answer, *want)
	}
}
