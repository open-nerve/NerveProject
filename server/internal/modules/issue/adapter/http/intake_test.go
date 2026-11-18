package httpadapter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeSubmit is submitIntakeIssue: each call is recorded as "caller
// project", with what it got; it answers answer, or err.
type fakeSubmit struct {
	calls  []string
	got    []domain.NewIntakeIssue
	answer domain.Issue
	err    error
}

func (f *fakeSubmit) Execute(ctx context.Context, projectID uuid.UUID, in domain.NewIntakeIssue) (domain.Issue, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Issue{}, err
	}
	f.calls = append(f.calls, actor.UserID.String()+" "+projectID.String())
	f.got = append(f.got, in)
	return f.answer, f.err
}

// fakeTriage is triageIntakeIssue: each call is recorded as "caller work
// item", with the triage it got; it answers answer, or err.
type fakeTriage struct {
	calls  []string
	got    []domain.Triage
	answer domain.Issue
	err    error
}

func (f *fakeTriage) Execute(ctx context.Context, id uuid.UUID, t domain.Triage) (domain.Issue, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Issue{}, err
	}
	f.calls = append(f.calls, actor.UserID.String()+" "+id.String())
	f.got = append(f.got, t)
	return f.answer, f.err
}

// The paths of the two operations: web's intake, and WEB-8's triage.
var (
	submitPath = "/api/v0/projects/" + webID.String() + "/intake-issues"
	triagePath = "/api/v0/intake-issues/" + webEight.ID.String() + "/triage"
	snoozeTill = time.Date(2026, 10, 20, 9, 0, 0, 123456000, time.UTC)
)

// inIntake is WEB-8 with the intake row f, and its detail's JSON with the
// row's as the answer shows it.
func inIntake(f domain.IntakeFacts, intake string) (domain.Issue, string) {
	i := webEight
	i.Intake = &f
	return i, strings.Replace(webEightDetailJSON, `"intake":null`, `"intake":`+intake, 1)
}

// POST goes to the use case for the caller and the path's project, with
// the body's six fields, each nil when not given and a list given empty
// kept empty; the answer is 201 with the work item the use case answers,
// its intake row with it: its status, and its time and the work item it
// duplicates each null when it has none.
func TestSubmitIntakeIssue(t *testing.T) {
	high, html := domain.PriorityHigh, "<p>It breaks</p>"
	pending, pendingJSON := inIntake(domain.IntakeFacts{Status: domain.IntakePending}, `{"duplicate_to_id":null,"snoozed_till":null,"status":"pending"}`)
	for _, tt := range []struct {
		body string
		want domain.NewIntakeIssue
	}{
		{`{"name":" Login breaks ","description_html":"<p>It breaks</p>","priority":"high","assignee_ids":["0199a2b4-0000-7000-8000-000000000002"],` +
			`"label_ids":["0199a2b4-0000-7000-8000-0000000000d1"],"target_date":"2026-10-19"}`,
			domain.NewIntakeIssue{Name: " Login breaks ", DescriptionHTML: &html, Priority: &high, AssigneeIDs: []uuid.UUID{bobID},
				LabelIDs: []uuid.UUID{bugID}, TargetDate: &targetDay}},
		{`{"name":"Fix"}`, domain.NewIntakeIssue{Name: "Fix"}},
		{`{"name":"Fix","assignee_ids":[],"label_ids":[]}`, domain.NewIntakeIssue{Name: "Fix", AssigneeIDs: []uuid.UUID{}, LabelIDs: []uuid.UUID{}}},
	} {
		submit := &fakeSubmit{answer: pending}
		res, body := do(t, newServer(t, httpadapter.UseCases{SubmitIntakeIssue: submit}), request(http.MethodPost, submitPath, "alice", tt.body))
		if res.StatusCode != http.StatusCreated || body != pendingJSON+"\n" {
			t.Errorf("POST %s = %d %s, want 201 %s", tt.body, res.StatusCode, body, pendingJSON)
		}
		if want := []string{aliceID.String() + " " + webID.String()}; !slices.Equal(submit.calls, want) ||
			!reflect.DeepEqual(submit.got, []domain.NewIntakeIssue{tt.want}) {
			t.Errorf("POST %s: calls %q with %+v; want %q with %+v", tt.body, submit.calls, submit.got, want, tt.want)
		}
	}
}

// A body without its name, with a field a submission may not have (the
// state, the start date, the parent, the cycle, the modules, a number or
// a sort order), null for a field that is not nullable, a field of another
// type, an id that is no id, or a day that is no day: refused as
// bad_request before the use case.
func TestSubmitIntakeIssueHoldsTheBodyToItsStructure(t *testing.T) {
	submit := &fakeSubmit{}
	h := newServer(t, httpadapter.UseCases{SubmitIntakeIssue: submit})
	for _, body := range []string{`{}`, `{"priority":"high"}`, `{"name":1}`, `{"name":"A","state_id":"0199a2b4-0000-7000-8000-0000000000c1"}`,
		`{"name":"A","start_date":"2026-10-12"}`, `{"name":"A","parent_id":"0199a2b4-0000-7000-8000-0000000000e6"}`,
		`{"name":"A","cycle_id":"0199a2b4-0000-7000-8000-0000000000b1"}`, `{"name":"A","module_ids":[]}`, `{"name":"A","sequence_id":3}`,
		`{"name":"A","sort_order":1}`, `{"name":"A","description_html":null}`, `{"name":"A","target_date":null}`, `{"name":"A","assignee_ids":"bob"}`,
		`{"name":"A","label_ids":["Bug"]}`, `{"name":"A","target_date":"2026-02-30"}`} {
		res, got := do(t, h, request(http.MethodPost, submitPath, "alice", body))
		var problem struct{ Code string }
		if err := json.Unmarshal([]byte(got), &problem); err != nil || res.StatusCode != http.StatusBadRequest || problem.Code != "bad_request" {
			t.Errorf("POST %s = %d %s, want 400 bad_request", body, res.StatusCode, got)
		}
	}
	if len(submit.calls) != 0 {
		t.Errorf("the use case got %q, want nothing", submit.calls)
	}
}

// intakeOff and the other refusals of the two operations, as the domain
// and the platform answer them.
const (
	intakeOff = `{"status":409,"code":"issue.intake_disabled","title":"Conflict",` +
		`"detail":"Intake is turned off in this project; turn it on to submit or triage work items."}`
	archivedProject = `{"status":409,"code":"project.archived","title":"Conflict","detail":"The project is archived; unarchive it to change it."}`
	forbidden       = `{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`
	failed          = `{"status":500,"code":"internal_error","title":"Internal Server Error"}`
)

// The use case's refusals, as the contract declares them, and its failure.
func TestSubmitIntakeIssueRefusals(t *testing.T) {
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"a label refused", shared.Invalid(shared.FieldError{Field: "label_ids[0]", Code: shared.FieldNotAllowed, Message: "must be a label of the project"}),
			http.StatusUnprocessableEntity, `{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"label_ids[0]","code":"not_allowed","message":"must be a label of the project"}]}`},
		{"no project", domain.ErrProjectNotFound, http.StatusNotFound,
			`{"status":404,"code":"project.not_found","title":"Not Found","detail":"The project does not exist, or you cannot see it."}`},
		{"a role that may not submit", shared.Forbidden(), http.StatusForbidden, forbidden},
		{"an archived project", domain.ErrProjectArchived, http.StatusConflict, archivedProject},
		{"its intake off", domain.ErrIntakeDisabled, http.StatusConflict, intakeOff},
		{"a busy project", shared.ServerBusy(time.Second), http.StatusServiceUnavailable, busy},
		{"an image not the work item's", shared.NewError(shared.KindNotFound, "asset.not_found", "The asset does not exist, or you cannot see it."),
			http.StatusNotFound, `{"status":404,"code":"asset.not_found","title":"Not Found","detail":"The asset does not exist, or you cannot see it."}`},
		{"a failure", errGone, http.StatusInternalServerError, failed},
	} {
		h := newServer(t, httpadapter.UseCases{SubmitIntakeIssue: &fakeSubmit{err: tt.err}})
		res, body := do(t, h, request(http.MethodPost, submitPath, "alice", `{"name":"Fix"}`))
		if res.StatusCode != tt.status || body != tt.want+"\n" || tt.want == busy && res.Header.Get("Retry-After") != "1" {
			t.Errorf("%s: POST = %d %s, Retry-After %q; want %d %s", tt.name, res.StatusCode, body, res.Header.Get("Retry-After"), tt.status, tt.want)
		}
	}
}
