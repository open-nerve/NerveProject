package httpadapter_test

import (
	"context"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// pagePreferences are the four use cases of the display settings on a
// cycle's and a module's page: each call is recorded as "<get|update>
// <page> caller id", a change with what it got too; each answers its page's
// settings, or err.
type pagePreferences struct {
	calls   []string
	cycles  []domain.CyclePreferencesPatch
	modules []domain.ModulePreferencesPatch
	cycle   domain.CyclePreferences
	module  domain.ModulePreferences
	err     error
}

func (f *pagePreferences) record(ctx context.Context, call string, id uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	f.calls = append(f.calls, call+" "+actor.UserID.String()+" "+id.String())
	return nil
}

type (
	getCyclePreferences     struct{ *pagePreferences }
	updateCyclePreferences  struct{ *pagePreferences }
	getModulePreferences    struct{ *pagePreferences }
	updateModulePreferences struct{ *pagePreferences }
)

func (f getCyclePreferences) Execute(ctx context.Context, id uuid.UUID) (domain.CyclePreferences, error) {
	if err := f.record(ctx, "get cycle", id); err != nil {
		return domain.CyclePreferences{}, err
	}
	return f.cycle, f.err
}

func (f updateCyclePreferences) Execute(ctx context.Context, id uuid.UUID, p domain.CyclePreferencesPatch) (domain.CyclePreferences, error) {
	if err := f.record(ctx, "update cycle", id); err != nil {
		return domain.CyclePreferences{}, err
	}
	f.cycles = append(f.cycles, p)
	return f.cycle, f.err
}

func (f getModulePreferences) Execute(ctx context.Context, id uuid.UUID) (domain.ModulePreferences, error) {
	if err := f.record(ctx, "get module", id); err != nil {
		return domain.ModulePreferences{}, err
	}
	return f.module, f.err
}

func (f updateModulePreferences) Execute(ctx context.Context, id uuid.UUID, p domain.ModulePreferencesPatch) (domain.ModulePreferences, error) {
	if err := f.record(ctx, "update module", id); err != nil {
		return domain.ModulePreferences{}, err
	}
	f.modules = append(f.modules, p)
	return f.module, f.err
}

// server serves the module's API over f.
func (f *pagePreferences) server(t *testing.T) http.Handler {
	return newServer(t, httpadapter.UseCases{GetCyclePreferences: getCyclePreferences{f}, UpdateCyclePreferences: updateCyclePreferences{f},
		GetModulePreferences: getModulePreferences{f}, UpdateModulePreferences: updateModulePreferences{f}})
}

const (
	displayDefaultJSON    = `{"calendar":{"layout":"month","show_weekends":false},"group_by":null,"layout":"list","order_by":"-created_at","show_empty_groups":true,"sub_group_by":null,"sub_issue":true}`
	propertiesDefaultJSON = `{"assignee":true,"attachment_count":true,"created_on":true,"cycle":true,"due_date":true,"key":true,"labels":true,"link":true,"modules":true,"priority":true,"start_date":true,"state":true,"sub_issue_count":true,"updated_on":true}`
	pageDefaultsJSON      = `{"display_filters":` + displayDefaultJSON + `,"display_properties":` + propertiesDefaultJSON + `,"filters":{}}`
	// The settings of bobsCycle and bobsModule: an empty array kept, the
	// other page's key, a grouping and display properties set apart from
	// the defaults.
	displayCustomJSON    = `{"calendar":{"layout":"week","show_weekends":true},"group_by":"priority","layout":"calendar","order_by":"cycle_name","show_empty_groups":false,"sub_group_by":null,"sub_issue":false}`
	propertiesCustomJSON = `{"assignee":false,"attachment_count":true,"created_on":true,"cycle":false,"due_date":true,"key":true,"labels":true,"link":false,"modules":true,"priority":true,"start_date":true,"state":true,"sub_issue_count":true,"updated_on":true}`
	cycleCustomJSON      = `{"display_filters":` + displayCustomJSON + `,"display_properties":` + propertiesCustomJSON + `,"filters":{"label_id":[],"module_id":["none"],"priority":["low"]}}`
	moduleCustomJSON     = `{"display_filters":` + displayCustomJSON + `,"display_properties":` + propertiesCustomJSON + `,"filters":{"cycle_id":["none"],"label_id":[],"priority":["low"]}}`
)

// customDisplay are displayCustomJSON's and propertiesCustomJSON's settings.
func customDisplay() (shared.ListDisplayFilters, shared.ListDisplayProperties) {
	group := "priority"
	d := shared.ListDisplayFilters{Layout: "calendar", GroupBy: &group, OrderBy: "cycle_name", Calendar: shared.ListCalendar{Layout: "week", ShowWeekends: true}}
	p := shared.DefaultListDisplayProperties()
	p.Assignee, p.Cycle, p.Link = false, false, false
	return d, p
}

func bobsCycle() domain.CyclePreferences {
	d, p := customDisplay()
	return domain.CyclePreferences{Filters: domain.CycleFilters{ListFilters: shared.ListFilters{LabelID: []string{}, Priority: []string{"low"}}, ModuleID: []string{"none"}},
		DisplayFilters: d, DisplayProperties: p}
}

func bobsModule() domain.ModulePreferences {
	d, p := customDisplay()
	return domain.ModulePreferences{Filters: domain.ModuleFilters{ListFilters: shared.ListFilters{LabelID: []string{}, Priority: []string{"low"}}, CycleID: []string{"none"}},
		DisplayFilters: d, DisplayProperties: p}
}

// GET goes to the page's reading use case for the caller and the path's
// cycle or module; the answer is 200 with the settings it answers, an
// omitted filter key left out and an empty one kept.
func TestGetPagePreferences(t *testing.T) {
	for _, tt := range []struct {
		path, call string
		f          pagePreferences
		want       string
	}{
		{"/api/v0/me/cycles/", "get cycle", pagePreferences{cycle: domain.DefaultPreferences[domain.CycleFilters]()}, pageDefaultsJSON},
		{"/api/v0/me/cycles/", "get cycle", pagePreferences{cycle: bobsCycle()}, cycleCustomJSON},
		{"/api/v0/me/modules/", "get module", pagePreferences{module: domain.DefaultPreferences[domain.ModuleFilters]()}, pageDefaultsJSON},
		{"/api/v0/me/modules/", "get module", pagePreferences{module: bobsModule()}, moduleCustomJSON},
	} {
		f := tt.f
		res, body := do(t, f.server(t), request(http.MethodGet, tt.path+webID.String()+"/preferences", "alice", ""))
		if res.StatusCode != http.StatusOK || body != tt.want+"\n" {
			t.Errorf("GET %s = %d %s, want 200 %s", tt.path, res.StatusCode, body, tt.want)
		}
		if want := []string{tt.call + " " + aliceID.String() + " " + webID.String()}; !slices.Equal(f.calls, want) {
			t.Errorf("GET %s: calls %q, want %q", tt.path, f.calls, want)
		}
	}
}

// The fields the body names go to the page's changing use case, each whole,
// its words as given (the use case checks them); a field left out nil, a
// filter key left out nil and an empty one empty; the other page's key is
// the page's filter. The answer is 200 with the settings it answers.
func TestUpdatePagePreferencesPassesTheChange(t *testing.T) {
	display := `{"layout":"calendar","group_by":"priority","sub_group_by":null,"order_by":"cycle_name","sub_issue":false,"show_empty_groups":false,"calendar":{"layout":"week","show_weekends":true}}`
	properties := `{"assignee":false,"start_date":true,"due_date":true,"labels":true,"key":true,"priority":true,"state":true,"sub_issue_count":true,"link":false,"attachment_count":true,"created_on":true,"updated_on":true,"cycle":false,"modules":true}`
	shared10 := `"state_id":["bad-uuid"],"state_group":["future"],"priority":["future"],"label_id":[],"assignee_id":["me"],"created_by_id":["me"],"mention_id":["me"],"subscriber_id":["me"],"start_date":["bad-date"],"target_date":["none"]`
	words := shared.ListFilters{StateID: []string{"bad-uuid"}, StateGroup: []string{"future"}, Priority: []string{"future"}, LabelID: []string{},
		AssigneeID: []string{"me"}, CreatedByID: []string{"me"}, MentionID: []string{"me"}, SubscriberID: []string{"me"}, StartDate: []string{"bad-date"},
		TargetDate: []string{"none"}}
	d, p := customDisplay()
	bodies := func(other string) []string {
		return []string{`{"filters":{` + shared10 + `,"` + other + `":["none","bad-uuid"]},"display_filters":` + display + `,"display_properties":` + properties + `}`,
			`{"filters":{}}`, `{"filters":{"label_id":[]}}`, `{"display_properties":` + properties + `}`, `{}`}
	}

	f := pagePreferences{cycle: bobsCycle(), module: bobsModule()}
	h := f.server(t)
	for _, body := range bodies("module_id") {
		if res, got := do(t, h, request(http.MethodPatch, "/api/v0/me/cycles/"+webID.String()+"/preferences", "alice", body)); res.StatusCode != http.StatusOK || got != cycleCustomJSON+"\n" {
			t.Errorf("PATCH cycle %s = %d %s, want 200 %s", body, res.StatusCode, got, cycleCustomJSON)
		}
	}
	for _, body := range bodies("cycle_id") {
		if res, got := do(t, h, request(http.MethodPatch, "/api/v0/me/modules/"+webID.String()+"/preferences", "alice", body)); res.StatusCode != http.StatusOK || got != moduleCustomJSON+"\n" {
			t.Errorf("PATCH module %s = %d %s, want 200 %s", body, res.StatusCode, got, moduleCustomJSON)
		}
	}
	wantCycles := []domain.CyclePreferencesPatch{
		{Filters: &domain.CycleFilters{ListFilters: words, ModuleID: []string{"none", "bad-uuid"}}, DisplayFilters: &d, DisplayProperties: &p},
		{Filters: &domain.CycleFilters{}}, {Filters: &domain.CycleFilters{ListFilters: shared.ListFilters{LabelID: []string{}}}}, {DisplayProperties: &p}, {},
	}
	wantModules := []domain.ModulePreferencesPatch{
		{Filters: &domain.ModuleFilters{ListFilters: words, CycleID: []string{"none", "bad-uuid"}}, DisplayFilters: &d, DisplayProperties: &p},
		{Filters: &domain.ModuleFilters{}}, {Filters: &domain.ModuleFilters{ListFilters: shared.ListFilters{LabelID: []string{}}}}, {DisplayProperties: &p}, {},
	}
	if !reflect.DeepEqual(f.cycles, wantCycles) || !reflect.DeepEqual(f.modules, wantModules) {
		t.Errorf("the use cases got %+v and %+v, want %+v and %+v", f.cycles, f.modules, wantCycles, wantModules)
	}
	u, m := "update cycle "+aliceID.String()+" "+webID.String(), "update module "+aliceID.String()+" "+webID.String()
	if want := []string{u, u, u, u, u, m, m, m, m, m}; !slices.Equal(f.calls, want) {
		t.Errorf("calls %q, want %q", f.calls, want)
	}
}

// The page's own key (M6 P3b spec R8), a key no page has, the project's
// navigation and place, a null, a value of another type, and display
// settings missing a key or with one more: refused as bad_request before
// the use case.
func TestPagePreferencesHoldTheBodyToItsStructure(t *testing.T) {
	f := pagePreferences{}
	h := f.server(t)
	common := []string{`{"filters":{"project_id":[]}}`, `{"filters":{"cycle_ids":[]}}`, `{"filters":{"module_ids":[]}}`, `{"filters":{"archived":[]}}`,
		`{"navigation":{"default_tab":"views","hide_in_more_menu":[]}}`, `{"sort_order":1}`, `{"extra":true}`, `{"filters":null}`, `{"display_filters":null}`,
		`{"display_properties":null}`, `{"filters":[]}`, `{"filters":{"label_id":null}}`, `{"filters":{"label_id":"x"}}`, `{"filters":{"label_id":[1]}}`,
		`{"display_properties":{"assignee":true}}`, `{"display_properties":` + propertiesDefaultJSON[:len(propertiesDefaultJSON)-1] + `,"link_count":true}}`,
		`{"display_filters":` + displayDefaultJSON[:len(displayDefaultJSON)-1] + `,"extra":true}}`, `{"display_filters":{"layout":"list"}}`, `[]`}
	for page, own := range map[string]string{"cycles": "cycle_id", "modules": "module_id"} {
		for _, body := range append([]string{`{"filters":{"` + own + `":[]}}`, `{"filters":{"` + own + `":["none"]}}`}, common...) {
			res, got := do(t, h, request(http.MethodPatch, "/api/v0/me/"+page+"/"+webID.String()+"/preferences", "alice", body))
			if res.StatusCode != http.StatusBadRequest {
				t.Errorf("PATCH %s %s = %d %s, want 400", page, body, res.StatusCode, got)
			}
		}
	}
	if len(f.calls) != 0 {
		t.Errorf("the use cases got %q, want nothing", f.calls)
	}
}

// The use cases' refusals, as the contract declares them, and their
// failure.
func TestPagePreferencesRefusals(t *testing.T) {
	invalid := shared.Invalid(shared.FieldError{Field: "filters.state_id[0]", Code: "not_allowed", Message: "is not a state id or none"})
	invalidJSON := `{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
		`"errors":[{"field":"filters.state_id[0]","code":"not_allowed","message":"is not a state id or none"}]}`
	for _, page := range []struct {
		path, notFound string
		err            error
	}{
		{"/api/v0/me/cycles/", cycleNotFoundJSON, domain.ErrCycleNotFound},
		{"/api/v0/me/modules/", moduleNotFoundJSON, domain.ErrModuleNotFound},
	} {
		for _, tt := range []struct {
			name, method string
			err          error
			status       int
			want         string
		}{
			{"no page", http.MethodGet, page.err, http.StatusNotFound, page.notFound},
			{"no member", http.MethodGet, shared.Forbidden(), http.StatusForbidden, forbiddenJSON},
			{"a failure", http.MethodGet, errGone, http.StatusInternalServerError, internalErrorJSON},
			{"a word", http.MethodPatch, invalid, http.StatusUnprocessableEntity, invalidJSON},
			{"no page", http.MethodPatch, page.err, http.StatusNotFound, page.notFound},
			{"no member", http.MethodPatch, shared.Forbidden(), http.StatusForbidden, forbiddenJSON},
			{"a busy project", http.MethodPatch, shared.ServerBusy(time.Second), http.StatusServiceUnavailable, busy},
			{"a failure", http.MethodPatch, errGone, http.StatusInternalServerError, internalErrorJSON},
		} {
			f := pagePreferences{err: tt.err}
			body := ""
			if tt.method == http.MethodPatch {
				body = `{"filters":{"state_id":["x"]}}`
			}
			res, got := do(t, f.server(t), request(tt.method, page.path+uuid.NewV7().String()+"/preferences", "alice", body))
			if res.StatusCode != tt.status || got != tt.want+"\n" || tt.want == busy && res.Header.Get("Retry-After") != "1" {
				t.Errorf("%s %s%s: %d %s; want %d %s", tt.name, tt.method, page.path, res.StatusCode, got, tt.status, tt.want)
			}
		}
	}
}
