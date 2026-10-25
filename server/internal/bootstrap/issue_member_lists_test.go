package bootstrap

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func memberIssuesPath(w issueWorld, target string) string {
	return "/api/v0/workspaces/acme/members/" + w.ids[target].String() + "/issues"
}
func memberListIDs(t *testing.T, w issueWorld, actor, target, relation, extra string, want ...uuid.UUID) issueListHTTP {
	t.Helper()
	p := checkListEnvelope(t, w.send(t, actor, http.MethodGet, memberIssuesPath(w, target)+"?relation="+relation+extra, "", 200), false, false)
	got := make([]uuid.UUID, len(p.Data))
	for n, row := range p.Data {
		got[n] = row.ID
	}
	want = slices.Clone(want)
	slices.SortFunc(want, uuid.UUID.Compare)
	slices.SortFunc(got, uuid.UUID.Compare)
	if p.Total != int64(len(want)) || !slices.Equal(got, want) || p.NextCursor != nil {
		t.Fatalf("%s reads %s/%s = %+v, want %v", actor, target, relation, p, want)
	}
	return p
}

// A=walt, B=bob, C=alice. Automatic subscriptions intentionally overlap.
func memberRelations(t *testing.T, w issueWorld) []uuid.UUID {
	t.Helper()
	w.makeMember(t, "walt", w.web, shared.RoleMember)
	create := func(actor string, assignees ...uuid.UUID) uuid.UUID {
		return w.newWorkItemBy(t, actor, w.web, fmt.Sprintf(`{"name":"Relationship","assignee_ids":%s,"start_date":"2026-10-12","target_date":"2026-10-15"}`, idArray(assignees...)))
	}
	ids := []uuid.UUID{create("alice", w.ids["bob"]), create("bob", w.ids["alice"]), create("alice"), create("walt", w.ids["walt"]), create("alice", w.ids["alice"])}
	before := queryIDs(t, w.pool, "SELECT subscriber_id FROM issue_subscribers WHERE issue_id=$1 AND deleted_at IS NULL", ids[2])
	if !slices.Equal(before, []uuid.UUID{w.ids["alice"]}) {
		t.Fatal(before)
	}
	var comment answeredComment
	decodeAnswer(t, w.send(t, "bob", http.MethodPost, commentsOf(ids[2]), `{"comment_html":"<p>Follow up</p>"}`, 201), &comment)
	if comment.ActorID != w.ids["bob"] || comment.IssueID != ids[2] {
		t.Fatal(comment)
	}
	for n, id := range ids {
		var author uuid.UUID
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT created_by_id FROM issues WHERE id=$1", id).Scan(&author); err != nil {
			t.Fatal(err)
		}
		creators := []string{"alice", "bob", "alice", "walt", "alice"}
		if author != w.ids[creators[n]] {
			t.Fatal(author)
		}
		assignees := queryIDs(t, w.pool, "SELECT assignee_id FROM issue_assignees WHERE issue_id=$1 AND deleted_at IS NULL ORDER BY assignee_id", id)
		wantAssigned := [][]uuid.UUID{{w.ids["bob"]}, {w.ids["alice"]}, {}, {w.ids["walt"]}, {w.ids["alice"]}}
		if !slices.Equal(assignees, wantAssigned[n]) {
			t.Fatal(assignees, wantAssigned[n])
		}
		got := queryIDs(t, w.pool, "SELECT subscriber_id FROM issue_subscribers WHERE issue_id=$1 AND deleted_at IS NULL ORDER BY subscriber_id", id)
		want := []uuid.UUID{w.ids["alice"], w.ids["bob"]}
		switch n {
		case 3:
			want = []uuid.UUID{w.ids["walt"]}
		case 4:
			want = []uuid.UUID{w.ids["alice"]}
		}
		slices.SortFunc(want, uuid.UUID.Compare)
		if !slices.Equal(got, want) {
			t.Fatal(got, want)
		}
	}
	return ids
}
func TestMemberIssueRelationTarget(t *testing.T) {
	w := newIssueWorld(t)
	ids := memberRelations(t, w)
	memberListIDs(t, w, "walt", "bob", "assigned", "", ids[0])
	memberListIDs(t, w, "walt", "bob", "created", "", ids[1])
	memberListIDs(t, w, "walt", "bob", "subscribed", "", ids[:3]...)
	for _, r := range []string{"assigned", "created", "subscribed"} {
		memberListIDs(t, w, "walt", "walt", r, "", ids[3])
	}
	memberListIDs(t, w, "walt", "alice", "assigned", "", ids[1], ids[4])
	memberListIDs(t, w, "walt", "alice", "created", "", ids[0], ids[2], ids[4])
	memberListIDs(t, w, "walt", "alice", "subscribed", "", ids[0], ids[1], ids[2], ids[4])
	// Both repeated date fields intersect the actual SQL result.
	memberListIDs(t, w, "walt", "bob", "assigned", "&start_date=2026-10-10..&start_date=..2026-10-13&target_date=2026-10-14..&target_date=..2026-10-16", ids[0])
	memberListIDs(t, w, "walt", "bob", "assigned", "&start_date=2026-10-10..&start_date=..2026-10-11")
	memberListIDs(t, w, "walt", "bob", "assigned", "&target_date=2026-10-16..&target_date=..2026-10-20")
	// Member group pages and continuation use the same closed envelope.
	path := memberIssuesPath(w, "bob") + "?relation=subscribed&group_by=project_id&limit=1&order_by=sort_order"
	p := checkListEnvelope(t, w.send(t, "walt", http.MethodGet, path, "", 200), true, false)
	if p.Total != 3 || len(p.Groups) != 1 || p.Groups[0].Key != w.web.String() || p.Groups[0].Total != 3 || len(p.Groups[0].Data) != 1 || p.Groups[0].NextCursor == nil {
		t.Fatal(p)
	}
	cursor := p.Groups[0].NextCursor
	seen := []uuid.UUID{p.Groups[0].Data[0].ID}
	for cursor != nil {
		p = checkListEnvelope(t, w.send(t, "walt", http.MethodGet, path+"&group_key="+w.web.String()+"&cursor="+url.QueryEscape(*cursor), "", 200), false, false)
		if p.Total != 3 || len(p.Data) != 1 {
			t.Fatal(p)
		}
		seen = append(seen, p.Data[0].ID)
		cursor = p.NextCursor
	}
	want := slices.Clone(ids[:3])
	slices.SortFunc(want, uuid.UUID.Compare)
	slices.SortFunc(seen, uuid.UUID.Compare)
	if !slices.Equal(seen, want) {
		t.Fatal(seen)
	}
}
func TestMemberIssueVisibility(t *testing.T) {
	w := newIssueWorld(t)
	ids := memberRelations(t, w)
	// Another visible project joins the caller scope; filtering cannot grant an unseen project.
	second := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Second", "SEC")
	w.makeMember(t, "bob", second, shared.RoleMember)
	w.makeMember(t, "walt", second, shared.RoleMember)
	other := w.newWorkItemBy(t, "bob", second, `{"name":"Second"}`)
	memberListIDs(t, w, "walt", "bob", "created", "", ids[1], other)
	memberListIDs(t, w, "walt", "bob", "created", "&project_id="+second.String(), other)
	// Caller OwnOnly, even when the caller is also workspace admin and target is a full member.
	ownAssigned := w.newWorkItemBy(t, "walt", w.web, fmt.Sprintf(`{"name":"Own assigned","assignee_ids":%s,"parent_id":%q}`, idArray(w.ids["bob"]), ids[0]))
	ownSubscribed := w.newWorkItemBy(t, "walt", w.web, `{"name":"Own subscribed"}`)
	w.send(t, "bob", http.MethodPost, commentsOf(ownSubscribed), `{"comment_html":"<p>Watch</p>"}`, 201)
	pm := queryIDs(t, w.pool, "SELECT id FROM project_members WHERE project_id=$1 AND member_id=$2", w.web, w.ids["walt"])[0]
	w.send(t, "alice", http.MethodPatch, "/api/v0/project-members/"+pm.String(), `{"role":5}`, 200)
	wm := queryIDs(t, w.pool, "SELECT id FROM workspace_members WHERE workspace_id=$1 AND member_id=$2", w.acme, w.ids["walt"])[0]
	w.send(t, "alice", http.MethodPatch, "/api/v0/workspace-members/"+wm.String(), `{"role":20}`, 200)
	p := memberListIDs(t, w, "walt", "bob", "assigned", "", ownAssigned)
	if p.Data[0].ParentID != nil {
		t.Fatal("hidden parent", p)
	}
	memberListIDs(t, w, "walt", "bob", "created", "", other)
	memberListIDs(t, w, "walt", "bob", "subscribed", "", ownAssigned, ownSubscribed, other)
	// Only the guest project's own rows appear in its grouping.
	g := checkListEnvelope(t, w.send(t, "walt", http.MethodGet, memberIssuesPath(w, "bob")+"?relation=subscribed&group_by=project_id", "", 200), true, false)
	if g.Total != 3 || len(g.Groups) != 2 {
		t.Fatal(g)
	}
	// A workspace admin not joined to a project gets no implicit scope.
	memberListIDs(t, w, "wanda", "bob", "subscribed", "")
	w.send(t, "walt", http.MethodPost, "/api/v0/projects/"+second.String()+"/leave", "", 204)
	memberListIDs(t, w, "walt", "bob", "created", "&project_id="+second.String())
	for _, sql := range []string{
		"UPDATE project_members SET is_active=false WHERE project_id=$1 AND member_id=$2",
		"UPDATE project_members SET is_active=true,deleted_at=now() WHERE project_id=$1 AND member_id=$2",
	} {
		if _, err := w.pool.Exec(pgtest.Soon(t), sql, w.web, w.ids["walt"]); err != nil {
			t.Fatal(err)
		}
		memberListIDs(t, w, "walt", "bob", "subscribed", "")
	}
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE project_members SET deleted_at=NULL,is_active=true WHERE project_id=$1 AND member_id=$2", w.web, w.ids["walt"]); err != nil {
		t.Fatal(err)
	}
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/archive", "", 200)
	memberListIDs(t, w, "walt", "bob", "subscribed", "")
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/unarchive", "", 200)
	w.send(t, "alice", http.MethodDelete, "/api/v0/projects/"+w.web.String(), "", 204)
	memberListIDs(t, w, "walt", "bob", "subscribed", "")
}
func TestListMemberIssuesTargetStates(t *testing.T) {
	w := newIssueWorld(t)
	ids := memberRelations(t, w)
	path := memberIssuesPath(w, "bob") + "?relation=created"
	for _, active := range []bool{false, true} {
		if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE workspace_members SET is_active=$3 WHERE workspace_id=$1 AND member_id=$2", w.acme, w.ids["bob"], active); err != nil {
			t.Fatal(err)
		}
		memberListIDs(t, w, "walt", "bob", "created", "", ids[1])
	}
	// A target's role does not impose the caller's workspace policy.
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE workspace_members SET role=5 WHERE workspace_id=$1 AND member_id=$2", w.acme, w.ids["bob"]); err != nil {
		t.Fatal(err)
	}
	memberListIDs(t, w, "walt", "bob", "created", "", ids[1])
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE workspace_members SET deleted_at=now() WHERE workspace_id=$1 AND member_id=$2", w.acme, w.ids["bob"]); err != nil {
		t.Fatal(err)
	}
	answer := w.send(t, "walt", http.MethodGet, path, "", 404)
	if problemCode(t, []byte(answer)) != "workspace.member_not_found" {
		t.Fatal(answer)
	}
	answer = w.send(t, "walt", http.MethodGet, memberIssuesPath(w, "olga")+"?relation=created", "", 404)
	if problemCode(t, []byte(answer)) != "workspace.member_not_found" {
		t.Fatal(answer)
	}
	w.send(t, "olga", http.MethodPost, "/api/v0/workspaces", `{"name":"Other","slug":"other"}`, 201)
	answer = w.send(t, "walt", http.MethodGet, memberIssuesPath(w, "olga")+"?relation=created", "", 404)
	if problemCode(t, []byte(answer)) != "workspace.member_not_found" {
		t.Fatal(answer)
	}
	// Workspace hiding wins before the target check; a real guest is forbidden.
	for _, tt := range []struct {
		actor, path, code string
		status            int
	}{
		{"olga", path, "workspace.not_found", 404}, {"walt", "/api/v0/workspaces/missing/members/" + w.ids["bob"].String() + "/issues?relation=created", "workspace.not_found", 404},
	} {
		answer = w.send(t, tt.actor, http.MethodGet, tt.path, "", tt.status)
		if problemCode(t, []byte(answer)) != tt.code {
			t.Fatal(answer)
		}
	}
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE workspace_members SET role=5 WHERE workspace_id=$1 AND member_id=$2", w.acme, w.ids["walt"]); err != nil {
		t.Fatal(err)
	}
	answer = w.send(t, "walt", http.MethodGet, path, "", 403)
	if problemCode(t, []byte(answer)) != "forbidden" {
		t.Fatal(answer)
	}
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE workspace_members SET role=15,is_active=false WHERE workspace_id=$1 AND member_id=$2", w.acme, w.ids["walt"]); err != nil {
		t.Fatal(err)
	}
	answer = w.send(t, "walt", http.MethodGet, path, "", 404)
	if problemCode(t, []byte(answer)) != "workspace.not_found" {
		t.Fatal(answer)
	}
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE workspace_members SET is_active=true,deleted_at=now() WHERE workspace_id=$1 AND member_id=$2", w.acme, w.ids["walt"]); err != nil {
		t.Fatal(err)
	}
	answer = w.send(t, "walt", http.MethodGet, path, "", 404)
	if problemCode(t, []byte(answer)) != "workspace.not_found" {
		t.Fatal(answer)
	}

}
