package bootstrap

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
	"uuid"
)

type issueListHTTP struct {
	Total int64 `json:"total"`
	Data  []struct {
		ID             uuid.UUID
		ParentID       *uuid.UUID  `json:"parent_id"`
		SubIssuesCount int         `json:"sub_issues_count"`
		LabelIDs       []uuid.UUID `json:"label_ids"`
		AssigneeIDs    []uuid.UUID `json:"assignee_ids"`
	} `json:"data"`
	NextCursor     *string         `json:"next_cursor"`
	Groups         []issueListHTTP `json:"groups"`
	Subgroups      []issueListHTTP `json:"subgroups"`
	SubgroupTotals []struct {
		Key   string
		Total int64
	} `json:"subgroup_totals"`
	Key string
}

// Check complete keys at every envelope level, including [] versus null.
func listKeys(t *testing.T, raw []byte, keys ...string) map[string]json.RawMessage {
	t.Helper()
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		t.Fatal(err, string(raw))
	}
	got := make([]string, 0, len(obj))
	for k := range obj {
		got = append(got, k)
	}
	slices.Sort(got)
	slices.Sort(keys)
	if !slices.Equal(got, keys) {
		t.Fatalf("keys %v want %v: %s", got, keys, raw)
	}
	return obj
}
func listArray(t *testing.T, raw json.RawMessage) []json.RawMessage {
	t.Helper()
	var a []json.RawMessage
	if len(raw) == 0 || raw[0] != '[' || json.Unmarshal(raw, &a) != nil {
		t.Fatalf("want array, got %s", raw)
	}
	return a
}
func checkListRows(t *testing.T, obj map[string]json.RawMessage) {
	t.Helper()
	if raw := obj["next_cursor"]; string(raw) != "null" {
		var cursor string
		if json.Unmarshal(raw, &cursor) != nil || cursor == "" {
			t.Fatalf("cursor %s", raw)
		}
	}
	for _, raw := range listArray(t, obj["data"]) {
		row := listKeys(t, raw, "id", "workspace_id", "project_id", "sequence_id", "name", "state_id", "priority", "start_date", "target_date", "parent_id", "sort_order", "completed_at", "archived_at", "assignee_ids", "label_ids", "sub_issues_count", "created_by_id", "updated_by_id", "created_at", "updated_at")
		listArray(t, row["assignee_ids"])
		listArray(t, row["label_ids"])
	}
}
func checkListEnvelope(t *testing.T, answer string, grouped, lanes bool) issueListHTTP {
	t.Helper()
	var page issueListHTTP
	decodeAnswer(t, answer, &page)
	if !grouped {
		obj := listKeys(t, []byte(answer), "total", "data", "next_cursor")
		checkListRows(t, obj)
		return page
	}
	keys := []string{"total", "groups"}
	if lanes {
		keys = append(keys, "subgroup_totals")
	}
	obj := listKeys(t, []byte(answer), keys...)
	for _, raw := range listArray(t, obj["groups"]) {
		if !lanes {
			checkListRows(t, listKeys(t, raw, "key", "total", "data", "next_cursor"))
			continue
		}
		group := listKeys(t, raw, "key", "total", "subgroups")
		for _, sub := range listArray(t, group["subgroups"]) {
			checkListRows(t, listKeys(t, sub, "key", "total", "data", "next_cursor"))
		}
	}
	if lanes {
		for _, raw := range listArray(t, obj["subgroup_totals"]) {
			listKeys(t, raw, "key", "total")
		}
	}
	return page
}
func TestIssueListEnvelope(t *testing.T) {
	w := newIssueWorld(t)
	path := "/api/v0/projects/" + w.web.String() + "/issues"
	for _, query := range []string{"", "?group_by=label_id", "?group_by=label_id&sub_group_by=assignee_id"} {
		p := checkListEnvelope(t, w.send(t, "alice", http.MethodGet, path+query, "", 200), query != "", strings.Contains(query, "sub_group"))
		if p.Total != 0 {
			t.Fatal(p)
		}
	}
	var label struct{ ID uuid.UUID }
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/labels", `{"name":"Other"}`, 201), &label)
	a, c := w.bug, label.ID
	people := idArray(w.ids["alice"], w.ids["bob"])
	first := w.newWorkItemIn(t, w.web, fmt.Sprintf(`{"name":"Both 1","label_ids":%s,"assignee_ids":%s,"start_date":"2026-10-12","target_date":"2026-10-14"}`, idArray(a, c), people))
	second := w.newWorkItemIn(t, w.web, fmt.Sprintf(`{"name":"Both 2","label_ids":%s,"assignee_ids":%s,"start_date":"2026-10-18","target_date":"2026-10-20"}`, idArray(a, c), people))
	w.newWorkItemIn(t, w.web, fmt.Sprintf(`{"name":"Only C","label_ids":%s,"assignee_ids":%s}`, idArray(c), people))
	query := "?label_id=" + a.String() + "&group_by=label_id&sub_group_by=assignee_id&order_by=sort_order&limit=1"
	page := checkListEnvelope(t, w.send(t, "alice", http.MethodGet, path+query, "", 200), true, true)
	if page.Total != 2 || len(page.Groups) != 2 || len(page.SubgroupTotals) != 2 {
		t.Fatal(page)
	}
	for _, g := range page.Groups {
		if g.Total != 2 || len(g.Subgroups) != 2 {
			t.Fatal(g)
		}
		for _, cell := range g.Subgroups {
			if cell.Total != 2 || len(cell.Data) != 1 || cell.Data[0].ID != first || len(cell.Data[0].LabelIDs) != 2 || len(cell.Data[0].AssigneeIDs) != 2 || cell.NextCursor == nil {
				t.Fatal(cell)
			}
			more := query + "&group_key=" + g.Key + "&sub_group_key=" + cell.Key + "&cursor=" + url.QueryEscape(*cell.NextCursor)
			next := checkListEnvelope(t, w.send(t, "alice", http.MethodGet, path+more, "", 200), false, false)
			if next.Total != 2 || len(next.Data) != 1 || next.Data[0].ID != second || next.NextCursor != nil {
				t.Fatal(next)
			}
		}
	}
	groups := checkListEnvelope(t, w.send(t, "alice", http.MethodGet, path+"?label_id="+a.String()+"&group_by=label_id&limit=1&order_by=sort_order", "", 200), true, false)
	for _, g := range groups.Groups {
		if g.NextCursor == nil {
			t.Fatal(g)
		}
		next := checkListEnvelope(t, w.send(t, "alice", http.MethodGet, path+"?label_id="+a.String()+"&group_by=label_id&limit=1&order_by=sort_order&group_key="+g.Key+"&cursor="+url.QueryEscape(*g.NextCursor), "", 200), false, false)
		if next.Total != 2 || len(next.Data) != 1 || next.Data[0].ID != second || next.NextCursor != nil {
			t.Fatal(next)
		}
	}
	for _, date := range []string{"start_date", "target_date"} {
		values := map[string]string{"start_date": "2026-10-10..2026-10-19&start_date=..2026-10-15", "target_date": "2026-10-10..2026-10-22&target_date=..2026-10-15"}
		p := checkListEnvelope(t, w.send(t, "alice", http.MethodGet, path+"?"+date+"="+values[date], "", 200), false, false)
		if p.Total != 1 || len(p.Data) != 1 || p.Data[0].ID != first {
			t.Fatal(p)
		}
	}
}
func TestIssueListQueryBoundary(t *testing.T) {
	w := newIssueWorld(t)
	path := "/api/v0/projects/" + w.web.String() + "/issues"
	check := func(path, token string, want int, field, code string) {
		t.Helper()
		req := newRequest(t, http.MethodGet, w.base+path, token, nil)
		raw := req.URL.RawQuery
		res, body := send(t, req)
		w.contract.CheckResponse(t, req, res)
		if res.StatusCode != want || req.URL.RawQuery != raw {
			t.Fatal(res.StatusCode, string(body))
		}
		if field != "" && oneError(t, body) != field+" "+code {
			t.Fatal(string(body))
		}
	}
	for _, field := range []string{"state_id", "state_group", "priority", "label_id", "assignee_id", "created_by_id", "mention_id", "subscriber_id", "sub_issue", "archived", "group_by", "sub_group_by", "order_by", "group_key", "sub_group_key", "limit", "cursor"} {
		t.Run("duplicate/"+field, func(t *testing.T) {
			check(path+"?"+field+"=x&"+field+"=y", w.tokens["alice"], 422, field, "not_allowed")
		})
	}
	for _, tt := range []struct {
		query, field, code string
		status             int
	}{{"state_id=bad", "state_id", "not_allowed", 422}, {"start_date=2026-02-30", "start_date", "not_allowed", 422}, {"sub_issue=yes", "sub_issue", "not_allowed", 422}, {"archived=1", "archived", "not_allowed", 422}, {"cycle_id=x", "cycle_id", "not_allowed", 422}, {"group_by=project_id", "group_by", "not_allowed", 422}, {"order_by=link_count", "order_by", "not_allowed", 422}, {"limit=x", "limit", "invalid_format", 400}, {"limit=101", "limit", "out_of_range", 422}, {"cursor=bad", "cursor", "invalid_format", 400}} {
		check(path+"?"+tt.query, w.tokens["alice"], tt.status, tt.field, tt.code)
	}
	check(path+"?limit=1&limit=2", "", 422, "limit", "not_allowed")
	check("/api/v0/projects/bad/issues?limit=1&limit=2", "", 400, "project_id", "invalid_format")
	check("/api/v0/projects/bad/issues?project_id=x&project_id=y", w.tokens["alice"], 400, "project_id", "invalid_format")
	check(path, "", 401, "", "")
	check("/api/v0/issues/"+uuid.NewV7().String()+"/comments?limit=1&limit=2", w.tokens["alice"], 400, "limit", "invalid_format")
}
func TestIssueListPermissions(t *testing.T) {
	w := newIssueWorld(t)
	w.adminGuest(t)
	parent := w.newWorkItem(t)
	own := w.newWorkItemBy(t, "walt", w.web, `{"name":"Own"}`)
	w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+own.String(), `{"parent_id":"`+parent.String()+`"}`, 200)
	w.newWorkItemBy(t, "walt", w.web, `{"name":"Own child","parent_id":"`+own.String()+`"}`)
	w.newWorkItem(t)
	path := "/api/v0/projects/" + w.web.String() + "/issues"
	p := checkListEnvelope(t, w.send(t, "walt", http.MethodGet, path, "", 200), false, false)
	if p.Total != 2 || len(p.Data) != 2 {
		t.Fatal(p)
	}
	for _, i := range p.Data {
		if i.ID == own && (i.ParentID != nil || i.SubIssuesCount != 1) {
			t.Fatal(i)
		}
	}
	other := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Other", "OTH")
	w.newWorkItemIn(t, other, `{"name":"Other"}`)
	p = checkListEnvelope(t, w.send(t, "alice", http.MethodGet, path, "", 200), false, false)
	if p.Total != 4 {
		t.Fatal("explicit project widened", p)
	}
	archived := w.doneWorkItem(t)
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+archived.String()+"/archive", "", 200)
	w.send(t, "alice", http.MethodPatch, "/api/v0/project-members/"+queryIDs(t, w.pool, "SELECT id FROM project_members WHERE project_id=$1 AND member_id=$2", w.web, w.ids["bob"])[0].String(), `{"role":5}`, 200)
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/archive", "", 200)
	for _, tt := range []struct {
		query string
		total int64
	}{{"", 4}, {"?archived=true", 1}} {
		p := checkListEnvelope(t, w.send(t, "alice", http.MethodGet, path+tt.query, "", 200), false, false)
		if p.Total != tt.total {
			t.Fatal(p)
		}
	}
	answer := w.send(t, "bob", http.MethodGet, path+"?archived=true", "", 403)
	if problemCode(t, []byte(answer)) != "forbidden" {
		t.Fatal(answer)
	}
}
