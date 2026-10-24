package httpadapter_test

import (
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

// The work items the use case answers. WEB-7, under WEB-6, from the 12th
// to the 19th of October, completed, with an assignee and a label, made by
// alice, archived half a minute later and changed last by bob a minute
// after its creation: each field that may be null has a value of its own,
// so each shows as its own. WEB-8, new, with none of them, without a
// creator or a last writer (created_by_id and updated_by_id null).
var (
	stateID   = uuid.MustParse("0199a2b4-0000-7000-8000-0000000000c1")
	bobID     = uuid.MustParse("0199a2b4-0000-7000-8000-000000000002")
	bugID     = uuid.MustParse("0199a2b4-0000-7000-8000-0000000000d1")
	webSixID  = uuid.MustParse("0199a2b4-0000-7000-8000-0000000000e6")
	startDay  = shared.DateOf(time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC))
	targetDay = shared.DateOf(time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC))
	completed = created.Add(-1)
	archived  = created.Add(30 * time.Second)
	updated   = created.Add(time.Minute)
	webSeven  = domain.Issue{ID: uuid.MustParse("0199a2b4-0000-7000-8000-0000000000e7"), WorkspaceID: acmeID, ProjectID: webID, SequenceID: 7,
		Name: "Fix the login", DescriptionHTML: "<p>Seven</p>", StateID: stateID, Priority: domain.PriorityHigh, StartDate: &startDay, TargetDate: &targetDay, ParentID: &webSixID,
		SortOrder: 75535.5, CompletedAt: &completed, ArchivedAt: &archived, AssigneeIDs: []uuid.UUID{bobID}, LabelIDs: []uuid.UUID{bugID},
		SubIssuesCount: 2, CreatedByID: &aliceID, UpdatedByID: &bobID, CreatedAt: created, UpdatedAt: updated}
	webSevenJSON = `{"archived_at":"2026-10-17T10:00:30.123456Z","assignee_ids":["0199a2b4-0000-7000-8000-000000000002"],` +
		`"completed_at":"2026-10-17T10:00:00.123455999Z","created_at":"2026-10-17T10:00:00.123456Z",` +
		`"created_by_id":"0199a2b4-0000-7000-8000-000000000001","id":"0199a2b4-0000-7000-8000-0000000000e7",` +
		`"label_ids":["0199a2b4-0000-7000-8000-0000000000d1"],"name":"Fix the login","parent_id":"0199a2b4-0000-7000-8000-0000000000e6",` +
		`"priority":"high","project_id":"0199a2b4-0000-7000-8000-0000000000a1","sequence_id":7,"sort_order":75535.5,"start_date":"2026-10-12",` +
		`"state_id":"0199a2b4-0000-7000-8000-0000000000c1","sub_issues_count":2,"target_date":"2026-10-19",` +
		`"updated_at":"2026-10-17T10:01:00.123456Z",` +
		`"updated_by_id":"0199a2b4-0000-7000-8000-000000000002","workspace_id":"0199a2b4-0000-7000-8000-00000000000a"}`
	webEight = domain.Issue{ID: uuid.MustParse("0199a2b4-0000-7000-8000-0000000000e8"), WorkspaceID: acmeID, ProjectID: webID, SequenceID: 8,
		Name: "Fix", DescriptionHTML: "<p></p>", StateID: stateID, Priority: domain.PriorityNone, SortOrder: 65535, CreatedAt: created, UpdatedAt: created}
	webEightJSON = `{"archived_at":null,"assignee_ids":[],"completed_at":null,"created_at":"2026-10-17T10:00:00.123456Z",` +
		`"created_by_id":null,"id":"0199a2b4-0000-7000-8000-0000000000e8","label_ids":[],"name":"Fix","parent_id":null,"priority":"none",` +
		`"project_id":"0199a2b4-0000-7000-8000-0000000000a1","sequence_id":8,"sort_order":65535,"start_date":null,` +
		`"state_id":"0199a2b4-0000-7000-8000-0000000000c1","sub_issues_count":0,"target_date":null,"updated_at":"2026-10-17T10:00:00.123456Z",` +
		`"updated_by_id":null,` +
		`"workspace_id":"0199a2b4-0000-7000-8000-00000000000a"}`
)

var (
	webSevenDetailJSON = strings.Replace(webSevenJSON, `"id":`, `"description_html":"\u003cp\u003eSeven\u003c/p\u003e","id":`, 1)
	webEightDetailJSON = strings.Replace(webEightJSON, `"id":`, `"description_html":"\u003cp\u003e\u003c/p\u003e","id":`, 1)
)

// POST goes to the use case for the caller and the path's project, with
// the body's fields: the state, the priority, the dates and the parent nil
// and the lists empty when not given; the answer is 201 with the work item
// the use case answers, every field of it, null for each it has none of,
// [] for a list without ids.
func TestCreateIssue(t *testing.T) {
	path := "/api/v0/projects/" + webID.String() + "/issues"
	high := domain.PriorityHigh
	html := "<p>Rich</p>"
	for _, tt := range []struct {
		body   string
		want   domain.NewIssue
		answer domain.Issue
		json   string
	}{
		{`{"name":" Fix the login ","state_id":"0199a2b4-0000-7000-8000-0000000000c1","priority":"high",` +
			`"assignee_ids":["0199a2b4-0000-7000-8000-000000000002"],"label_ids":["0199a2b4-0000-7000-8000-0000000000d1"],` +
			`"start_date":"2026-10-12","target_date":"2026-10-19","parent_id":"0199a2b4-0000-7000-8000-0000000000e6"}`,
			domain.NewIssue{Name: " Fix the login ", StateID: &stateID, Priority: &high, AssigneeIDs: []uuid.UUID{bobID},
				LabelIDs: []uuid.UUID{bugID}, StartDate: &startDay, TargetDate: &targetDay, ParentID: &webSixID}, webSeven, webSevenDetailJSON},
		{`{"name":"Fix","description_html":"<p>Rich</p>"}`, domain.NewIssue{Name: "Fix", DescriptionHTML: &html}, webSeven, webSevenDetailJSON},
		{`{"name":"Fix"}`, domain.NewIssue{Name: "Fix"}, webEight, webEightDetailJSON},
		{`{"name":"Fix","assignee_ids":[],"label_ids":[]}`, domain.NewIssue{Name: "Fix", AssigneeIDs: []uuid.UUID{}, LabelIDs: []uuid.UUID{}},
			webEight, webEightDetailJSON},
	} {
		create := &fakeCreate{answer: tt.answer}
		h := newServer(t, httpadapter.UseCases{CreateIssue: create})
		if res, body := do(t, h, request(http.MethodPost, path, "alice", tt.body)); res.StatusCode != http.StatusCreated || body != tt.json+"\n" {
			t.Errorf("POST %s = %d %s, want 201 %s", tt.body, res.StatusCode, body, tt.json)
		}
		if want := []string{aliceID.String() + " " + webID.String()}; !slices.Equal(create.calls, want) ||
			!reflect.DeepEqual(create.got, []domain.NewIssue{tt.want}) {
			t.Errorf("POST %s: calls %q with %+v; want %q with %+v", tt.body, create.calls, create.got, want, tt.want)
		}
	}
}

// A priority given empty is given, not left out: the body's shape does not
// check an enum's values, so it goes to the use case as it came, whose
// check refuses it like any value outside the five (422 priority
// invalid_format; the fake answers that check's own refusal).
func TestCreateIssueTakesAnEmptyPriorityAsGiven(t *testing.T) {
	empty := domain.Priority("")
	in := domain.NewIssue{Name: "Fix", Priority: &empty}
	_, refused := domain.CheckNewIssue(in)
	create := &fakeCreate{err: refused}
	res, body := do(t, newServer(t, httpadapter.UseCases{CreateIssue: create}), request(http.MethodPost, "/api/v0/projects/"+webID.String()+"/issues", "alice",
		`{"name":"Fix","priority":""}`))
	want := `{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
		`"errors":[{"field":"priority","code":"invalid_format","message":"must be one of urgent, high, medium, low, none"}]}`
	if res.StatusCode != http.StatusUnprocessableEntity || body != want+"\n" || !reflect.DeepEqual(create.got, []domain.NewIssue{in}) {
		t.Errorf("POST = %d %s, the use case got %+v; want 422 %s, the use case %+v", res.StatusCode, body, create.got, want, in)
	}
}

// A body without its name, with a field it may not have (a number or a
// sort order, which the project gives: M4 design 3.7; a description, which
// comes in P4), null for a field that is not nullable (a date left out for
// none), a field of another type, an id that is no id, or a date that is
// no day or has a time: refused as bad_request before the use case.
func TestCreateIssueHoldsTheBodyToItsStructure(t *testing.T) {
	create := &fakeCreate{}
	h := newServer(t, httpadapter.UseCases{CreateIssue: create})
	for _, body := range []string{`{}`, `{"priority":"high"}`, `{"name":1}`, `{"name":"A","sequence_id":3}`, `{"name":"A","sort_order":1}`,
		`{"name":"A","description_html":null}`, `{"name":"A","state_id":"Backlog"}`, `{"name":"A","state_id":null}`,
		`{"name":"A","target_date":null}`, `{"name":"A","assignee_ids":"bob"}`, `{"name":"A","label_ids":["Bug"]}`,
		`{"name":"A","start_date":"2026-02-30"}`, `{"name":"A","target_date":"2026-10-19T00:00:00Z"}`} {
		res, got := do(t, h, request(http.MethodPost, "/api/v0/projects/"+webID.String()+"/issues", "alice", body))
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
func TestCreateIssueRefusals(t *testing.T) {
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"an assignee refused", shared.Invalid(shared.FieldError{Field: "assignee_ids[1]", Code: shared.FieldNotAllowed,
			Message: "must be an active member of the project who is not its guest"}), http.StatusUnprocessableEntity,
			`{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"assignee_ids[1]","code":"not_allowed","message":"must be an active member of the project who is not its guest"}]}`},
		{"no project", domain.ErrProjectNotFound, http.StatusNotFound,
			`{"status":404,"code":"project.not_found","title":"Not Found","detail":"The project does not exist, or you cannot see it."}`},
		{"a project guest", shared.Forbidden(), http.StatusForbidden,
			`{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
		{"an archived project", domain.ErrProjectArchived, http.StatusConflict,
			`{"status":409,"code":"project.archived","title":"Conflict","detail":"The project is archived; unarchive it to change it."}`},
		{"a busy project", shared.ServerBusy(time.Second), http.StatusServiceUnavailable, busy},
		{"a failure", errGone, http.StatusInternalServerError,
			`{"status":500,"code":"internal_error","title":"Internal Server Error"}`},
	} {
		h := newServer(t, httpadapter.UseCases{CreateIssue: &fakeCreate{err: tt.err}})
		res, body := do(t, h, request(http.MethodPost, "/api/v0/projects/"+webID.String()+"/issues", "alice", `{"name":"Fix"}`))
		if res.StatusCode != tt.status || body != tt.want+"\n" || tt.want == busy && res.Header.Get("Retry-After") != "1" {
			t.Errorf("%s: POST = %d %s, Retry-After %q; want %d %s", tt.name, res.StatusCode, body, res.Header.Get("Retry-After"), tt.status,
				tt.want)
		}
	}
}

// busy is the answer to a write the server is too busy for (shared's
// server_busy), which comes with Retry-After: 1.
const busy = `{"status":503,"code":"server_busy","title":"Service Unavailable","detail":"The server is busy; retry shortly."}`

// GET goes to the use case for the caller and the path's work item; the
// answer is 200 with the work item the use case answers, as POST shows it.
func TestGetIssue(t *testing.T) {
	for _, tt := range []struct {
		answer domain.Issue
		json   string
	}{{webSeven, webSevenDetailJSON}, {webEight, webEightDetailJSON}} {
		get := &fakeGet{answer: tt.answer}
		res, body := do(t, newServer(t, httpadapter.UseCases{GetIssue: get}), request(http.MethodGet, "/api/v0/issues/"+tt.answer.ID.String(),
			"alice", ""))
		if res.StatusCode != http.StatusOK || body != tt.json+"\n" {
			t.Errorf("GET = %d %s, want 200 %s", res.StatusCode, body, tt.json)
		}
		if want := []string{aliceID.String() + " " + tt.answer.ID.String()}; !slices.Equal(get.calls, want) {
			t.Errorf("GET: calls %q, want %q", get.calls, want)
		}
	}
}

// The use case's refusals, as the contract declares them, and its failure.
func TestGetIssueRefusals(t *testing.T) {
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"no work item", domain.ErrIssueNotFound, http.StatusNotFound,
			`{"status":404,"code":"issue.not_found","title":"Not Found","detail":"The work item does not exist, or you cannot see it."}`},
		{"a workspace admin who is no member of the project", shared.Forbidden(), http.StatusForbidden,
			`{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
		{"a failure", errGone, http.StatusInternalServerError,
			`{"status":500,"code":"internal_error","title":"Internal Server Error"}`},
	} {
		h := newServer(t, httpadapter.UseCases{GetIssue: &fakeGet{err: tt.err}})
		res, body := do(t, h, request(http.MethodGet, "/api/v0/issues/"+webSeven.ID.String(), "alice", ""))
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: GET = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}
