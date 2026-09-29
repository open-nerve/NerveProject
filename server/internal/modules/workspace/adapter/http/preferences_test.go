package httpadapter_test

import (
	"net/http"
	"slices"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// GET answers the use case's settings for the caller and the slug of the
// path.
func TestGetWorkspacePreferences(t *testing.T) {
	prefs := &fakePrefs{answer: domain.Preferences{NavigationControl: "TABBED", NavigationProjectLimit: 0}}
	h := newServer(t, fakes{prefs: prefs})
	for _, c := range []struct{ token, slug string }{{"alice", "acme"}, {"bob", "beta"}} {
		res, body := do(t, h, request(http.MethodGet, "/api/v0/me/workspaces/"+c.slug+"/preferences", c.token, ""))
		if want := `{"navigation_control_preference":"TABBED","navigation_project_limit":0}` + "\n"; res.StatusCode != http.StatusOK || body != want {
			t.Errorf("%s: GET = %d %s, want 200 %s", c.token, res.StatusCode, body, want)
		}
	}
	if want := []string{"GET alice acme", "GET bob beta"}; !slices.Equal(prefs.calls, want) {
		t.Errorf("calls = %q, want %q", prefs.calls, want)
	}
}

// The body becomes the patch, a field left out nil; the answer is 200 with
// the settings as stored.
func TestUpdateWorkspacePreferences(t *testing.T) {
	prefs := &fakePrefs{answer: domain.Preferences{NavigationControl: "ACCORDION", NavigationProjectLimit: 3}}
	h := newServer(t, fakes{prefs: prefs})
	for _, body := range []string{`{"navigation_control_preference":"TABBED","navigation_project_limit":3}`, `{"navigation_project_limit":0}`, `{}`} {
		res, got := do(t, h, request(http.MethodPatch, "/api/v0/me/workspaces/beta/preferences", "bob", body))
		if want := `{"navigation_control_preference":"ACCORDION","navigation_project_limit":3}` + "\n"; res.StatusCode != http.StatusOK || got != want {
			t.Errorf("PATCH %s = %d %s, want 200 %s", body, res.StatusCode, got, want)
		}
	}
	tabbed, three, zero := "TABBED", 3, 0
	want := []domain.PreferencesPatch{{NavigationControl: &tabbed, NavigationProjectLimit: &three}, {NavigationProjectLimit: &zero}, {}}
	if len(prefs.patches) != len(want) {
		t.Fatalf("patches = %+v, want %+v", prefs.patches, want)
	}
	for i := range want {
		got := prefs.patches[i]
		sameMode := (got.NavigationControl == nil) == (want[i].NavigationControl == nil) &&
			(got.NavigationControl == nil || *got.NavigationControl == *want[i].NavigationControl)
		sameLimit := (got.NavigationProjectLimit == nil) == (want[i].NavigationProjectLimit == nil) &&
			(got.NavigationProjectLimit == nil || *got.NavigationProjectLimit == *want[i].NavigationProjectLimit)
		if !sameMode || !sameLimit {
			t.Errorf("patch %d = %+v, want %+v", i, got, want[i])
		}
	}
	if calls := []string{"PATCH bob beta", "PATCH bob beta", "PATCH bob beta"}; !slices.Equal(prefs.calls, calls) {
		t.Errorf("calls = %q, want %q", prefs.calls, calls)
	}
}

// The use cases' refusals, as the contract declares them; a limit that is
// not a whole number is refused before them.
func TestWorkspacePreferencesRefusals(t *testing.T) {
	notFound := `{"status":404,"code":"workspace.not_found","title":"Not Found","detail":"The workspace does not exist, or you are not a member of it."}`
	tests := []struct {
		name, method, body string
		err                error
		status             int
		want               string
	}{
		{"GET, not visible", http.MethodGet, "", domain.ErrNotFound, http.StatusNotFound, notFound},
		{"PATCH, not visible", http.MethodPatch, `{"navigation_project_limit":3}`, domain.ErrNotFound, http.StatusNotFound, notFound},
		{"PATCH, invalid values", http.MethodPatch, `{"navigation_project_limit":-1}`,
			shared.Invalid(shared.FieldError{Field: "navigation_project_limit", Code: shared.FieldOutOfRange, Message: "must be between 0 and 2147483647"}),
			http.StatusUnprocessableEntity, `{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"navigation_project_limit","code":"out_of_range","message":"must be between 0 and 2147483647"}]}`},
	}
	for _, tt := range tests {
		h := newServer(t, fakes{prefs: &fakePrefs{err: tt.err}})
		res, body := do(t, h, request(tt.method, "/api/v0/me/workspaces/acme/preferences", "alice", tt.body))
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
	prefs := &fakePrefs{}
	h := newServer(t, fakes{prefs: prefs})
	if res, _ := do(t, h, request(http.MethodPatch, "/api/v0/me/workspaces/acme/preferences", "alice", `{"navigation_project_limit":2.5}`)); res.StatusCode != http.StatusBadRequest || len(prefs.calls) != 0 {
		t.Errorf("PATCH with a fraction = %d, calls %q; want 400 and no call", res.StatusCode, prefs.calls)
	}
}
