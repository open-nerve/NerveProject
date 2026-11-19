package bootstrap

import (
	"net/http"
	"reflect"
	"slices"
	"testing"
	"uuid"
)

// The rows of the permission matrix of a project's intake list, its
// items' content edit and deletion, and a workspace's pending counts (M7
// P2b spec 2.3–2.6, 2.8). The list, under the project of each column, and
// the edit and the deletion, of the column's work item by its id
// (/intake-issues/{issue_id}), on a copy where each cell's prepare put that
// work item into the intake (inTheIntake, DP-24), pending, by its maker: so
// that the guest who sees only his own lists his own and not acme's admin's,
// and the relative rules (R11) part the columns by who made it. The
// counts, at the workspace level, on the reads' copy, where no work item
// is in an intake.

func intakeItemMatrixRows() []matrixRow {
	list := toProject(http.MethodGet, "/intake-issues", "")
	rename, deprioritize := toIntakeItem(http.MethodPatch, `{"name":"Mend"}`), toIntakeItem(http.MethodPatch, `{"priority":"high"}`)
	remove := toIntakeItem(http.MethodDelete, "")
	return []matrixRow{
		// Every active member of the project, its guests too (M7 design
		// 3.10, intake.read); a GET that writes, for its prepare.
		{op: "listIntakeIssues", write: true, columns: issueColumns, prepare: inTheIntake, request: list,
			cells: onTheIssues(cellOK, cellOK, cellOK), check: listsTheIntake},
		// An archived project's intake is read, its toggle off as well.
		{op: "listIntakeIssues", variant: "archived project", write: true, columns: archivedColumns, prepare: inTheIntake, request: list,
			cells: ofArchived(cellOK, cellForbidden), check: listsTheIntake},
		// Every active member of the workspace (intake.count, DP-19).
		{op: "getIntakeCounts", request: toWorkspace(http.MethodGet, "/intake-counts", ""), cells: inWorkspace(cellOK, cellOK, cellOK),
			check: countsTheIntakes},
		// The project's admins, and the member or the guest who submitted it
		// (spec 2.4, R11); not a member nor a guest who sees it, on acme's
		// admin's.
		{op: "updateIntakeIssue", write: true, columns: issueColumns, prepare: inTheIntake, request: rename,
			cells: bySubmitter(cellOK, cellForbidden, cellOK, cellOK), check: changesTheIntakeItem(func(i *answeredIssue) { i.Name = "Mend" })},
		// A guest who submitted it gives its name, its description and
		// skip_activity alone (R4): its priority is refused him.
		{op: "updateIntakeIssue", variant: "a field but its content", write: true, columns: issueColumns, prepare: inTheIntake, request: deprioritize,
			cells: bySubmitter(cellOK, cellForbidden, cellOK, cellForbidden), check: changesTheIntakeItem(func(i *answeredIssue) { i.Priority = "high" })},
		// An archived project's intake item is not changed, after the work
		// item's intake row is found, before its intake, off, is looked at.
		{op: "updateIntakeIssue", variant: "archived", write: true, columns: archivedColumns, prepare: inTheIntake, request: rename,
			cells: intakeRows.ofArchived(cellProjectArchived, cellForbidden)},
		// A work item in no intake is not found, after the decision, before
		// its project's 409.
		{op: "updateIntakeIssue", variant: "archived, in no intake", write: true, columns: archivedColumns, request: rename,
			cells: intakeRows.ofArchived(cellIntakeNotFound, cellForbidden)},
		// The project's admins, and the member who submitted it, while he is
		// a member (spec 2.5, AllowCreator): not a guest, who made it or not.
		{op: "deleteIntakeIssue", write: true, columns: issueColumns, prepare: inTheIntake, request: remove,
			cells: bySubmitter(cellNoContent, cellForbidden, cellNoContent, cellForbidden)},
		{op: "deleteIntakeIssue", variant: "archived", write: true, columns: archivedColumns, prepare: inTheIntake, request: remove,
			cells: intakeRows.ofArchived(cellProjectArchived, cellForbidden)},
		{op: "deleteIntakeIssue", variant: "archived, in no intake", write: true, columns: archivedColumns, request: remove,
			cells: intakeRows.ofArchived(cellIntakeNotFound, cellForbidden)},
	}
}

// toIntakeItem is the request of a row whose callers each send method, with
// body, to their column's work item (issueOf), under /intake-issues.
func toIntakeItem(method, body string) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		return intakeRows.to(method, "", issueOf(c), body)(c, s)
	}
}

// bySubmitter are the cells of a row of the issue table that changes or
// deletes an intake item by who submitted it (spec 2.4, 2.5, R11): its
// project's admin's (admin), on any; other, of the member and of the guest
// who sees every work item, each on acme's admin's; the member's who made
// it (member); the guest's who sees only his own, on his own (guest); and
// onAnIntakeItem's for the columns that do not see it or the project.
func bySubmitter(admin, other, member, guest cell) map[caller]cell {
	cells := onAnIntakeItem(admin, other, member, guest)
	cells[callerGuestAll] = other
	return cells
}

// listsTheIntake: the column's work item, which its prepare put into the
// intake (inTheIntake), pending, alone and without a cursor; none for a
// guest of acme's public project, which does not show its guests every
// work item, the workspace admin who is only its guest too, on another's.
func listsTheIntake(t *testing.T, c caller, s seeded, answer string) {
	var page struct {
		Total int64 `json:"total"`
		Data  []struct {
			ID     uuid.UUID      `json:"id"`
			Intake answeredIntake `json:"intake"`
		} `json:"data"`
		NextCursor *string `json:"next_cursor"`
	}
	listKeys(t, []byte(answer), "total", "data", "next_cursor")
	decodeAnswer(t, answer, &page)
	own := c == callerGuestOwn || c == callerGuestOthers || c == callerAdminGuest || c == callerAdminGuestOwn
	want := []uuid.UUID{}
	if _, maker := numberOf(projectOf(c), issueOf(c)); !own || maker == accountOf(c) {
		want = append(want, s.issue(projectOf(c), issueOf(c)))
	}
	got := make([]uuid.UUID, len(page.Data))
	for n, row := range page.Data {
		got[n] = row.ID
		if row.Intake != (answeredIntake{Status: "pending"}) {
			t.Errorf("%s lists %s; want its intake pending", c, answer)
		}
	}
	if page.Total != int64(len(want)) || !slices.Equal(got, want) || page.NextCursor != nil {
		t.Errorf("%s lists %s; want %v", c, answer, want)
	}
}

// changesTheIntakeItem: the column's work item as seeded (seededIssue), in
// its triage state, where its prepare put it, with change made to it, the
// caller its last writer; its intake row still pending.
func changesTheIntakeItem(change func(i *answeredIssue)) func(*testing.T, caller, seeded, string) {
	return func(t *testing.T, c caller, s seeded, answer string) {
		answered(t, c, s, answer, false, func(i *answeredIssue) {
			i.StateID = s.state(projectOf(c), "Triage")
			change(i)
		})
		if got, want := intakeIn(t, answer), (&answeredIntake{Status: "pending"}); !reflect.DeepEqual(got, want) {
			t.Errorf("%s changes %s; want its intake %+v", c, answer, *want)
		}
	}
}

// countsTheIntakes: each of acme's projects whose intake prepareMatrix
// turned on (intakeOn) that the column is an active member of, in id
// order, none pending: the public and the private one for the guest, a
// guest of both; none for the admin and the member, members of neither
// (the member's membership of the private one deleted, partingStates).
func countsTheIntakes(t *testing.T, c caller, s seeded, answer string) {
	want := []answeredCount{}
	if c == callerGuest {
		for _, id := range slices.SortedFunc(slices.Values([]uuid.UUID{s.project("acme/public"), s.project("acme/private")}), uuid.UUID.Compare) {
			want = append(want, answeredCount{ProjectID: id})
		}
	}
	if got := countsIn(t, answer); !slices.Equal(got, want) {
		t.Errorf("%s counts %s; want %+v", c, answer, want)
	}
}

// answeredCount is a project's count of pending intake work items as
// getIntakeCounts answers it.
type answeredCount struct {
	ProjectID uuid.UUID `json:"project_id"`
	Pending   int64     `json:"pending"`
}

// countsIn is the counts of the answer raw of getIntakeCounts, which has
// its data alone, a list, never null.
func countsIn(t *testing.T, raw string) []answeredCount {
	t.Helper()
	listKeys(t, []byte(raw), "data")
	var counts struct {
		Data []answeredCount `json:"data"`
	}
	decodeAnswer(t, raw, &counts)
	if counts.Data == nil {
		t.Fatalf("the counts %s: data must be a list", raw)
	}
	return counts.Data
}
