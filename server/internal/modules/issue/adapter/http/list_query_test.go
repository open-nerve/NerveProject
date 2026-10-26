package httpadapter

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)

func TestListQueryCopy(t *testing.T) {
	raw := map[string][]string{"start_date": {"a", "b"}, "unknown": {"kept"}}
	copied := copyListQuery(raw)
	copied["start_date"][0] = "changed"
	delete(copied, "unknown")
	if raw["start_date"][0] != "a" || len(raw) != 2 {
		t.Fatal("copy aliases input", raw)
	}
	for _, op := range []string{"ListIssues", "ListMemberIssues", "ListIssueComments"} {
		r := httptest.NewRequest(http.MethodGet, "/?start_date=a&start_date=b&unknown=kept", nil)
		next := func(ctx context.Context, _ http.ResponseWriter, _ *http.Request, req any) (any, error) {
			got := rawListQuery(ctx)
			if op == "ListIssueComments" {
				if got != nil {
					t.Fatal(got)
				}
			} else if !reflect.DeepEqual(got, raw) {
				t.Fatal(got, raw)
			}
			return req, nil
		}
		if v, err := withListQuery(next, op)(context.Background(), httptest.NewRecorder(), r, "request"); err != nil || v != "request" {
			t.Fatal(v, err)
		}
	}
}
func TestListBindingErrorIsLocal(t *testing.T) {
	api := &httpserver.API{Errors: httpserver.NewAPIErrors(slog.New(slog.DiscardHandler))}
	project := "GET /api/v0/projects/{project_id}/issues"
	member := "GET /api/v0/workspaces/{slug}/members/{user_id}/issues"
	for _, tt := range []struct {
		name, pattern, param, query string
		err                         error
		want                        int
	}{
		{"project duplicate", project, "limit", "limit=1&limit=2", nil, 422},
		{"member duplicate", member, "project_id", "project_id=a&project_id=b", nil, 422},
		{"project query not declared", project, "project_id", "project_id=a&project_id=b", nil, 400},
		{"member query not declared", member, "archived", "archived=true&archived=false", nil, 400},
		{"date", project, "start_date", "start_date=a&start_date=b", nil, 400},
		{"single", project, "limit", "limit=x", nil, 400},
		{"other API", "GET /api/v0/issues/{issue_id}/comments", "limit", "limit=1&limit=2", nil, 400},
		{"similar route", "GET /api/v0/other/projects/{project_id}/issues", "limit", "limit=1&limit=2", nil, 400},
		{"wrong error", project, "limit", "limit=1&limit=2", &gen.RequiredParamError{ParamName: "limit"}, 400},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/?"+tt.query, nil)
			r.Pattern = tt.pattern
			rec := httptest.NewRecorder()
			err := tt.err
			if err == nil {
				err = fmt.Errorf("wrapped: %w", &gen.InvalidParamFormatError{ParamName: tt.param, Err: fmt.Errorf("bad")})
			}
			listQueryBindingError(api)(rec, r, err)
			if rec.Code != tt.want {
				t.Fatal(rec.Code, rec.Body.String())
			}
		})
	}
}
