package bootstrap

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

func commentPath(id uuid.UUID) string { return "/api/v0/issue-comments/" + id.String() }
func (w issueWorld) newComment(t *testing.T, by string, id uuid.UUID) answeredComment {
	t.Helper()
	var c answeredComment
	decodeAnswer(t, w.send(t, by, http.MethodPost, commentsOf(id), `{"comment_html":"<p>Before</p>"}`, 201), &c)
	return c
}

func TestCommentPermissions(t *testing.T) {
	w := newIssueWorld(t)

	id := w.newWorkItem(t)
	c := w.newComment(t, "bob", id)
	// The issue creator and administrator is not the comment author.
	w.send(t, "alice", http.MethodPatch, commentPath(c.ID), `{"comment_html":"<p>Denied</p>"}`, 403)
	var edited answeredComment
	decodeAnswer(t, w.send(t, "bob", http.MethodPatch, commentPath(c.ID), fmt.Sprintf(`{"comment_html":%q}`, mentionHTML(w.ids["alice"])), 200), &edited)
	if edited.ActorID != w.ids["bob"] || edited.EditedAt == nil || !edited.EditedAt.Equal(edited.UpdatedAt) || !edited.CreatedAt.Equal(c.CreatedAt) {
		t.Fatal(edited)
	}
	var by uuid.UUID
	var at time.Time
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT updated_by_id,updated_at FROM issues WHERE id=$1", id).Scan(&by, &at); err != nil || by != w.ids["bob"] || !at.Equal(edited.UpdatedAt) {
		t.Fatal("issue stamp", by, at, err)
	}
	var before, after string
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT to_jsonb(c)::text FROM issue_comments c WHERE id=$1", c.ID).Scan(&before); err != nil {
		t.Fatal(err)
	}
	decodeAnswer(t, w.send(t, "bob", http.MethodPatch, commentPath(c.ID), fmt.Sprintf(`{"comment_html":%q}`, edited.HTML+"<script>removed()</script>"), 200), &edited)
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT to_jsonb(c)::text FROM issue_comments c WHERE id=$1", c.ID).Scan(&after); err != nil || before != after {
		t.Fatal("no-op changed row", err)
	}
	neighbor := w.newComment(t, "bob", id)
	w.send(t, "alice", http.MethodDelete, commentPath(c.ID), "", 204)
	w.send(t, "bob", http.MethodPatch, commentPath(c.ID), `{"comment_html":"<p>Gone</p>"}`, 404)
	w.send(t, "bob", http.MethodDelete, commentPath(c.ID), "", 404)
	w.send(t, "bob", http.MethodDelete, commentPath(neighbor.ID), "", 204)
	w.adminGuest(t)
	own := w.newWorkItemBy(t, "walt", w.web, `{"name":"Own"}`)
	ownComment := w.newComment(t, "alice", own)
	w.send(t, "walt", http.MethodDelete, commentPath(ownComment.ID), "", 204)
	hidden := w.newComment(t, "alice", id)
	for _, method := range []string{http.MethodPatch, http.MethodDelete} {
		body := `{"comment_html":"<p>Hidden</p>"}`
		if method == http.MethodDelete {
			body = ""
		}
		h := w.send(t, "walt", method, commentPath(hidden.ID), body, 404)
		m := w.send(t, "walt", method, commentPath(uuid.NewV7()), body, 404)
		if !reflect.DeepEqual(h, m) {
			t.Fatal("child existence leaked", h, m)
		}
	}
}

func TestCommentPermissionsRefuseBeforeExternalProjectLock(t *testing.T) {
	w := newIssueWorldWith(t, func(cfg *config.Config) { cfg.Issue.WriteLockTimeout = 3 * time.Second })
	id := w.newWorkItem(t)
	c := w.newComment(t, "alice", id)
	held := holding(t, w.pool, "SELECT 1 FROM projects WHERE id=$1 FOR UPDATE", w.web)
	for _, method := range []string{http.MethodPatch, http.MethodDelete} {
		req := newRequest(t, method, w.base+commentPath(c.ID), w.tokens["bob"], []byte(`{"comment_html":"<p>No</p>"}`))
		if method == http.MethodDelete {
			req = newRequest(t, method, w.base+commentPath(c.ID), w.tokens["bob"], nil)
		}
		a := receiveWithin(t, sendInBackground(req), time.Second, "non-author precheck")
		if a.err != nil || a.res.StatusCode != 403 {
			t.Fatal(a.err, a.res, string(a.body))
		}
		w.contract.CheckResponse(t, req, a.res)
	}
	if err := held.Rollback(pgtest.Soon(t)); err != nil {
		t.Fatal(err)
	}
}

func TestCommentPermissionsEditSubscriptionsAndDeletionAudit(t *testing.T) {
	w := newIssueWorld(t)
	w.makeMember(t, "walt", w.web, 15)
	id := w.newWorkItem(t)
	activityRows := func() string {
		t.Helper()
		var rows string
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT coalesce(jsonb_agg(to_jsonb(a) ORDER BY id),'[]'::jsonb)::text FROM issue_activities a WHERE issue_id=$1", id).Scan(&rows); err != nil {
			t.Fatal(err)
		}
		return rows
	}
	activitiesBefore := activityRows()
	c := w.newComment(t, "bob", id)
	old := w.newComment(t, "alice", id)
	w.send(t, "alice", http.MethodDelete, commentPath(old.ID), "", 204)
	neighbor := w.newComment(t, "alice", id)
	snapshot := func(table string, id uuid.UUID) string {
		t.Helper()
		var row string
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT to_jsonb(r)::text FROM "+table+" r WHERE id=$1", id).Scan(&row); err != nil {
			t.Fatal(err)
		}
		return row
	}
	oldAudit, neighborAudit := snapshot("issue_comments", old.ID), snapshot("issue_comments", neighbor.ID)
	before := queryIDs(t, w.pool, "SELECT id FROM issue_subscribers WHERE issue_id=$1 ORDER BY id", id)
	stamps := make(map[uuid.UUID]string)
	for _, s := range before {
		stamps[s] = snapshot("issue_subscribers", s)
	}
	w.send(t, "bob", http.MethodPatch, commentPath(c.ID), fmt.Sprintf(`{"comment_html":%q}`, mentionHTML(w.ids["walt"], w.ids["olga"])), 200)
	subs := queryIDs(t, w.pool, "SELECT subscriber_id FROM issue_subscribers WHERE issue_id=$1 AND deleted_at IS NULL", id)
	if len(subs) != 3 {
		t.Fatal("only new visible member should subscribe", subs)
	}
	for _, s := range before {
		if snapshot("issue_subscribers", s) != stamps[s] {
			t.Fatal("existing subscription audit changed")
		}
	}
	w.send(t, "bob", http.MethodPatch, commentPath(c.ID), `{"comment_html":"<p>Removed mention</p>"}`, 200)
	if len(queryIDs(t, w.pool, "SELECT id FROM issue_subscribers WHERE issue_id=$1 AND deleted_at IS NULL", id)) != 3 {
		t.Fatal("mention removal unsubscribed")
	}
	w.send(t, "bob", http.MethodDelete, commentPath(c.ID), "", 204)
	if oldAudit != snapshot("issue_comments", old.ID) || neighborAudit != snapshot("issue_comments", neighbor.ID) {
		t.Fatal("deletion changed neighboring audit")
	}
	if activitiesBefore != activityRows() {
		t.Fatal("comment write added activity")
	}
	var n int
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM issue_comments c JOIN issues i ON i.id=c.issue_id WHERE c.id=$1 AND c.deleted_at=c.updated_at AND c.updated_by_id=i.updated_by_id AND c.updated_at=i.updated_at AND c.created_at<>c.deleted_at", c.ID).Scan(&n); err != nil || n != 1 {
		t.Fatal("deletion stamp", n, err)
	}
}

func TestCommentPermissionsArchiveBeforeUnchangedHTML(t *testing.T) {
	w := newIssueWorld(t)
	id := w.doneWorkItem(t)
	c := w.newComment(t, "bob", id)
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+id.String()+"/archive", "", 200)
	for _, method := range []string{http.MethodPatch, http.MethodDelete} {
		body := `{"comment_html":"<p>Before</p>"}`
		if method == http.MethodDelete {
			body = ""
		}
		answer := w.send(t, "bob", method, commentPath(c.ID), body, 409)
		if refusalOf(t, []byte(answer), false) != "issue.archived" {
			t.Fatal(answer)
		}
	}
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/archive", "", 200)
	for _, method := range []string{http.MethodPatch, http.MethodDelete} {
		body := `{"comment_html":"<p>Before</p>"}`
		if method == http.MethodDelete {
			body = ""
		}
		answer := w.send(t, "bob", method, commentPath(c.ID), body, 409)
		if refusalOf(t, []byte(answer), false) != "project.archived" {
			t.Fatal(answer)
		}
	}
}
