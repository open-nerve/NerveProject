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

// fakeCreateModule is createModule: each call is recorded as "caller
// project", with what it got; it answers answer, or err.
type fakeCreateModule struct {
	calls  []string
	got    []domain.NewModule
	answer domain.Module
	err    error
}

func (f *fakeCreateModule) Execute(ctx context.Context, projectID uuid.UUID, in domain.NewModule) (domain.Module, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Module{}, err
	}
	f.calls = append(f.calls, actor.UserID.String()+" "+projectID.String())
	f.got = append(f.got, in)
	return f.answer, f.err
}

var (
	bobID     = uuid.MustParse("0199a2b4-0000-7000-8000-000000000002")
	authID    = uuid.MustParse("0199a2b4-0000-7000-8000-0000000000d1")
	billingID = uuid.MustParse("0199a2b4-0000-7000-8000-0000000000d2")
)

// web's modules as the use cases answer them: Billing, with every value,
// archived (as only a read shows it); Auth, without dates, lead or members,
// whose accounts are gone.
func modules(t *testing.T) (billing, auth domain.Module) {
	archived := created.Add(time.Hour)
	billing = domain.Module{ID: billingID, ProjectID: webID, WorkspaceID: acmeID, Name: "Billing", Description: "Pay", StartDate: day(t, "2026-11-02"),
		TargetDate: day(t, "2026-11-13"), Status: domain.ModuleInProgress, LeadID: &bobID, MemberIDs: []uuid.UUID{aliceID, bobID},
		SortOrder: 55535.5, ArchivedAt: &archived, CreatedAt: created, UpdatedAt: created.Add(1), CreatedByID: &aliceID, UpdatedByID: &bobID}
	auth = domain.Module{ID: authID, ProjectID: webID, WorkspaceID: acmeID, Name: "Auth", Status: domain.ModulePlanned, SortOrder: 65535,
		CreatedAt: created, UpdatedAt: created}
	return billing, auth
}

const (
	billingJSON = `{"archived_at":"2026-11-07T11:00:00.123456Z","created_at":"2026-11-07T10:00:00.123456Z",` +
		`"created_by_id":"0199a2b4-0000-7000-8000-000000000001","description":"Pay","id":"0199a2b4-0000-7000-8000-0000000000d2",` +
		`"lead_id":"0199a2b4-0000-7000-8000-000000000002","member_ids":["0199a2b4-0000-7000-8000-000000000001",` +
		`"0199a2b4-0000-7000-8000-000000000002"],"name":"Billing","project_id":"0199a2b4-0000-7000-8000-0000000000a1","sort_order":55535.5,` +
		`"start_date":"2026-11-02","status":"in-progress","target_date":"2026-11-13","updated_at":"2026-11-07T10:00:00.123456001Z",` +
		`"updated_by_id":"0199a2b4-0000-7000-8000-000000000002","workspace_id":"0199a2b4-0000-7000-8000-00000000000a"}`
	authJSON = `{"archived_at":null,"created_at":"2026-11-07T10:00:00.123456Z","created_by_id":null,"description":"",` +
		`"id":"0199a2b4-0000-7000-8000-0000000000d1","lead_id":null,"member_ids":[],"name":"Auth",` +
		`"project_id":"0199a2b4-0000-7000-8000-0000000000a1","sort_order":65535,"start_date":null,"status":"planned","target_date":null,` +
		`"updated_at":"2026-11-07T10:00:00.123456Z","updated_by_id":null,"workspace_id":"0199a2b4-0000-7000-8000-00000000000a"}`
)

// POST goes to the use case for the caller and the path's project with the
// body's values as given (the use case cleans and checks them, a status not
// of the six too): a description left out as empty, dates, a status and a
// lead left out as none, a lead given as null as none, members left out as
// none. The answer is 201 with every field of the module the use case
// answers, null for each value it has not, its members an array.
func TestCreateModule(t *testing.T) {
	billing, auth := modules(t)
	path := "/api/v0/projects/" + webID.String() + "/modules"
	inProgress, done := domain.ModuleInProgress, domain.ModuleStatus("done")
	for _, tt := range []struct {
		body   string
		in     domain.NewModule
		answer domain.Module
		want   string
	}{
		{`{"name":" Billing ","description":"Pay","start_date":"2026-11-02","target_date":"2026-11-13","status":"in-progress",` +
			`"lead_id":"` + bobID.String() + `","member_ids":["` + bobID.String() + `","` + aliceID.String() + `"]}`,
			domain.NewModule{Name: " Billing ", Description: "Pay", StartDate: day(t, "2026-11-02"), TargetDate: day(t, "2026-11-13"),
				Status: &inProgress, LeadID: &bobID, MemberIDs: []uuid.UUID{bobID, aliceID}}, billing, billingJSON},
		{`{"name":"Auth"}`, domain.NewModule{Name: "Auth"}, auth, authJSON},
		{`{"name":"Auth","lead_id":null,"member_ids":[]}`, domain.NewModule{Name: "Auth", MemberIDs: []uuid.UUID{}}, auth, authJSON},
		{`{"name":"Half","target_date":"2026-11-13","status":"done"}`, domain.NewModule{Name: "Half", TargetDate: day(t, "2026-11-13"),
			Status: &done}, auth, authJSON},
	} {
		create := &fakeCreateModule{answer: tt.answer}
		res, body := do(t, newServer(t, httpadapter.UseCases{CreateModule: create}), request(http.MethodPost, path, "alice", tt.body))
		if res.StatusCode != http.StatusCreated || body != tt.want+"\n" {
			t.Errorf("POST %s = %d %s, want 201 %s", tt.body, res.StatusCode, body, tt.want)
		}
		if want := []string{aliceID.String() + " " + webID.String()}; !slices.Equal(create.calls, want) ||
			!reflect.DeepEqual(create.got, []domain.NewModule{tt.in}) {
			t.Errorf("POST %s: the use case got %q %+v, want %q %+v", tt.body, create.calls, create.got, want, tt.in)
		}
	}
}

// A body without its name, with a field it may not have (the sort order and
// the archive are the server's), null for a field that is not nullable (a
// date, the description, the members, the status left out for none), a
// field of another type, an id that is no UUID, or a date that is no day:
// refused as bad_request before the use case.
func TestCreateModuleHoldsTheBodyToItsStructure(t *testing.T) {
	create := &fakeCreateModule{}
	h := newServer(t, httpadapter.UseCases{CreateModule: create})
	for _, body := range []string{`{}`, `{"description":"x"}`, `{"name":1}`, `{"name":"A","sort_order":1}`,
		`{"name":"A","archived_at":"2026-11-07T10:00:00Z"}`, `{"name":"A","description":null}`, `{"name":"A","start_date":null}`,
		`{"name":"A","target_date":null}`, `{"name":"A","status":null}`, `{"name":"A","member_ids":null}`, `{"name":"A","status":1}`,
		`{"name":"A","lead_id":"bob"}`, `{"name":"A","member_ids":["bob"]}`, `{"name":"A","member_ids":"` + bobID.String() + `"}`,
		`{"name":"A","start_date":"2026-02-30"}`, `{"name":"A","target_date":"2026-11-13T00:00:00Z"}`} {
		res, got := do(t, h, request(http.MethodPost, "/api/v0/projects/"+webID.String()+"/modules", "alice", body))
		var problem struct{ Code string }
		if err := json.Unmarshal([]byte(got), &problem); err != nil || res.StatusCode != http.StatusBadRequest || problem.Code != "bad_request" {
			t.Errorf("POST %s = %d %s, want 400 bad_request", body, res.StatusCode, got)
		}
	}
	if len(create.calls) != 0 {
		t.Errorf("the use case got %q, want nothing", create.calls)
	}
}

// The use case's refusals, as the contract declares them, and its failure.
func TestCreateModuleRefusals(t *testing.T) {
	member := uuid.NewV7()
	refused := domain.NewModule{Name: "A", MemberIDs: []uuid.UUID{member}}.CheckRoles(nil)
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"a member who may not be", refused, http.StatusUnprocessableEntity,
			`{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"member_ids[0]","code":"not_allowed","message":"must be an active member of the project who is not its guest"}]}`},
		{"no project", domain.ErrProjectNotFound, http.StatusNotFound,
			`{"status":404,"code":"project.not_found","title":"Not Found","detail":"The project does not exist, or you cannot see it."}`},
		{"a project guest", shared.Forbidden(), http.StatusForbidden,
			`{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
		{"an archived project", domain.ErrProjectArchived, http.StatusConflict,
			`{"status":409,"code":"project.archived","title":"Conflict","detail":"The project is archived; unarchive it to change it."}`},
		{"modules turned off", domain.ErrModulesDisabled, http.StatusConflict,
			`{"status":409,"code":"planning.modules_disabled","title":"Conflict","detail":"Modules are turned off in this project; turn them on to change them."}`},
		{"a name taken", domain.ErrModuleNameTaken, http.StatusConflict,
			`{"status":409,"code":"planning.module_name_taken","title":"Conflict","detail":"A module of this project has this name."}`},
		{"a busy project", shared.ServerBusy(time.Second), http.StatusServiceUnavailable, busy},
		{"a failure", errGone, http.StatusInternalServerError, `{"status":500,"code":"internal_error","title":"Internal Server Error"}`},
	} {
		h := newServer(t, httpadapter.UseCases{CreateModule: &fakeCreateModule{err: tt.err}})
		res, body := do(t, h, request(http.MethodPost, "/api/v0/projects/"+webID.String()+"/modules", "alice", `{"name":"A"}`))
		if res.StatusCode != tt.status || body != tt.want+"\n" || tt.want == busy && res.Header.Get("Retry-After") != "1" {
			t.Errorf("%s: POST = %d %s, Retry-After %q; want %d %s", tt.name, res.StatusCode, body, res.Header.Get("Retry-After"), tt.status,
				tt.want)
		}
	}
}
