package httpadapter_test

import (
	"net/http"
	"slices"
	"testing"
	"time"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// DELETE goes to the use case for the caller and the path's work item; the
// answer is 204, without a body.
func TestDeleteIssue(t *testing.T) {
	deletion := &fakeDelete{}
	res, body := do(t, newServer(t, httpadapter.UseCases{DeleteIssue: deletion}),
		request(http.MethodDelete, "/api/v0/issues/"+webSeven.ID.String(), "alice", ""))
	if res.StatusCode != http.StatusNoContent || body != "" {
		t.Errorf("DELETE = %d %q, want 204 without a body", res.StatusCode, body)
	}
	if want := []string{aliceID.String() + " " + webSeven.ID.String()}; !slices.Equal(deletion.calls, want) {
		t.Errorf("calls %q, want %q", deletion.calls, want)
	}
}

// The use case's refusals, as the contract declares them, and its failure.
func TestDeleteIssueRefusals(t *testing.T) {
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"no work item", domain.ErrIssueNotFound, http.StatusNotFound,
			`{"status":404,"code":"issue.not_found","title":"Not Found","detail":"The work item does not exist, or you cannot see it."}`},
		{"a project member, another's work item", shared.Forbidden(), http.StatusForbidden,
			`{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
		{"an archived project", domain.ErrProjectArchived, http.StatusConflict,
			`{"status":409,"code":"project.archived","title":"Conflict","detail":"The project is archived; unarchive it to change it."}`},
		{"a busy project", shared.ServerBusy(time.Second), http.StatusServiceUnavailable, busy},
		{"a failure", errGone, http.StatusInternalServerError,
			`{"status":500,"code":"internal_error","title":"Internal Server Error"}`},
	} {
		h := newServer(t, httpadapter.UseCases{DeleteIssue: &fakeDelete{err: tt.err}})
		res, body := do(t, h, request(http.MethodDelete, "/api/v0/issues/"+webSeven.ID.String(), "alice", ""))
		if res.StatusCode != tt.status || body != tt.want+"\n" || tt.want == busy && res.Header.Get("Retry-After") != "1" {
			t.Errorf("%s: DELETE = %d %s, Retry-After %q; want %d %s", tt.name, res.StatusCode, body, res.Header.Get("Retry-After"), tt.status,
				tt.want)
		}
	}
}
