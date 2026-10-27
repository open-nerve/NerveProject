package domain

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// These cases catch loss of key presence, scalar multiplicity, Unicode
// boundaries, indexed date errors, and aggregation of independent failures.
func TestSearchQuery(t *testing.T) {
	projectID := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	issueID := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	field := func(path, code, message string) []shared.FieldError {
		return []shared.FieldError{{Field: path, Code: code, Message: message}}
	}
	type queryCase struct {
		name   string
		raw    map[string][]string
		want   SearchQuery
		fields []shared.FieldError
		bad    bool
	}
	cases := []queryCase{
		{name: "omitted", want: SearchQuery{Limit: 50}},
		{name: "empty", raw: map[string][]string{"q": {""}}, want: SearchQuery{Limit: 50}},
		{name: "whitespace", raw: map[string][]string{"q": {" \t\n\u2003"}}, want: SearchQuery{Limit: 50}},
		{name: "trim", raw: map[string][]string{"q": {"\u2003 title \n"}}, want: SearchQuery{Text: "title", Limit: 50}},
		{name: "100_unicode", raw: map[string][]string{"q": {strings.Repeat("中", 100)}}, want: SearchQuery{Text: strings.Repeat("中", 100), Limit: 50}},
		{name: "101_unicode", raw: map[string][]string{"q": {strings.Repeat("中", 101)}}, fields: field("q", "too_long", "must be at most 100 characters")},
		{name: "nul", raw: map[string][]string{"q": {"a\x00b"}}, fields: field("q", "invalid_format", "must not contain a NUL character")},
		{name: "invalid_utf8", raw: map[string][]string{"q": {"a\xffb"}}, fields: field("q", "invalid_format", "must be valid UTF-8")},
		{name: "limit_min", raw: map[string][]string{"limit": {"1"}}, want: SearchQuery{Limit: 1}},
		{name: "limit_max", raw: map[string][]string{"limit": {"100"}}, want: SearchQuery{Limit: 100}},
		{name: "limit_zero", raw: map[string][]string{"limit": {"0"}}, fields: field("limit", "out_of_range", "must be between 1 and 100")},
		{name: "limit_large", raw: map[string][]string{"limit": {"101"}}, fields: field("limit", "out_of_range", "must be between 1 and 100")},
		{name: "limit_text", raw: map[string][]string{"limit": {"x"}}, bad: true, fields: field("limit", "invalid_format", "must have the required parameter format")},
		{name: "limit_empty", raw: map[string][]string{"limit": {""}}, bad: true, fields: field("limit", "invalid_format", "must have the required parameter format")},
		{name: "unknown", raw: map[string][]string{"cursor": {"anything"}}, fields: field("cursor", "not_allowed", "is not allowed for this search")},
		{name: "empty_for", raw: map[string][]string{"for": {""}}, fields: field("for", "not_allowed", "must be one of parent, sub_issue, relation")},
		{name: "unknown_for", raw: map[string][]string{"for": {"children"}}, fields: field("for", "not_allowed", "must be one of parent, sub_issue, relation")},
		{name: "for_requires_source", raw: map[string][]string{"for": {"parent"}}, fields: field("issue_id", "required", "is required when for is provided")},
		{name: "source_requires_for", raw: map[string][]string{"issue_id": {issueID.String()}}, fields: field("issue_id", "not_allowed", "requires for")},
		{name: "empty_source", raw: map[string][]string{"for": {"parent"}, "issue_id": {""}}, fields: field("issue_id", "invalid_format", "must be a UUID")},
		{name: "empty_project", raw: map[string][]string{"project_id": {""}}, fields: field("project_id", "invalid_format", "must be a UUID")},
		{name: "bad_source", raw: map[string][]string{"for": {"parent"}, "issue_id": {"none"}}, fields: field("issue_id", "invalid_format", "must be a UUID")},
		{name: "bad_project", raw: map[string][]string{"project_id": {"me"}}, fields: field("project_id", "invalid_format", "must be a UUID")},
		{name: "project_only", raw: map[string][]string{"project_id": {projectID.String()}}, want: SearchQuery{ProjectID: &projectID, Limit: 50}},
		{name: "no_values", raw: map[string][]string{"q": nil}, fields: field("q", "invalid_format", "must be given once")},
		{name: "date_no_values", raw: map[string][]string{"start_date": nil}, fields: field("start_date", "invalid_format", "must have at least one condition")},
		{name: "aggregate", raw: map[string][]string{"zz": {"x"}, "q": {strings.Repeat("中", 101)}, "project_id": {"bad"}, "limit": {"0"}}, fields: []shared.FieldError{
			{Field: "limit", Code: "out_of_range", Message: "must be between 1 and 100"},
			{Field: "project_id", Code: "invalid_format", Message: "must be a UUID"},
			{Field: "q", Code: "too_long", Message: "must be at most 100 characters"},
			{Field: "zz", Code: "not_allowed", Message: "is not allowed for this search"},
		}},
	}
	cases = append(cases,
		queryCase{name: "duplicate_q", raw: map[string][]string{"q": {"first", "second"}}, fields: field("q", "invalid_format", "must be given once")},
		queryCase{name: "duplicate_for", raw: map[string][]string{"for": {"parent", "relation"}, "issue_id": {issueID.String()}}, fields: field("for", "invalid_format", "must be given once")},
		queryCase{name: "duplicate_project_id", raw: map[string][]string{"project_id": {projectID.String(), issueID.String()}}, fields: field("project_id", "invalid_format", "must be given once")},
		queryCase{name: "duplicate_issue_id", raw: map[string][]string{"for": {"parent"}, "issue_id": {projectID.String(), issueID.String()}}, fields: field("issue_id", "invalid_format", "must be given once")},
		queryCase{name: "duplicate_limit", raw: map[string][]string{"limit": {"1", "100"}}, fields: field("limit", "invalid_format", "must be given once")},
	)
	for _, mode := range []SearchFor{"parent", "sub_issue", "relation"} {
		cases = append(cases, queryCase{name: string(mode), raw: map[string][]string{"for": {string(mode)}, "issue_id": {issueID.String()}, "project_id": {projectID.String()}}, want: SearchQuery{For: mode, IssueID: &issueID, ProjectID: &projectID, Limit: 50}})
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseSearchQuery(tc.raw)
			if tc.fields == nil {
				if err != nil || !reflect.DeepEqual(got, tc.want) {
					t.Fatalf("ParseSearchQuery = %+v, %v; want %+v", got, err, tc.want)
				}
				return
			}
			assertSearchError(t, err, tc.bad, tc.fields)
		})
	}
	t.Run("dates", testSearchDates)
}

func assertSearchError(t *testing.T, err error, bad bool, fields []shared.FieldError) {
	t.Helper()
	var problem *shared.Error
	if !errors.As(err, &problem) {
		t.Fatalf("error = %v; want shared.Error", err)
	}
	kind, code, detail := shared.KindInvalid, "validation_failed", "The request has invalid values."
	if bad {
		kind, code, detail = shared.KindBadRequest, "bad_request", "The search parameter is malformed."
	}
	if problem.Kind != kind || problem.Code != code || problem.Detail != detail || !reflect.DeepEqual(problem.Fields, fields) {
		t.Fatalf("problem = %+v; want kind=%v code=%s detail=%q fields=%+v", problem, kind, code, detail, fields)
	}
}

func testSearchDates(t *testing.T) {
	for _, key := range []string{"start_date", "target_date"} {
		t.Run(key, func(t *testing.T) {
			for _, tc := range []struct {
				name          string
				values        []string
				index         int
				code, message string
			}{
				{"none_date", []string{"none", "2026-10-27"}, 0, "not_allowed", "must not be combined with another date condition"},
				{"date_none", []string{"2026-10-27", "none"}, 1, "not_allowed", "must not be combined with another date condition"},
				{"none_none", []string{"none", "none"}, 0, "not_allowed", "must not be combined with another date condition"},
				{"bad_before_none", []string{"bad", "none"}, 1, "not_allowed", "must not be combined with another date condition"},
				{"bad_date", []string{"2026-10-27", "2026-02-30"}, 1, "invalid_format", "must be a date, date range, or none"},
				{"empty_date", []string{""}, 0, "invalid_format", "must be a date, date range, or none"},
				{"reversed", []string{"2026-10-28..2026-10-27"}, 0, "invalid_format", "must be a date, date range, or none"},
			} {
				t.Run(tc.name, func(t *testing.T) {
					_, err := ParseSearchQuery(map[string][]string{key: tc.values})
					path := key + "[0]"
					if tc.index == 1 {
						path = key + "[1]"
					}
					assertSearchError(t, err, false, []shared.FieldError{{Field: path, Code: tc.code, Message: tc.message}})
				})
			}
			for _, tc := range []struct {
				name   string
				values []string
				bounds [][2]string
				none   bool
			}{
				{"none", []string{"none"}, [][2]string{{"", ""}}, true},
				{"day", []string{"2026-10-27"}, [][2]string{{"2026-10-27", "2026-10-27"}}, false},
				{"and", []string{"2026-10-01..2026-10-31", "2026-10-10..", "..2026-10-20"}, [][2]string{{"2026-10-01", "2026-10-31"}, {"2026-10-10", ""}, {"", "2026-10-20"}}, false},
			} {
				t.Run(tc.name, func(t *testing.T) {
					got, err := ParseSearchQuery(map[string][]string{key: tc.values})
					if err != nil {
						t.Fatal(err)
					}
					filters := got.StartDate
					if key == "target_date" {
						filters = got.TargetDate
					}
					if len(filters) != len(tc.bounds) {
						t.Fatalf("filters = %+v; want %d conditions", filters, len(tc.bounds))
					}
					for i, filter := range filters {
						from, to := "", ""
						if filter.From != nil {
							from = filter.From.String()
						}
						if filter.To != nil {
							to = filter.To.String()
						}
						if filter.None != tc.none || [2]string{from, to} != tc.bounds[i] {
							t.Fatalf("condition %d = %+v (%s..%s); want none=%v bounds=%v", i, filter, from, to, tc.none, tc.bounds[i])
						}
					}
				})
			}
		})
	}
}
