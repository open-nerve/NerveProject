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

type fakeMemberList struct {
	actor, id uuid.UUID
	slug      string
	raw       map[string][]string
	page      app.IssueListPage
	err       error
}

func (f *fakeMemberList) Execute(ctx context.Context, slug string, id uuid.UUID, raw map[string][]string) (app.IssueListPage, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return app.IssueListPage{}, err
	}
	f.actor, f.id, f.raw, f.slug = actor.UserID, id, raw, slug
	if _, err := domain.ParseListQuery(raw, domain.MemberList, actor.UserID); err != nil {
		return app.IssueListPage{}, err
	}
	return f.page, f.err
}

var memberListPath = "/api/v0/workspaces/acme/members/" + webID.String() + "/issues"

func TestMemberIssueListEnvelope(t *testing.T) {
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
		{"group continuation", "?group_by=priority&group_key=high", app.IssueListPage{}, `{"total":0,"data":[],"next_cursor":null}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeMemberList{page: tt.page}
			res, body := do(t, newServer(t, httpadapter.UseCases{ListMemberIssues: f}), request(http.MethodGet, memberListPath+"?relation=assigned&"+strings.TrimPrefix(tt.query, "?"), "alice", ""))
			var got, want any
			if json.Unmarshal([]byte(body), &got) != nil || json.Unmarshal([]byte(tt.want), &want) != nil || !reflect.DeepEqual(got, want) || res.StatusCode != 200 || f.actor != aliceID || f.id != webID || f.slug != "acme" {
				t.Fatal(res.StatusCode, body, tt.want)
			}
		})
	}
}
func TestMemberIssueListQueryBoundary(t *testing.T) {
	values := map[string]string{"state_group": "started", "priority": "high", "label_id": bugID.String(), "project_id": webID.String(), "sub_issue": "true", "group_by": "priority", "order_by": "created_at", "group_key": "high", "limit": "1", "cursor": "bad", "relation": "assigned"}
	check := func(t *testing.T, path, token string, status int, field, code string) {
		t.Helper()
		f := &fakeMemberList{}
		req := request(http.MethodGet, path, token, "")
		before := req.URL.RawQuery
		res, body := do(t, newServer(t, httpadapter.UseCases{ListMemberIssues: f}), req)
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
			query := field + "=" + url.QueryEscape(value) + "&" + field + "=" + url.QueryEscape(value)
			if field != "relation" {
				query = "relation=assigned&" + query
			}
			check(t, memberListPath+"?"+query, "alice", 422, field, "not_allowed")
		})
	}
	for _, tt := range []struct {
		query, field, code string
		status             int
	}{
		{"unknown=x", "unknown", "not_allowed", 422}, {"project_id=wrong", "project_id", "not_allowed", 422}, {"start_date=2026-02-30", "start_date", "not_allowed", 422}, {"sub_issue=1", "sub_issue", "not_allowed", 422}, {"group_by=assignee_id", "group_by", "not_allowed", 422}, {"order_by=link_count", "order_by", "not_allowed", 422}, {"limit=no", "limit", "invalid_format", 400}, {"limit=0", "limit", "out_of_range", 422}, {"limit=101", "limit", "out_of_range", 422}, {"cursor=bad", "cursor", "invalid_format", 400},
		{"state_id=x", "state_id", "not_allowed", 422}, {"assignee_id=me", "assignee_id", "not_allowed", 422}, {"created_by_id=me", "created_by_id", "not_allowed", 422}, {"mention_id=me", "mention_id", "not_allowed", 422}, {"subscriber_id=me", "subscriber_id", "not_allowed", 422}, {"archived=false", "archived", "not_allowed", 422}, {"sub_group_by=priority", "sub_group_by", "not_allowed", 422}, {"sub_group_key=high", "sub_group_key", "not_allowed", 422}, {"target_member_id=x", "target_member_id", "not_allowed", 422}, {"user_id=x", "user_id", "not_allowed", 422},
		{"group_key=high", "group_key", "not_allowed", 422},
	} {
		t.Run(tt.query, func(t *testing.T) {
			check(t, memberListPath+"?relation=assigned&"+tt.query, "alice", tt.status, tt.field, tt.code)
		})
	}
	check(t, memberListPath+"?relation=assigned&limit=1&limit=2", "", 422, "limit", "not_allowed")
	check(t, "/api/v0/workspaces/acme/members/bad/issues?relation=assigned&limit=1&limit=2", "", 400, "user_id", "invalid_format")
	check(t, "/api/v0/workspaces/acme/members/bad/issues?relation=assigned&user_id=a&user_id=b", "alice", 400, "user_id", "invalid_format")
	check(t, memberListPath+"?relation=assigned", "", 401, "", "")
	check(t, memberListPath, "alice", 400, "relation", "required")
	check(t, memberListPath+"?relation=other", "alice", 422, "relation", "not_allowed")

	raw := "relation=assigned&project_id=" + stateID.String() + "," + webID.String() + "&start_date=2026-10-01..&start_date=..2026-10-20&target_date=2026-10-12&target_date=2026-10-12"
	f := &fakeMemberList{}
	res, body := do(t, newServer(t, httpadapter.UseCases{ListMemberIssues: f}), request(http.MethodGet, memberListPath+"?"+raw, "alice", ""))
	parsed, _ := url.ParseQuery(raw)
	if res.StatusCode != 200 || !reflect.DeepEqual(f.raw, map[string][]string(parsed)) {
		t.Fatal(res.StatusCode, body, f.raw)
	}
}
func TestListMemberIssuesHTTPRefusals(t *testing.T) {
	for _, tt := range []struct {
		err    error
		status int
	}{{domain.ErrWorkspaceNotFound, 404}, {domain.ErrMemberNotFound, 404}, {shared.Forbidden(), 403}, {shared.Invalid(shared.FieldError{Field: "priority", Code: "not_allowed", Message: "is not allowed for this list"}), 422}, {errGone, 500}} {
		f := &fakeMemberList{err: tt.err}
		res, body := do(t, newServer(t, httpadapter.UseCases{ListMemberIssues: f}), request(http.MethodGet, memberListPath+"?relation=assigned", "alice", ""))
		if res.StatusCode != tt.status {
			t.Fatal(res.StatusCode, body)
		}
	}
}
