package httpadapter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeIntakeList is listIntakeIssues: it records the caller, the project
// and the query it got, which it reads as the use case does
// (domain.ParseIntakeQuery), its 400 or 422 the answer; else it answers
// page, or err.
type fakeIntakeList struct {
	actor, id uuid.UUID
	raw       map[string][]string
	page      app.IntakeIssuePage
	err       error
}

func (f *fakeIntakeList) Execute(ctx context.Context, id uuid.UUID, raw map[string][]string) (app.IntakeIssuePage, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return app.IntakeIssuePage{}, err
	}
	f.actor, f.id, f.raw = actor.UserID, id, raw
	if _, err := domain.ParseIntakeQuery(raw, actor.UserID); err != nil {
		return app.IntakeIssuePage{}, err
	}
	return f.page, f.err
}

// fakeIntakeCounts is getIntakeCounts: it records the caller and the
// workspace's slug; it answers counts, or err.
type fakeIntakeCounts struct {
	actor  uuid.UUID
	slug   string
	counts []app.IntakeCount
	err    error
}

func (f *fakeIntakeCounts) Execute(ctx context.Context, slug string) ([]app.IntakeCount, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	f.actor, f.slug = actor.UserID, slug
	return f.counts, f.err
}

var (
	intakeListPath   = "/api/v0/projects/" + webID.String() + "/intake-issues"
	intakeCountsPath = "/api/v0/workspaces/acme/intake-counts"
)

// intakeListed is a work item i as a row of the intake list with the
// intake row f, and the row's JSON: the list row's, rowJSON, with the
// intake row's, intake.
func intakeListed(i domain.Issue, f domain.IntakeFacts, rowJSON, intake string) (domain.Issue, string) {
	i.Intake = &f
	return i, strings.Replace(rowJSON, `"label_ids":`, `"intake":`+intake+`,"label_ids":`, 1)
}

// GET goes to the use case for the caller and the path's project; the
// answer is its page: its total, each work item as a project list shows
// it with its intake row as its detail shows it, and its cursor, null on
// the last page.
func TestListIntakeIssues(t *testing.T) {
	seven, sevenJSON := intakeListed(webSeven, domain.IntakeFacts{Status: domain.IntakeAccepted}, webSevenJSON,
		`{"duplicate_to_id":null,"snoozed_till":null,"status":"accepted"}`)
	eight, eightJSON := intakeListed(webEight, domain.IntakeFacts{Status: domain.IntakeDuplicate, DuplicateToID: &webSixID}, webEightJSON,
		`{"duplicate_to_id":"0199a2b4-0000-7000-8000-0000000000e6","snoozed_till":null,"status":"duplicate"}`)
	snoozed, snoozedJSON := intakeListed(webEight, domain.IntakeFacts{Status: domain.IntakeSnoozed, SnoozedTill: &snoozeTill}, webEightJSON,
		`{"duplicate_to_id":null,"snoozed_till":"2026-10-20T09:00:00.123456Z","status":"snoozed"}`)
	for _, tt := range []struct {
		name string
		page app.IntakeIssuePage
		want string
	}{
		{"none", app.IntakeIssuePage{Data: []domain.Issue{}}, `{"total":0,"data":[],"next_cursor":null}`},
		{"the last page", app.IntakeIssuePage{Total: 2, Data: []domain.Issue{seven, eight}}, `{"total":2,"data":[` + sevenJSON + `,` + eightJSON + `],"next_cursor":null}`},
		{"a page before another", app.IntakeIssuePage{Total: 3, Data: []domain.Issue{snoozed}, NextCursor: "next"},
			`{"total":3,"data":[` + snoozedJSON + `],"next_cursor":"next"}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeIntakeList{page: tt.page}
			res, body := do(t, newServer(t, httpadapter.UseCases{ListIntakeIssues: f}), request(http.MethodGet, intakeListPath, "alice", ""))
			var got, want any
			if res.StatusCode != http.StatusOK || json.Unmarshal([]byte(body), &got) != nil || json.Unmarshal([]byte(tt.want), &want) != nil ||
				!reflect.DeepEqual(got, want) || f.actor != aliceID || f.id != webID {
				t.Fatalf("GET = %d %s, the use case for %s on %s; want 200 %s, for alice on web", res.StatusCode, body, f.actor, f.id, tt.want)
			}
		})
	}
}

// The query goes to the use case as it came, every parameter and value,
// a CSV's too; each parameter of the list given twice is 422 not_allowed
// on it, its days' too, before the token is looked at; a query the use
// case does not read is its 422 or 400; a limit that is no number is 400.
func TestIntakeListQuery(t *testing.T) {
	check := func(t *testing.T, query, token string, status int, field, code string) {
		t.Helper()
		f := &fakeIntakeList{}
		res, body := do(t, newServer(t, httpadapter.UseCases{ListIntakeIssues: f}), request(http.MethodGet, intakeListPath+"?"+query, token, ""))
		var p struct{ Errors []shared.FieldError }
		_ = json.Unmarshal([]byte(body), &p)
		if res.StatusCode != status || field != "" && (len(p.Errors) != 1 || p.Errors[0].Field != field || p.Errors[0].Code != code) {
			t.Fatalf("GET ?%s = %d %s; want %d %s %s", query, res.StatusCode, body, status, field, code)
		}
	}
	values := map[string]string{"status": "pending,snoozed", "priority": "high", "label_id": "none", "assignee_id": "me", "created_by_id": "me",
		"created_at": "2026-10-01..2026-10-20", "updated_at": "2026-10-01..", "order_by": "-sequence_id", "limit": "1", "cursor": "bad"}
	for field, value := range values {
		t.Run("twice/"+field, func(t *testing.T) {
			check(t, field+"="+url.QueryEscape(value)+"&"+field+"="+url.QueryEscape(value), "alice", 422, field, "not_allowed")
			check(t, field+"="+url.QueryEscape(value)+"&"+field+"="+url.QueryEscape(value), "", 422, field, "not_allowed")
		})
	}
	for _, tt := range []struct {
		query, field, code string
		status             int
	}{
		{"unknown=x", "unknown", "not_allowed", 422}, {"status=open", "status", "not_allowed", 422}, {"created_at=none", "created_at", "not_allowed", 422},
		{"order_by=priority", "order_by", "not_allowed", 422}, {"limit=no", "limit", "invalid_format", 400}, {"limit=0", "limit", "out_of_range", 422},
		{"cursor=bad", "cursor", "invalid_format", 400},
	} {
		t.Run(tt.query, func(t *testing.T) { check(t, tt.query, "alice", tt.status, tt.field, tt.code) })
	}
	check(t, "", "", http.StatusUnauthorized, "", "")
	raw := "status=pending,snoozed&priority=high,none&label_id=" + bugID.String() + ",none&assignee_id=me&created_by_id=" + bobID.String() +
		"&created_at=2026-10-01..2026-10-20&updated_at=..2026-10-20&order_by=-sequence_id&limit=20"
	f := &fakeIntakeList{}
	res, body := do(t, newServer(t, httpadapter.UseCases{ListIntakeIssues: f}), request(http.MethodGet, intakeListPath+"?"+raw, "alice", ""))
	if parsed, _ := url.ParseQuery(raw); res.StatusCode != http.StatusOK || !reflect.DeepEqual(f.raw, map[string][]string(parsed)) {
		t.Fatalf("GET ?%s = %d %s, the use case got %v; want 200 and the query as it came", raw, res.StatusCode, body, f.raw)
	}
}

// The use case's refusals, as the contract declares them, and its failure.
func TestListIntakeIssuesRefusals(t *testing.T) {
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"a value refused", shared.Invalid(shared.FieldError{Field: "status", Code: shared.FieldNotAllowed, Message: "is not allowed for this list"}),
			http.StatusUnprocessableEntity, `{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"status","code":"not_allowed","message":"is not allowed for this list"}]}`},
		{"no project", domain.ErrProjectNotFound, http.StatusNotFound,
			`{"status":404,"code":"project.not_found","title":"Not Found","detail":"The project does not exist, or you cannot see it."}`},
		{"a role that may not read it", shared.Forbidden(), http.StatusForbidden, forbidden},
		{"a failure", errGone, http.StatusInternalServerError, failed},
	} {
		h := newServer(t, httpadapter.UseCases{ListIntakeIssues: &fakeIntakeList{err: tt.err}})
		res, body := do(t, h, request(http.MethodGet, intakeListPath, "alice", ""))
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: GET = %d %s; want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

// GET goes to the use case for the caller and the path's workspace; the
// answer is each project it counts, in its order, none too.
func TestGetIntakeCounts(t *testing.T) {
	for _, tt := range []struct {
		name   string
		counts []app.IntakeCount
		want   string
	}{
		{"none", []app.IntakeCount{}, `{"data":[]}`},
		{"two", []app.IntakeCount{{ProjectID: webID, Pending: 2}, {ProjectID: webSixID, Pending: 0}},
			`{"data":[{"pending":2,"project_id":"0199a2b4-0000-7000-8000-0000000000a1"},{"pending":0,"project_id":"0199a2b4-0000-7000-8000-0000000000e6"}]}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeIntakeCounts{counts: tt.counts}
			res, body := do(t, newServer(t, httpadapter.UseCases{CountIntakeIssues: f}), request(http.MethodGet, intakeCountsPath, "alice", ""))
			if res.StatusCode != http.StatusOK || body != tt.want+"\n" || f.actor != aliceID || f.slug != "acme" {
				t.Fatalf("GET = %d %s, the use case for %s in %q; want 200 %s, for alice in acme", res.StatusCode, body, f.actor, f.slug, tt.want)
			}
		})
	}
}

// The use case's refusals, as the contract declares them, and its failure.
func TestGetIntakeCountsRefusals(t *testing.T) {
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"no workspace", domain.ErrWorkspaceNotFound, http.StatusNotFound,
			`{"status":404,"code":"workspace.not_found","title":"Not Found","detail":"The workspace does not exist, or you are not a member of it."}`},
		{"a role that may not count", shared.Forbidden(), http.StatusForbidden, forbidden},
		{"a failure", errGone, http.StatusInternalServerError, failed},
	} {
		h := newServer(t, httpadapter.UseCases{CountIntakeIssues: &fakeIntakeCounts{err: tt.err}})
		res, body := do(t, h, request(http.MethodGet, intakeCountsPath, "alice", ""))
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: GET = %d %s; want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}
