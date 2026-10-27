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

type fakeProjectList struct {
	actor, id uuid.UUID
	raw       map[string][]string
	page      app.IssueListPage
	err       error
}

func (f *fakeProjectList) Execute(ctx context.Context, id uuid.UUID, raw map[string][]string) (app.IssueListPage, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return app.IssueListPage{}, err
	}
	f.actor, f.id, f.raw = actor.UserID, id, raw
	if _, err := domain.ParseListQuery(raw, domain.ProjectList, actor.UserID); err != nil {
		return app.IssueListPage{}, err
	}
	return f.page, f.err
}

var projectListPath = "/api/v0/projects/" + webID.String() + "/issues"

func TestIssueListEnvelope(t *testing.T) {
	row := []domain.Issue{webSeven, webEight}
	for _, tt := range []struct {
		name, query string
		page        app.IssueListPage
		want        string
	}{
		{"flat empty", "", app.IssueListPage{}, `{"total":0,"data":[],"next_cursor":null}`},
		{"flat rows", "", app.IssueListPage{Total: 2, Data: row, NextCursor: "next"}, `{"total":2,"data":[` + webSevenJSON + `,` + webEightJSON + `],"next_cursor":"next"}`},
		{"groups empty", "?group_by=priority", app.IssueListPage{}, `{"total":0,"groups":[]}`},
		{"groups", "?group_by=priority", app.IssueListPage{Total: 2, Groups: []app.IssueListGroup{{Key: "high", Total: 2, Data: row}}}, `{"total":2,"groups":[{"key":"high","total":2,"data":[` + webSevenJSON + `,` + webEightJSON + `],"next_cursor":null}]}`},
		{"lanes empty", "?group_by=priority&sub_group_by=label_id", app.IssueListPage{}, `{"total":0,"groups":[],"subgroup_totals":[]}`},
		{"lanes", "?group_by=priority&sub_group_by=label_id", app.IssueListPage{Total: 2, Groups: []app.IssueListGroup{{Key: "high", Total: 2, Subgroups: []app.IssueListGroup{{Key: "none", Total: 2, Data: row, NextCursor: "next"}}}}, SubgroupTotals: []app.ListTotal{{Key: "none", Total: 2}}}, `{"total":2,"groups":[{"key":"high","total":2,"subgroups":[{"key":"none","total":2,"data":[` + webSevenJSON + `,` + webEightJSON + `],"next_cursor":"next"}]}],"subgroup_totals":[{"key":"none","total":2}]}`},
		{"group continuation", "?group_by=priority&group_key=high", app.IssueListPage{}, `{"total":0,"data":[],"next_cursor":null}`},
		{"cell continuation", "?group_by=priority&sub_group_by=label_id&group_key=high&sub_group_key=none", app.IssueListPage{Total: 2, Data: row}, `{"total":2,"data":[` + webSevenJSON + `,` + webEightJSON + `],"next_cursor":null}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeProjectList{page: tt.page}
			res, body := do(t, newServer(t, httpadapter.UseCases{ListIssues: f}), request(http.MethodGet, projectListPath+tt.query, "alice", ""))
			var got, want any
			if json.Unmarshal([]byte(body), &got) != nil || json.Unmarshal([]byte(tt.want), &want) != nil || !reflect.DeepEqual(got, want) || res.StatusCode != 200 || f.actor != aliceID || f.id != webID {
				t.Fatal(res.StatusCode, body, tt.want)
			}
		})
	}
}
func TestIssueListQueryBoundary(t *testing.T) {
	values := map[string]string{"state_id": stateID.String(), "state_group": "started", "priority": "high", "label_id": bugID.String(), "assignee_id": "me", "created_by_id": "me", "mention_id": "me", "subscriber_id": "me", "sub_issue": "true", "archived": "false", "group_by": "priority", "sub_group_by": "label_id", "order_by": "created_at", "group_key": "high", "sub_group_key": "none", "limit": "1", "cursor": "bad"}
	check := func(t *testing.T, path, token string, status int, field, code string) {
		t.Helper()
		f := &fakeProjectList{}
		req := request(http.MethodGet, path, token, "")
		before := req.URL.RawQuery
		res, body := do(t, newServer(t, httpadapter.UseCases{ListIssues: f}), req)
		var p struct {
			Code   string
			Errors []shared.FieldError
		}
		_ = json.Unmarshal([]byte(body), &p)
		if res.StatusCode != status || req.URL.RawQuery != before {
			t.Fatal(res.StatusCode, body, before, req.URL.RawQuery)
		}
		if field != "" && (len(p.Errors) != 1 || p.Errors[0].Field != field || p.Errors[0].Code != code) {
			t.Fatal(body)
		}
	}
	for field, value := range values {
		t.Run("duplicate/"+field, func(t *testing.T) {
			check(t, projectListPath+"?"+field+"="+url.QueryEscape(value)+"&"+field+"="+url.QueryEscape(value), "alice", 422, field, "not_allowed")
		})
	}
	for _, tt := range []struct {
		query, field, code string
		status             int
	}{
		{"unknown=x", "unknown", "not_allowed", 422}, {"project_id=" + webID.String(), "project_id", "not_allowed", 422}, {"state_id=wrong", "state_id", "not_allowed", 422}, {"start_date=2026-02-30", "start_date", "not_allowed", 422}, {"sub_issue=1", "sub_issue", "not_allowed", 422}, {"archived=TRUE", "archived", "not_allowed", 422}, {"group_by=cycle_id", "group_by", "not_allowed", 422}, {"order_by=unknown_count", "order_by", "not_allowed", 422}, {"limit=no", "limit", "invalid_format", 400}, {"limit=0", "limit", "out_of_range", 422}, {"limit=101", "limit", "out_of_range", 422}, {"cursor=bad", "cursor", "invalid_format", 400},
	} {
		t.Run(tt.query, func(t *testing.T) { check(t, projectListPath+"?"+tt.query, "alice", tt.status, tt.field, tt.code) })
	}
	check(t, projectListPath+"?limit=1&limit=2", "", 422, "limit", "not_allowed")
	check(t, "/api/v0/projects/bad/issues?limit=1&limit=2", "", 400, "project_id", "invalid_format")
	check(t, "/api/v0/projects/bad/issues?project_id=a&project_id=b", "alice", 400, "project_id", "invalid_format")
	check(t, projectListPath, "", 401, "", "")
	for _, field := range []string{"limit", "cursor"} {
		check(t, commentsPath+"?"+field+"=1&"+field+"=2", "alice", 400, field, "invalid_format")
	}
	raw := "state_id=" + stateID.String() + "," + webID.String() + "&start_date=2026-10-01..&start_date=..2026-10-20&target_date=2026-10-12&target_date=2026-10-12"
	f := &fakeProjectList{}
	res, body := do(t, newServer(t, httpadapter.UseCases{ListIssues: f}), request(http.MethodGet, projectListPath+"?"+raw, "alice", ""))
	parsed, _ := url.ParseQuery(raw)
	if res.StatusCode != 200 || !reflect.DeepEqual(f.raw, map[string][]string(parsed)) {
		t.Fatal(res.StatusCode, body, f.raw)
	}
}
func TestListIssuesHTTPRefusals(t *testing.T) {
	for _, tt := range []struct {
		err    error
		status int
	}{{domain.ErrProjectNotFound, 404}, {shared.Forbidden(), 403}, {shared.Invalid(shared.FieldError{Field: "priority", Code: "not_allowed", Message: "is not allowed for this list"}), 422}, {errGone, 500}} {
		f := &fakeProjectList{err: tt.err}
		res, body := do(t, newServer(t, httpadapter.UseCases{ListIssues: f}), request(http.MethodGet, projectListPath, "alice", ""))
		if res.StatusCode != tt.status {
			t.Fatal(res.StatusCode, body)
		}
	}
}
