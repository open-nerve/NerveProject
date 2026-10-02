package httpadapter_test

import (
	"net/http"
	"slices"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// DELETE goes to the use case for the caller and the path's project, and
// answers 204 with no body; the use case's refusals, as the contract
// declares them.
func TestDeleteProject(t *testing.T) {
	del := &fakeDelete{}
	h := newServer(t, fakes{delete: del})
	path := "/api/v0/projects/" + webID.String()
	if res, body := do(t, h, request(http.MethodDelete, path, "bob", "")); res.StatusCode != http.StatusNoContent || body != "" {
		t.Errorf("DELETE = %d %q, want 204 and no body", res.StatusCode, body)
	}
	if want := []string{"bob " + webID.String()}; !slices.Equal(del.calls, want) {
		t.Errorf("calls = %q, want %q", del.calls, want)
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
		h := newServer(t, fakes{delete: &fakeDelete{err: tt.err}})
		if res, body := do(t, h, request(http.MethodDelete, path, "alice", "")); res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("DELETE = %d %s, want %d %s", res.StatusCode, body, tt.status, tt.want)
		}
	}
}
