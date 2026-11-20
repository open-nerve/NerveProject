package httpadapter_test

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fullView is bob's view of Web with every value it may have, each a value
// of its own: an icon of both kinds, filters with a key given empty, the
// project's cycle and module keys, a grouped board and some properties
// hidden, changed after its creation.
func fullView() domain.View {
	web := webID
	return domain.View{ID: boardID, WorkspaceID: acmeID, ProjectID: &web, OwnedByID: bobID, CreatedAt: created, UpdatedAt: changed,
		Content: domain.Content{Name: "Board", Description: "Open bugs, by state.",
			LogoProps: shared.LogoProps{InUse: new(shared.LogoInUseIcon), Emoji: &shared.LogoEmoji{Value: new("1f41e")},
				Icon: &shared.LogoIcon{Name: new("bug"), Color: new("#f00"), BackgroundColor: new("#fff")}},
			Filters: domain.Filters{ListFilters: shared.ListFilters{StateGroup: []string{"started", "unstarted"}, Priority: []string{}},
				CycleID: []string{"none"}, ModuleID: []string{"0199a2b4-0000-7000-8000-0000000000f1"}},
			DisplayFilters: shared.ListDisplayFilters{Layout: "kanban", GroupBy: new("state"), SubGroupBy: new("priority"), OrderBy: "sort_order",
				Calendar: shared.ListCalendar{Layout: "week", ShowWeekends: true}},
			DisplayProperties: shared.ListDisplayProperties{Assignee: true, DueDate: true, Key: true, Priority: true, State: true, Link: true,
				CreatedOn: true, Cycle: true}}}
}

// bareView is alice's view of acme with none of the values it may lack: of
// no project, without a description, an icon or a filter, shown as the
// defaults.
func bareView() domain.View {
	return domain.View{ID: sheetID, WorkspaceID: acmeID, OwnedByID: aliceID, CreatedAt: created, UpdatedAt: created,
		Content: domain.Content{Name: "Sheet", DisplayFilters: shared.DefaultListDisplayFilters(), DisplayProperties: shared.DefaultListDisplayProperties()}}
}

const (
	fullJSON = `{"id":"0199a2b4-0000-7000-8000-0000000000e1","workspace_id":"0199a2b4-0000-7000-8000-00000000000a",` +
		`"project_id":"0199a2b4-0000-7000-8000-0000000000a1","name":"Board","description":"Open bugs, by state.",` +
		`"logo_props":{"in_use":"icon","emoji":{"value":"1f41e"},"icon":{"name":"bug","color":"#f00","background_color":"#fff"}},` +
		`"filters":{"state_group":["started","unstarted"],"priority":[],"cycle_id":["none"],"module_id":["0199a2b4-0000-7000-8000-0000000000f1"]},` +
		`"display_filters":{"layout":"kanban","group_by":"state","sub_group_by":"priority","order_by":"sort_order","sub_issue":false,` +
		`"show_empty_groups":false,"calendar":{"layout":"week","show_weekends":true}},` +
		`"display_properties":{"assignee":true,"start_date":false,"due_date":true,"labels":false,"key":true,"priority":true,"state":true,` +
		`"sub_issue_count":false,"link":true,"attachment_count":false,"created_on":true,"updated_on":false,"cycle":true,"modules":false},` +
		`"owned_by_id":"0199a2b4-0000-7000-8000-000000000002","created_at":"2026-11-20T10:00:00.123456Z","updated_at":"2026-11-20T11:00:00.123456Z"}`
	bareJSON = `{"id":"0199a2b4-0000-7000-8000-0000000000e2","workspace_id":"0199a2b4-0000-7000-8000-00000000000a","project_id":null,` +
		`"name":"Sheet","description":"","logo_props":{},"filters":{},` +
		`"display_filters":{"layout":"list","group_by":null,"sub_group_by":null,"order_by":"-created_at","sub_issue":true,` +
		`"show_empty_groups":true,"calendar":{"layout":"month","show_weekends":false}},` +
		`"display_properties":{"assignee":true,"start_date":true,"due_date":true,"labels":true,"key":true,"priority":true,"state":true,` +
		`"sub_issue_count":true,"link":true,"attachment_count":true,"created_on":true,"updated_on":true,"cycle":true,"modules":true},` +
		`"owned_by_id":"0199a2b4-0000-7000-8000-000000000001","created_at":"2026-11-20T10:00:00.123456Z","updated_at":"2026-11-20T10:00:00.123456Z"}`
)

// The display settings of a body, and as the use case takes them: a grouped
// spreadsheet, none of its words checked here, and some properties hidden.
const (
	displayFiltersJSON = `{"layout":"spreadsheet","group_by":"labels","sub_group_by":null,"order_by":"priority","sub_issue":false,` +
		`"show_empty_groups":true,"calendar":{"layout":"day","show_weekends":false}}`
	displayPropertiesJSON = `{"assignee":false,"start_date":true,"due_date":false,"labels":true,"key":false,"priority":true,"state":false,` +
		`"sub_issue_count":true,"link":false,"attachment_count":true,"created_on":false,"updated_on":true,"cycle":false,"modules":true}`
)

var (
	bodyDisplayFilters = shared.ListDisplayFilters{Layout: "spreadsheet", GroupBy: new("labels"), OrderBy: "priority", ShowEmptyGroups: true,
		Calendar: shared.ListCalendar{Layout: "day"}}
	bodyDisplayProperties = shared.ListDisplayProperties{StartDate: true, Labels: true, Priority: true, SubIssueCount: true,
		AttachmentCount: true, UpdatedOn: true, Modules: true}
)

// GET of either collection goes to its use case for the caller and the
// path's project or workspace; the answer is 200 with its views in their
// order, each with every key, and [] for none.
func TestListingTheViews(t *testing.T) {
	for _, route := range []struct {
		path string
		want call
	}{
		{projectPath, call{Caller: aliceID, Op: "listProjectViews", ID: webID}},
		{workspacePath, call{Caller: aliceID, Op: "listWorkspaceViews", Slug: "acme"}},
	} {
		for _, tt := range []struct {
			list []domain.View
			want string
		}{
			{[]domain.View{fullView(), bareView()}, `{"data":[` + fullJSON + `,` + bareJSON + `]}`},
			{[]domain.View{}, `{"data":[]}`},
			{nil, `{"data":[]}`},
		} {
			f := &fakes{list: tt.list}
			res, body := do(t, newServer(t, f.useCases()), request(http.MethodGet, route.path, ""))
			if res.StatusCode != http.StatusOK || !sameJSON(t, body, tt.want) {
				t.Errorf("GET %s of %d = %d %s, want 200 %s", route.path, len(tt.list), res.StatusCode, body, tt.want)
			}
			if !reflect.DeepEqual(f.calls, []call{route.want}) {
				t.Errorf("GET %s: the use cases got %+v, want %+v alone", route.path, f.calls, route.want)
			}
		}
	}
}

// GET of a view goes to the use case for the caller and the path's view;
// the answer is 200 with the view, of either kind.
func TestGettingAView(t *testing.T) {
	for _, tt := range []struct {
		view domain.View
		want string
	}{{fullView(), fullJSON}, {bareView(), bareJSON}} {
		f := &fakes{view: tt.view}
		res, body := do(t, newServer(t, f.useCases()), request(http.MethodGet, viewPath, ""))
		if res.StatusCode != http.StatusOK || !sameJSON(t, body, tt.want) {
			t.Errorf("GET = %d %s, want 200 %s", res.StatusCode, body, tt.want)
		}
		if want := []call{{Caller: aliceID, Op: "getView", ID: boardID}}; !reflect.DeepEqual(f.calls, want) {
			t.Errorf("the use cases got %+v, want %+v alone", f.calls, want)
		}
	}
}

// POST of either collection goes to its use case for the caller and the
// path's project or workspace with the body's values as given, unchecked:
// a description, an icon and filters left out are none, display settings
// left out nil; a filter key given empty stays apart from one left out.
// The answer is 201 with the new view.
func TestCreatingAView(t *testing.T) {
	for _, tt := range []struct {
		body string
		want domain.NewView
	}{
		{`{"name":"Board"}`, domain.NewView{Name: "Board"}},
		{`{"name":"x","description":"","logo_props":{},"filters":{}}`, domain.NewView{Name: "x"}},
		{`{"name":"  Board ","description":"Open bugs","logo_props":{"in_use":"other","icon":{"name":"bug"}},` +
			`"filters":{"state_id":[],"cycle_id":["none"],"project_id":["p"]},"display_filters":` + displayFiltersJSON +
			`,"display_properties":` + displayPropertiesJSON + `}`,
			domain.NewView{Name: "  Board ", Description: "Open bugs", LogoProps: shared.LogoProps{InUse: new("other"), Icon: &shared.LogoIcon{Name: new("bug")}},
				Filters:        domain.Filters{ListFilters: shared.ListFilters{StateID: []string{}}, CycleID: []string{"none"}, ProjectID: []string{"p"}},
				DisplayFilters: &bodyDisplayFilters, DisplayProperties: &bodyDisplayProperties}},
	} {
		for _, route := range []struct {
			path string
			want call
		}{
			{projectPath, call{Caller: aliceID, Op: "createProjectView", ID: webID, New: tt.want}},
			{workspacePath, call{Caller: aliceID, Op: "createWorkspaceView", Slug: "acme", New: tt.want}},
		} {
			f := &fakes{view: fullView()}
			res, body := do(t, newServer(t, f.useCases()), request(http.MethodPost, route.path, tt.body))
			if res.StatusCode != http.StatusCreated || !sameJSON(t, body, fullJSON) {
				t.Errorf("POST %s %s = %d %s, want 201 %s", route.path, tt.body, res.StatusCode, body, fullJSON)
			}
			if !reflect.DeepEqual(f.calls, []call{route.want}) {
				t.Errorf("POST %s %s: the use cases got %+v, want %+v alone", route.path, tt.body, f.calls, route.want)
			}
		}
	}
}

// PATCH goes to the use case for the caller and the path's view with each
// field the body gives, whole and unchecked, and each it leaves out nil:
// {} gives nothing, a field given empty is given, and within the filters a
// key given empty stays apart from one left out. The answer is 200 with
// the view.
func TestUpdatingAView(t *testing.T) {
	for _, tt := range []struct {
		body string
		want domain.Patch
	}{
		{`{}`, domain.Patch{}},
		{`{"name":"Renamed"}`, domain.Patch{Name: new("Renamed")}},
		{`{"description":""}`, domain.Patch{Description: new("")}},
		{`{"logo_props":{}}`, domain.Patch{LogoProps: &shared.LogoProps{}}},
		{`{"logo_props":{"in_use":"emoji","emoji":{"url":"u"}}}`,
			domain.Patch{LogoProps: &shared.LogoProps{InUse: new(shared.LogoInUseEmoji), Emoji: &shared.LogoEmoji{URL: new("u")}}}},
		{`{"filters":{}}`, domain.Patch{Filters: &domain.Filters{}}},
		{`{"filters":{"priority":[],"module_id":["none"],"project_id":["p"]}}`,
			domain.Patch{Filters: &domain.Filters{ListFilters: shared.ListFilters{Priority: []string{}}, ModuleID: []string{"none"}, ProjectID: []string{"p"}}}},
		{`{"display_filters":` + displayFiltersJSON + `}`, domain.Patch{DisplayFilters: &bodyDisplayFilters}},
		{`{"display_properties":` + displayPropertiesJSON + `}`, domain.Patch{DisplayProperties: &bodyDisplayProperties}},
	} {
		f := &fakes{view: bareView()}
		res, body := do(t, newServer(t, f.useCases()), request(http.MethodPatch, viewPath, tt.body))
		if res.StatusCode != http.StatusOK || !sameJSON(t, body, bareJSON) {
			t.Errorf("PATCH %s = %d %s, want 200 %s", tt.body, res.StatusCode, body, bareJSON)
		}
		if want := []call{{Caller: aliceID, Op: "updateView", ID: boardID, Patch: tt.want}}; !reflect.DeepEqual(f.calls, want) {
			t.Errorf("PATCH %s: the use cases got %+v, want %+v alone", tt.body, f.calls, want)
		}
	}
}

// DELETE goes to the use case for the caller and the path's view; the
// answer is 204 without a body.
func TestDeletingAView(t *testing.T) {
	f := &fakes{}
	res, body := do(t, newServer(t, f.useCases()), request(http.MethodDelete, viewPath, ""))
	if res.StatusCode != http.StatusNoContent || body != "" {
		t.Errorf("DELETE = %d %q, want 204 and no body", res.StatusCode, body)
	}
	if want := []call{{Caller: aliceID, Op: "deleteView", ID: boardID}}; !reflect.DeepEqual(f.calls, want) {
		t.Errorf("the use cases got %+v, want %+v alone", f.calls, want)
	}
}

// An id in the path that is no UUID answers 400 bad_request before any use
// case.
func TestAnIDThatIsNoUUIDAnswers400(t *testing.T) {
	f := &fakes{}
	h := newServer(t, f.useCases())
	for _, req := range []*http.Request{request(http.MethodGet, "/api/v0/projects/web/views", ""), request(http.MethodGet, "/api/v0/views/board", ""),
		request(http.MethodPatch, "/api/v0/views/board", `{}`), request(http.MethodDelete, viewPath+"x", "")} {
		if res, body := do(t, h, req); res.StatusCode != http.StatusBadRequest || problemCode(t, body) != "bad_request" {
			t.Errorf("%s %s = %d %s, want 400 bad_request", req.Method, req.URL.Path, res.StatusCode, body)
		}
	}
	if len(f.calls) != 0 {
		t.Errorf("the use cases got %+v, want none", f.calls)
	}
}
