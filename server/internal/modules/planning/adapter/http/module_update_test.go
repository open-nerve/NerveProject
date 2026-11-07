package httpadapter_test

import (
	"context"
	"encoding/json"
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

// fakeUpdateModule is updateModule: each call is recorded as "caller module", with what it got; it answers answer,
// or err.
type fakeUpdateModule struct {
	calls  []string
	got    []domain.ModulePatch
	answer domain.Module
	err    error
}

func (f *fakeUpdateModule) Execute(ctx context.Context, id uuid.UUID, p domain.ModulePatch) (domain.Module, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Module{}, err
	}
	f.calls = append(f.calls, actor.UserID.String()+" "+id.String())
	f.got = append(f.got, p)
	return f.answer, f.err
}

// PATCH goes to the use case for the caller and the path's module with the fields given as given (the use case cleans
// and checks them, a status not of the six too): a date or the lead left out not set, one null set to none;
// member_ids, given, the whole set, empty too. The answer is 200 with the module the use case answers.
func TestUpdateModule(t *testing.T) {
	billing, _ := modules(t)
	order, paused, done := -2.5, domain.ModulePaused, domain.ModuleStatus("done")
	for _, tt := range []struct {
		body string
		want domain.ModulePatch
	}{
		{`{}`, domain.ModulePatch{}},
		{`{"name":" Two ","description":"","status":"paused","sort_order":-2.5}`,
			domain.ModulePatch{Name: text(" Two "), Description: text(""), Status: &paused, SortOrder: &order}},
		{`{"start_date":"2026-11-02","target_date":null,"lead_id":null}`,
			domain.ModulePatch{SetStartDate: true, StartDate: day(t, "2026-11-02"), SetTargetDate: true, SetLeadID: true}},
		{`{"target_date":"2026-11-20","lead_id":"` + bobID.String() + `","member_ids":["` + bobID.String() + `","` + aliceID.String() + `"]}`,
			domain.ModulePatch{SetTargetDate: true, TargetDate: day(t, "2026-11-20"), SetLeadID: true, LeadID: &bobID,
				MemberIDs: &[]uuid.UUID{bobID, aliceID}}},
		{`{"member_ids":[],"status":"done"}`, domain.ModulePatch{MemberIDs: &[]uuid.UUID{}, Status: &done}},
	} {
		update := &fakeUpdateModule{answer: billing}
		res, body := do(t, newServer(t, httpadapter.UseCases{UpdateModule: update}),
			request(http.MethodPatch, "/api/v0/modules/"+billing.ID.String(), "alice", tt.body))
		if res.StatusCode != http.StatusOK || body != billingJSON+"\n" {
			t.Errorf("PATCH %s = %d %s, want 200 %s", tt.body, res.StatusCode, body, billingJSON)
		}
		if want := []string{aliceID.String() + " " + billing.ID.String()}; !slices.Equal(update.calls, want) ||
			!reflect.DeepEqual(update.got, []domain.ModulePatch{tt.want}) {
			t.Errorf("PATCH %s: the use case got %q %+v, want %q %+v", tt.body, update.calls, update.got, want, tt.want)
		}
	}
}

// A body with a field it may not have (the archive, the project and the creation are the server's), null for a field
// that is not nullable (the name, the description, the status, the members, the sort order), a field of another
// type, an id that is no UUID, or a date that is no day: refused as bad_request before the use case.
func TestUpdateModuleHoldsTheBodyToItsStructure(t *testing.T) {
	update := &fakeUpdateModule{}
	h := newServer(t, httpadapter.UseCases{UpdateModule: update})
	for _, body := range []string{`{"archived_at":null}`, `{"project_id":"` + webID.String() + `"}`, `{"created_by_id":null}`, `{"id":null}`,
		`{"name":null}`, `{"description":null}`, `{"status":null}`, `{"member_ids":null}`, `{"sort_order":null}`, `{"sort_order":"1"}`,
		`{"name":1}`, `{"status":1}`, `{"lead_id":"bob"}`, `{"member_ids":["bob"]}`, `{"member_ids":"` + bobID.String() + `"}`,
		`{"start_date":"2026-02-30"}`, `{"target_date":"2026-11-13T00:00:00Z"}`} {
		res, got := do(t, h, request(http.MethodPatch, "/api/v0/modules/"+uuid.NewV7().String(), "alice", body))
		var problem struct{ Code string }
		if err := json.Unmarshal([]byte(got), &problem); err != nil || res.StatusCode != http.StatusBadRequest || problem.Code != "bad_request" {
			t.Errorf("PATCH %s = %d %s, want 400 bad_request", body, res.StatusCode, got)
		}
	}
	if len(update.calls) != 0 {
		t.Errorf("the use case got %q, want nothing", update.calls)
	}
}

// The use case's refusals, as the contract declares them, and its failure.
func TestUpdateModuleRefusals(t *testing.T) {
	refused, _, err := domain.ModulePatch{SetLeadID: true, LeadID: &bobID}.Apply(domain.Module{}, nil)
	if err == nil {
		t.Fatalf("a lead who may not be: %+v", refused)
	}
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"a lead who may not be", err, http.StatusUnprocessableEntity,
			`{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"lead_id","code":"not_allowed","message":"must be an active member of the project who is not its guest"}]}`},
		{"no module", domain.ErrModuleNotFound, http.StatusNotFound, moduleNotFoundJSON},
		{"a project guest", shared.Forbidden(), http.StatusForbidden, forbiddenJSON},
		{"an archived project", domain.ErrProjectArchived, http.StatusConflict,
			`{"status":409,"code":"project.archived","title":"Conflict","detail":"The project is archived; unarchive it to change it."}`},
		{"modules turned off", domain.ErrModulesDisabled, http.StatusConflict,
			`{"status":409,"code":"planning.modules_disabled","title":"Conflict","detail":"Modules are turned off in this project; turn them on to change them."}`},
		{"an archived module", domain.ErrArchived, http.StatusConflict,
			`{"status":409,"code":"planning.archived","title":"Conflict","detail":"The cycle or module is archived; unarchive it to change it."}`},
		{"a name taken", domain.ErrModuleNameTaken, http.StatusConflict,
			`{"status":409,"code":"planning.module_name_taken","title":"Conflict","detail":"A module of this project has this name."}`},
		{"a busy project", shared.ServerBusy(time.Second), http.StatusServiceUnavailable, busy},
		{"a failure", errGone, http.StatusInternalServerError, internalErrorJSON},
	} {
		h := newServer(t, httpadapter.UseCases{UpdateModule: &fakeUpdateModule{err: tt.err}})
		res, body := do(t, h, request(http.MethodPatch, "/api/v0/modules/"+uuid.NewV7().String(), "alice", `{"name":"A"}`))
		if res.StatusCode != tt.status || body != tt.want+"\n" || tt.want == busy && res.Header.Get("Retry-After") != "1" {
			t.Errorf("%s: PATCH = %d %s; want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}
