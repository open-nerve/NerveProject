package httpadapter

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"slices"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type listQueryKey struct{}

func listQueryBindingError(api *httpserver.API) func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		if r.Method == http.MethodGet && (r.Pattern == draftListPattern || r.Pattern == searchIssuesPattern) {
			raw, parseErr := url.ParseQuery(r.URL.RawQuery)
			if parseErr != nil {
				api.Errors.BadRequest(w, r, parseErr)
				return
			}
			check := checkDraftQuery
			if r.Pattern == searchIssuesPattern {
				check = checkSearchScalars
			}
			if queryErr := check(raw); queryErr != nil {
				api.Errors.Write(w, r, queryErr)
				return
			}
		}
		var format *gen.InvalidParamFormatError
		if r.Method == http.MethodGet && errors.As(err, &format) && listSingleQuery(r.Pattern, format.ParamName) && len(r.URL.Query()[format.ParamName]) > 1 {
			api.Errors.Write(w, r, shared.Invalid(shared.FieldError{Field: format.ParamName, Code: shared.FieldNotAllowed, Message: "is not allowed for this list"}))
			return
		}
		api.Errors.BadRequest(w, r, err)
	}
}

// intakeListPattern is the route of a project's intake list (M7 P2b spec
// 2.3), and intakeListQuery its query's parameters, each given once, its
// days too (domain.ParseIntakeQuery).
const intakeListPattern = "GET /api/v0/projects/{project_id}/intake-issues"

var intakeListQuery = []string{"status", "priority", "label_id", "assignee_id", "created_by_id", "created_at", "updated_at", "order_by", "limit", "cursor"}

// Each route declares its own single-valued query vocabulary; path and date
// parameters must never be remapped just because a query uses the same name.
func listSingleQuery(pattern, name string) bool {
	if pattern == intakeListPattern {
		return slices.Contains(intakeListQuery, name)
	}
	project := pattern == "GET /api/v0/projects/{project_id}/issues"
	member := pattern == "GET /api/v0/workspaces/{slug}/members/{user_id}/issues"
	if !project && !member {
		return false
	}
	switch name {
	case "state_group", "priority", "label_id", "sub_issue", "group_by", "order_by", "group_key", "limit", "cursor":
		return true
	case "state_id", "assignee_id", "created_by_id", "mention_id", "subscriber_id", "cycle_id", "module_id", "archived", "sub_group_by", "sub_group_key":
		return project
	case "project_id", "relation":
		return member
	}
	return false
}

func withListQuery(next gen.StrictHandlerFunc, operationID string) gen.StrictHandlerFunc {
	if !slices.Contains([]string{"ListIssues", "ListMemberIssues", "ListDrafts", "SearchIssues", "ListIntakeIssues"}, operationID) {
		return next
	}
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request any) (any, error) {
		// URL.Query silently drops malformed pairs; a list must never lose a
		// filter or cursor and execute the remaining, broader query instead.
		raw, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			return nil, shared.NewError(shared.KindBadRequest, shared.CodeBadRequest, "The request query could not be decoded.")
		}
		return next(context.WithValue(ctx, listQueryKey{}, copyListQuery(raw)), w, r, request)
	}
}
func copyListQuery(raw map[string][]string) map[string][]string {
	copied := make(map[string][]string, len(raw))
	for k, v := range raw {
		copied[k] = slices.Clone(v)
	}
	return copied
}
func rawListQuery(ctx context.Context) map[string][]string {
	raw, _ := ctx.Value(listQueryKey{}).(map[string][]string)
	return raw
}
