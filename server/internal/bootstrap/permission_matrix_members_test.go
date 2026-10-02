package bootstrap

import (
	"fmt"
	"net/http"
	"testing"
	"uuid"
)

// The project members' rows of the permission matrix (M3 design 9.2):
// listing, adding and joining. prepareMatrix's preconditions hold the
// accounts the rows add, and the joiners, to what the rows say of them.

func memberMatrixRows() []matrixRow {
	return []matrixRow{
		{op: "listProjectMembers", columns: projectColumns, request: toProject(http.MethodGet, "/members", ""),
			cells: ofProject(cellOK, cellOK, cellOK, cellOK, cellForbidden, cellForbidden), check: listsTheProjectMembers},
		// The project's admins, and its members who are the workspace's
		// admins (M3 design 3.5): the workspace's member, of none of the
		// projects, is added as a member.
		{op: "addProjectMembers", write: true, columns: projectColumns, request: addsToProject(callerMember, 15),
			cells: ofProject(cellCreated, cellForbidden, cellForbidden, cellCreated, cellForbidden, cellForbidden), check: addsTheMember(callerMember, 15)},
		// A target refused after the decision (M3 design 3.6 convention 3,
		// 9.2): who may add gets the 422 of the row's refusal, and no other,
		// and who may not his 403 or 404 as with a valid target, learning
		// nothing of it. X's account is no member of acme; WG-'s is its
		// guest, asked for as a member, and WA-'s its admin, asked for as a
		// member (M3 design 3.5: each joins with his own role alone); PM's is
		// the project's active member.
		{op: "addProjectMembers", variant: "a target who is no member of the workspace", write: true, columns: projectColumns,
			request: addsToProject(callerNever, 15),
			cells:   ofProject(cellValidationFailed, cellForbidden, cellForbidden, cellValidationFailed, cellForbidden, cellForbidden),
			refusal: "members[0].member_id not_allowed"},
		{op: "addProjectMembers", variant: "a workspace guest as a member", write: true, columns: projectColumns,
			request: addsToProject(callerGuestOnly, 15),
			cells:   ofProject(cellValidationFailed, cellForbidden, cellForbidden, cellValidationFailed, cellForbidden, cellForbidden),
			refusal: "members[0].role not_allowed"},
		{op: "addProjectMembers", variant: "a workspace admin as a member", write: true, columns: projectColumns,
			request: addsToProject(callerAdmin, 15),
			cells:   ofProject(cellValidationFailed, cellForbidden, cellForbidden, cellValidationFailed, cellForbidden, cellForbidden),
			refusal: "members[0].role not_allowed"},
		{op: "addProjectMembers", variant: "an active member of the project", write: true, columns: projectColumns,
			request: addsToProject(callerProjectMember, 15),
			cells:   ofProject(cellValidationFailed, cellForbidden, cellForbidden, cellValidationFailed, cellForbidden, cellForbidden),
			refusal: "members[0].member_id duplicate"},
		// An archived project's members are added as any other's (M3 design
		// 3.19).
		{op: "addProjectMembers", variant: "archived", write: true, columns: archivedColumns, request: addsToProject(callerMember, 15),
			cells: map[caller]cell{callerArchivedAdmin: cellCreated}, check: addsTheMember(callerMember, 15)},
		// Whoever sees the project but a workspace guest, its member too
		// (M3 design 3.5): an active member is left as he is; the
		// workspace's admin and member, of no project, join with their
		// workspace roles.
		{op: "joinProject", write: true, columns: projectColumns, request: toProject(http.MethodPost, "/join", ""),
			cells: ofProject(cellOK, cellOK, cellForbidden, cellOK, cellOK, cellOK), check: joinsAs},
		{op: "joinProject", variant: "archived", write: true, columns: archivedColumns, request: toProject(http.MethodPost, "/join", ""),
			cells: map[caller]cell{callerArchivedAdmin: cellOK}, check: joinsAs},
	}
}

// joinsAs: the column's project as the caller sees it, his role in it and
// its place in his sidebar: PA's 20 and PM's and PM+WA's 15, the archived
// project's admin's 20, as they were; WA-'s 20 and WM-公's 15, their
// workspace roles; each at 65535, the joiners' place (M3 design 3.18) and
// the seeded members'.
func joinsAs(t *testing.T, c caller, s seeded, answer string) {
	var p struct {
		ID         uuid.UUID `json:"id"`
		MemberRole *int      `json:"member_role"`
		SortOrder  float64   `json:"sort_order"`
	}
	decodeAnswer(t, answer, &p)
	want := map[caller]int{callerProjectAdmin: 20, callerProjectMember: 15, callerMemberAndAdmin: 15, callerArchivedAdmin: 20,
		callerAdminOnly: 20, callerMemberPublic: 15}[c]
	if p.ID != s.project(projectOf(c)) || p.MemberRole == nil || *p.MemberRole != want || p.SortOrder != 65535 {
		t.Errorf("%s joins %s; want %s, his role %d, at 65535", c, answer, projectOf(c), want)
	}
}

// addsToProject is the request of a row whose callers each add target's
// account, as role, to the project their column targets.
func addsToProject(target caller, role int) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		return http.MethodPost, "/api/v0/projects/" + s.project(projectOf(c)).String() + "/members",
			fmt.Sprintf(`{"members":[{"member_id":"%s","role":%d}]}`, s.account(target), role)
	}
}

// addsTheMember: the column's project's membership of target's account,
// with role.
func addsTheMember(target caller, role int) func(t *testing.T, c caller, s seeded, answer string) {
	return func(t *testing.T, c caller, s seeded, answer string) {
		var list struct {
			Data []struct {
				ProjectID uuid.UUID `json:"project_id"`
				MemberID  uuid.UUID `json:"member_id"`
				Role      int       `json:"role"`
			} `json:"data"`
		}
		decodeAnswer(t, answer, &list)
		if len(list.Data) != 1 || list.Data[0].ProjectID != s.project(projectOf(c)) || list.Data[0].MemberID != s.account(target) ||
			list.Data[0].Role != role {
			t.Errorf("%s adds %s; want %s's membership of %s as %d", c, answer, target, projectOf(c), role)
		}
	}
}

// listsTheProjectMembers: acme's public project's active members, each
// with his role, in the order prepareMatrix made them: PA, PM, the
// workspace's guest (PG's account), PM+WA and the removed member, whose
// membership of the project the stand-in left active (the list reads
// project_members alone, M3 design 5.2); not WG-, whose membership
// partingStates ended.
func listsTheProjectMembers(t *testing.T, c caller, s seeded, answer string) {
	var list struct {
		Data []struct {
			ProjectID uuid.UUID `json:"project_id"`
			MemberID  uuid.UUID `json:"member_id"`
			Role      int       `json:"role"`
		} `json:"data"`
	}
	decodeAnswer(t, answer, &list)
	want := []struct {
		c    caller
		role int
	}{{callerProjectAdmin, 20}, {callerProjectMember, 15}, {callerGuest, 5}, {callerMemberAndAdmin, 15}, {callerRemoved, 15}}
	ok := len(list.Data) == len(want)
	for i := 0; ok && i < len(want); i++ {
		m := list.Data[i]
		ok = m.ProjectID == s.project("acme/public") && m.MemberID == s.account(want[i].c) && m.Role == want[i].role
	}
	if !ok {
		t.Errorf("%s lists %s; want %+v of acme/public, in that order", c, answer, want)
	}
}
