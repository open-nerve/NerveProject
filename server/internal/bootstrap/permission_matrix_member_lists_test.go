package bootstrap

import (
	"net/http"
	"testing"
	"uuid"
)

const memberIssuesPattern = "/api/v0/workspaces/{slug}/members/{user_id}/issues"

// Static completeness needs distinct account IDs; newSeeded's nil contract stays intact.
func memberTargetSeed(t *testing.T) seeded {
	s := newSeeded().in(t)
	for _, c := range matrixAccounts {
		s.accounts[c] = uuid.NewV7()
	}
	return s
}
func TestMemberIssueMatrixAccountTargets(t *testing.T) {
	s := memberTargetSeed(t)
	path := func(slug string, id uuid.UUID) string {
		return "/api/v0/workspaces/" + slug + "/members/" + id.String() + "/issues"
	}
	for _, tt := range []struct {
		name, pattern, path string
		c                   caller
		valid               bool
	}{
		{"account target differs from caller", memberIssuesPattern, path("acme", s.account(callerProjectMember)), callerAdmin, true},
		{"ended historical account", memberIssuesPattern, path("acme", s.account(callerRemoved)), callerAdmin, true},
		{"gone historical account", memberIssuesPattern, path("gone", s.account(callerMember)), callerDeleted, true},
		{"wrong workspace account", memberIssuesPattern, path("acme", s.account(callerNever)), callerAdmin, false},
		{"gone with acme only account", memberIssuesPattern, path("gone", s.account(callerProjectMember)), callerDeleted, false},
		{"wrong workspace path", memberIssuesPattern, path("other", s.account(callerNever)), callerAdmin, false},
		{"unregistered", memberIssuesPattern, path("acme", uuid.NewV7()), callerAdmin, false},
		{"membership row is no account", memberIssuesPattern, path("acme", s.membership("acme", callerProjectMember)), callerAdmin, false},
		{"invitation row is no account", memberIssuesPattern, path("acme", s.invitation("acme", "newcomer@example.com")), callerAdmin, false},
		{"nil is no registered account", memberIssuesPattern, path("acme", uuid.Nil()), callerAdmin, false},
		{"other endpoint remains strict", "/api/v0/workspaces/{slug}/other/{user_id}", "/api/v0/workspaces/acme/other/" + s.account(callerProjectMember).String(), callerAdmin, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			v := targetViolation(tt.pattern, tt.path, tt.c, s, func(string) bool { return false })
			if (v == "") != tt.valid {
				t.Fatal(v, tt.valid)
			}
		})
	}
	delete(s.memberships, "acme/"+string(callerProjectMember))
	if targetViolation(memberIssuesPattern, path("acme", s.account(callerProjectMember)), callerAdmin, s, func(string) bool { return false }) == "" {
		t.Fatal("account without seeded membership accepted")
	}
	s = memberTargetSeed(t)
	rows := memberIssueMatrixRows()
	for _, r := range rows {
		for _, c := range workspaceColumns {
			_, p, _ := r.request(c, s)
			target := callerProjectMember
			if c == callerDeleted {
				target = callerMember
			}
			want := path(workspaceOf(c), s.account(target)) + "?relation=" + r.variant
			if p != want {
				t.Fatal(c, p, want)
			}
		}
	}
}

// The two allowed workspace columns have no ordinary live project membership:
// admin was never joined, member's private project membership was deleted.
// Their empty scope is asserted here; A/B/C positive SQL sets live in the API tests.
func memberIssueMatrixRows() []matrixRow {
	rows := []matrixRow{}
	for _, relation := range []string{"assigned", "created", "subscribed"} {
		rows = append(rows, matrixRow{op: "listMemberIssues", variant: relation, cells: inWorkspace(cellOK, cellOK, cellForbidden), request: func(c caller, s seeded) (string, string, string) {
			target := callerProjectMember
			if c == callerDeleted {
				target = callerMember
			}
			return http.MethodGet, "/api/v0/workspaces/" + workspaceOf(c) + "/members/" + s.account(target).String() + "/issues?relation=" + relation, ""
		}, check: func(t *testing.T, _ caller, _ seeded, answer string) {
			p := checkListEnvelope(t, answer, false, false)
			if p.Total != 0 || len(p.Data) != 0 || p.NextCursor != nil {
				t.Fatal(answer)
			}
		}})
	}
	return rows
}
