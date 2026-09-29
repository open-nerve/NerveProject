package bootstrap

import (
	"net/http"
	"slices"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
)

// The workspace module's rows of the permission matrix (M3 design 9.2).

var (
	cellWorkspaceNotFound = cell{http.StatusNotFound, "workspace.not_found"}
	cellCreationDisabled  = cell{http.StatusForbidden, "workspace.creation_disabled"}
)

// inWorkspace are the cells of a workspace-level row: the answer of the
// workspace's admin, member and guest, and workspace.not_found for the
// callers the workspace is not visible to.
func inWorkspace(admin, member, guest cell) map[caller]cell {
	return map[caller]cell{callerAdmin: admin, callerMember: member, callerGuest: guest,
		callerNever: cellWorkspaceNotFound, callerRemoved: cellWorkspaceNotFound, callerDeleted: cellWorkspaceNotFound}
}

// toWorkspace is the request of a row whose callers each send method to the
// path under the workspace their column targets.
func toWorkspace(method, path, body string) func(caller) (string, string, string) {
	return func(c caller) (string, string, string) {
		return method, "/api/v0/workspaces/" + workspaceOf(c) + path, body
	}
}

func workspaceMatrixRows() []matrixRow {
	return []matrixRow{
		// The account level: any valid credential (6.4).
		{op: "listWorkspaces", request: sameRequest(http.MethodGet, "/api/v0/workspaces", ""), cells: every(cellOK),
			check: listsItsWorkspaces},
		{op: "checkWorkspaceSlug", request: sameRequest(http.MethodGet, "/api/v0/workspace-slugs/acme", ""), cells: every(cellOK)},
		{op: "createWorkspace", write: true, request: sameRequest(http.MethodPost, "/api/v0/workspaces", `{"name":"New","slug":"new"}`),
			cells: every(cellCreated)},
		{op: "createWorkspace", variant: "creation switched off", write: true,
			config:  func(cfg *config.Config) { cfg.Workspace.CreationEnabled = false },
			request: sameRequest(http.MethodPost, "/api/v0/workspaces", `{"name":"New","slug":"new"}`), cells: every(cellCreationDisabled)},
		// The workspace level.
		{op: "getWorkspace", request: toWorkspace(http.MethodGet, "", ""), cells: inWorkspace(cellOK, cellOK, cellOK),
			check: readsItsRole},
		{op: "updateWorkspace", write: true, request: toWorkspace(http.MethodPatch, "", `{"name":"Renamed"}`),
			cells: inWorkspace(cellOK, cellForbidden, cellForbidden), check: renamesIt},
		{op: "getWorkspacePreferences", request: toPreferences(http.MethodGet, ""), cells: inWorkspace(cellOK, cellOK, cellOK),
			check: preferencesAre(navigation{"TABBED", 3}, navigation{"ACCORDION", 10})},
		{op: "updateWorkspacePreferences", write: true, request: toPreferences(http.MethodPatch, `{"navigation_project_limit":5}`),
			cells: inWorkspace(cellOK, cellOK, cellOK), check: preferencesAre(navigation{"TABBED", 5}, navigation{"ACCORDION", 5})},
	}
}

// toPreferences is the request of a row whose callers each send method to
// their own display settings in the workspace their column targets.
func toPreferences(method, body string) func(caller) (string, string, string) {
	return func(c caller) (string, string, string) {
		return method, "/api/v0/me/workspaces/" + workspaceOf(c) + "/preferences", body
	}
}

// navigation is a caller's display settings as an answer holds them.
type navigation struct {
	Mode  string `json:"navigation_control_preference"`
	Limit int    `json:"navigation_project_limit"`
}

// preferencesAre: each caller reads and changes his own settings; the
// admin's answer is admin, the member's and the guest's others.
func preferencesAre(admin, others navigation) func(t *testing.T, c caller, answer string) {
	return func(t *testing.T, c caller, answer string) {
		var got navigation
		decodeAnswer(t, answer, &got)
		want := others
		if c == callerAdmin {
			want = admin
		}
		if got != want {
			t.Errorf("%s's settings are %+v, want %+v", c, got, want)
		}
	}
}

// renamesIt: the admin's answer is acme renamed, with the admin's role.
func renamesIt(t *testing.T, c caller, answer string) {
	var w struct {
		Slug string `json:"slug"`
		Name string `json:"name"`
		Role int    `json:"role"`
	}
	decodeAnswer(t, answer, &w)
	if w.Slug != "acme" || w.Name != "Renamed" || w.Role != 20 {
		t.Errorf("%s's update answers %+v, want acme named Renamed, role 20", c, w)
	}
}

// listsItsWorkspaces: each caller's list is the workspaces it is an active
// member of, never acme for the callers it is not visible to, and not the
// deleted gone.
func listsItsWorkspaces(t *testing.T, c caller, answer string) {
	var list struct {
		Data []struct {
			Slug string `json:"slug"`
		} `json:"data"`
	}
	decodeAnswer(t, answer, &list)
	want := map[caller][]string{callerAdmin: {"acme"}, callerMember: {"acme"}, callerGuest: {"acme"},
		callerNever: {"other"}, callerRemoved: {"other"}, callerDeleted: {}}[c]
	got := []string{}
	for _, w := range list.Data {
		got = append(got, w.Slug)
	}
	if !slices.Equal(got, want) {
		t.Errorf("%s lists %q, want %q", c, got, want)
	}
}

// readsItsRole: the workspace read is acme with the caller's own role.
func readsItsRole(t *testing.T, c caller, answer string) {
	var w struct {
		Slug string `json:"slug"`
		Role int    `json:"role"`
	}
	decodeAnswer(t, answer, &w)
	want := map[caller]int{callerAdmin: 20, callerMember: 15, callerGuest: 5}[c]
	if w.Slug != "acme" || w.Role != want {
		t.Errorf("%s reads %s as role %d, want acme as role %d", c, w.Slug, w.Role, want)
	}
}
