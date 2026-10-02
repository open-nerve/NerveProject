package httpadapter_test

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

var (
	aliceEmail  = "alice@corp.com"
	aliceMember = domain.Member{
		Membership: domain.Membership{ID: uuid.MustParse("0199a2b4-0000-7000-8000-0000000000a1"), WorkspaceID: acmeID, MemberID: aliceID,
			Role: shared.RoleAdmin, IsActive: true, CreatedAt: created},
		User: domain.MemberUser{ID: aliceID, DisplayName: "al", FirstName: "Alice", LastName: "Liddell", Email: &aliceEmail},
	}
	bobMember = domain.Member{
		Membership: domain.Membership{ID: uuid.MustParse("0199a2b4-0000-7000-8000-0000000000b1"), WorkspaceID: acmeID, MemberID: bobID,
			Role: shared.RoleGuest, CreatedAt: created},
		User: domain.MemberUser{ID: bobID, DisplayName: "bob"},
	}
)

const (
	aliceMemberJSON = `{"created_at":"2026-09-29T10:00:00.123456Z","id":"0199a2b4-0000-7000-8000-0000000000a1","is_active":true,` +
		`"member":{"avatar_url":null,"display_name":"al","email":"alice@corp.com","first_name":"Alice",` +
		`"id":"0199a2b4-0000-7000-8000-000000000001","last_name":"Liddell"},"role":20,"workspace_id":"0199a2b4-0000-7000-8000-00000000000a"}`
	bobMemberJSON = `{"created_at":"2026-09-29T10:00:00.123456Z","id":"0199a2b4-0000-7000-8000-0000000000b1","is_active":false,` +
		`"member":{"avatar_url":null,"display_name":"bob","email":null,"first_name":"","id":"0199a2b4-0000-7000-8000-000000000002",` +
		`"last_name":""},"role":5,"workspace_id":"0199a2b4-0000-7000-8000-00000000000a"}`
)

// PATCH changes the role of the membership of the path for the caller and
// answers the membership; the role is the use case's to check.
func TestUpdateWorkspaceMember(t *testing.T) {
	role := &fakeUpdateMember{answer: aliceMember}
	h := newServer(t, fakes{role: role})
	for _, body := range []string{`{"role":20}`, `{"role":10}`} {
		res, got := do(t, h, request(http.MethodPatch, "/api/v0/workspace-members/"+aliceMember.ID.String(), "bob", body))
		if res.StatusCode != http.StatusOK || got != aliceMemberJSON+"\n" {
			t.Errorf("PATCH %s = %d %s, want 200 %s", body, res.StatusCode, got, aliceMemberJSON)
		}
	}
	id := aliceMember.ID.String()
	if want := []string{"bob " + id + " 20", "bob " + id + " 10"}; !slices.Equal(role.calls, want) {
		t.Errorf("calls = %q, want %q", role.calls, want)
	}
}

// The use case's refusals, as the contract declares them; a body without a
// role is refused before it.
func TestUpdateWorkspaceMemberRefusals(t *testing.T) {
	tests := []struct {
		err    error
		status int
		want   string
	}{
		{shared.Invalid(shared.FieldError{Field: "role", Code: shared.FieldInvalidFormat, Message: "is not 5, 15 or 20"}), http.StatusUnprocessableEntity,
			`{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"role","code":"invalid_format","message":"is not 5, 15 or 20"}]}`},
		{domain.ErrMemberNotFound, http.StatusNotFound,
			`{"status":404,"code":"workspace.member_not_found","title":"Not Found","detail":"The member does not exist, or you cannot see the workspace."}`},
		{shared.Forbidden(), http.StatusForbidden, `{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
		{domain.ErrOwnMembership, http.StatusConflict,
			`{"status":409,"code":"workspace.own_membership","title":"Conflict","detail":"You cannot change your own membership."}`},
	}
	path := "/api/v0/workspace-members/" + bobMember.ID.String()
	for _, tt := range tests {
		h := newServer(t, fakes{role: &fakeUpdateMember{err: tt.err}})
		if res, body := do(t, h, request(http.MethodPatch, path, "alice", `{"role":5}`)); res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("PATCH refused with %v = %d %s, want %d %s", tt.err, res.StatusCode, body, tt.status, tt.want)
		}
	}
	role := &fakeUpdateMember{}
	h := newServer(t, fakes{role: role})
	if res, _ := do(t, h, request(http.MethodPatch, path, "alice", `{}`)); res.StatusCode != http.StatusBadRequest || len(role.calls) != 0 {
		t.Errorf("PATCH without a role = %d, calls %q; want 400 and no call", res.StatusCode, role.calls)
	}
}

// The list is the use case's for the caller and the slug of the path: an
// address shown, one null, the avatar null, an ended membership as it is.
func TestListWorkspaceMembers(t *testing.T) {
	members := &fakeMembers{lists: map[string][]domain.Member{"acme": {aliceMember, bobMember}}}
	h := newServer(t, fakes{member: members})
	for _, tt := range []struct{ token, slug, want string }{
		{"alice", "acme", `{"data":[` + aliceMemberJSON + `,` + bobMemberJSON + `]}`},
		{"bob", "beta", `{"data":[]}`},
	} {
		res, body := do(t, h, request(http.MethodGet, "/api/v0/workspaces/"+tt.slug+"/members", tt.token, ""))
		if res.StatusCode != http.StatusOK || body != tt.want+"\n" {
			t.Errorf("%s GET %s's members = %d %s, want 200 %s", tt.token, tt.slug, res.StatusCode, body, tt.want)
		}
	}
	if want := []string{"alice acme", "bob beta"}; !slices.Equal(members.calls, want) {
		t.Errorf("calls = %q, want %q", members.calls, want)
	}
	h = newServer(t, fakes{member: &fakeMembers{err: domain.ErrNotFound}})
	res, body := do(t, h, request(http.MethodGet, "/api/v0/workspaces/acme/members", "bob", ""))
	if want := `{"status":404,"code":"workspace.not_found","title":"Not Found","detail":"The workspace does not exist, or you are not a member of it."}`; res.StatusCode != http.StatusNotFound || body != want+"\n" {
		t.Errorf("GET refused = %d %s, want 404 %s", res.StatusCode, body, want)
	}
}

// DELETE removes the membership of the path for the caller and answers 204
// with no body.
func TestRemoveWorkspaceMember(t *testing.T) {
	remove := &fakeRemoveMember{}
	h := newServer(t, fakes{remove: remove})
	for _, token := range []string{"alice", "bob"} {
		if res, body := do(t, h, request(http.MethodDelete, "/api/v0/workspace-members/"+bobMember.ID.String(), token, "")); res.StatusCode != http.StatusNoContent ||
			body != "" {
			t.Errorf("%s's DELETE = %d %q, want 204 and no body", token, res.StatusCode, body)
		}
	}
	if want := []string{"alice " + bobMember.ID.String(), "bob " + bobMember.ID.String()}; !slices.Equal(remove.calls, want) {
		t.Errorf("calls = %q, want %q", remove.calls, want)
	}
}

// The use case's refusals, as the contract declares them: the project
// module's project.sole_admin comes through as it is, a 409 of the
// workspace's operation (M3 design 9.4); and its failure, a 500, never a
// 204 or another problem.
func TestRemoveWorkspaceMemberRefusals(t *testing.T) {
	// The project module's ErrSoleAdmin, which this module does not import:
	// its kind, code and detail.
	soleAdmin := shared.NewError(shared.KindConflict, "project.sole_admin",
		"Ending the membership would leave a project that has other members without an admin; make another of its members an admin first.")
	tests := []struct {
		err    error
		status int
		want   string
	}{
		{domain.ErrMemberNotFound, http.StatusNotFound,
			`{"status":404,"code":"workspace.member_not_found","title":"Not Found","detail":"The member does not exist, or you cannot see the workspace."}`},
		{shared.Forbidden(), http.StatusForbidden, `{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
		{domain.ErrOwnMembership, http.StatusConflict,
			`{"status":409,"code":"workspace.own_membership","title":"Conflict","detail":"You cannot change your own membership."}`},
		{fmt.Errorf("end the member's project memberships: %w", soleAdmin), http.StatusConflict,
			`{"status":409,"code":"project.sole_admin","title":"Conflict","detail":"Ending the membership would leave a project that has ` +
				`other members without an admin; make another of its members an admin first."}`},
		{errors.New("the database is gone"), http.StatusInternalServerError,
			`{"status":500,"code":"internal_error","title":"Internal Server Error"}`},
	}
	for _, tt := range tests {
		h := newServer(t, fakes{remove: &fakeRemoveMember{err: tt.err}})
		res, body := do(t, h, request(http.MethodDelete, "/api/v0/workspace-members/"+bobMember.ID.String(), "alice", ""))
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("DELETE refused with %v = %d %s, want %d %s", tt.err, res.StatusCode, body, tt.status, tt.want)
		}
	}
	remove := &fakeRemoveMember{}
	h := newServer(t, fakes{remove: remove})
	if res, _ := do(t, h, request(http.MethodDelete, "/api/v0/workspace-members/not-a-uuid", "alice", "")); res.StatusCode != http.StatusBadRequest ||
		len(remove.calls) != 0 {
		t.Errorf("DELETE of no membership id = %d, calls %q; want 400 and no call", res.StatusCode, remove.calls)
	}
}
