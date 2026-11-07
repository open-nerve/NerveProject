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

// fakeCreate is createCycle: each call is recorded as "caller project", with
// what it got; it answers answer, or err.
type fakeCreate struct {
	calls  []string
	got    []domain.NewCycle
	answer domain.ReadCycle
	err    error
}

func (f *fakeCreate) Execute(ctx context.Context, projectID uuid.UUID, in domain.NewCycle) (domain.ReadCycle, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.ReadCycle{}, err
	}
	f.calls = append(f.calls, actor.UserID.String()+" "+projectID.String())
	f.got = append(f.got, in)
	return f.answer, f.err
}

// day is the day s, YYYY-MM-DD.
func day(t *testing.T, s string) *shared.Date {
	t.Helper()
	d, err := shared.ParseDate(s)
	if err != nil {
		t.Fatal(err)
	}
	return &d
}

// web's cycles as the use cases answer them: Sprint 1, dated, current and
// archived (as only the read shows it), with an owner and its writers;
// Backlog, a draft whose accounts are gone.
func cycles(t *testing.T) (sprint, backlog domain.ReadCycle) {
	archived := created.Add(time.Hour)
	sprint = domain.ReadCycle{Cycle: domain.Cycle{ID: uuid.MustParse("0199a2b4-0000-7000-8000-0000000000c1"), ProjectID: webID,
		WorkspaceID: acmeID, Name: "Sprint 1", Description: "First", Dates: domain.Dates{Start: day(t, "2026-11-02"), End: day(t, "2026-11-13")},
		OwnedByID: &aliceID, SortOrder: 55535.5, ArchivedAt: &archived, CreatedAt: created, UpdatedAt: created.Add(1), CreatedByID: &aliceID,
		UpdatedByID: &aliceID}, Status: domain.StatusCurrent}
	backlog = domain.ReadCycle{Cycle: domain.Cycle{ID: uuid.MustParse("0199a2b4-0000-7000-8000-0000000000c2"), ProjectID: webID,
		WorkspaceID: acmeID, Name: "Backlog", SortOrder: 65535, CreatedAt: created, UpdatedAt: created}, Status: domain.StatusDraft}
	return sprint, backlog
}

const (
	sprintJSON = `{"archived_at":"2026-11-07T11:00:00.123456Z","created_at":"2026-11-07T10:00:00.123456Z",` +
		`"created_by_id":"0199a2b4-0000-7000-8000-000000000001","description":"First","end_date":"2026-11-13",` +
		`"id":"0199a2b4-0000-7000-8000-0000000000c1","name":"Sprint 1","owned_by_id":"0199a2b4-0000-7000-8000-000000000001",` +
		`"project_id":"0199a2b4-0000-7000-8000-0000000000a1","sort_order":55535.5,"start_date":"2026-11-02","status":"current",` +
		`"updated_at":"2026-11-07T10:00:00.123456001Z","updated_by_id":"0199a2b4-0000-7000-8000-000000000001",` +
		`"workspace_id":"0199a2b4-0000-7000-8000-00000000000a"}`
	backlogJSON = `{"archived_at":null,"created_at":"2026-11-07T10:00:00.123456Z","created_by_id":null,"description":"","end_date":null,` +
		`"id":"0199a2b4-0000-7000-8000-0000000000c2","name":"Backlog","owned_by_id":null,"project_id":"0199a2b4-0000-7000-8000-0000000000a1",` +
		`"sort_order":65535,"start_date":null,"status":"draft","updated_at":"2026-11-07T10:00:00.123456Z","updated_by_id":null,` +
		`"workspace_id":"0199a2b4-0000-7000-8000-00000000000a"}`
)

// POST goes to the use case for the caller and the path's project with the
// body's values as given (the use case cleans them), a description left out
// as empty and dates left out as none; the answer is 201 with every field of
// the cycle the use case answers, null for each value it has not.
func TestCreateCycle(t *testing.T) {
	sprint, backlog := cycles(t)
	path := "/api/v0/projects/" + webID.String() + "/cycles"
	for _, tt := range []struct {
		body   string
		in     domain.NewCycle
		answer domain.ReadCycle
		want   string
	}{
		{`{"name":" Sprint 1 ","description":"First","start_date":"2026-11-02","end_date":"2026-11-13"}`,
			domain.NewCycle{Name: " Sprint 1 ", Description: "First", Dates: domain.Dates{Start: day(t, "2026-11-02"), End: day(t, "2026-11-13")}},
			sprint, sprintJSON},
		{`{"name":"Backlog"}`, domain.NewCycle{Name: "Backlog"}, backlog, backlogJSON},
		{`{"name":"Half","end_date":"2026-11-13"}`, domain.NewCycle{Name: "Half", Dates: domain.Dates{End: day(t, "2026-11-13")}}, backlog,
			backlogJSON},
	} {
		create := &fakeCreate{answer: tt.answer}
		res, body := do(t, newServer(t, httpadapter.UseCases{CreateCycle: create}), request(http.MethodPost, path, "alice", tt.body))
		if res.StatusCode != http.StatusCreated || body != tt.want+"\n" {
			t.Errorf("POST %s = %d %s, want 201 %s", tt.body, res.StatusCode, body, tt.want)
		}
		if want := []string{aliceID.String() + " " + webID.String()}; !slices.Equal(create.calls, want) ||
			!reflect.DeepEqual(create.got, []domain.NewCycle{tt.in}) {
			t.Errorf("POST %s: the use case got %q %+v, want %q %+v", tt.body, create.calls, create.got, want, tt.in)
		}
	}
}

// A body without its name, with a field it may not have (the status, owner
// and sort order are the server's), null for a field that is not nullable
// (a date left out for none), a field of another type, or a date that is no
// day or has a time: refused as bad_request before the use case.
func TestCreateCycleHoldsTheBodyToItsStructure(t *testing.T) {
	create := &fakeCreate{}
	h := newServer(t, httpadapter.UseCases{CreateCycle: create})
	for _, body := range []string{`{}`, `{"description":"x"}`, `{"name":1}`, `{"name":"A","status":"draft"}`, `{"name":"A","sort_order":1}`,
		`{"name":"A","owned_by_id":"0199a2b4-0000-7000-8000-000000000001"}`, `{"name":"A","description":null}`,
		`{"name":"A","start_date":null,"end_date":null}`, `{"name":"A","start_date":"2026-02-30","end_date":"2026-03-02"}`,
		`{"name":"A","start_date":"2026-11-02","end_date":"2026-11-13T00:00:00Z"}`} {
		res, got := do(t, h, request(http.MethodPost, "/api/v0/projects/"+webID.String()+"/cycles", "alice", body))
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
func TestCreateCycleRefusals(t *testing.T) {
	_, reversed := domain.CheckNewCycle(domain.NewCycle{Name: "A", Dates: domain.Dates{Start: day(t, "2026-11-13"), End: day(t, "2026-11-02")}})
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"reversed dates", reversed, http.StatusUnprocessableEntity,
			`{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"end_date","code":"not_allowed","message":"must not be before the start date"}]}`},
		{"no project", domain.ErrProjectNotFound, http.StatusNotFound,
			`{"status":404,"code":"project.not_found","title":"Not Found","detail":"The project does not exist, or you cannot see it."}`},
		{"a project guest", shared.Forbidden(), http.StatusForbidden,
			`{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
		{"an archived project", domain.ErrProjectArchived, http.StatusConflict,
			`{"status":409,"code":"project.archived","title":"Conflict","detail":"The project is archived; unarchive it to change it."}`},
		{"cycles turned off", domain.ErrCyclesDisabled, http.StatusConflict,
			`{"status":409,"code":"planning.cycles_disabled","title":"Conflict","detail":"Cycles are turned off in this project; turn them on to change them."}`},
		{"overlapping dates", domain.ErrDatesOverlap, http.StatusConflict,
			`{"status":409,"code":"planning.cycle_dates_overlap","title":"Conflict","detail":"The dates overlap another cycle of this project."}`},
		{"a busy project", shared.ServerBusy(time.Second), http.StatusServiceUnavailable, busy},
		{"a failure", errGone, http.StatusInternalServerError, `{"status":500,"code":"internal_error","title":"Internal Server Error"}`},
	} {
		h := newServer(t, httpadapter.UseCases{CreateCycle: &fakeCreate{err: tt.err}})
		res, body := do(t, h, request(http.MethodPost, "/api/v0/projects/"+webID.String()+"/cycles", "alice", `{"name":"A"}`))
		if res.StatusCode != tt.status || body != tt.want+"\n" || tt.want == busy && res.Header.Get("Retry-After") != "1" {
			t.Errorf("%s: POST = %d %s, Retry-After %q; want %d %s", tt.name, res.StatusCode, body, res.Header.Get("Retry-After"), tt.status,
				tt.want)
		}
	}
}
