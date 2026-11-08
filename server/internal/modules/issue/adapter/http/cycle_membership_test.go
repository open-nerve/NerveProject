package httpadapter_test

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// sprintJSON is sprintID as a body names it.
const sprintJSON = `"0199a2b4-0000-7000-8000-0000000000b1"`

// POST takes the cycle the work item is created in, nil when left out (M6
// P3a spec 2.3); null, or a value that is no id, is refused as bad_request
// before the use case: a creation names a cycle or none.
func TestCreateIssueTakesTheCycle(t *testing.T) {
	path := "/api/v0/projects/" + webID.String() + "/issues"
	for body, want := range map[string]domain.NewIssue{
		`{"name":"Fix","cycle_id":` + sprintJSON + `}`: {Name: "Fix", CycleID: &sprintID},
		`{"name":"Fix"}`: {Name: "Fix"},
	} {
		create := &fakeCreate{answer: webSeven}
		if res, got := do(t, newServer(t, httpadapter.UseCases{CreateIssue: create}), request(http.MethodPost, path, "alice", body)); res.StatusCode != http.StatusCreated ||
			!reflect.DeepEqual(create.got, []domain.NewIssue{want}) {
			t.Errorf("POST %s = %d %s with %+v; want 201 with %+v", body, res.StatusCode, got, create.got, want)
		}
	}
	create := &fakeCreate{}
	for _, body := range []string{`{"name":"Fix","cycle_id":null}`, `{"name":"Fix","cycle_id":"Sprint"}`} {
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

// PATCH takes the cycle given, null given as none, which takes the work
// item out of its cycle; left out, it is not given (M6 P3a spec 2.3). A
// value that is no id is refused as bad_request before the use case.
func TestUpdateIssueTakesTheCycle(t *testing.T) {
	path := "/api/v0/issues/" + webSeven.ID.String()
	for body, want := range map[string]domain.IssuePatch{
		`{"cycle_id":` + sprintJSON + `}`: {SetCycleID: true, CycleID: &sprintID},
		`{"cycle_id":null}`:               {SetCycleID: true},
		`{"name":"Fix"}`:                  {Name: new("Fix")},
	} {
		update := &fakeUpdate{answer: webSeven}
		if res, got := do(t, newServer(t, httpadapter.UseCases{UpdateIssue: update}), request(http.MethodPatch, path, "alice", body)); res.StatusCode != http.StatusOK ||
			!reflect.DeepEqual(update.got, []domain.IssuePatch{want}) {
			t.Errorf("PATCH %s = %d %s with %+v; want 200 with %+v", body, res.StatusCode, got, update.got, want)
		}
	}
	update := &fakeUpdate{}
	res, got := do(t, newServer(t, httpadapter.UseCases{UpdateIssue: update}), request(http.MethodPatch, path, "alice", `{"cycle_id":"Sprint"}`))
	if res.StatusCode != http.StatusBadRequest || len(update.calls) != 0 {
		t.Errorf("PATCH a cycle that is no id = %d %s, the use case got %q; want 400 bad_request, nothing", res.StatusCode, got, update.calls)
	}
}

// planningRefusals are the use cases' refusals of a work item's cycle (M6
// P3a spec 2.3), as createIssue and updateIssue declare them: planning's
// codes, the work item's details.
var planningRefusals = []struct {
	name string
	err  error
	want string
}{
	{"cycles turned off", domain.ErrCyclesDisabled, `{"status":409,"code":"planning.cycles_disabled","title":"Conflict",` +
		`"detail":"Cycles are turned off in this project; turn them on to change a work item's cycle."}`},
	{"an archived cycle", domain.ErrPlanningArchived, `{"status":409,"code":"planning.archived","title":"Conflict",` +
		`"detail":"The cycle or module is archived; a work item cannot join it."}`},
	{"an ended cycle", domain.ErrCycleCompleted, `{"status":409,"code":"planning.cycle_completed","title":"Conflict",` +
		`"detail":"The cycle is completed; a work item cannot join it."}`},
}

// Each refusal of the cycle, answered to a creation.
func TestCreateIssueCycleRefusals(t *testing.T) {
	for _, tt := range planningRefusals {
		t.Run(tt.name, func(t *testing.T) {
			h := newServer(t, httpadapter.UseCases{CreateIssue: &fakeCreate{err: tt.err}})
			body := `{"name":"Fix","cycle_id":` + sprintJSON + `}`
			if res, got := do(t, h, request(http.MethodPost, "/api/v0/projects/"+webID.String()+"/issues", "alice", body)); res.StatusCode != http.StatusConflict ||
				got != tt.want+"\n" {
				t.Errorf("POST = %d %s, want 409 %s", res.StatusCode, got, tt.want)
			}
		})
	}
}

// Each refusal of the cycle, answered to a change.
func TestUpdateIssueCycleRefusals(t *testing.T) {
	for _, tt := range planningRefusals {
		t.Run(tt.name, func(t *testing.T) {
			h := newServer(t, httpadapter.UseCases{UpdateIssue: &fakeUpdate{err: tt.err}})
			body := `{"cycle_id":` + sprintJSON + `}`
			if res, got := do(t, h, request(http.MethodPatch, "/api/v0/issues/"+webSeven.ID.String(), "alice", body)); res.StatusCode != http.StatusConflict ||
				got != tt.want+"\n" {
				t.Errorf("PATCH = %d %s, want 409 %s", res.StatusCode, got, tt.want)
			}
		})
	}
}
