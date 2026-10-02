package bootstrap

import (
	"fmt"
	"net/http"
	"slices"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
)

// The workspace module's rows of the permission matrix (M3 design 9.2).

var (
	cellWorkspaceNotFound = cell{http.StatusNotFound, "workspace.not_found"}
	cellCreationDisabled  = cell{http.StatusForbidden, "workspace.creation_disabled"}
	cellMemberNotFound    = cell{http.StatusNotFound, "workspace.member_not_found"}
	cellOwnMembership     = cell{http.StatusConflict, "workspace.own_membership"}
	cellValidationFailed  = cell{http.StatusUnprocessableEntity, "validation_failed"}
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
func toWorkspace(method, path, body string) func(caller, seeded) (string, string, string) {
	return func(c caller, _ seeded) (string, string, string) {
		return method, "/api/v0/workspaces/" + workspaceOf(c) + path, body
	}
}

// ofMember are the cells of a row that names a membership: the answers of
// the workspace's admin, member and guest, and workspace.member_not_found
// for the callers the workspace is not visible to.
func ofMember(admin, member, guest cell) map[caller]cell {
	return map[caller]cell{callerAdmin: admin, callerMember: member, callerGuest: guest,
		callerNever: cellMemberNotFound, callerRemoved: cellMemberNotFound, callerDeleted: cellMemberNotFound}
}

// toMembership is the request of a row whose callers each send method,
// with body, to the membership target names for their column: a
// workspace's slug and whose membership of it.
func toMembership(method string, target func(caller) (string, caller), body string) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		slug, who := target(c)
		return method, "/api/v0/workspace-members/" + s.membership(slug, who).String(), body
	}
}

// anotherMember is, for each column, a membership of another account in the
// workspace the column targets: the member's, or the guest's for the member
// himself.
func anotherMember(c caller) (string, caller) {
	switch c {
	case callerMember:
		return "acme", callerGuest
	case callerDeleted:
		return "gone", callerMember
	}
	return "acme", callerMember
}

// endedMembership is, for each column, an ended membership of the
// workspace the column targets: the removed member's of acme; for the
// deleted workspace's column, the member's of gone, deleted with it.
func endedMembership(c caller) (string, caller) {
	if c == callerDeleted {
		return "gone", callerMember
	}
	return "acme", callerRemoved
}

// ownMembership is, for each column, the caller's own membership of the
// workspace the column targets: ended for the removed member, deleted with
// gone. Never a member has none in acme: his cell names the member's.
func ownMembership(c caller) (string, caller) {
	switch c {
	case callerNever:
		return "acme", callerMember
	case callerDeleted:
		return "gone", callerDeleted
	}
	return "acme", c
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
		// The answers to an invitation: each column's own, and acme's
		// newcomer's, to another address (M3 design 9.2).
		{op: "acceptWorkspaceInvitation", variant: "one's own", write: true, request: toOwnInvitation("accept"), cells: every(cellOK),
			check: joinsAsAMember},
		{op: "acceptWorkspaceInvitation", variant: "another's", write: true, request: toNewcomersInvitation("accept"),
			cells: every(cellEmailMismatch)},
		{op: "declineWorkspaceInvitation", variant: "one's own", write: true, request: toOwnInvitation("decline"), cells: every(cellNoContent)},
		{op: "declineWorkspaceInvitation", variant: "another's", write: true, request: toNewcomersInvitation("decline"),
			cells: every(cellEmailMismatch)},
		// The workspace level.
		{op: "getWorkspace", request: toWorkspace(http.MethodGet, "", ""), cells: inWorkspace(cellOK, cellOK, cellOK),
			check: readsItsRole},
		{op: "updateWorkspace", write: true, request: toWorkspace(http.MethodPatch, "", `{"name":"Renamed"}`),
			cells: inWorkspace(cellOK, cellForbidden, cellForbidden), check: renamesIt},
		{op: "deleteWorkspace", write: true, request: toWorkspace(http.MethodDelete, "", ""),
			cells: inWorkspace(cellNoContent, cellForbidden, cellForbidden)},
		{op: "listWorkspaceMembers", request: toWorkspace(http.MethodGet, "/members", ""), cells: inWorkspace(cellOK, cellOK, cellOK),
			check: listsTheMembers},
		{op: "updateWorkspaceMember", variant: "another member", write: true, request: toMembership(http.MethodPatch, anotherMember, `{"role":5}`),
			cells: ofMember(cellOK, cellForbidden, cellForbidden), check: demotesTheMember},
		{op: "updateWorkspaceMember", variant: "one's own", write: true, request: toMembership(http.MethodPatch, ownMembership, `{"role":15}`),
			cells: ofMember(cellOwnMembership, cellForbidden, cellForbidden)},
		// The admins remove another member (M3 design 9.2); to them alone an
		// ended membership is not found, and their own is the 409 of one's
		// own membership (3.4): a member or a guest is refused before any
		// check of the target.
		{op: "removeWorkspaceMember", variant: "another member", write: true, request: toMembership(http.MethodDelete, anotherMember, ""),
			cells: ofMember(cellNoContent, cellForbidden, cellForbidden)},
		{op: "removeWorkspaceMember", variant: "one's own", write: true, request: toMembership(http.MethodDelete, ownMembership, ""),
			cells: ofMember(cellOwnMembership, cellForbidden, cellForbidden)},
		{op: "removeWorkspaceMember", variant: "an ended membership", write: true, request: toMembership(http.MethodDelete, endedMembership, ""),
			cells: ofMember(cellMemberNotFound, cellForbidden, cellForbidden)},
		{op: "listWorkspaceInvitations", request: toWorkspace(http.MethodGet, "/invitations", ""),
			cells: inWorkspace(cellOK, cellForbidden, cellForbidden), check: listsTheInvitations},
		{op: "createWorkspaceInvitations", write: true, request: toWorkspace(http.MethodPost, "/invitations", inviting("invitee@example.com")),
			cells: inWorkspace(cellCreated, cellForbidden, cellForbidden), check: invitesTheInvitee},
		// The addresses the workspace refuses, read through the wired
		// MemberProfiles and the store: an active member's, an invited one's.
		{op: "createWorkspaceInvitations", variant: "an active member's address", write: true,
			request: toWorkspace(http.MethodPost, "/invitations", inviting("member@example.com")),
			cells:   inWorkspace(cellValidationFailed, cellForbidden, cellForbidden)},
		{op: "createWorkspaceInvitations", variant: "an invited address", write: true,
			request: toWorkspace(http.MethodPost, "/invitations", inviting("newcomer@example.com")),
			cells:   inWorkspace(cellValidationFailed, cellForbidden, cellForbidden)},
		{op: "updateWorkspaceInvitation", write: true, request: toInvitation(http.MethodPatch, "newcomer@example.com", `{"role":20}`),
			cells: ofInvitation(cellOK, cellForbidden, cellForbidden), check: promotesTheNewcomer},
		{op: "deleteWorkspaceInvitation", write: true, request: toInvitation(http.MethodDelete, "newcomer@example.com", ""),
			cells: ofInvitation(cellNoContent, cellForbidden, cellForbidden)},
		{op: "getWorkspacePreferences", request: toPreferences(http.MethodGet, ""), cells: inWorkspace(cellOK, cellOK, cellOK),
			check: preferencesAre(navigation{"TABBED", 3}, navigation{"ACCORDION", 10})},
		{op: "updateWorkspacePreferences", write: true, request: toPreferences(http.MethodPatch, `{"navigation_project_limit":5}`),
			cells: inWorkspace(cellOK, cellOK, cellOK), check: preferencesAre(navigation{"TABBED", 5}, navigation{"ACCORDION", 5})},
	}
}

// demotesTheMember: the admin's answer is the member's membership, now a
// guest's, with the member's address.
func demotesTheMember(t *testing.T, c caller, _ seeded, answer string) {
	var m struct {
		Role   int `json:"role"`
		Member struct {
			Email string `json:"email"`
		} `json:"member"`
	}
	decodeAnswer(t, answer, &m)
	if m.Role != 5 || m.Member.Email != "member@example.com" {
		t.Errorf("%s's change answers %+v, want the member as a guest", c, m)
	}
}

// listsTheMembers: acme's nine memberships, the removed member's ended; the
// admin and the member see every address, the guest none, his own neither
// (M3 design 3.4, 9.2).
func listsTheMembers(t *testing.T, c caller, _ seeded, answer string) {
	var list struct {
		Data []struct {
			IsActive bool `json:"is_active"`
			Member   struct {
				DisplayName string  `json:"display_name"`
				Email       *string `json:"email"`
			} `json:"member"`
		} `json:"data"`
	}
	decodeAnswer(t, answer, &list)
	var got, want []string
	for _, m := range list.Data {
		email := "null"
		if m.Member.Email != nil {
			email = *m.Member.Email
		}
		got = append(got, fmt.Sprintf("%s %s active %v", m.Member.DisplayName, email, m.IsActive))
	}
	for _, name := range []string{"admin", "member", "guest", "removed", "project-admin", "project-member", "project-member-and-workspace-admin",
		"workspace-guest-only", "project-member-before"} {
		email := name + "@example.com"
		if c == callerGuest {
			email = "null"
		}
		want = append(want, fmt.Sprintf("%s %s active %v", name, email, name != "removed"))
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("%s sees the members %q, want %q", c, got, want)
	}
}

// toPreferences is the request of a row whose callers each send method to
// their own display settings in the workspace their column targets.
func toPreferences(method, body string) func(caller, seeded) (string, string, string) {
	return func(c caller, _ seeded) (string, string, string) {
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
func preferencesAre(admin, others navigation) func(t *testing.T, c caller, s seeded, answer string) {
	return func(t *testing.T, c caller, _ seeded, answer string) {
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
func renamesIt(t *testing.T, c caller, _ seeded, answer string) {
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
// deleted gone. A column without a list stated here fails: it would pass by
// listing nothing.
func listsItsWorkspaces(t *testing.T, c caller, _ seeded, answer string) {
	var list struct {
		Data []struct {
			Slug string `json:"slug"`
		} `json:"data"`
	}
	decodeAnswer(t, answer, &list)
	want, stated := map[caller][]string{callerAdmin: {"acme"}, callerMember: {"acme"}, callerGuest: {"acme"},
		callerNever: {"other"}, callerRemoved: {"other"}, callerDeleted: {}}[c]
	if !stated {
		t.Fatalf("no list stated for %s", c)
	}
	got := []string{}
	for _, w := range list.Data {
		got = append(got, w.Slug)
	}
	if !slices.Equal(got, want) {
		t.Errorf("%s lists %q, want %q", c, got, want)
	}
}

// readsItsRole: the workspace read is acme with the caller's own role.
func readsItsRole(t *testing.T, c caller, _ seeded, answer string) {
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
