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
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeRelate is createIssueRelations: each call is recorded as "caller
// work item type", with the work items it got; it answers answer, or err.
type fakeRelate struct {
	calls  []string
	got    [][]uuid.UUID
	answer []app.IssueRelation
	err    error
}

func (f *fakeRelate) Execute(ctx context.Context, id uuid.UUID, t domain.RelationType, ids []uuid.UUID) ([]app.IssueRelation, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	f.calls = append(f.calls, actor.UserID.String()+" "+id.String()+" "+string(t))
	f.got = append(f.got, ids)
	return f.answer, f.err
}

// relationsPath is the path of WEB-6's relations; relatedAnswers, the
// answers of the use case: WEB-6 blocked by WEB-8 and a duplicate of
// WEB-7, each relation with an id, a type and a moment of its own, so that
// each shows as its own, the other work item as getIssue shows it, in the
// use case's order; or none.
var (
	relationsPath  = "/api/v0/issues/" + webSixID.String() + "/relations"
	relatedAnswers = []struct {
		relations []app.IssueRelation
		json      string
	}{
		{[]app.IssueRelation{
			{ID: uuid.MustParse("0199a2b4-0000-7000-8000-0000000000f1"), Type: domain.BlockedBy, Issue: webEight, CreatedAt: created.Add(time.Hour)},
			{ID: uuid.MustParse("0199a2b4-0000-7000-8000-0000000000f2"), Type: domain.Duplicate, Issue: webSeven, CreatedAt: created.Add(2 * time.Hour)},
		}, `{"data":[{"created_at":"2026-10-17T11:00:00.123456Z","id":"0199a2b4-0000-7000-8000-0000000000f1","issue":` + webEightJSON +
			`,"relation_type":"blocked_by"},{"created_at":"2026-10-17T12:00:00.123456Z","id":"0199a2b4-0000-7000-8000-0000000000f2","issue":` +
			webSevenJSON + `,"relation_type":"duplicate"}]}`},
		{nil, `{"data":[]}`},
	}
)

// POST goes to the use case for the caller and the path's work item, with
// the body's type and work items as they came: a type that is none of the
// four too, a work item named twice too, or none, in their order; the
// answer is 201 with the relations it answers, in its order, [] for none.
func TestCreateIssueRelations(t *testing.T) {
	for _, tt := range []struct {
		body string
		kind domain.RelationType
		ids  []uuid.UUID
	}{
		{`{"relation_type":"blocking","issue_ids":["0199a2b4-0000-7000-8000-0000000000e8","0199a2b4-0000-7000-8000-0000000000e7",` +
			`"0199a2b4-0000-7000-8000-0000000000e8"]}`, domain.Blocking, []uuid.UUID{webEight.ID, webSeven.ID, webEight.ID}},
		{`{"relation_type":"start_before","issue_ids":[]}`, "start_before", []uuid.UUID{}},
	} {
		for _, answer := range relatedAnswers {
			relate := &fakeRelate{answer: answer.relations}
			res, body := do(t, newServer(t, httpadapter.UseCases{CreateIssueRelations: relate}), request(http.MethodPost, relationsPath, "alice", tt.body))
			if res.StatusCode != http.StatusCreated || body != answer.json+"\n" {
				t.Errorf("POST %s = %d %s, want 201 %s", tt.body, res.StatusCode, body, answer.json)
			}
			if want := []string{aliceID.String() + " " + webSixID.String() + " " + string(tt.kind)}; !slices.Equal(relate.calls, want) ||
				!reflect.DeepEqual(relate.got, [][]uuid.UUID{tt.ids}) {
				t.Errorf("POST %s: calls %q with %v; want %q with %v", tt.body, relate.calls, relate.got, want, tt.ids)
			}
		}
	}
}

// A body without the type or the work items, with null or another type
// for either, an id that is no id, or a field it may not have: refused as
// bad_request before the use case.
func TestCreateIssueRelationsHoldsTheBodyToItsStructure(t *testing.T) {
	relate := &fakeRelate{}
	h := newServer(t, httpadapter.UseCases{CreateIssueRelations: relate})
	for _, body := range []string{`{}`, `{"issue_ids":["0199a2b4-0000-7000-8000-0000000000e8"]}`, `{"relation_type":"blocking"}`,
		`{"relation_type":null,"issue_ids":[]}`, `{"relation_type":1,"issue_ids":[]}`, `{"relation_type":"blocking","issue_ids":null}`,
		`{"relation_type":"blocking","issue_ids":"WEB-8"}`, `{"relation_type":"blocking","issue_ids":["WEB-8"]}`,
		`{"relation_type":"blocking","issue_ids":[],"sub_issue_ids":[]}`} {
		res, got := do(t, h, request(http.MethodPost, relationsPath, "alice", body))
		var problem struct{ Code string }
		if err := json.Unmarshal([]byte(got), &problem); err != nil || res.StatusCode != http.StatusBadRequest || problem.Code != "bad_request" {
			t.Errorf("POST %s = %d %s, want 400 bad_request", body, res.StatusCode, got)
		}
	}
	if len(relate.calls) != 0 {
		t.Errorf("the use case got %q, want nothing", relate.calls)
	}
}

// The use case's refusals, as the contract declares them, and its
// failure.
func TestCreateIssueRelationsRefusals(t *testing.T) {
	invalid := `{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.","errors":[`
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"a type of none of the four, and none named", domain.CheckNewRelations("start_before", nil), http.StatusUnprocessableEntity,
			invalid + `{"field":"relation_type","code":"invalid_format","message":"must be one of relates_to, duplicate, blocked_by, blocking"},` +
				`{"field":"issue_ids","code":"too_short","message":"must name a work item"}]}`},
		{"a work item refused, and one related already", shared.Invalid(domain.RelatedNotAllowed(1), domain.RelatedAlready(2)),
			http.StatusUnprocessableEntity, invalid + `{"field":"issue_ids[1]","code":"not_allowed","message":"must be another work item of ` +
				`the workspace that you see, not archived, in a project that is not archived"},` +
				`{"field":"issue_ids[2]","code":"duplicate","message":"relates to this work item already"}]}`},
		{"no work item", domain.ErrIssueNotFound, http.StatusNotFound,
			`{"status":404,"code":"issue.not_found","title":"Not Found","detail":"The work item does not exist, or you cannot see it."}`},
		{"a project guest", shared.Forbidden(), http.StatusForbidden,
			`{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
		{"an archived project", domain.ErrProjectArchived, http.StatusConflict,
			`{"status":409,"code":"project.archived","title":"Conflict","detail":"The project is archived; unarchive it to change it."}`},
		{"an archived work item", domain.ErrIssueArchived, http.StatusConflict,
			`{"status":409,"code":"issue.archived","title":"Conflict","detail":"The work item is archived; unarchive it to change it."}`},
		{"a busy project", shared.ServerBusy(time.Second), http.StatusServiceUnavailable, busy},
		{"a failure", errGone, http.StatusInternalServerError, `{"status":500,"code":"internal_error","title":"Internal Server Error"}`},
	} {
		h := newServer(t, httpadapter.UseCases{CreateIssueRelations: &fakeRelate{err: tt.err}})
		body := `{"relation_type":"blocking","issue_ids":["0199a2b4-0000-7000-8000-0000000000e8"]}`
		res, got := do(t, h, request(http.MethodPost, relationsPath, "alice", body))
		if res.StatusCode != tt.status || got != tt.want+"\n" || tt.want == busy && res.Header.Get("Retry-After") != "1" {
			t.Errorf("%s: POST = %d %s, Retry-After %q; want %d %s", tt.name, res.StatusCode, got, res.Header.Get("Retry-After"), tt.status, tt.want)
		}
	}
}
