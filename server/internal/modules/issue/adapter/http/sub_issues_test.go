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

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeList is listSubIssues: each call is recorded as "caller work item";
// it answers answer, or err.
type fakeList struct {
	calls  []string
	answer []domain.Issue
	err    error
}

func (f *fakeList) Execute(ctx context.Context, id uuid.UUID) ([]domain.Issue, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	f.calls = append(f.calls, actor.UserID.String()+" "+id.String())
	return f.answer, f.err
}

// fakeAdd is addSubIssues: each call is recorded as "caller work item",
// with the work items it got; it answers answer, or err.
type fakeAdd struct {
	calls  []string
	got    [][]uuid.UUID
	answer []domain.Issue
	err    error
}

func (f *fakeAdd) Execute(ctx context.Context, id uuid.UUID, ids []uuid.UUID) ([]domain.Issue, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	f.calls = append(f.calls, actor.UserID.String()+" "+id.String())
	f.got = append(f.got, ids)
	return f.answer, f.err
}

// subIssuesPath is the path of WEB-7's sub-items; listed, the answers of
// both operations: WEB-7 and WEB-8, each as getIssue shows it, in the use
// case's order, or none.
var (
	subIssuesPath = "/api/v0/issues/" + webSeven.ID.String() + "/sub-issues"
	listed        = []struct {
		items []domain.Issue
		json  string
	}{{[]domain.Issue{webSeven, webEight}, `{"data":[` + webSevenJSON + `,` + webEightJSON + `]}`}, {nil, `{"data":[]}`}}
)

// GET goes to the use case for the caller and the path's work item; the
// answer is 200 with the work items it answers, in its order, [] for none.
func TestListSubIssues(t *testing.T) {
	for _, answer := range listed {
		list := &fakeList{answer: answer.items}
		res, body := do(t, newServer(t, httpadapter.UseCases{ListSubIssues: list}), request(http.MethodGet, subIssuesPath, "alice", ""))
		if res.StatusCode != http.StatusOK || body != answer.json+"\n" {
			t.Errorf("GET = %d %s, want 200 %s", res.StatusCode, body, answer.json)
		}
		if want := []string{aliceID.String() + " " + webSeven.ID.String()}; !slices.Equal(list.calls, want) {
			t.Errorf("calls %q, want %q", list.calls, want)
		}
	}
}

// POST goes to the use case for the caller and the path's work item, with
// the body's work items as they came, in their order, one named twice too,
// or none; the answer is 200 with the work items it answers, as GET shows
// them.
func TestAddSubIssues(t *testing.T) {
	for _, tt := range []struct {
		body string
		ids  []uuid.UUID
	}{
		{`{"sub_issue_ids":["0199a2b4-0000-7000-8000-0000000000e8","0199a2b4-0000-7000-8000-0000000000e6",` +
			`"0199a2b4-0000-7000-8000-0000000000e8"]}`, []uuid.UUID{webEight.ID, webSixID, webEight.ID}},
		{`{"sub_issue_ids":[]}`, []uuid.UUID{}},
	} {
		for _, answer := range listed {
			add := &fakeAdd{answer: answer.items}
			res, body := do(t, newServer(t, httpadapter.UseCases{AddSubIssues: add}), request(http.MethodPost, subIssuesPath, "alice", tt.body))
			if res.StatusCode != http.StatusOK || body != answer.json+"\n" {
				t.Errorf("POST %s = %d %s, want 200 %s", tt.body, res.StatusCode, body, answer.json)
			}
			if want := []string{aliceID.String() + " " + webSeven.ID.String()}; !slices.Equal(add.calls, want) ||
				!reflect.DeepEqual(add.got, [][]uuid.UUID{tt.ids}) {
				t.Errorf("POST %s: calls %q with %v; want %q with %v", tt.body, add.calls, add.got, want, tt.ids)
			}
		}
	}
}

// A body without the work items, with null or another type for them, an id
// that is no id, or a field it may not have: refused as bad_request before
// the use case.
func TestAddSubIssuesHoldsTheBodyToItsStructure(t *testing.T) {
	add := &fakeAdd{}
	h := newServer(t, httpadapter.UseCases{AddSubIssues: add})
	for _, body := range []string{`{}`, `{"sub_issue_ids":null}`, `{"sub_issue_ids":"WEB-6"}`, `{"sub_issue_ids":["WEB-6"]}`,
		`{"sub_issue_ids":[],"parent_id":"0199a2b4-0000-7000-8000-0000000000e6"}`} {
		res, got := do(t, h, request(http.MethodPost, subIssuesPath, "alice", body))
		var problem struct{ Code string }
		if err := json.Unmarshal([]byte(got), &problem); err != nil || res.StatusCode != http.StatusBadRequest || problem.Code != "bad_request" {
			t.Errorf("POST %s = %d %s, want 400 bad_request", body, res.StatusCode, got)
		}
	}
	if len(add.calls) != 0 {
		t.Errorf("the use case got %q, want nothing", add.calls)
	}
}

// The use cases' refusals, as the contract declares them, and their
// failure: listSubIssues' are getIssue's; addSubIssues' updateIssue's.
func TestSubIssuesRefusals(t *testing.T) {
	post, both := []string{http.MethodPost}, []string{http.MethodGet, http.MethodPost}
	for _, tt := range []struct {
		name    string
		methods []string
		err     error
		status  int
		want    string
	}{
		{"none named", post, domain.CheckSubIssueIDs(nil), http.StatusUnprocessableEntity,
			`{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"sub_issue_ids","code":"too_short","message":"must name a work item"}]}`},
		{"a work item refused", post, shared.Invalid(domain.SubIssueNotAllowed(1)), http.StatusUnprocessableEntity,
			`{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"sub_issue_ids[1]","code":"not_allowed","message":"must be a work item of the project that you see, not ` +
				`archived, and neither this one nor above it"}]}`},
		{"no work item", both, domain.ErrIssueNotFound, http.StatusNotFound,
			`{"status":404,"code":"issue.not_found","title":"Not Found","detail":"The work item does not exist, or you cannot see it."}`},
		{"a project guest", both, shared.Forbidden(), http.StatusForbidden,
			`{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
		{"an archived project", post, domain.ErrProjectArchived, http.StatusConflict,
			`{"status":409,"code":"project.archived","title":"Conflict","detail":"The project is archived; unarchive it to change it."}`},
		{"an archived work item", post, domain.ErrIssueArchived, http.StatusConflict,
			`{"status":409,"code":"issue.archived","title":"Conflict","detail":"The work item is archived; unarchive it to change it."}`},
		{"a busy project", post, shared.ServerBusy(time.Second), http.StatusServiceUnavailable, busy},
		{"a failure", both, errGone, http.StatusInternalServerError, `{"status":500,"code":"internal_error","title":"Internal Server Error"}`},
	} {
		for _, method := range tt.methods {
			h := newServer(t, httpadapter.UseCases{ListSubIssues: &fakeList{err: tt.err}, AddSubIssues: &fakeAdd{err: tt.err}})
			body := ""
			if method == http.MethodPost {
				body = `{"sub_issue_ids":["0199a2b4-0000-7000-8000-0000000000e8"]}`
			}
			res, got := do(t, h, request(method, subIssuesPath, "alice", body))
			if res.StatusCode != tt.status || got != tt.want+"\n" || tt.want == busy && res.Header.Get("Retry-After") != "1" {
				t.Errorf("%s: %s = %d %s, Retry-After %q; want %d %s", tt.name, method, res.StatusCode, got, res.Header.Get("Retry-After"),
					tt.status, tt.want)
			}
		}
	}
}
