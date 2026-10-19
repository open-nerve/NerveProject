package httpadapter_test

import (
	"net/http"
	"slices"
	"testing"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// GET goes to the use case for the caller, the path's workspace, and the
// identifier as the path gives it, in its case, a Turkish letter sent
// escaped as it came; the answer is 200 with the work item the use case
// answers, as getIssue shows it.
func TestGetIssueByIdentifier(t *testing.T) {
	for _, tt := range []struct{ path, identifier string }{
		{"/api/v0/workspaces/acme/issues/WEB-7", "WEB-7"},
		{"/api/v0/workspaces/acme/issues/web-7", "web-7"},
		{"/api/v0/workspaces/acme/issues/i%C5%9F-7", "iş-7"},
	} {
		get := &fakeGetByIdentifier{answer: webSeven}
		res, body := do(t, newServer(t, httpadapter.UseCases{GetIssueByIdentifier: get}), request(http.MethodGet, tt.path, "alice", ""))
		if res.StatusCode != http.StatusOK || body != webSevenJSON+"\n" {
			t.Errorf("GET %s = %d %s, want 200 %s", tt.path, res.StatusCode, body, webSevenJSON)
		}
		if want := []string{aliceID.String() + " acme " + tt.identifier}; !slices.Equal(get.calls, want) {
			t.Errorf("GET %s: calls %q, want %q", tt.path, get.calls, want)
		}
	}
}

// The use case's refusals, as the contract declares them, and its failure.
func TestGetIssueByIdentifierRefusals(t *testing.T) {
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"no workspace", domain.ErrWorkspaceNotFound, http.StatusNotFound,
			`{"status":404,"code":"workspace.not_found","title":"Not Found","detail":"The workspace does not exist, or you are not a member of it."}`},
		{"no work item", domain.ErrIssueNotFound, http.StatusNotFound,
			`{"status":404,"code":"issue.not_found","title":"Not Found","detail":"The work item does not exist, or you cannot see it."}`},
		{"a workspace admin who is no member of the project", shared.Forbidden(), http.StatusForbidden,
			`{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
		{"a failure", errGone, http.StatusInternalServerError,
			`{"status":500,"code":"internal_error","title":"Internal Server Error"}`},
	} {
		h := newServer(t, httpadapter.UseCases{GetIssueByIdentifier: &fakeGetByIdentifier{err: tt.err}})
		res, body := do(t, h, request(http.MethodGet, "/api/v0/workspaces/acme/issues/WEB-7", "alice", ""))
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: GET = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}
