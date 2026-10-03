package bootstrap

import (
	"context"
	"net/http"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The rows of the permission matrix of the writes on one project
// membership (M3 design 9.2): changing a member's role, removing a member.
// Each names the membership by its id
// (/project-members/{project_member_id}), in its column's project;
// prepareMatrix's preconditions hold each membership a row names to the
// state the row says.

var (
	cellProjectMemberNotFound = cell{http.StatusNotFound, "project.member_not_found"}
	cellProjectOwnMembership  = cell{http.StatusConflict, "project.own_membership"}
	cellRoleTooHigh           = cell{http.StatusForbidden, "project.role_too_high"}
)

// ofMembership are the cells of a row of a write on a project membership:
// the answers of PA, PM, PG, PM+WA, WA- and WM-公, and
// project.member_not_found for the columns that do not see their project:
// WM-私, WG-, P-前 and X.
func ofMembership(pa, pm, pg, pmwa, wa, wm cell) map[caller]cell {
	return map[caller]cell{callerProjectAdmin: pa, callerProjectMember: pm, callerProjectGuest: pg, callerMemberAndAdmin: pmwa,
		callerAdminOnly: wa, callerMemberPublic: wm, callerMemberPrivate: cellProjectMemberNotFound, callerGuestOnly: cellProjectMemberNotFound,
		callerBefore: cellProjectMemberNotFound, callerNever: cellProjectMemberNotFound, callerRemoved: cellProjectMemberNotFound,
		callerDeleted: cellProjectMemberNotFound}
}

// aMembership names, for a column, the membership of its project a row's
// cell names: the project's key, and the column whose membership it is.
type aMembership func(c caller) (key string, member caller)

// projectMemberOf: PM's membership of the column's project, active, 15;
// in gone's project, where PM has none, gone's member's, deleted with it.
func projectMemberOf(c caller) (string, caller) {
	if key := projectOf(c); key == "gone/project" {
		return key, callerMember
	}
	return projectOf(c), callerProjectMember
}

// ownMembershipOf: the column's own membership of its project, for the
// columns that have one (PA, PM, PG, the workspace's guest's account, and
// PM+WA); PM's for the others.
func ownMembershipOf(c caller) (string, caller) {
	switch c {
	case callerProjectAdmin, callerProjectMember, callerMemberAndAdmin:
		return projectOf(c), c
	case callerProjectGuest:
		return projectOf(c), callerGuest
	}
	return projectMemberOf(c)
}

// endedMembershipOf: an ended membership of the column's project: the
// removed member's of the public one, which his removal ended, the member
// before's of the private one; in gone's project, gone's member's.
func endedMembershipOf(c caller) (string, caller) {
	switch key := projectOf(c); key {
	case "acme/public":
		return key, callerRemoved
	case "acme/private":
		return key, callerBefore
	}
	return projectMemberOf(c)
}

// workspaceGuestOf: the membership of the column's project of the
// workspace's guest (PG's account), a guest's; in gone's project, gone's
// member's.
func workspaceGuestOf(c caller) (string, caller) {
	if key := projectOf(c); key != "gone/project" {
		return key, callerGuest
	}
	return projectMemberOf(c)
}

// toProjectMembership is the request of a row whose callers each send
// method, with body, to the membership which names for their column.
func toProjectMembership(method, body string, which aMembership) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		key, member := which(c)
		return method, "/api/v0/project-members/" + s.projectMember(key, member).String(), body
	}
}

func membershipMatrixRows() []matrixRow {
	return []matrixRow{
		// The project's admins, and its members who are the workspace's
		// admins (M3 design 3.5, 9.2): PM's membership, his own in his
		// column, made a guest's. PM gets the rule's 403, not the 409 of his
		// own: the decision comes first.
		{op: "updateProjectMember", write: true, columns: projectColumns, request: toProjectMembership(http.MethodPatch, `{"role":5}`, projectMemberOf),
			cells: ofMembership(cellOK, cellForbidden, cellForbidden, cellOK, cellForbidden, cellForbidden), check: changesTheRole(projectMemberOf, 5)},
		// One's own role, to an admin's: the project admin who is no
		// workspace admin may not (409); the workspace's admin who is a
		// project member may (M3 design 3.5's exception).
		{op: "updateProjectMember", variant: "one's own membership", write: true, columns: projectColumns,
			request: toProjectMembership(http.MethodPatch, `{"role":20}`, ownMembershipOf),
			cells:   ofMembership(cellProjectOwnMembership, cellForbidden, cellForbidden, cellOK, cellForbidden, cellForbidden),
			check:   changesTheRole(ownMembershipOf, 20)},
		// An ended membership is 404 to who may change roles, after the
		// decision: who may not gets his 403, and learns nothing of it.
		{op: "updateProjectMember", variant: "an ended membership", write: true, columns: projectColumns,
			request: toProjectMembership(http.MethodPatch, `{"role":5}`, endedMembershipOf),
			cells:   ofMembership(cellProjectMemberNotFound, cellForbidden, cellForbidden, cellProjectMemberNotFound, cellForbidden, cellForbidden)},
		// A workspace guest stays a guest, whoever changes his role (M3
		// design 3.5), after the decision.
		{op: "updateProjectMember", variant: "a workspace guest made a member", write: true, columns: projectColumns,
			request: toProjectMembership(http.MethodPatch, `{"role":15}`, workspaceGuestOf),
			cells:   ofMembership(cellValidationFailed, cellForbidden, cellForbidden, cellValidationFailed, cellForbidden, cellForbidden),
			refusal: "role not_allowed"},
		// As updateProjectMember (M3 design 3.5, 9.2): PM's membership ends;
		// PM+WA, a member, removes a member, of his own role.
		{op: "removeProjectMember", write: true, columns: projectColumns, request: toProjectMembership(http.MethodDelete, "", projectMemberOf),
			cells: ofMembership(cellNoContent, cellForbidden, cellForbidden, cellNoContent, cellForbidden, cellForbidden)},
		// Nobody removes his own membership, the workspace's admin neither:
		// he leaves the project (M3 design 3.5).
		{op: "removeProjectMember", variant: "one's own membership", write: true, columns: projectColumns,
			request: toProjectMembership(http.MethodDelete, "", ownMembershipOf),
			cells:   ofMembership(cellProjectOwnMembership, cellForbidden, cellForbidden, cellProjectOwnMembership, cellForbidden, cellForbidden)},
		{op: "removeProjectMember", variant: "an ended membership", write: true, columns: projectColumns,
			request: toProjectMembership(http.MethodDelete, "", endedMembershipOf),
			cells:   ofMembership(cellProjectMemberNotFound, cellForbidden, cellForbidden, cellProjectMemberNotFound, cellForbidden, cellForbidden)},
		// The admin's membership: PA's own (409); PM+WA, a member, may not
		// remove a role above his own, though he is the workspace's admin
		// (M3 design 3.5: no exception here).
		{op: "removeProjectMember", variant: "an admin's membership", write: true, columns: projectColumns,
			request: toProjectMembership(http.MethodDelete, "", adminOf),
			cells:   ofMembership(cellProjectOwnMembership, cellForbidden, cellForbidden, cellRoleTooHigh, cellForbidden, cellForbidden)},
	}
}

// adminOf: PA's membership of the column's project, an admin's; in gone's
// project, gone's admin's.
func adminOf(c caller) (string, caller) {
	if key := projectOf(c); key != "gone/project" {
		return key, callerProjectAdmin
	}
	return "gone/project", callerDeleted
}

// memberships checks the project memberships the rows of the writes on one
// name (membershipMatrixRows), each in the state its row says: PM's, PA's,
// PM+WA's and the workspace guest's (PG's account) active, of their roles;
// the removed member's of the public project and the member before's of
// the private one ended, not deleted; gone's member's and admin's of gone's
// project deleted with gone. The workspace guest is acme's active guest, so that a
// role's 422 is his workspace role's.
func (s projectSeed) memberships(sd seeded) {
	s.t.Helper()
	ctx := context.Background()
	for _, tt := range []struct {
		key           string
		c             caller
		role          shared.Role
		found, active bool
	}{
		{"acme/public", callerProjectMember, shared.RoleMember, true, true}, {"acme/private", callerProjectMember, shared.RoleMember, true, true},
		{"acme/public", callerProjectAdmin, shared.RoleAdmin, true, true}, {"acme/private", callerProjectAdmin, shared.RoleAdmin, true, true},
		{"acme/public", callerMemberAndAdmin, shared.RoleMember, true, true},
		{"acme/public", callerGuest, shared.RoleGuest, true, true}, {"acme/private", callerGuest, shared.RoleGuest, true, true},
		{"acme/public", callerRemoved, shared.RoleMember, true, false}, {"acme/private", callerBefore, shared.RoleMember, true, false},
		{"gone/project", callerMember, 0, false, false}, {"gone/project", callerDeleted, 0, false, false},
	} {
		m, found, err := s.store.MemberByID(ctx, sd.projectMember(tt.key, tt.c))
		if err != nil || found != tt.found || found && (m.ProjectID != sd.project(tt.key) || m.MemberID != s.ids[tt.c] || m.Role != tt.role ||
			m.Active != tt.active) {
			s.t.Fatalf("%s's membership of %s = %+v, %v, %v; want found %v, active %v, of role %d", tt.c, tt.key, m, found, err, tt.found,
				tt.active, tt.role)
		}
	}
	if role, active, err := s.matrixSeed.store.ActiveRole(ctx, sd.workspace("acme"), s.ids[callerGuest]); err != nil || !active ||
		role != shared.RoleGuest {
		s.t.Fatalf("%s's role in acme = %d, %v, %v; want its active guest", callerGuest, role, active, err)
	}
}

// changesTheRole: the membership which names for the column, of its
// project and member, with role.
func changesTheRole(which aMembership, role int) func(t *testing.T, c caller, s seeded, answer string) {
	return func(t *testing.T, c caller, s seeded, answer string) {
		var m struct {
			ID        uuid.UUID `json:"id"`
			ProjectID uuid.UUID `json:"project_id"`
			MemberID  uuid.UUID `json:"member_id"`
			Role      int       `json:"role"`
		}
		decodeAnswer(t, answer, &m)
		key, member := which(c)
		if m.ID != s.projectMember(key, member) || m.ProjectID != s.project(key) || m.MemberID != s.account(accountOf(member)) || m.Role != role {
			t.Errorf("%s changes %s; want %s's membership of %s as %d", c, answer, member, key, role)
		}
	}
}
