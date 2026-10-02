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

// fakePreferences is both use cases of the display settings: each call is
// recorded as "get <caller> <project>" or "update <caller> <project>", a
// change with what it got too; both answer answer, or err.
type fakePreferences struct {
	calls  []string
	got    []domain.PreferencesPatch
	answer domain.Preferences
	err    error
}

func (f *fakePreferences) Execute(ctx context.Context, projectID uuid.UUID) (domain.Preferences, error) {
	f.calls = append(f.calls, "get "+caller(ctx)+" "+projectID.String())
	return f.answer, f.err
}

// fakeUpdatePreferences is fakePreferences' change.
type fakeUpdatePreferences struct{ *fakePreferences }

func (f fakeUpdatePreferences) Execute(ctx context.Context, projectID uuid.UUID, p domain.PreferencesPatch) (domain.Preferences, error) {
	f.calls = append(f.calls, "update "+caller(ctx)+" "+projectID.String())
	f.got = append(f.got, p)
	return f.answer, f.err
}

var (
	bobsTabs = domain.Preferences{Navigation: domain.Navigation{DefaultTab: "modules", HideInMoreMenu: []string{"views", "intake"}}, SortOrder: -2.5}
	tabsJSON = `{"navigation":{"default_tab":"modules","hide_in_more_menu":["views","intake"]},"sort_order":-2.5}`
)

// GET goes to the reading use case for the caller and the path's project;
// the answer is 200 with the settings it answers, an empty hidden list as
// [] too, a nil one as well.
func TestGetProjectPreferences(t *testing.T) {
	path := "/api/v0/me/projects/" + webID.String() + "/preferences"
	defaults := `{"navigation":{"default_tab":"work_items","hide_in_more_menu":[]},"sort_order":65535}`
	for _, tt := range []struct {
		answer domain.Preferences
		want   string
	}{
		{bobsTabs, tabsJSON},
		{domain.DefaultPreferences(), defaults},
		{domain.Preferences{Navigation: domain.Navigation{DefaultTab: "work_items"}, SortOrder: 65535}, defaults},
	} {
		prefs := &fakePreferences{answer: tt.answer}
		h := newServer(t, fakes{prefs: prefs})
		if res, body := do(t, h, request(http.MethodGet, path, "bob", "")); res.StatusCode != http.StatusOK || body != tt.want+"\n" {
			t.Errorf("GET = %d %s, want 200 %s", res.StatusCode, body, tt.want)
		}
		if want := []string{"get bob " + webID.String()}; !slices.Equal(prefs.calls, want) {
			t.Errorf("calls = %q, want %q", prefs.calls, want)
		}
	}
}

// The fields the body names go to the changing use case, the navigation
// whole, a tab the contract does not list too (the domain refuses it); a
// field left out nil. The answer is 200 with the settings it answers.
func TestUpdateProjectPreferencesPassesTheChange(t *testing.T) {
	prefs := &fakePreferences{answer: bobsTabs}
	h := newServer(t, fakes{prefs: prefs})
	path := "/api/v0/me/projects/" + webID.String() + "/preferences"
	for _, body := range []string{
		`{"navigation":{"default_tab":"cycles","hide_in_more_menu":["intake","views"]},"sort_order":-2.5}`,
		`{}`,
		`{"navigation":{"default_tab":"pages","hide_in_more_menu":[]}}`,
		`{"sort_order":0}`,
	} {
		if res, got := do(t, h, request(http.MethodPatch, path, "alice", body)); res.StatusCode != http.StatusOK || got != tabsJSON+"\n" {
			t.Errorf("PATCH %s = %d %s, want 200 %s", body, res.StatusCode, got, tabsJSON)
		}
	}
	want := []domain.PreferencesPatch{
		{Navigation: &domain.Navigation{DefaultTab: "cycles", HideInMoreMenu: []string{"intake", "views"}}, SortOrder: ptr(-2.5)},
		{},
		{Navigation: &domain.Navigation{DefaultTab: "pages", HideInMoreMenu: []string{}}},
		{SortOrder: ptr(0.0)},
	}
	if !reflect.DeepEqual(prefs.got, want) {
		t.Errorf("inputs = %+v, want %+v", prefs.got, want)
	}
	if u := "update alice " + webID.String(); !slices.Equal(prefs.calls, []string{u, u, u, u}) {
		t.Errorf("calls = %q, want four updates by alice of web", prefs.calls)
	}
}

// A field the body may not have, a navigation without one of its two
// fields, a null, a value of another type: refused as bad_request before
// the use case.
func TestUpdateProjectPreferencesHoldsTheBodyToItsStructure(t *testing.T) {
	prefs := &fakePreferences{answer: bobsTabs}
	h := newServer(t, fakes{prefs: prefs})
	for _, body := range []string{`{"pages":{}}`, `{"navigation":{"default_tab":"views"}}`, `{"navigation":{"hide_in_more_menu":[]}}`,
		`{"navigation":null}`, `{"sort_order":null}`, `{"sort_order":"1"}`, `{"navigation":{"default_tab":"views","hide_in_more_menu":"views"}}`,
		`{"navigation":{"default_tab":"views","hide_in_more_menu":[],"pages":[]}}`, `[]`} {
		if res, got := do(t, h, request(http.MethodPatch, "/api/v0/me/projects/"+webID.String()+"/preferences", "alice", body)); res.StatusCode != http.StatusBadRequest {
			t.Errorf("PATCH %s = %d %s, want 400", body, res.StatusCode, got)
		}
	}
	if len(prefs.calls) != 0 {
		t.Errorf("the use case got %q, want nothing", prefs.calls)
	}
}

// The use cases' refusals, as the contract declares them.
func TestProjectPreferencesRefusals(t *testing.T) {
	path := "/api/v0/me/projects/" + webID.String() + "/preferences"
	tests := []struct {
		name   string
		err    error
		method string
		status int
		want   string
	}{
		{"no project", domain.ErrNotFound, http.MethodGet, http.StatusNotFound,
			`{"status":404,"code":"project.not_found","title":"Not Found","detail":"The project does not exist, or you cannot see it."}`},
		{"no member", shared.Forbidden(), http.MethodGet, http.StatusForbidden,
			`{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
		{"no project", domain.ErrNotFound, http.MethodPatch, http.StatusNotFound,
			`{"status":404,"code":"project.not_found","title":"Not Found","detail":"The project does not exist, or you cannot see it."}`},
		{"no member", shared.Forbidden(), http.MethodPatch, http.StatusForbidden,
			`{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
		{"an unknown tab", shared.Invalid(shared.FieldError{Field: "navigation.default_tab", Code: "invalid_format",
			Message: "is not one of work_items, cycles, modules, views, intake"}), http.MethodPatch, http.StatusUnprocessableEntity,
			`{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"navigation.default_tab","code":"invalid_format","message":"is not one of work_items, cycles, modules, views, intake"}]}`},
	}
	for _, tt := range tests {
		h := newServer(t, fakes{prefs: &fakePreferences{err: tt.err}})
		body := ""
		if tt.method == http.MethodPatch {
			body = `{"sort_order":1}`
		}
		if res, got := do(t, h, request(tt.method, path, "alice", body)); res.StatusCode != tt.status || got != tt.want+"\n" {
			t.Errorf("%s %s = %d %s, want %d %s", tt.name, tt.method, res.StatusCode, got, tt.status, tt.want)
		}
	}
}
