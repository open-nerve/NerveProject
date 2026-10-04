package httpadapter_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeUpdateMember is updateProjectMember: each call is recorded as
// "caller id role"; it answers answer, or err.
type fakeUpdateMember struct {
	calls  []string
	answer domain.Member
	err    error
}

func (f *fakeUpdateMember) Execute(ctx context.Context, id uuid.UUID, role shared.Role) (domain.Member, error) {
	f.calls = append(f.calls, fmt.Sprintf("%s %s %d", caller(ctx), id, role))
	return f.answer, f.err
}

// The answers of the writes on a project membership, as the contract
// declares them: each refusal, and a failure's 500, never a 200, a 204 or
// another problem.
const (
	memberNotFoundJSON = `{"status":404,"code":"project.member_not_found","title":"Not Found",` +
		`"detail":"The project member does not exist, or you cannot see the project."}`
	forbiddenJSON     = `{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`
	ownMembershipJSON = `{"status":409,"code":"project.own_membership","title":"Conflict","detail":"You cannot remove your own membership ` +
		`of the project, nor change your own role in it unless you are a workspace admin."}`
	roleTooHighJSON = `{"status":403,"code":"project.role_too_high","title":"Forbidden","detail":"The role is too high for you. Unless you ` +
		`are a workspace admin, you change only members whose project role is below yours, to roles below yours. You remove only members ` +
		`whose project role is not above yours."}`
	internalErrorJSON = `{"status":500,"code":"internal_error","title":"Internal Server Error"}`
)

// errGone is the error of a use case that failed.
var errGone = errors.New("the database is gone")

// PATCH goes to the use case for the caller, the path's membership and the
// role given, one outside the three too; the answer is 200 with the
// membership the use case answers.
func TestUpdateProjectMember(t *testing.T) {
	for _, role := range []int{5, 10} {
		update := &fakeUpdateMember{answer: bobInWeb}
		h := newServer(t, fakes{updateMember: update})
		path := "/api/v0/project-members/" + bobInWeb.ID.String()
		want := `{"created_at":"2026-10-01T10:00:00.123456Z","id":"0199a2b4-0000-7000-8000-0000000000b2",` +
			`"member_id":"0199a2b4-0000-7000-8000-000000000002","project_id":"0199a2b4-0000-7000-8000-0000000000a1","role":5}`
		if res, body := do(t, h, request(http.MethodPatch, path, "alice", fmt.Sprintf(`{"role":%d}`, role))); res.StatusCode != http.StatusOK ||
			body != want+"\n" {
			t.Errorf("PATCH role %d = %d %s, want 200 %s", role, res.StatusCode, body, want)
		}
		if want := []string{fmt.Sprintf("alice %s %d", bobInWeb.ID, role)}; !slices.Equal(update.calls, want) {
			t.Errorf("calls = %q, want %q", update.calls, want)
		}
	}
}

// A body without its role, with a field it may not have, or a role of
// another type: refused as bad_request before the use case.
func TestUpdateProjectMemberHoldsTheBodyToItsStructure(t *testing.T) {
	update := &fakeUpdateMember{}
	h := newServer(t, fakes{updateMember: update})
	for _, body := range []string{`{}`, `{"role":null}`, `{"role":"5"}`, `{"role":5,"member_id":"0199a2b4-0000-7000-8000-000000000002"}`} {
		res, got := do(t, h, request(http.MethodPatch, "/api/v0/project-members/"+bobInWeb.ID.String(), "alice", body))
		var problem struct{ Code string }
		if err := json.Unmarshal([]byte(got), &problem); err != nil || res.StatusCode != http.StatusBadRequest || problem.Code != "bad_request" {
			t.Errorf("PATCH %s = %d %s, want 400 bad_request", body, res.StatusCode, got)
		}
	}
	if len(update.calls) != 0 {
		t.Errorf("the use case got %q, want nothing", update.calls)
	}
}

// The use case's refusals, as the contract declares them, and its failure.
func TestUpdateProjectMemberRefusals(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"a role outside the three", shared.Invalid(shared.FieldError{Field: "role", Code: shared.FieldInvalidFormat,
			Message: "is not 5, 15 or 20"}), http.StatusUnprocessableEntity,
			`{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"role","code":"invalid_format","message":"is not 5, 15 or 20"}]}`},
		{"no membership", domain.ErrMemberNotFound, http.StatusNotFound, memberNotFoundJSON},
		{"a project member", shared.Forbidden(), http.StatusForbidden, forbiddenJSON},
		{"his own", domain.ErrOwnMembership, http.StatusConflict, ownMembershipJSON},
		{"an admin's role", domain.ErrRoleTooHigh, http.StatusForbidden, roleTooHighJSON},
		{"a workspace guest made a member", shared.Invalid(shared.FieldError{Field: "role", Code: shared.FieldNotAllowed,
			Message: "must be 5: the member is a guest of the workspace"}), http.StatusUnprocessableEntity,
			`{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"role","code":"not_allowed","message":"must be 5: the member is a guest of the workspace"}]}`},
		{"a failure", errGone, http.StatusInternalServerError, internalErrorJSON},
	}
	for _, tt := range tests {
		h := newServer(t, fakes{updateMember: &fakeUpdateMember{err: tt.err}})
		res, body := do(t, h, request(http.MethodPatch, "/api/v0/project-members/"+bobInWeb.ID.String(), "alice", `{"role":5}`))
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: PATCH = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

// DELETE goes to the use case for the caller and the path's membership,
// and answers 204 with no body; the use case's refusals, as the contract
// declares them, and its failure.
func TestRemoveProjectMember(t *testing.T) {
	remove := &fakeDelete{}
	h := newServer(t, fakes{removeMember: remove})
	path := "/api/v0/project-members/" + bobInWeb.ID.String()
	if res, body := do(t, h, request(http.MethodDelete, path, "alice", "")); res.StatusCode != http.StatusNoContent || body != "" {
		t.Errorf("DELETE = %d %q, want 204 and no body", res.StatusCode, body)
	}
	if want := []string{"alice " + bobInWeb.ID.String()}; !slices.Equal(remove.calls, want) {
		t.Errorf("calls = %q, want %q", remove.calls, want)
	}
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"no membership", domain.ErrMemberNotFound, http.StatusNotFound, memberNotFoundJSON},
		{"a project member", shared.Forbidden(), http.StatusForbidden, forbiddenJSON},
		{"his own", domain.ErrOwnMembership, http.StatusConflict, ownMembershipJSON},
		{"a higher role", domain.ErrRoleTooHigh, http.StatusForbidden, roleTooHighJSON},
		{"a failure", errGone, http.StatusInternalServerError, internalErrorJSON},
	} {
		h := newServer(t, fakes{removeMember: &fakeDelete{err: tt.err}})
		if res, body := do(t, h, request(http.MethodDelete, path, "alice", "")); res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: DELETE = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

// POST /projects/{project_id}/leave is the use case's 204, with no body, for
// the caller and the project of the path; each refusal the contract
// declares for it is its problem, a failure a 500.
func TestLeaveProject(t *testing.T) {
	leave := &fakeDelete{}
	h := newServer(t, fakes{leave: leave})
	path := "/api/v0/projects/" + webID.String() + "/leave"
	if res, body := do(t, h, request(http.MethodPost, path, "alice", "")); res.StatusCode != http.StatusNoContent || body != "" {
		t.Errorf("POST = %d %q, want 204 and no body", res.StatusCode, body)
	}
	if want := []string{"alice " + webID.String()}; !slices.Equal(leave.calls, want) {
		t.Errorf("calls = %q, want %q", leave.calls, want)
	}
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"no project", domain.ErrNotFound, http.StatusNotFound,
			`{"status":404,"code":"project.not_found","title":"Not Found","detail":"The project does not exist, or you cannot see it."}`},
		{"not its member", shared.Forbidden(), http.StatusForbidden, forbiddenJSON},
		{"its only admin", domain.ErrSoleAdmin, http.StatusConflict, `{"status":409,"code":"project.sole_admin","title":"Conflict",` +
			`"detail":"The project would be left without an admin: its only active admin cannot leave it, nor can his membership end while ` +
			`it has other active members. It must first be given another admin, or be deleted."}`},
		{"a failure", errGone, http.StatusInternalServerError, internalErrorJSON},
	} {
		h := newServer(t, fakes{leave: &fakeDelete{err: tt.err}})
		if res, body := do(t, h, request(http.MethodPost, path, "alice", "")); res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: POST = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}
