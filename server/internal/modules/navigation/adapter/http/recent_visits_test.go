package httpadapter_test

import (
	"context"
	"net/http"
	"reflect"
	"testing"
	"time"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/navigation/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

var (
	itemVisitID    = uuid.MustParse("0199a2b4-0000-7000-8000-0000000000c1")
	projectVisitID = uuid.MustParse("0199a2b4-0000-7000-8000-0000000000c2")
	itemID         = uuid.MustParse("0199a2b4-0000-7000-8000-0000000000b1")
	todoID         = uuid.MustParse("0199a2b4-0000-7000-8000-0000000000d1")
	visited        = created.Add(2 * time.Hour)
)

// The recent visits' paths: the caller's recording, and his list in acme.
const (
	recordPath = "/api/v0/me/recent-visits"
	recentPath = "/api/v0/workspaces/acme/recent-visits"
)

// recentCall is what a recent visits' use case was asked: the caller, the
// operation, the workspace's slug, a recording's kind and id, and a list's
// kind, nil for both.
type recentCall struct {
	Caller uuid.UUID
	Op     string
	Slug   string
	Kind   domain.RecentVisitKind
	ID     uuid.UUID
	Of     *domain.RecentVisitKind
}

// recentFakes are the recent visits' use cases, each a fake that records
// its call and answers list, or err.
type recentFakes struct {
	calls []recentCall
	list  []app.ShownRecentVisit
	err   error
}

// record keeps c, asked by the caller of ctx (the zero id when none).
func (f *recentFakes) record(ctx context.Context, c recentCall) {
	if actor, err := shared.RequireActor(ctx); err == nil {
		c.Caller = actor.UserID
	}
	f.calls = append(f.calls, c)
}

type (
	fakeRecordRecentVisit struct{ *recentFakes }
	fakeListRecentVisits  struct{ *recentFakes }
)

func (f fakeRecordRecentVisit) Execute(ctx context.Context, kind domain.RecentVisitKind, id uuid.UUID) error {
	f.record(ctx, recentCall{Op: "recordRecentVisit", Kind: kind, ID: id})
	return f.err
}

func (f fakeListRecentVisits) Execute(ctx context.Context, slug string, kind *domain.RecentVisitKind) ([]app.ShownRecentVisit, error) {
	f.record(ctx, recentCall{Op: "listRecentVisits", Slug: slug, Of: kind})
	return f.list, f.err
}

// useCases are f as the module's recent visits' use cases.
func (f *recentFakes) useCases() httpadapter.UseCases {
	return httpadapter.UseCases{RecordRecentVisit: fakeRecordRecentVisit{f}, ListRecentVisits: fakeListRecentVisits{f}}
}

// itemVisit is alice's visit of the work item WEB-7 in Web, with each value
// a work item's answer has: Web's icon, its state, its priority and two
// assignees.
func itemVisit() app.ShownRecentVisit {
	return app.ShownRecentVisit{
		Visit: domain.RecentVisit{ID: itemVisitID, WorkspaceID: acmeID, ProjectID: webID, UserID: aliceID, Kind: domain.RecentVisitIssue, EntityID: itemID,
			VisitedAt: visited},
		Project: app.ProjectTarget{WorkspaceID: acmeID, Name: "Web", Identifier: "WEB",
			LogoProps: shared.LogoProps{InUse: new(shared.LogoInUseEmoji), Emoji: &shared.LogoEmoji{Value: new("128640")}}},
		Issue: &app.IssueSummary{ID: itemID, ProjectID: webID, SequenceID: 7, Name: "Fix the login", StateID: todoID, Priority: "high",
			AssigneeIDs: []uuid.UUID{aliceID, bobID}}}
}

// projectVisit is alice's visit of Web itself, an archived project without
// an icon, which the API does not tell.
func projectVisit() app.ShownRecentVisit {
	return app.ShownRecentVisit{
		Visit: domain.RecentVisit{ID: projectVisitID, WorkspaceID: acmeID, ProjectID: webID, UserID: aliceID, Kind: domain.RecentVisitProject,
			EntityID: webID, VisitedAt: created},
		Project: app.ProjectTarget{WorkspaceID: acmeID, Name: "Web", Identifier: "WEB", Archived: true}}
}

// itemOfNone is alice's visit of WEB-7 assigned to no one, its assignees
// nil as a use case may give them.
func itemOfNone() app.ShownRecentVisit {
	v := itemVisit()
	v.Issue = &app.IssueSummary{ID: itemID, ProjectID: webID, SequenceID: 7, Name: "Fix the login", StateID: todoID, Priority: "none"}
	return v
}

const (
	itemVisitJSON = `{"id":"0199a2b4-0000-7000-8000-0000000000c1","entity_type":"issue","entity_id":"0199a2b4-0000-7000-8000-0000000000b1",` +
		`"visited_at":"2026-11-20T12:00:00.123456Z","project":{"id":"0199a2b4-0000-7000-8000-0000000000a1","name":"Web","identifier":"WEB",` +
		`"logo_props":{"in_use":"emoji","emoji":{"value":"128640"}}},"issue":{"id":"0199a2b4-0000-7000-8000-0000000000b1","sequence_id":7,` +
		`"name":"Fix the login","state_id":"0199a2b4-0000-7000-8000-0000000000d1","priority":"high",` +
		`"assignee_ids":["0199a2b4-0000-7000-8000-000000000001","0199a2b4-0000-7000-8000-000000000002"]}}`
	projectVisitJSON = `{"id":"0199a2b4-0000-7000-8000-0000000000c2","entity_type":"project","entity_id":"0199a2b4-0000-7000-8000-0000000000a1",` +
		`"visited_at":"2026-11-20T10:00:00.123456Z","project":{"id":"0199a2b4-0000-7000-8000-0000000000a1","name":"Web","identifier":"WEB",` +
		`"logo_props":{}},"issue":null}`
	itemOfNoneJSON = `{"id":"0199a2b4-0000-7000-8000-0000000000c1","entity_type":"issue","entity_id":"0199a2b4-0000-7000-8000-0000000000b1",` +
		`"visited_at":"2026-11-20T12:00:00.123456Z","project":{"id":"0199a2b4-0000-7000-8000-0000000000a1","name":"Web","identifier":"WEB",` +
		`"logo_props":{"in_use":"emoji","emoji":{"value":"128640"}}},"issue":{"id":"0199a2b4-0000-7000-8000-0000000000b1","sequence_id":7,` +
		`"name":"Fix the login","state_id":"0199a2b4-0000-7000-8000-0000000000d1","priority":"none","assignee_ids":[]}}`
)

// POST goes to the use case for the caller with the body's kind and id as
// given, unchecked: a kind none of the two among them, which the use case
// refuses (422). Its workspace and its project are the target's, never the
// body's. A new visit and a revisit alike answer 204 without a body.
func TestRecordingARecentVisit(t *testing.T) {
	for _, tt := range []struct {
		body string
		kind domain.RecentVisitKind
		id   uuid.UUID
	}{
		{`{"entity_type":"issue","entity_id":"0199a2b4-0000-7000-8000-0000000000b1"}`, domain.RecentVisitIssue, itemID},
		{`{"entity_type":"project","entity_id":"0199a2b4-0000-7000-8000-0000000000a1"}`, domain.RecentVisitProject, webID},
		{`{"entity_id":"0199a2b4-0000-7000-8000-0000000000e1","entity_type":"view"}`, "view", boardID},
		{`{"entity_type":"","entity_id":"0199a2b4-0000-7000-8000-0000000000b1"}`, "", itemID},
	} {
		f := &recentFakes{}
		res, body := do(t, newServer(t, f.useCases()), request(http.MethodPost, recordPath, tt.body))
		if res.StatusCode != http.StatusNoContent || body != "" {
			t.Errorf("POST %s = %d %q, want 204 and no body", tt.body, res.StatusCode, body)
		}
		if want := []recentCall{{Caller: aliceID, Op: "recordRecentVisit", Kind: tt.kind, ID: tt.id}}; !reflect.DeepEqual(f.calls, want) {
			t.Errorf("POST %s: the use cases got %+v, want %+v alone", tt.body, f.calls, want)
		}
	}
}

// GET of one's recent visits goes to the use case for the caller and the
// path's workspace, of no kind when the query names none; the answer is 200
// with the visits in their order, each with every key: a work item's with
// the work item, its assignees [] for none, a project's with issue null, and
// each project's icon {} for none.
func TestListingTheRecentVisits(t *testing.T) {
	for _, tt := range []struct {
		list []app.ShownRecentVisit
		want string
	}{
		{[]app.ShownRecentVisit{itemVisit(), projectVisit()}, `{"data":[` + itemVisitJSON + `,` + projectVisitJSON + `]}`},
		{[]app.ShownRecentVisit{itemOfNone()}, `{"data":[` + itemOfNoneJSON + `]}`},
		{[]app.ShownRecentVisit{}, `{"data":[]}`},
		{nil, `{"data":[]}`},
	} {
		f := &recentFakes{list: tt.list}
		res, body := do(t, newServer(t, f.useCases()), request(http.MethodGet, recentPath, ""))
		if res.StatusCode != http.StatusOK || !sameJSON(t, body, tt.want) {
			t.Errorf("GET of %d = %d %s, want 200 %s", len(tt.list), res.StatusCode, body, tt.want)
		}
		if want := []recentCall{{Caller: aliceID, Op: "listRecentVisits", Slug: "acme"}}; !reflect.DeepEqual(f.calls, want) {
			t.Errorf("GET of %d: the use cases got %+v, want %+v alone", len(tt.list), f.calls, want)
		}
	}
}

// The query's entity_type goes to the use case as given, unchecked: a kind
// none of the two and an empty one among them, which the use case refuses
// (422); left out, no kind. A parameter the list does not take is passed
// over, as navigation's other collections do.
func TestTheRecentVisitsQueryGoesToTheListAsGiven(t *testing.T) {
	kind := func(k domain.RecentVisitKind) *domain.RecentVisitKind { return &k }
	for _, tt := range []struct {
		query string
		want  *domain.RecentVisitKind
	}{
		{"?entity_type=issue", kind(domain.RecentVisitIssue)},
		{"?entity_type=project", kind(domain.RecentVisitProject)},
		{"?entity_type=page", kind("page")},
		{"?entity_type=", kind("")},
		{"", nil},
		{"?kind=issue", nil},
	} {
		f := &recentFakes{}
		if res, body := do(t, newServer(t, f.useCases()), request(http.MethodGet, recentPath+tt.query, "")); res.StatusCode != http.StatusOK {
			t.Errorf("GET %s = %d %s, want 200", tt.query, res.StatusCode, body)
		}
		if want := []recentCall{{Caller: aliceID, Op: "listRecentVisits", Slug: "acme", Of: tt.want}}; !reflect.DeepEqual(f.calls, want) {
			t.Errorf("GET %s: the use cases got %+v, want %+v alone", tt.query, f.calls, want)
		}
	}
}

// recentRefusals are every refusal the recent visits' operations declare,
// by name, and a failure: the views' (refusals) and their own.
func recentRefusals(t *testing.T) map[string]refusal {
	t.Helper()
	all := refusals(t)
	all["invalid kind"] = refusal{domain.CheckRecentVisitKind("page"), http.StatusUnprocessableEntity,
		`{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
			`"errors":[{"field":"entity_type","code":"invalid_format","message":"must be one of issue, project"}]}`}
	all["no issue"] = refusal{domain.ErrIssueNotFound, http.StatusNotFound,
		`{"status":404,"code":"issue.not_found","title":"Not Found","detail":"The work item does not exist, or you cannot see it."}`}
	return all
}

// Each recent visits' operation answers each refusal its x-problem-codes
// declare, and a failure, as the use case gives it: the problem, its status
// and its code. A target the caller may not record is its own 404, never
// 403, and never the workspace's: the recording names no workspace.
func TestTheRefusalsOfEachRecentVisitOperation(t *testing.T) {
	all := recentRefusals(t)
	for _, op := range []struct {
		method, path, body string
		refusals           []string
	}{
		{http.MethodPost, recordPath, `{"entity_type":"issue","entity_id":"0199a2b4-0000-7000-8000-0000000000b1"}`,
			[]string{"invalid kind", "no issue", "no project", "failure"}},
		{http.MethodGet, recentPath, "", []string{"invalid kind", "no workspace", "failure"}},
	} {
		for _, name := range op.refusals {
			tt := all[name]
			f := &recentFakes{err: tt.err}
			res, body := do(t, newServer(t, f.useCases()), request(op.method, op.path, op.body))
			if res.StatusCode != tt.status || body != tt.want+"\n" {
				t.Errorf("%s %s, %s: %d %s, want %d %s", op.method, op.path, name, res.StatusCode, body, tt.status, tt.want)
			}
			if len(f.calls) != 1 {
				t.Errorf("%s %s, %s: the use cases got %+v, want one call", op.method, op.path, name, f.calls)
			}
		}
	}
}

// A body that is not the recording's structure answers 400 bad_request
// before any use case: not an object, either field left out, null or a
// value of another type for either, an id that is no UUID, and a field it
// does not have (the workspace and the project, which the target gives,
// and Plane's entity_name and entity_identifier). Which kinds there are is
// the use case's (422).
func TestARecentVisitsBodyHeldToItsStructure(t *testing.T) {
	const item = `"0199a2b4-0000-7000-8000-0000000000b1"`
	f := &recentFakes{}
	h := newServer(t, f.useCases())
	for _, body := range []string{`[]`, `"issue"`, `null`, `{}`, `{"entity_type":"issue"}`, `{"entity_id":` + item + `}`,
		`{"entity_type":null,"entity_id":` + item + `}`, `{"entity_type":1,"entity_id":` + item + `}`, `{"entity_type":["issue"],"entity_id":` + item + `}`,
		`{"entity_type":"issue","entity_id":null}`, `{"entity_type":"issue","entity_id":"web"}`, `{"entity_type":"issue","entity_id":7}`,
		`{"entity_type":"issue","entity_id":` + item + `,"project_id":` + item + `}`, `{"entity_type":"issue","entity_id":` + item + `,"slug":"acme"}`,
		`{"entity_name":"issue","entity_id":` + item + `}`, `{"entity_type":"issue","entity_identifier":` + item + `}`} {
		if res, got := do(t, h, request(http.MethodPost, recordPath, body)); res.StatusCode != http.StatusBadRequest || problemCode(t, got) != "bad_request" {
			t.Errorf("POST %s = %d %s, want 400 bad_request", body, res.StatusCode, got)
		}
	}
	if len(f.calls) != 0 {
		t.Errorf("the use cases got %+v, want none", f.calls)
	}
}
