package httpadapter_test

import (
	"context"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeAdd is addProjectMembers: each call is recorded as "caller id" with
// what it got; it answers list, or err.
type fakeAdd struct {
	calls []string
	got   [][]domain.NewMember
	list  []domain.Member
	err   error
}

func (f *fakeAdd) Execute(ctx context.Context, projectID uuid.UUID, in []domain.NewMember) ([]domain.Member, error) {
	f.calls = append(f.calls, caller(ctx)+" "+projectID.String())
	f.got = append(f.got, in)
	return f.list, f.err
}

// The members go to the use case for the caller and the path's project, in
// their order, a role outside the three too; the answer is 201 with the
// list the use case answers.
func TestAddProjectMembersPassesTheMembers(t *testing.T) {
	add := &fakeAdd{list: []domain.Member{aliceInWeb, bobInWeb}}
	h := newServer(t, fakes{add: add})
	body := `{"members":[{"member_id":"0199a2b4-0000-7000-8000-000000000001","role":20},{"member_id":"0199a2b4-0000-7000-8000-000000000002","role":10}]}`
	res, got := do(t, h, request(http.MethodPost, "/api/v0/projects/"+webID.String()+"/members", "bob", body))
	if res.StatusCode != http.StatusCreated || got != membersJSON+"\n" {
		t.Errorf("POST = %d %s, want 201 %s", res.StatusCode, got, membersJSON)
	}
	want := [][]domain.NewMember{{{MemberID: aliceID, Role: shared.RoleAdmin}, {MemberID: bobID, Role: 10}}}
	if !reflect.DeepEqual(add.got, want) || !slices.Equal(add.calls, []string{"bob " + webID.String()}) {
		t.Errorf("the use case got %q %+v, want bob's %+v", add.calls, add.got, want)
	}
}

// A body without members, with null ones, a member without his role or
// account, a field a member may not have, a value of another type: refused
// as bad_request before the use case.
func TestAddProjectMembersHoldsTheBodyToItsStructure(t *testing.T) {
	add := &fakeAdd{}
	h := newServer(t, fakes{add: add})
	for _, body := range []string{`{}`, `{"members":null}`, `{"members":[{"member_id":"0199a2b4-0000-7000-8000-000000000001"}]}`,
		`{"members":[{"role":15}]}`, `{"members":[{"member_id":"0199a2b4-0000-7000-8000-000000000001","role":15,"email":"a@b.c"}]}`,
		`{"members":[{"member_id":"alice","role":15}]}`, `{"members":[{"member_id":"0199a2b4-0000-7000-8000-000000000001","role":"15"}]}`,
		`{"members":{}}`, `{"members":[],"role":15}`} {
		if res, got := do(t, h, request(http.MethodPost, "/api/v0/projects/"+webID.String()+"/members", "bob", body)); res.StatusCode != http.StatusBadRequest {
			t.Errorf("POST %s = %d %s, want 400", body, res.StatusCode, got)
		}
	}
	if len(add.calls) != 0 {
		t.Errorf("the use case got %q, want nothing", add.calls)
	}
}

// The use case's refusals, as the contract declares them.
func TestAddProjectMembersRefusals(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"no project", domain.ErrNotFound, http.StatusNotFound,
			`{"status":404,"code":"project.not_found","title":"Not Found","detail":"The project does not exist, or you cannot see it."}`},
		{"a project member", shared.Forbidden(), http.StatusForbidden,
			`{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
		{"the targets", shared.Invalid(shared.FieldError{Field: "members[0].member_id", Code: "not_allowed",
			Message: "must be an active member of the workspace"}, shared.FieldError{Field: "members[1].role", Code: "not_allowed",
			Message: "is not one his workspace role allows: a workspace admin is added as an admin, a guest as a guest"}),
			http.StatusUnprocessableEntity,
			`{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"members[0].member_id","code":"not_allowed","message":"must be an active member of the workspace"},` +
				`{"field":"members[1].role","code":"not_allowed","message":"is not one his workspace role allows: a workspace admin is added ` +
				`as an admin, a guest as a guest"}]}`},
	}
	for _, tt := range tests {
		h := newServer(t, fakes{add: &fakeAdd{err: tt.err}})
		res, body := do(t, h, request(http.MethodPost, "/api/v0/projects/"+webID.String()+"/members", "alice",
			`{"members":[{"member_id":"0199a2b4-0000-7000-8000-000000000002","role":15}]}`))
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: POST = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}
