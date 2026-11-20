package bootstrap

import (
	"net/http"
	"slices"
	"testing"
	"uuid"
)

// Archived-item rows prepare a separate database per cell (write:true);
// their actual request remains GET and takes no write gate or lock.
func issueListMatrixRows() []matrixRow {
	return []matrixRow{
		{op: "listIssues", columns: issueColumns, request: toProject(http.MethodGet, "/issues", ""), cells: onTheIssues(cellOK, cellOK, cellOK), check: listsProjectIssues(false)},
		{op: "listIssues", variant: "archived issues", columns: issueColumns, write: true, prepare: inDone(true), request: toProject(http.MethodGet, "/issues?archived=true", ""), cells: onTheIssues(cellOK, cellOK, cellForbidden), check: listsProjectIssues(true)},
		{op: "listIssues", variant: "archived project", columns: archivedColumns, request: toProject(http.MethodGet, "/issues", ""), cells: ofArchived(cellOK, cellForbidden), check: listsProjectIssues(false)},
		{op: "listIssues", variant: "archived issues in archived project", columns: archivedColumns, write: true, prepare: inDone(true), request: toProject(http.MethodGet, "/issues?archived=true", ""), cells: ofArchived(cellOK, cellForbidden), check: listsProjectIssues(true)},
		// Every active member of the workspace (workspace_issue.list, M7 P3a
		// spec 2.3), each over the projects he is an active member of.
		{op: "listWorkspaceIssues", request: toWorkspace(http.MethodGet, "/issues", ""), cells: inWorkspace(cellOK, cellOK, cellOK), check: listsWorkspaceIssues},
	}
}

// listsWorkspaceIssues checks a workspace column's list: the admin's and the
// member's columns are active members of no project of acme, so theirs is
// empty; the guest's is every work item of the project that shows its guests
// all (guestViewAll) and his own of the public one, whose guest sees only
// his own (spec 2.2), as the search's scope (searchMatrixRows).
func listsWorkspaceIssues(t *testing.T, c caller, s seeded, answer string) {
	page := checkListEnvelope(t, answer, false, false)
	want := []uuid.UUID{}
	if c == callerGuest {
		for _, item := range seededItems() {
			if item.key == guestViewAll || item.key == "acme/public" && item.by == callerGuest {
				want = append(want, s.issue(item.key, item.name))
			}
		}
	}
	got := make([]uuid.UUID, len(page.Data))
	for n, i := range page.Data {
		got[n] = i.ID
	}
	slices.SortFunc(got, uuid.UUID.Compare)
	slices.SortFunc(want, uuid.UUID.Compare)
	if page.Total != int64(len(want)) || !slices.Equal(got, want) || page.NextCursor != nil {
		t.Fatalf("%s lists %s; want %v", c, answer, want)
	}
}
func listsProjectIssues(archived bool) func(*testing.T, caller, seeded, string) {
	return func(t *testing.T, c caller, s seeded, answer string) {
		page := checkListEnvelope(t, answer, false, false)
		own := c == callerGuestOwn || c == callerGuestOthers || c == callerAdminGuest || c == callerAdminGuestOwn
		want := []uuid.UUID{}
		for _, item := range seededItems() {
			if item.key == projectOf(c) && (!archived || item.name == issueOf(c)) && (!own || item.by == accountOf(c)) {
				want = append(want, s.issue(item.key, item.name))
			}
		}
		got := make([]uuid.UUID, len(page.Data))
		for n, i := range page.Data {
			got[n] = i.ID
		}
		slices.SortFunc(got, uuid.UUID.Compare)
		slices.SortFunc(want, uuid.UUID.Compare)
		if page.Total != int64(len(want)) || !slices.Equal(got, want) || page.NextCursor != nil {
			t.Fatalf("%s lists %s; want %v", c, answer, want)
		}
	}
}
