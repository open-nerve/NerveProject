package bootstrap

import (
	"fmt"
	"net/url"
	"slices"
	"strings"
	"testing"
	"uuid"
)

func windowIDs(want ...uuid.UUID) func(*testing.T, string) {
	return func(t *testing.T, body string) {
		t.Helper()
		var page struct{ Data []struct{ ID uuid.UUID } }
		decodeAnswer(t, body, &page)
		got := make([]uuid.UUID, len(page.Data))
		for i, row := range page.Data {
			got[i] = row.ID
		}
		if !slices.Equal(got, want) {
			t.Fatalf("ordered IDs=%v want=%v; %s", got, want, body)
		}
	}
}

func windowPageIDs(total int64, want ...uuid.UUID) func(*testing.T, string) {
	return func(t *testing.T, body string) {
		t.Helper()
		windowIDs(want...)(t, body)
		var page struct {
			Total int64
			Next  *string `json:"next_cursor"`
		}
		decodeAnswer(t, body, &page)
		if page.Total != total {
			t.Fatalf("total=%d want %d", page.Total, total)
		}
		if int64(len(want)) == total && page.Next != nil {
			t.Fatal("terminal page has cursor", body)
		}
	}
}

// These positions come from pre-deletion API pages, including nested cells.
// They are never recomputed after a project disappears from the directory.
func (f deletionWindowFixture) pageRequests(t *testing.T, workspace bool) []deletionWindowRequest {
	t.Helper()
	w := f.w
	var out []deletionWindowRequest
	add := func(name, path string, want int, check func(*testing.T, string)) {
		r := deletionWindowRequest{name: name, actor: "alice", method: "GET", path: path, want: want, check: check}
		if want == 404 {
			r.absentPath = strings.Replace(path, w.web.String(), uuid.NewV7().String(), 1)
			if strings.Contains(path, "/workspaces/acme/") {
				r.absentPath = strings.Replace(path, "/workspaces/acme/", "/workspaces/absent/", 1)
			}
		}
		out = append(out, r)
	}
	for _, target := range []struct{ name, path string }{
		{"project", "/api/v0/projects/" + w.web.String() + "/issues"},
		{"member", memberIssuesPath(w, "bob") + "?relation=created"},
	} {
		for _, shape := range []string{"flat", "group", "subgroup"} {
			// Member lists expose groups, but only project lists expose subgroups.
			if target.name == "member" && shape == "subgroup" {
				add("member/unsupported-subgroup", target.path+"&group_by=priority&sub_group_by=state_group", 422, windowField("sub_group_by not_allowed"))
				continue
			}
			query := "order_by=-created_at&limit=1"
			if shape != "flat" {
				query += "&group_by=priority"
			}
			if shape == "subgroup" {
				query += "&sub_group_by=state_group"
			}
			sep := "?"
			if strings.Contains(target.path, "?") {
				sep = "&"
			}
			path := target.path + sep + query
			p := checkListEnvelope(t, w.send(t, "alice", "GET", path, "", 200), shape != "flat", shape == "subgroup")
			cursor := p.NextCursor
			selected := ""
			if shape != "flat" {
				// Choose a group/cell with a real continuation, never a forged cursor.
				for _, g := range p.Groups {
					if shape == "group" && g.NextCursor != nil {
						cursor = g.NextCursor
						selected = "&group_key=" + url.QueryEscape(g.Key)
						break
					}
					for _, sub := range g.Subgroups {
						if sub.NextCursor != nil {
							cursor = sub.NextCursor
							selected = "&group_key=" + url.QueryEscape(g.Key) + "&sub_group_key=" + url.QueryEscape(sub.Key)
							break
						}
					}
					if cursor != nil {
						break
					}
				}
			}
			if cursor == nil {
				t.Fatalf("%s/%s fixture has no continuation: %+v", target.name, shape, p)
			}
			want := 404
			var check func(*testing.T, string)
			if target.name == "member" && !workspace {
				want = 200
				check = func(t *testing.T, body string) {
					page := checkListEnvelope(t, body, shape != "flat", shape == "subgroup")
					if page.Total != 3 {
						t.Fatal("deleted member rows counted", body)
					}
					rows := page
					if shape == "group" {
						if len(page.Groups) != 1 || page.Groups[0].Key != "high" || page.Groups[0].Total != 3 {
							t.Fatal("deleted group retained or live group lost", body)
						}
						rows = page.Groups[0]
					}
					if len(rows.Data) != 1 || rows.Data[0].ID != f.kept[2] || rows.NextCursor == nil {
						t.Fatal("member first-page order/lookahead", body)
					}
				}
			}
			add(target.name+"/"+shape, path, want, check)
			continued := path + selected + "&cursor=" + url.QueryEscape(*cursor)
			if want == 200 {
				check = func(t *testing.T, body string) {
					p := checkListEnvelope(t, body, false, false)
					if p.Total != 3 {
						t.Fatal("cursor retained deleted directory", body)
					}
					first := f.kept[2]
					if shape == "group" {
						first = f.kept[1]
					}
					if len(p.Data) != 1 || p.Data[0].ID != first || p.NextCursor == nil {
						t.Fatal("old cursor lost exact position/order/lookahead", body)
					}
				}
			}
			add(target.name+"/"+shape+"/old-cursor", continued, want, check)
		}
	}
	// The member cursor starts at the newest (now deleted) bob issue.
	member := memberIssuesPath(w, "bob") + "?relation=created&order_by=-created_at&limit=1"
	p := checkListEnvelope(t, w.send(t, "alice", "GET", member, "", 200), false, false)
	if len(p.Data) != 1 || p.Data[0].ID != f.memberGone || p.NextCursor == nil {
		t.Fatal("member first-page discriminator", p)
	}
	want := 200
	if workspace {
		want = 404
	}
	check := windowPageIDs(3, f.kept[2], f.kept[1], f.kept[0])
	if workspace {
		check = nil
	}
	add("member/old-cursor-all", strings.Replace(member, "limit=1", "limit=100", 1)+"&cursor="+url.QueryEscape(*p.NextCursor), want, check)
	for _, relation := range []string{"assigned", "subscribed"} {
		check = windowPageIDs(3, f.kept[2], f.kept[1], f.kept[0])
		if workspace {
			check = nil
		}
		add("member/"+relation, memberIssuesPath(w, "bob")+"?relation="+relation+"&order_by=-created_at", want, check)
	}
	var drafts struct {
		Data []draftHTTP
		Next *string `json:"next_cursor"`
	}
	decodeAnswer(t, w.send(t, "alice", "GET", draftsPath+"?limit=1", "", 200), &drafts)
	if len(drafts.Data) != 1 || drafts.Data[0].ID != f.x.draft || drafts.Next == nil {
		t.Fatal("draft cursor discriminator", drafts)
	}
	check = windowPageIDs(2, f.noProject, f.qDraft)
	if workspace {
		check = nil
	}
	add("drafts/old-cursor", draftsPath+"?limit=100&cursor="+url.QueryEscape(*drafts.Next), want, check)
	for _, path := range []string{commentsOf(f.x.issue), activitiesOf(f.x.issue), versionsOf(f.x.issue)} {
		var page struct {
			Next *string `json:"next_cursor"`
		}
		decodeAnswer(t, w.send(t, "alice", "GET", path+"?limit=1", "", 200), &page)
		if page.Next == nil {
			t.Fatal("child page has no real cursor", path)
		}
		r := deletionWindowRequest{name: "child/old-cursor/" + path, actor: "alice", method: "GET", path: path + "?limit=1&cursor=" + url.QueryEscape(*page.Next), want: 404}
		r.absentPath = strings.Replace(r.path, f.x.issue.String(), uuid.NewV7().String(), 1)
		out = append(out, r)
	}
	for _, query := range []string{"q=Window&limit=1", "q=Window&project_id=" + f.q.String(), "q=Window&for=relation&issue_id=" + f.kept[2].String()} {
		check = windowIDs(f.kept[2], f.kept[1], f.kept[0])
		if strings.Contains(query, "limit=1") {
			check = windowIDs(f.kept[2])
		}
		if strings.Contains(query, "for=relation") {
			// A relation candidate collection excludes its source itself.
			check = windowIDs(f.kept[1], f.kept[0])
		}
		if workspace {
			check = nil
		}
		add("search/"+query, "/api/v0/workspaces/acme/issue-search?"+query, want, check)
	}
	// Live Q remains exactly unchanged by deleting P, including order and count.
	if !workspace {
		path := fmt.Sprintf("/api/v0/projects/%s/issues?order_by=-created_at", f.q)
		before := deletionWindowRequest{actor: "alice", method: "GET", path: path, want: 200}.answer(t, w)
		add("live-project/whole-page", path, 200, func(t *testing.T, body string) {
			if body != before.body {
				t.Fatal("live project page changed", body, before.body)
			}
		})
	}
	path := "/api/v0/workspaces/acme/issue-search?project_id=" + f.q.String()
	emptyQuery := deletionWindowRequest{actor: "alice", method: "GET", path: path, want: 200}.answer(t, w)
	check = func(t *testing.T, body string) {
		if body != emptyQuery.body {
			t.Fatal("empty search order/content changed", body, emptyQuery.body)
		}
	}
	if workspace {
		check = nil
	}
	add("search/empty-query", path, want, check)
	return out
}
