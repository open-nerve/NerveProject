package httpadapter_test

import (
	"net/http"
	"slices"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// POST .../join goes to the use case for the caller and the path's project,
// and to no other; the answer is 200 with the project the use case
// answers. Its refusals, as the contract declares them.
func TestJoinProject(t *testing.T) {
	join, archive := &fakeOnProject{answer: web}, &fakeOnProject{answer: web}
	h := newServer(t, fakes{join: join, archive: archive})
	path := "/api/v0/projects/" + webID.String() + "/join"
	if res, body := do(t, h, request(http.MethodPost, path, "bob", "")); res.StatusCode != http.StatusOK || body != webJSON+"\n" {
		t.Errorf("POST %s = %d %s, want 200 %s", path, res.StatusCode, body, webJSON)
	}
	if want := []string{"bob " + webID.String()}; !slices.Equal(join.calls, want) || len(archive.calls) != 0 {
		t.Errorf("calls %q and %q; want %q, and none to archiving", join.calls, archive.calls, want)
	}
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"not seen", domain.ErrNotFound, http.StatusNotFound,
			`{"status":404,"code":"project.not_found","title":"Not Found","detail":"The project does not exist, or you cannot see it."}`},
		{"a workspace guest", shared.Forbidden(), http.StatusForbidden,
			`{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
	} {
		h := newServer(t, fakes{join: &fakeOnProject{err: tt.err}})
		if res, body := do(t, h, request(http.MethodPost, path, "alice", "")); res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: POST = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}
