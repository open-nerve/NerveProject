package httpadapter_test

import (
	"net/http"
	"slices"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// POST .../archive goes to the archiving use case and .../unarchive to the
// other, each for the caller and the path's project, and nothing else; the
// answer is 200 with the project the use case answers. Each has its
// refusals, as the contract declares them.
func TestArchiveAndUnarchiveProject(t *testing.T) {
	for _, op := range []string{"archive", "unarchive"} {
		archive, unarchive := &fakeOnProject{answer: web}, &fakeOnProject{answer: web}
		h := newServer(t, fakes{archive: archive, unarchive: unarchive})
		path := "/api/v0/projects/" + webID.String() + "/" + op
		if res, body := do(t, h, request(http.MethodPost, path, "bob", "")); res.StatusCode != http.StatusOK || body != webJSON+"\n" {
			t.Errorf("POST %s = %d %s, want 200 %s", path, res.StatusCode, body, webJSON)
		}
		called, idle := archive, unarchive
		if op == "unarchive" {
			called, idle = unarchive, archive
		}
		if want := []string{"bob " + webID.String()}; !slices.Equal(called.calls, want) || len(idle.calls) != 0 {
			t.Errorf("%s: calls %q and %q; want %q, and none to the other", op, called.calls, idle.calls, want)
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
			refusing := &fakeOnProject{err: tt.err}
			h := newServer(t, fakes{archive: refusing, unarchive: refusing})
			if res, body := do(t, h, request(http.MethodPost, path, "alice", "")); res.StatusCode != tt.status || body != tt.want+"\n" {
				t.Errorf("%s: POST = %d %s, want %d %s", op, res.StatusCode, body, tt.status, tt.want)
			}
		}
	}
}
