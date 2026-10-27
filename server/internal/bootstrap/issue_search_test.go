package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func searchExecute(t *testing.T, w issueWorld, actor string, raw map[string][]string) []domain.Issue {
	t.Helper()
	u := snapshotSearchUseCase(w, newListProbe(w, ""))
	rows, err := u.Execute(shared.WithActor(context.Background(), shared.Actor{UserID: w.ids[actor]}), "acme", raw)
	if err != nil {
		t.Fatal(err)
	}
	return rows
}
func checkSearchIDs(t *testing.T, rows []domain.Issue, want ...uuid.UUID) {
	t.Helper()
	got := make([]uuid.UUID, len(rows))
	for n, row := range rows {
		got[n] = row.ID
	}
	if !slices.Equal(got, want) {
		t.Fatalf("IDs %v; want %v", got, want)
	}
}

// The entire chain must be traversed even when its middle is invisible;
// role widening, a one-level traversal or a one-direction edge leaks rows.
func TestSearchIssueScopeAndExclusions(t *testing.T) {
	t.Run("hierarchy", func(t *testing.T) {
		w := newIssueWorld(t)
		w.makeMember(t, "walt", w.web, shared.RoleMember)
		root := w.newWorkItemBy(t, "walt", w.web, `{"name":"Tree"}`)
		a := w.newWorkItemBy(t, "walt", w.web, `{"name":"Tree"}`)
		b := w.newWorkItemBy(t, "bob", w.web, `{"name":"Tree"}`)
		c := w.newWorkItemBy(t, "walt", w.web, `{"name":"Tree"}`)
		d := w.newWorkItemBy(t, "walt", w.web, `{"name":"Tree"}`)
		allowed := w.newWorkItemBy(t, "walt", w.web, `{"name":"Tree"}`)
		listSeedSQL(t, w, "UPDATE issues SET parent_id=$2 WHERE id=$1", a, root)
		listSeedSQL(t, w, "UPDATE issues SET parent_id=$2 WHERE id=$1", b, a)
		listSeedSQL(t, w, "UPDATE issues SET parent_id=$2 WHERE id=$1", c, b)
		listSeedSQL(t, w, "UPDATE issues SET parent_id=$2 WHERE id=$1", d, c)
		listSeedSQL(t, w, "UPDATE project_members SET role=5 WHERE project_id=$1 AND member_id=$2", w.web, w.ids["walt"])
		raw := func(kind string, source uuid.UUID) map[string][]string {
			return map[string][]string{"q": {"Tree"}, "for": {kind}, "issue_id": {source.String()}}
		}
		for _, middle := range []string{"hidden", "archived", "deleted"} {
			if middle != "hidden" {
				listSeedSQL(t, w, "UPDATE issues SET archived_at=NULL,deleted_at=NULL WHERE id=$1", b)
				listSeedSQL(t, w, "UPDATE issues SET "+map[string]string{"archived": "archived_at", "deleted": "deleted_at"}[middle]+"=now() WHERE id=$1", b)
			}
			t.Run(middle, func(t *testing.T) {
				checkSearchIDs(t, searchExecute(t, w, "walt", raw("parent", a)), allowed)
				checkSearchIDs(t, searchExecute(t, w, "walt", raw("sub_issue", d)), allowed)
			})
		}
		for _, term := range []string{"WEB-4", "4"} {
			checkSearchIDs(t, searchExecute(t, w, "walt", map[string][]string{"q": {term}, "for": {"parent"}, "issue_id": {a.String()}}))
		}
		for _, term := range []string{"WEB-1", "1"} {
			checkSearchIDs(t, searchExecute(t, w, "walt", map[string][]string{"q": {term}, "for": {"sub_issue"}, "issue_id": {d.String()}}))
		}
		// A malformed cycle must terminate; the same disallowed nodes remain out.
		listSeedSQL(t, w, "UPDATE issues SET parent_id=$2 WHERE id=$1", root, d)
		checkSearchIDs(t, searchExecute(t, w, "walt", raw("parent", a)), allowed)
	})
	t.Run("all live relation types both directions", func(t *testing.T) {
		w := newIssueWorld(t)
		source := w.newWorkItemIn(t, w.web, `{"name":"Relation"}`)
		for _, kind := range []string{"blocked_by", "duplicate", "relates_to"} {
			for _, reverse := range []bool{false, true} {
				id := w.newWorkItemIn(t, w.web, `{"name":"Relation"}`)
				from, to := source, id
				if reverse {
					from, to = to, from
				}
				relate(t, w.pool, from, to, kind, w.ids["alice"])
			}
		}
		for _, term := range []string{"WEB-1", "1", "WEB-2", "2"} {
			checkSearchIDs(t, searchExecute(t, w, "alice", map[string][]string{"q": {term}, "for": {"relation"}, "issue_id": {source.String()}}))
		}
		for _, reverse := range []bool{false, true} {
			t.Run(fmt.Sprint("deleted-", reverse), func(t *testing.T) {
				id := w.newWorkItemIn(t, w.web, `{"name":"Relation"}`)
				from, to := source, id
				if reverse {
					from, to = to, from
				}
				relate(t, w.pool, from, to, "duplicate", w.ids["alice"])
				listSeedSQL(t, w, "UPDATE issue_relations SET deleted_at=now() WHERE issue_id=$1 AND related_issue_id=$2", from, to)
				listSeedSQL(t, w, "UPDATE issues SET created_at='2020-01-01' WHERE id=$1", id)
				checkSearchIDs(t, searchExecute(t, w, "alice", map[string][]string{"q": {"Relation"}, "for": {"relation"}, "issue_id": {source.String()}}), id)
				checkSearchIDs(t, searchExecute(t, w, "alice", map[string][]string{"q": {"Relation"}, "for": {"relation"}, "issue_id": {source.String()}, "limit": {"1"}}), id)
				listSeedSQL(t, w, "UPDATE issues SET deleted_at=now() WHERE id=$1", id)
			})
		}
	})
	t.Run("ordinary directory and admin guest", func(t *testing.T) {
		w := newIssueWorld(t)
		w.adminGuest(t)
		hidden := w.newWorkItemBy(t, "bob", w.web, `{"name":"Scope"}`)
		own := w.newWorkItemBy(t, "walt", w.web, `{"name":"Scope"}`)
		archived := w.newWorkItemBy(t, "walt", w.web, `{"name":"Scope"}`)
		deleted := w.newWorkItemBy(t, "walt", w.web, `{"name":"Scope"}`)
		listSeedSQL(t, w, "UPDATE issues SET archived_at=now() WHERE id=$1", archived)
		listSeedSQL(t, w, "UPDATE issues SET deleted_at=now() WHERE id=$1", deleted)
		outside := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Outside", "OUT")
		w.newWorkItemIn(t, outside, `{"name":"Scope"}`)
		// Foreign workspace membership is active too; it still cannot enter acme.
		w.send(t, "alice", "POST", "/api/v0/workspaces", `{"name":"Foreign","slug":"foreign"}`, 201)
		foreign := createdProject(t, w.contract, w.base, w.tokens["alice"], "foreign", "Foreign", "FOREIGN")
		inWorkspaceOf(t, w.pool, foreign, w.ids["walt"], w.ids["alice"], shared.RoleMember)
		w.makeMember(t, "walt", foreign, shared.RoleMember)
		foreignIssue := w.newWorkItemBy(t, "walt", foreign, `{"name":"Scope"}`)
		_, sourceErr := snapshotSearchUseCase(w, newListProbe(w, "")).Execute(shared.WithActor(context.Background(), shared.Actor{UserID: w.ids["walt"]}), "acme", map[string][]string{"for": {"relation"}, "issue_id": {foreignIssue.String()}})
		if !errors.Is(sourceErr, domain.ErrIssueNotFound) {
			t.Fatal("foreign source", sourceErr)
		}
		checkSearchIDs(t, searchExecute(t, w, "walt", map[string][]string{"q": {"Scope"}}), own)
		listSeedSQL(t, w, "UPDATE issues SET created_at='2027-01-01' WHERE id=$1", hidden)
		checkSearchIDs(t, searchExecute(t, w, "walt", map[string][]string{"q": {"Scope"}, "limit": {"1"}}), own)
		for _, column := range []string{"archived_at", "deleted_at"} {
			listSeedSQL(t, w, "UPDATE projects SET "+column+"=now() WHERE id=$1", w.web)
			checkSearchIDs(t, searchExecute(t, w, "walt", map[string][]string{"q": {"Scope"}}))
			listSeedSQL(t, w, "UPDATE projects SET "+column+"=NULL WHERE id=$1", w.web)
		}
	})
	t.Run("dates AND before limit", func(t *testing.T) {
		w := newIssueWorld(t)
		a := w.newWorkItemIn(t, w.web, `{"name":"Dates"}`)
		b := w.newWorkItemIn(t, w.web, `{"name":"Dates"}`)
		c := w.newWorkItemIn(t, w.web, `{"name":"Dates"}`)
		listSeedSQL(t, w, "UPDATE issues SET start_date='2026-01-02',target_date='2026-02-02' WHERE id=$1", a)
		listSeedSQL(t, w, "UPDATE issues SET start_date='2026-01-03',target_date='2026-02-03' WHERE id=$1", b)
		for _, field := range []string{"start_date", "target_date"} {
			month := "01"
			if field == "target_date" {
				month = "02"
			}
			for _, tc := range []struct {
				conditions []string
				want       []uuid.UUID
			}{
				{[]string{"none"}, []uuid.UUID{c}},
				{[]string{"2026-" + month + "-02..2026-" + month + "-03"}, []uuid.UUID{b, a}},
				{[]string{"2026-" + month + "-01..", "..2026-" + month + "-02"}, []uuid.UUID{a}},
				{[]string{"2026-" + month + "-03..", "..2026-" + month + "-02"}, nil},
			} {
				checkSearchIDs(t, searchExecute(t, w, "alice", map[string][]string{"q": {"Dates"}, field: tc.conditions}), tc.want...)
			}
		}
		checkSearchIDs(t, searchExecute(t, w, "alice", map[string][]string{"q": {"Dates"}, "start_date": {"2026-01-02"}, "target_date": {"2026-02-02"}, "limit": {"1"}}), a)
	})
	t.Run("source scope and error order", func(t *testing.T) { searchSourceScope(t) })
}

func searchSourceScope(t *testing.T) {
	w := newIssueWorld(t)
	w.makeMember(t, "walt", w.web, shared.RoleMember)
	other := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Other", "OTHER")
	w.makeMember(t, "walt", other, shared.RoleMember)
	source := w.newWorkItemBy(t, "walt", w.web, `{"name":"Source"}`)
	candidate := w.newWorkItemIn(t, other, `{"name":"Candidate"}`)
	execute := func(ctx context.Context, slug string, raw map[string][]string) ([]domain.Issue, error) {
		return snapshotSearchUseCase(w, newListProbe(w, "")).Execute(ctx, slug, raw)
	}
	ctx := shared.WithActor(context.Background(), shared.Actor{UserID: w.ids["walt"]})
	for _, kind := range []string{"parent", "sub_issue", "relation"} {
		raw := map[string][]string{"project_id": {other.String()}, "for": {kind}, "issue_id": {source.String()}}
		rows, err := execute(ctx, "acme", raw)
		if kind == "relation" {
			if err != nil {
				t.Fatal(err)
			}
			checkSearchIDs(t, rows, candidate)
		} else {
			var problem *shared.Error
			if !errors.As(err, &problem) || problem.Kind != shared.KindInvalid || len(problem.Fields) != 1 || problem.Fields[0].Field != "project_id" || problem.Fields[0].Code != shared.FieldNotAllowed {
				t.Fatalf("%s: %v", kind, err)
			}
		}
	}
	for _, change := range []string{"own", "archived", "deleted", "outside"} {
		listSeedSQL(t, w, "UPDATE issues SET archived_at=NULL,deleted_at=NULL,created_by_id=$2 WHERE id=$1", source, w.ids["walt"])
		listSeedSQL(t, w, "UPDATE project_members SET role=15,is_active=true WHERE project_id=$1 AND member_id=$2", w.web, w.ids["walt"])
		switch change {
		case "own":
			listSeedSQL(t, w, "UPDATE project_members SET role=5 WHERE project_id=$1 AND member_id=$2", w.web, w.ids["walt"])
			listSeedSQL(t, w, "UPDATE issues SET created_by_id=$2 WHERE id=$1", source, w.ids["bob"])
		case "archived", "deleted":
			listSeedSQL(t, w, "UPDATE issues SET "+map[string]string{"archived": "archived_at", "deleted": "deleted_at"}[change]+"=now() WHERE id=$1", source)
		case "outside":
			listSeedSQL(t, w, "UPDATE project_members SET is_active=false WHERE project_id=$1 AND member_id=$2", w.web, w.ids["walt"])
		}
		_, err := execute(ctx, "acme", map[string][]string{"project_id": {other.String()}, "for": {"parent"}, "issue_id": {source.String()}})
		if !errors.Is(err, domain.ErrIssueNotFound) {
			t.Fatalf("%s source should precede conflict: %v", change, err)
		}
	}
	_, err := execute(ctx, "acme", map[string][]string{"project_id": {w.web.String()}, "for": {"parent"}, "issue_id": {uuid.NewV7().String()}})
	if !errors.Is(err, domain.ErrProjectNotFound) {
		t.Fatal("project scope must precede source", err)
	}
}

// The registered API must preserve raw multiplicity and domain field paths.
// Dropping issue_id without for, decoding away a bad pair, or binding UUIDs
// as transport types broadens searches or changes their public errors.
func TestSearchHTTPContract(t *testing.T) {
	w := newIssueWorld(t)
	source := w.newWorkItemIn(t, w.web, `{"name":"Source"}`)
	searchHTTPIDs(t, w, "alice", "q="+url.QueryEscape(strings.Repeat("界", 100)))
	for _, query := range []string{"", "q=", "q=+%E3%80%80"} {
		searchHTTPIDs(t, w, "alice", query, source)
	}
	for _, tc := range []struct {
		query, field, code string
		status             int
	}{
		{"cursor=x", "cursor", "not_allowed", 422}, {"order_by=name", "order_by", "not_allowed", 422},
		{"unknown=x", "unknown", "not_allowed", 422}, {"limit=x", "limit", "invalid_format", 400},
		{"limit=0", "limit", "out_of_range", 422}, {"limit=101", "limit", "out_of_range", 422},
		{"project_id=", "project_id", "invalid_format", 422}, {"project_id=bad", "project_id", "invalid_format", 422},
		{"issue_id=", "issue_id", "invalid_format", 422}, {"issue_id=bad", "issue_id", "invalid_format", 422},
		{"issue_id=" + source.String(), "issue_id", "not_allowed", 422},
		{"for=", "for", "not_allowed", 422}, {"for=wrong", "for", "not_allowed", 422},
		{"for=parent", "issue_id", "required", 422}, {"q=%00", "q", "invalid_format", 422},
		{"q=%FF", "q", "invalid_format", 422}, {"q=" + url.QueryEscape(strings.Repeat("界", 101)), "q", "too_long", 422},
	} {
		t.Run(tc.query, func(t *testing.T) { searchHTTPField(t, w, tc.query, tc.status, tc.field, tc.code) })
	}
	for key, value := range map[string]string{"q": "Source", "project_id": w.web.String(), "issue_id": source.String(), "for": "parent", "limit": "1"} {
		t.Run("duplicate/"+key, func(t *testing.T) {
			searchHTTPField(t, w, key+"="+value+"&"+key+"="+value, 422, key, "invalid_format")
		})
	}
	for _, query := range []string{"q=%GG", "q=a&q=b&bad=%GG", "limit=1&limit=2&bad=%GG", "limit=x&bad=%GG"} {
		t.Run("malformed/"+query, func(t *testing.T) { draftRequest(t, w, "alice", "GET", searchPath+"?"+query, "", 400) })
	}
	for _, field := range []string{"start_date", "target_date"} {
		for _, tc := range []struct{ values, path, code string }{
			{"none&" + field + "=2026-01-01", "[0]", "not_allowed"},
			{"2026-01-01&" + field + "=none", "[1]", "not_allowed"},
			{"none&" + field + "=none", "[0]", "not_allowed"}, {"2026-02-30", "[0]", "invalid_format"},
		} {
			searchHTTPField(t, w, field+"="+tc.values, 422, field+tc.path, tc.code)
		}
		listSeedSQL(t, w, "UPDATE issues SET "+field+"='2026-01-02' WHERE id=$1", source)
		searchHTTPIDs(t, w, "alice", field+"=2026-01-01..&"+field+"=..2026-01-02", source)
		searchHTTPIDs(t, w, "alice", field+"=2026-01-03..&"+field+"=..2026-01-02")
	}
	t.Run("source and project precedence", func(t *testing.T) {
		other := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Other", "OTHER")
		private := createdProjectFrom(t, w.contract, w.base, w.tokens["alice"], "acme", `{"name":"Private","identifier":"PRIVATE","network":0}`)
		w.makeMember(t, "bob", other, shared.RoleMember)
		w.makeMember(t, "walt", other, shared.RoleGuest)
		w.makeMember(t, "walt", w.web, shared.RoleGuest)
		archived := w.newWorkItem(t)
		deleted := w.newWorkItem(t)
		hiddenProject := w.newWorkItemIn(t, private, `{"name":"Private"}`)
		listSeedSQL(t, w, "UPDATE issues SET archived_at=now() WHERE id=$1", archived)
		w.send(t, "alice", "DELETE", "/api/v0/issues/"+deleted.String(), "", 204)
		w.send(t, "alice", "POST", "/api/v0/workspaces", `{"name":"Foreign","slug":"foreign"}`, 201)
		foreign := createdProject(t, w.contract, w.base, w.tokens["alice"], "foreign", "Foreign", "FOREIGN")
		foreignIssue := w.newWorkItemIn(t, foreign, `{"name":"Foreign"}`)
		missing := uuid.NewV7()
		sourceQuery := func(id uuid.UUID) string {
			return "for=parent&issue_id=" + id.String() + "&project_id=" + other.String()
		}
		absent := draftRequest(t, w, "bob", "GET", searchPath+"?"+sourceQuery(missing), "", 404)
		if problemObject(t, absent)["code"] != "issue.not_found" {
			t.Fatal(absent)
		}
		for _, id := range []uuid.UUID{hiddenProject, archived, deleted, foreignIssue} {
			got := draftRequest(t, w, "bob", "GET", searchPath+"?"+sourceQuery(id), "", 404)
			if !reflect.DeepEqual(problemObject(t, got), problemObject(t, absent)) {
				t.Fatal(got, absent)
			}
		}
		hidden := draftRequest(t, w, "walt", "GET", searchPath+"?"+sourceQuery(source), "", 404)
		if !reflect.DeepEqual(problemObject(t, hidden), problemObject(t, absent)) {
			t.Fatal(hidden, absent)
		}
		for _, kind := range []string{"parent", "sub_issue"} {
			searchHTTPField(t, w, "for="+kind+"&issue_id="+source.String()+"&project_id="+other.String(), 422, "project_id", "not_allowed")
		}
		outside := draftRequest(t, w, "bob", "GET", searchPath+"?for=parent&issue_id="+source.String()+"&project_id="+private.String(), "", 404)
		randomProject := draftRequest(t, w, "bob", "GET", searchPath+"?for=parent&issue_id="+source.String()+"&project_id="+uuid.NewV7().String(), "", 404)
		if !reflect.DeepEqual(problemObject(t, outside), problemObject(t, randomProject)) || problemObject(t, outside)["code"] != "project.not_found" {
			t.Fatal(outside, randomProject)
		}
	})
}
