package bootstrap

import (
	"encoding/json"
	"net/url"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

const searchPath = "/api/v0/workspaces/acme/issue-search"

// Search is a workspace read for every active role. Admin and member fixture
// columns have no live memberships; the guest retains its own/all scopes.
func searchMatrixRows() []matrixRow {
	return []matrixRow{{op: "searchIssues", cells: inWorkspace(cellOK, cellOK, cellOK), request: toWorkspace("GET", "/issue-search", ""), check: func(t *testing.T, c caller, s seeded, body string) {
		rows := searchHTTPRows(t, body)
		var got, want []uuid.UUID
		for _, row := range rows {
			got = append(got, row.ID)
		}
		if c == callerGuest {
			for _, item := range seededItems() {
				if item.key == "acme/private" || item.key == "acme/public" && item.by == callerGuest {
					want = append(want, s.issue(item.key, item.name))
				}
			}
		}
		slices.SortFunc(got, uuid.UUID.Compare)
		slices.SortFunc(want, uuid.UUID.Compare)
		if !slices.Equal(got, want) {
			t.Fatal(body, want)
		}
	}}}
}

func searchHTTPRows(t *testing.T, body string) []struct {
	ID       uuid.UUID  `json:"id"`
	ParentID *uuid.UUID `json:"parent_id"`
} {
	t.Helper()
	listKeys(t, []byte(body), "data")
	var p struct {
		Data []struct {
			ID       uuid.UUID  `json:"id"`
			ParentID *uuid.UUID `json:"parent_id"`
		} `json:"data"`
	}
	decodeAnswer(t, body, &p)
	if p.Data == nil {
		t.Fatal("search data must be an array:", body)
	}
	return p.Data
}
func searchHTTPIDs(t *testing.T, w issueWorld, actor, query string, want ...uuid.UUID) {
	t.Helper()
	body := draftRequest(t, w, actor, "GET", searchPath+"?"+query, "", 200)
	rows := searchHTTPRows(t, body)
	got := make([]uuid.UUID, len(rows))
	for i, row := range rows {
		got[i] = row.ID
	}
	// Scope comparisons are sets; ranking and date order have dedicated tests.
	slices.SortFunc(got, uuid.UUID.Compare)
	want = slices.Clone(want)
	slices.SortFunc(want, uuid.UUID.Compare)
	if !slices.Equal(got, want) {
		t.Fatalf("%s query %s: %s; want %v", actor, query, body, want)
	}
}
func searchHTTPField(t *testing.T, w issueWorld, query string, status int, field, code string) {
	t.Helper()
	body := draftRequest(t, w, "alice", "GET", searchPath+"?"+query, "", status)
	var p struct {
		Code   string
		Errors []shared.FieldError
	}
	if err := json.Unmarshal([]byte(body), &p); err != nil {
		t.Fatal(err)
	}
	wantCode := "validation_failed"
	if status == 400 {
		wantCode = "bad_request"
	}
	if p.Code != wantCode || len(p.Errors) != 1 || p.Errors[0].Field != field || p.Errors[0].Code != code {
		t.Fatalf("%s: %s; want %s/%s", query, body, field, code)
	}
}

// A workspace admin with a project guest grant stays OwnOnly. All term and
// picker classes must use that same scope, including the parent projection.
func TestPermissionMatrixSearch(t *testing.T) {
	w := newIssueWorld(t)
	w.makeMember(t, "walt", w.web, shared.RoleMember)
	hidden := w.newWorkItemBy(t, "bob", w.web, `{"name":"Scope"}`)
	own := w.newWorkItemBy(t, "walt", w.web, `{"name":"Scope"}`)
	source := w.newWorkItemBy(t, "walt", w.web, `{"name":"Origin"}`)
	listSeedSQL(t, w, "UPDATE issues SET parent_id=$2 WHERE id=$1", own, hidden)
	outside := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Outside", "OUT")
	outsideIssue := w.newWorkItemIn(t, outside, `{"name":"Scope"}`)
	w.makeMember(t, "wanda", w.web, shared.RoleAdmin)
	listSeedSQL(t, w, "UPDATE workspace_members SET role=5 WHERE workspace_id=$1 AND member_id=$2", w.acme, w.ids["walt"])
	listSeedSQL(t, w, "UPDATE project_members SET role=5 WHERE project_id=$1 AND member_id=$2", w.web, w.ids["walt"])
	for _, phase := range []string{"workspace guest", "admin guest"} {
		if phase == "admin guest" {
			listSeedSQL(t, w, "UPDATE workspace_members SET role=20 WHERE workspace_id=$1 AND member_id=$2", w.acme, w.ids["walt"])
		}
		t.Run(phase, func(t *testing.T) {
			for _, kind := range []string{"", "parent", "sub_issue", "relation"} {
				for _, term := range []string{"", "1", "WEB-1", "2", "WEB-2", "Scope"} {
					query := url.Values{"q": {term}}
					if kind != "" {
						query.Set("for", kind)
						query.Set("issue_id", source.String())
					}
					want := []uuid.UUID{}
					if term == "" || term == "Scope" || term == "2" || term == "WEB-2" {
						want = append(want, own)
					}
					if kind == "sub_issue" {
						want = nil
					}
					if kind == "" && term == "" {
						want = append(want, source)
					}
					searchHTTPIDs(t, w, "walt", query.Encode(), want...)
				}
			}
			body := draftRequest(t, w, "walt", "GET", searchPath+"?q=WEB-2", "", 200)
			rows := searchHTTPRows(t, body)
			if len(rows) != 1 || rows[0].ID != own || rows[0].ParentID != nil {
				t.Fatal("hidden parent leaked:", body)
			}
		})
	}
	for _, who := range []string{"alice", "bob", "wanda"} {
		for _, kind := range []string{"", "parent", "sub_issue", "relation"} {
			for _, term := range []string{"", "1", "WEB-1", "Scope"} {
				query := url.Values{"q": {term}}
				if kind != "" {
					query.Set("for", kind)
					query.Set("issue_id", source.String())
				}
				want := []uuid.UUID{hidden}
				if (term == "" || term == "Scope") && kind != "sub_issue" {
					want = append(want, own)
				}
				if term == "" && kind == "" {
					want = append(want, source)
				}
				if who == "alice" && term != "WEB-1" && (kind == "" || kind == "relation") {
					want = append(want, outsideIssue)
				}
				searchHTTPIDs(t, w, who, query.Encode(), want...)
			}
		}
	}
	listSeedSQL(t, w, "UPDATE workspace_members SET deleted_at=now() WHERE workspace_id=$1 AND member_id=$2", w.acme, w.ids["bob"])
	absent := draftRequest(t, w, "alice", "GET", "/api/v0/workspaces/absent/issue-search", "", 404)
	if problemObject(t, absent)["code"] != "workspace.not_found" {
		t.Fatal(absent)
	}
	for _, who := range []string{"bob", "olga"} {
		got := draftRequest(t, w, who, "GET", searchPath, "", 404)
		if !reflect.DeepEqual(problemObject(t, got), problemObject(t, absent)) {
			t.Fatal(got, absent)
		}
	}
	listSeedSQL(t, w, "UPDATE project_members SET is_active=false WHERE member_id=$1", w.ids["walt"])
	searchHTTPIDs(t, w, "walt", "")
}
