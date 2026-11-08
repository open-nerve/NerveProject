package httpadapter_test

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// modulesJSON is moduleIDs, the second first, as a body names them.
const modulesJSON = `["0199a2b4-0000-7000-8000-0000000000b3","0199a2b4-0000-7000-8000-0000000000b2"]`

// POST takes the modules the work item is created in, as given, in their
// order, empty when given empty and nil when left out (M6 P3a spec 2.3);
// null, or a value that is no id, is refused as bad_request before the use
// case.
func TestCreateIssueTakesTheModules(t *testing.T) {
	path := "/api/v0/projects/" + webID.String() + "/issues"
	for body, want := range map[string]domain.NewIssue{
		`{"name":"Fix","module_ids":` + modulesJSON + `}`: {Name: "Fix", ModuleIDs: []uuid.UUID{moduleIDs[1], moduleIDs[0]}},
		`{"name":"Fix","module_ids":[]}`:                  {Name: "Fix", ModuleIDs: []uuid.UUID{}},
		`{"name":"Fix"}`:                                  {Name: "Fix"},
	} {
		create := &fakeCreate{answer: webSeven}
		if res, got := do(t, newServer(t, httpadapter.UseCases{CreateIssue: create}), request(http.MethodPost, path, "alice", body)); res.StatusCode != http.StatusCreated ||
			!reflect.DeepEqual(create.got, []domain.NewIssue{want}) {
			t.Errorf("POST %s = %d %s with %+v; want 201 with %+v", body, res.StatusCode, got, create.got, want)
		}
	}
	create := &fakeCreate{}
	for _, body := range []string{`{"name":"Fix","module_ids":null}`, `{"name":"Fix","module_ids":["Auth"]}`} {
		res, got := do(t, newServer(t, httpadapter.UseCases{CreateIssue: create}), request(http.MethodPost, path, "alice", body))
		var problem struct{ Code string }
		if err := json.Unmarshal([]byte(got), &problem); err != nil || res.StatusCode != http.StatusBadRequest || problem.Code != "bad_request" {
			t.Errorf("POST %s = %d %s, want 400 bad_request", body, res.StatusCode, got)
		}
	}
	if len(create.calls) != 0 {
		t.Errorf("the use case got %q, want nothing", create.calls)
	}
}

// PATCH takes the work item's new modules as given, in their order, empty
// when given empty; left out, they are not given (M6 P3a spec 2.3). null,
// or a value that is no id, is refused as bad_request before the use case.
func TestUpdateIssueTakesTheModules(t *testing.T) {
	path := "/api/v0/issues/" + webSeven.ID.String()
	for body, want := range map[string]domain.IssuePatch{
		`{"module_ids":` + modulesJSON + `}`: {ModuleIDs: &[]uuid.UUID{moduleIDs[1], moduleIDs[0]}},
		`{"module_ids":[]}`:                  {ModuleIDs: &[]uuid.UUID{}},
		`{"name":"Fix"}`:                     {Name: new("Fix")},
	} {
		update := &fakeUpdate{answer: webSeven}
		if res, got := do(t, newServer(t, httpadapter.UseCases{UpdateIssue: update}), request(http.MethodPatch, path, "alice", body)); res.StatusCode != http.StatusOK ||
			!reflect.DeepEqual(update.got, []domain.IssuePatch{want}) {
			t.Errorf("PATCH %s = %d %s with %+v; want 200 with %+v", body, res.StatusCode, got, update.got, want)
		}
	}
	update := &fakeUpdate{}
	for _, body := range []string{`{"module_ids":null}`, `{"module_ids":["Auth"]}`} {
		res, got := do(t, newServer(t, httpadapter.UseCases{UpdateIssue: update}), request(http.MethodPatch, path, "alice", body))
		if res.StatusCode != http.StatusBadRequest {
			t.Errorf("PATCH %s = %d %s; want 400 bad_request", body, res.StatusCode, got)
		}
	}
	if len(update.calls) != 0 {
		t.Errorf("the use case got %q, want nothing", update.calls)
	}
}

// moduleRefusals are the use cases' refusals of a work item's modules (M6
// P3a spec 2.3), as createIssue and updateIssue declare them: planning's
// codes, the work item's details; an archived module is refused as an
// archived cycle is.
var moduleRefusals = []struct {
	name string
	err  error
	want string
}{
	{"modules turned off", domain.ErrModulesDisabled, `{"status":409,"code":"planning.modules_disabled","title":"Conflict",` +
		`"detail":"Modules are turned off in this project; turn them on to change a work item's modules."}`},
	{"an archived module", domain.ErrPlanningArchived, `{"status":409,"code":"planning.archived","title":"Conflict",` +
		`"detail":"The cycle or module is archived; a work item cannot join it."}`},
}

// Each refusal of the modules, answered to a creation.
func TestCreateIssueModuleRefusals(t *testing.T) {
	for _, tt := range moduleRefusals {
		t.Run(tt.name, func(t *testing.T) {
			h := newServer(t, httpadapter.UseCases{CreateIssue: &fakeCreate{err: tt.err}})
			body := `{"name":"Fix","module_ids":` + modulesJSON + `}`
			if res, got := do(t, h, request(http.MethodPost, "/api/v0/projects/"+webID.String()+"/issues", "alice", body)); res.StatusCode != http.StatusConflict ||
				got != tt.want+"\n" {
				t.Errorf("POST = %d %s, want 409 %s", res.StatusCode, got, tt.want)
			}
		})
	}
}

// Each refusal of the modules, answered to a change.
func TestUpdateIssueModuleRefusals(t *testing.T) {
	for _, tt := range moduleRefusals {
		t.Run(tt.name, func(t *testing.T) {
			h := newServer(t, httpadapter.UseCases{UpdateIssue: &fakeUpdate{err: tt.err}})
			body := `{"module_ids":` + modulesJSON + `}`
			if res, got := do(t, h, request(http.MethodPatch, "/api/v0/issues/"+webSeven.ID.String(), "alice", body)); res.StatusCode != http.StatusConflict ||
				got != tt.want+"\n" {
				t.Errorf("PATCH = %d %s, want 409 %s", res.StatusCode, got, tt.want)
			}
		})
	}
}
