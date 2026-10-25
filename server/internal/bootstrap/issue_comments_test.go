package bootstrap

import (
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type answeredComment struct {
	ID        uuid.UUID  `json:"id"`
	IssueID   uuid.UUID  `json:"issue_id"`
	ActorID   uuid.UUID  `json:"actor_id"`
	HTML      string     `json:"comment_html"`
	EditedAt  *time.Time `json:"edited_at"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	Reactions []any      `json:"reactions"`
}

func commentsOf(id uuid.UUID) string { return "/api/v0/issues/" + id.String() + "/comments" }
func mentionHTML(ids ...uuid.UUID) string {
	html := "<p>Hello</p>"
	for _, id := range ids {
		html += `<mention-component entity_name="user_mention" entity_identifier="` + id.String() + `"></mention-component>`
	}
	return html
}

func TestCommentsCreateAndMentionVisibility(t *testing.T) {
	w := newIssueWorld(t)
	// walt is an admin of the workspace who is a project guest, and wanda
	// an ordinary project guest. Neither sees alice's issue while the flag is false.
	inWorkspaceOf(t, w.pool, w.web, w.ids["olga"], w.ids["alice"], shared.RoleMember)
	w.makeMember(t, "olga", w.web, shared.RoleMember)
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE project_members SET is_active=false WHERE project_id=$1 AND member_id=$2", w.web, w.ids["olga"]); err != nil {
		t.Fatal(err)
	}
	w.adminGuest(t)
	membership := queryIDs(t, w.pool, "SELECT id FROM workspace_members WHERE workspace_id=$1 AND member_id=$2", w.acme, w.ids["wanda"])[0]
	w.send(t, "alice", http.MethodPatch, "/api/v0/workspace-members/"+membership.String(), `{"role":15}`, http.StatusOK)
	w.makeMember(t, "wanda", w.web, shared.RoleGuest)
	id := w.newWorkItemIn(t, w.web, `{"name":"Comments"}`)
	before := queryIDs(t, w.pool, "SELECT id FROM issue_subscribers WHERE issue_id=$1", id)
	var old string
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT to_jsonb(s)::text FROM issue_subscribers s WHERE issue_id=$1 AND subscriber_id=$2", id, w.ids["alice"]).Scan(&old); err != nil {
		t.Fatal(err)
	}
	// Reverse order, repetitions, a non-member and both hidden guest roles.
	html := mentionHTML(w.ids["olga"], w.ids["wanda"], w.ids["walt"], w.ids["bob"], w.ids["bob"])
	var c answeredComment
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, commentsOf(id), fmt.Sprintf(`{"comment_html":%q}`, html), 201), &c)
	if c.ActorID != w.ids["alice"] || c.IssueID != id || c.EditedAt != nil || len(c.Reactions) != 0 || c.Reactions == nil || !c.CreatedAt.Equal(c.UpdatedAt) {
		t.Fatal(c)
	}
	got := queryIDs(t, w.pool, "SELECT subscriber_id FROM issue_subscribers WHERE issue_id=$1 AND deleted_at IS NULL ORDER BY subscriber_id", id)
	want := []uuid.UUID{w.ids["alice"], w.ids["bob"]}
	slices.SortFunc(want, uuid.UUID.Compare)
	if !slices.Equal(got, want) {
		t.Fatalf("subscription visibility %v, want %v", got, want)
	}
	var after string
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT to_jsonb(s)::text FROM issue_subscribers s WHERE id=$1", before[0]).Scan(&after); err != nil || after != old {
		t.Fatalf("existing subscription changed: %s, %v", after, err)
	}
	for _, table := range []string{"issues", "issue_subscribers", "issue_comments"} {
		var n int
		condition := "id=$1"
		if table != "issues" {
			condition = "issue_id=$1"
		}
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM "+table+" WHERE "+condition+" AND updated_by_id=$2 AND updated_at=$3", id, w.ids["alice"], c.CreatedAt).Scan(&n); err != nil || n != 1 {
			t.Fatalf("%s stamp: %d, %v", table, n, err)
		}
	}
	if activities := queryIDs(t, w.pool, "SELECT id FROM issue_activities WHERE issue_id=$1", id); len(activities) != 1 {
		t.Fatal("comment created activity", activities)
	}
	// Once the project opens guests' visibility, both can subscribe.
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"guest_view_all_features":true}`, 200)
	w.send(t, "alice", http.MethodPost, commentsOf(id), fmt.Sprintf(`{"comment_html":%q}`, mentionHTML(w.ids["walt"], w.ids["wanda"])), 201)
	got = queryIDs(t, w.pool, "SELECT subscriber_id FROM issue_subscribers WHERE issue_id=$1 AND deleted_at IS NULL ORDER BY subscriber_id", id)
	if len(got) != 4 {
		t.Fatal(got)
	}
}

func TestCommentsListUsesIssuePermissionAndPositionOnly(t *testing.T) {
	w := newIssueWorld(t)
	id, other := w.newWorkItem(t), w.newWorkItem(t)
	var cs [3]answeredComment
	for n := range cs {
		decodeAnswer(t, w.send(t, "alice", http.MethodPost, commentsOf(id), fmt.Sprintf(`{"comment_html":"<p>%d</p>"}`, n), 201), &cs[n])
	}
	w.send(t, "alice", http.MethodPost, commentsOf(other), `{"comment_html":"<p>Other</p>"}`, 201)
	// Same timestamps and reverse ids ensure page order comes from (at,id).
	for _, c := range cs {
		if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE issue_comments SET created_at=$2 WHERE id=$1", c.ID, cs[0].CreatedAt); err != nil {
			t.Fatal(err)
		}
	}
	var page struct {
		Data []answeredComment
		Next *string `json:"next_cursor"`
	}
	decodeAnswer(t, w.send(t, "alice", http.MethodGet, commentsOf(id)+"?limit=1", "", 200), &page)
	if len(page.Data) != 1 || page.Next == nil || page.Data[0].ID != cs[0].ID {
		t.Fatal(page)
	}
	decodeAnswer(t, w.send(t, "alice", http.MethodGet, commentsOf(id)+"?limit=1&cursor="+*page.Next, "", 200), &page)
	if len(page.Data) != 1 || page.Data[0].ID != cs[1].ID {
		t.Fatal(page)
	}
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE issue_comments SET deleted_at=now() WHERE id=$1", cs[2].ID); err != nil {
		t.Fatal(err)
	}
	decodeAnswer(t, w.send(t, "alice", http.MethodGet, commentsOf(id)+"?limit=100", "", 200), &page)
	if len(page.Data) != 2 || page.Next != nil {
		t.Fatal(page)
	}
	w.makeMember(t, "walt", w.web, shared.RoleGuest)
	// A valid cursor obtained by another reader never authorizes its target.
	cursor, err := shared.EncodeCursor(domain.CommentCursor{CreatedAt: cs[0].CreatedAt, ID: cs[0].ID})
	if err != nil {
		t.Fatal(err)
	}
	w.send(t, "walt", http.MethodGet, commentsOf(id)+"?cursor="+cursor, "", 404)
	for _, tt := range []struct {
		query  string
		status int
	}{{"?cursor=broken&limit=0", 400}, {"?limit=0", 422}} {
		req := newRequest(t, http.MethodGet, w.base+commentsOf(id)+tt.query, w.tokens["alice"], nil)
		res, body := send(t, req)
		w.contract.CheckResponse(t, req, res)
		if res.StatusCode != tt.status {
			t.Fatal(res.StatusCode, string(body))
		}
	}
}
