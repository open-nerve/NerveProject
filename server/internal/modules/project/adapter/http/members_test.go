package httpadapter_test

import (
	"context"
	"net/http"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeMembers is the members' use cases: each call is recorded as "caller
// id"; the list answers list, or err.
type fakeMembers struct {
	calls []string
	list  []domain.Member
	err   error
}

func (f *fakeMembers) Execute(ctx context.Context, projectID uuid.UUID) ([]domain.Member, error) {
	f.calls = append(f.calls, caller(ctx)+" "+projectID.String())
	return f.list, f.err
}

var (
	aliceInWeb = domain.Member{ID: uuid.MustParse("0199a2b4-0000-7000-8000-0000000000b1"), ProjectID: webID, MemberID: aliceID,
		Role: shared.RoleAdmin, CreatedAt: created}
	bobInWeb = domain.Member{ID: uuid.MustParse("0199a2b4-0000-7000-8000-0000000000b2"), ProjectID: webID, MemberID: bobID,
		Role: shared.RoleGuest, CreatedAt: created}
	membersJSON = `{"data":[{"created_at":"2026-10-01T10:00:00.123456Z","id":"0199a2b4-0000-7000-8000-0000000000b1",` +
		`"member_id":"0199a2b4-0000-7000-8000-000000000001","project_id":"0199a2b4-0000-7000-8000-0000000000a1","role":20},` +
		`{"created_at":"2026-10-01T10:00:00.123456Z","id":"0199a2b4-0000-7000-8000-0000000000b2",` +
		`"member_id":"0199a2b4-0000-7000-8000-000000000002","project_id":"0199a2b4-0000-7000-8000-0000000000a1","role":5}]}`
)

// GET goes to the use case for the caller and the path's project; the
// answer is 200 with its list, in its order, and [] for none.
func TestListProjectMembers(t *testing.T) {
	path := "/api/v0/projects/" + webID.String() + "/members"
	for _, tt := range []struct {
		list []domain.Member
		want string
	}{{[]domain.Member{aliceInWeb, bobInWeb}, membersJSON}, {nil, `{"data":[]}`}} {
		list := &fakeMembers{list: tt.list}
		h := newServer(t, fakes{members: list})
		if res, body := do(t, h, request(http.MethodGet, path, "bob", "")); res.StatusCode != http.StatusOK || body != tt.want+"\n" {
			t.Errorf("GET = %d %s, want 200 %s", res.StatusCode, body, tt.want)
		}
		if want := []string{"bob " + webID.String()}; !slices.Equal(list.calls, want) {
			t.Errorf("calls = %q, want %q", list.calls, want)
		}
	}
	for _, tt := range []struct {
		err    error
		status int
		want   string
	}{
		{domain.ErrNotFound, http.StatusNotFound,
			`{"status":404,"code":"project.not_found","title":"Not Found","detail":"The project does not exist, or you cannot see it."}`},
		{shared.Forbidden(), http.StatusForbidden, `{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
	} {
		h := newServer(t, fakes{members: &fakeMembers{err: tt.err}})
		if res, body := do(t, h, request(http.MethodGet, path, "alice", "")); res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("GET = %d %s, want %d %s", res.StatusCode, body, tt.status, tt.want)
		}
	}
}
