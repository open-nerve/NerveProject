package httpadapter_test

import (
	"context"
	"encoding/json"
	"net/url"
	"reflect"
	"testing"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type fakeSearch struct {
	raw  map[string][]string
	slug string
	rows []domain.Issue
	err  error
}

func (f *fakeSearch) Execute(_ context.Context, slug string, raw map[string][]string) ([]domain.Issue, error) {
	f.raw, f.slug = raw, slug
	if _, err := domain.ParseSearchQuery(raw); err != nil {
		return nil, err
	}
	return f.rows, f.err
}

const searchPath = "/api/v0/workspaces/acme/issue-search"

// The complete collection response must not acquire paging or detail fields.
func TestSearchIssuesHTTPCollection(t *testing.T) {
	for _, tc := range []struct {
		name string
		rows []domain.Issue
		want string
	}{
		{"empty", nil, `{"data":[]}`}, {"rows", []domain.Issue{webSeven, webEight}, `{"data":[` + webSevenJSON + `,` + webEightJSON + `]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeSearch{rows: tc.rows}
			raw := "q=Fix&start_date=2026-01-01..&start_date=..2026-02-01&target_date=none"
			res, body := do(t, newServer(t, httpadapter.UseCases{SearchIssues: f}), request("GET", searchPath+"?"+raw, "alice", ""))
			parsed, _ := url.ParseQuery(raw)
			if res.StatusCode != 200 || body != tc.want+"\n" || f.slug != "acme" || !reflect.DeepEqual(f.raw, map[string][]string(parsed)) {
				t.Fatal(res.StatusCode, body, f.raw)
			}
		})
	}
}

// Every declared own problem code crosses the generated binding/strict chain;
// complete objects also prevent detail leaks and accidental error remapping.
func TestSearchIssuesHTTPRefusals(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		status int
		body   string
	}{
		{"workspace", domain.ErrWorkspaceNotFound, 404, `{"status":404,"code":"workspace.not_found","title":"Not Found","detail":"The workspace does not exist, or you are not a member of it."}`},
		{"project", domain.ErrProjectNotFound, 404, `{"status":404,"code":"project.not_found","title":"Not Found","detail":"The project does not exist, or you cannot see it."}`},
		{"source", domain.ErrIssueNotFound, 404, `{"status":404,"code":"issue.not_found","title":"Not Found","detail":"The work item does not exist, or you cannot see it."}`},
		{"forbidden", shared.Forbidden(), 403, `{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
		{"invalid", shared.Invalid(shared.FieldError{Field: "project_id", Code: "not_allowed", Message: "must match the source issue's project"}), 422, `{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.","errors":[{"field":"project_id","code":"not_allowed","message":"must match the source issue's project"}]}`},
		{"internal", errGone, 500, `{"status":500,"code":"internal_error","title":"Internal Server Error"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res, body := do(t, newServer(t, httpadapter.UseCases{SearchIssues: &fakeSearch{err: tc.err}}), request("GET", searchPath, "alice", ""))
			if res.StatusCode != tc.status || body != tc.body+"\n" {
				t.Fatal(res.StatusCode, body, tc.body)
			}
		})
	}
	f := &fakeSearch{}
	res, body := do(t, newServer(t, httpadapter.UseCases{SearchIssues: f}), request("GET", searchPath, "", ""))
	if res.StatusCode != 401 || f.raw != nil {
		t.Fatal(res.StatusCode, body)
	}
}

func TestSearchIssuesHTTPQueryBoundary(t *testing.T) {
	for _, tc := range []struct {
		query       string
		status      int
		field, code string
	}{
		{"q=a&q=b", 422, "q", "invalid_format"}, {"q=a&q=b&bad=%GG", 400, "", ""},
		{"limit=1&limit=2", 422, "limit", "invalid_format"}, {"limit=1&limit=2&bad=%GG", 400, "", ""},
		{"limit=x", 400, "limit", "invalid_format"}, {"project_id=not-a-uuid", 422, "project_id", "invalid_format"},
		{"start_date=2026-02-30", 422, "start_date[0]", "invalid_format"}, {"cursor=abc", 422, "cursor", "not_allowed"},
	} {
		t.Run(tc.query, func(t *testing.T) {
			res, body := do(t, newServer(t, httpadapter.UseCases{SearchIssues: &fakeSearch{}}), request("GET", searchPath+"?"+tc.query, "alice", ""))
			if res.StatusCode != tc.status {
				t.Fatal(res.StatusCode, body)
			}
			if tc.field != "" {
				var p struct{ Errors []shared.FieldError }
				if json.Unmarshal([]byte(body), &p) != nil || len(p.Errors) != 1 || p.Errors[0].Field != tc.field || p.Errors[0].Code != tc.code {
					t.Fatal(body)
				}
			}
		})
	}
}
