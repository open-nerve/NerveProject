package httpadapter_test

import (
	"context"
	"net/http"
	"slices"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

// fakeListWorkspaceLabels is listWorkspaceLabels: each call is recorded as
// "caller slug"; it answers list, or err.
type fakeListWorkspaceLabels struct {
	calls []string
	list  []domain.Label
	err   error
}

func (f *fakeListWorkspaceLabels) Execute(ctx context.Context, slug string) ([]domain.Label, error) {
	f.calls = append(f.calls, caller(ctx)+" "+slug)
	return f.list, f.err
}

// GET goes to the use case for the caller and the path's workspace; the
// answer is 200 with its list, in its order, every field of each label, UI's
// parent's id and Bug's null, and [] for none. Its refusal and its failure,
// as the contract declares them.
func TestListWorkspaceLabels(t *testing.T) {
	for _, tt := range []struct {
		list []domain.Label
		want string
	}{{[]domain.Label{webUILabel, webBugLabel}, `{"data":[` + webUILabelJSON + `,` + webBugLabelJSON + `]}`}, {nil, `{"data":[]}`}} {
		list := &fakeListWorkspaceLabels{list: tt.list}
		h := newServer(t, fakes{listWorkspaceLabels: list})
		if res, body := do(t, h, request(http.MethodGet, "/api/v0/workspaces/acme/labels", "bob", "")); res.StatusCode != http.StatusOK ||
			body != tt.want+"\n" {
			t.Errorf("GET = %d %s, want 200 %s", res.StatusCode, body, tt.want)
		}
		if want := []string{"bob acme"}; !slices.Equal(list.calls, want) {
			t.Errorf("calls = %q, want %q", list.calls, want)
		}
	}
	for _, tt := range []struct {
		err    error
		status int
		want   string
	}{
		{domain.ErrWorkspaceNotFound, http.StatusNotFound, workspaceNotFoundJSON},
		{errGone, http.StatusInternalServerError, internalErrorJSON},
	} {
		h := newServer(t, fakes{listWorkspaceLabels: &fakeListWorkspaceLabels{err: tt.err}})
		if res, body := do(t, h, request(http.MethodGet, "/api/v0/workspaces/acme/labels", "bob", "")); res.StatusCode != tt.status ||
			body != tt.want+"\n" {
			t.Errorf("GET answering %v = %d %s, want %d %s", tt.err, res.StatusCode, body, tt.status, tt.want)
		}
	}
}
