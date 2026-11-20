package httpadapter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"reflect"
	"testing"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeWorkspaceList is listWorkspaceIssues: it records the caller, the slug
// and the raw query it got, and refuses that query as the use case does.
type fakeWorkspaceList struct {
	actor uuid.UUID
	slug  string
	raw   map[string][]string
	page  app.IssueListPage
	err   error
}

func (f *fakeWorkspaceList) Execute(ctx context.Context, slug string, raw map[string][]string) (app.IssueListPage, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return app.IssueListPage{}, err
	}
	f.actor, f.slug, f.raw = actor.UserID, slug, raw
	if _, err := domain.ParseListQuery(raw, domain.WorkspaceList, actor.UserID); err != nil {
		return app.IssueListPage{}, err
	}
	return f.page, f.err
}

const workspaceListPath = "/api/v0/workspaces/acme/issues"

// The answer is always the flat page (M7 P3a spec 2.3): total, data and
// next_cursor, data never null; a cursor past the end keeps the total.
func TestWorkspaceIssueListEnvelope(t *testing.T) {
	row := []domain.Issue{webSeven, webEight}
	for _, tt := range []struct {
		name string
		page app.IssueListPage
		want string
	}{
		{"empty", app.IssueListPage{}, `{"total":0,"data":[],"next_cursor":null}`},
		{"past the end", app.IssueListPage{Total: 2}, `{"total":2,"data":[],"next_cursor":null}`},
		{"rows", app.IssueListPage{Total: 2, Data: row, NextCursor: "next"}, `{"total":2,"data":[` + webSevenJSON + `,` + webEightJSON + `],"next_cursor":"next"}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeWorkspaceList{page: tt.page}
			res, body := do(t, newServer(t, httpadapter.UseCases{ListWorkspaceIssues: f}), request(http.MethodGet, workspaceListPath, "alice", ""))
			var got, want any
			if json.Unmarshal([]byte(body), &got) != nil || json.Unmarshal([]byte(tt.want), &want) != nil || !reflect.DeepEqual(got, want) || res.StatusCode != 200 || f.actor != aliceID || f.slug != "acme" {
				t.Fatal(res.StatusCode, body, tt.want)
			}
		})
	}
}

// Each declared single-valued word given twice is 422 at binding, before
// the token is checked; every other word reaches the use case's parser
// whole, so a word the list does not take is 422 there, never dropped.
func TestWorkspaceIssueListQueryBoundary(t *testing.T) {
	check := func(t *testing.T, query, token string, status int, field, code string) {
		t.Helper()
		f := &fakeWorkspaceList{}
		req := request(http.MethodGet, workspaceListPath+"?"+query, token, "")
		res, body := do(t, newServer(t, httpadapter.UseCases{ListWorkspaceIssues: f}), req)
		var p struct {
			Code   string
			Errors []shared.FieldError
		}
		_ = json.Unmarshal([]byte(body), &p)
		if res.StatusCode != status {
			t.Fatal(res.StatusCode, body)
		}
		if field != "" && (len(p.Errors) != 1 || p.Errors[0].Field != field || p.Errors[0].Code != code) {
			t.Fatal(body)
		}
	}
	singles := map[string]string{"project_id": webID.String(), "state_id": stateID.String(), "state_group": "started", "priority": "high",
		"label_id": bugID.String(), "assignee_id": "me", "created_by_id": "me", "mention_id": "me", "subscriber_id": "me",
		"sub_issue": "true", "order_by": "created_at", "limit": "1", "cursor": "bad"}
	for field, value := range singles {
		t.Run("duplicate/"+field, func(t *testing.T) {
			query := field + "=" + url.QueryEscape(value) + "&" + field + "=" + url.QueryEscape(value)
			check(t, query, "alice", 422, field, "not_allowed")
			check(t, query, "", 422, field, "not_allowed")
		})
	}
	for _, tt := range []struct {
		query, field, code string
		status             int
	}{
		{"group_by=priority", "group_by", "not_allowed", 422}, {"group_by=a&group_by=b", "group_by", "not_allowed", 422},
		{"sub_group_by=priority", "sub_group_by", "not_allowed", 422}, {"group_key=high", "group_key", "not_allowed", 422},
		{"sub_group_key=high", "sub_group_key", "not_allowed", 422}, {"cycle_id=none", "cycle_id", "not_allowed", 422},
		{"module_id=none", "module_id", "not_allowed", 422}, {"archived=false", "archived", "not_allowed", 422},
		{"relation=assigned", "relation", "not_allowed", 422}, {"unknown=x", "unknown", "not_allowed", 422},
		{"project_id=wrong", "project_id", "not_allowed", 422}, {"start_date=2026-02-30", "start_date", "not_allowed", 422},
		{"sub_issue=1", "sub_issue", "not_allowed", 422}, {"order_by=sequence_id", "order_by", "not_allowed", 422},
		{"limit=no", "limit", "invalid_format", 400}, {"limit=0", "limit", "out_of_range", 422}, {"limit=101", "limit", "out_of_range", 422},
		{"cursor=bad", "cursor", "invalid_format", 400},
	} {
		t.Run(tt.query, func(t *testing.T) {
			check(t, tt.query, "alice", tt.status, tt.field, tt.code)
		})
	}
	check(t, "assignee_id=me", "", 401, "", "")

	raw := "assignee_id=me,none&created_by_id=me&subscriber_id=me&mention_id=" + aliceID.String() + "&project_id=" + webID.String() + "," + acmeID.String() +
		"&state_group=started&start_date=2026-10-01..&start_date=..2026-10-20&target_date=none&order_by=-priority&limit=2"
	f := &fakeWorkspaceList{}
	res, body := do(t, newServer(t, httpadapter.UseCases{ListWorkspaceIssues: f}), request(http.MethodGet, workspaceListPath+"?"+raw, "alice", ""))
	parsed, _ := url.ParseQuery(raw)
	if res.StatusCode != 200 || !reflect.DeepEqual(f.raw, map[string][]string(parsed)) {
		t.Fatal(res.StatusCode, body, f.raw)
	}
}

// The codes the operation declares (spec 2.3): validation_failed and
// workspace.not_found, a workspace not there or not seen alike; no 403.
func TestListWorkspaceIssuesHTTPRefusals(t *testing.T) {
	for _, tt := range []struct {
		err    error
		status int
	}{{domain.ErrWorkspaceNotFound, 404}, {shared.Invalid(shared.FieldError{Field: "priority", Code: "not_allowed", Message: "is not allowed for this list"}), 422}, {errGone, 500}} {
		f := &fakeWorkspaceList{err: tt.err}
		res, body := do(t, newServer(t, httpadapter.UseCases{ListWorkspaceIssues: f}), request(http.MethodGet, workspaceListPath, "alice", ""))
		if res.StatusCode != tt.status {
			t.Fatal(res.StatusCode, body)
		}
	}
}
