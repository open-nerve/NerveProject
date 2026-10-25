package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/content"
	issuepg "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	issueapp "github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	issuedomain "github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type answeredCommentReaction struct {
	ID        uuid.UUID `json:"id"`
	CommentID uuid.UUID `json:"comment_id"`
	ActorID   uuid.UUID `json:"actor_id"`
	Reaction  string    `json:"reaction"`
	CreatedAt time.Time `json:"created_at"`
}

func commentReactionsOf(id uuid.UUID) string { return commentPath(id) + "/reactions" }
func (w issueWorld) newCommentReactionPath(t *testing.T) string {
	t.Helper()
	c := w.newComment(t, "alice", w.newWorkItem(t))
	w.send(t, "alice", http.MethodPost, commentReactionsOf(c.ID), `{"reaction":"128077"}`, 201)
	return commentReactionsOf(c.ID) + "/128077"
}
func commentRow(t *testing.T, w issueWorld, id uuid.UUID) string {
	t.Helper()
	var s string
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT to_jsonb(c)::text FROM issue_comments c WHERE id=$1", id).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}
func issueRow(t *testing.T, w issueWorld, id uuid.UUID) string {
	t.Helper()
	var s string
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT to_jsonb(c)::text FROM issues c WHERE id=$1", id).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}
func addCommentReaction(t *testing.T, w issueWorld, actor string, id uuid.UUID, value string) answeredCommentReaction {
	t.Helper()
	var r answeredCommentReaction
	decodeAnswer(t, w.send(t, actor, http.MethodPost, commentReactionsOf(id), fmt.Sprintf(`{"reaction":%q}`, value), 201), &r)
	return r
}
func TestCommentReactions(t *testing.T) {
	w := newIssueWorld(t)
	id := w.newWorkItemIn(t, w.web, w.fullIssue())
	c := w.newComment(t, "alice", id)
	beforeIssue, beforeComment := issueRow(t, w, id), commentRow(t, w, c.ID)
	activities, subscribers := reactionSnapshot(t, w, id, "issue_activities"), reactionSnapshot(t, w, id, "issue_subscribers")
	first := addCommentReaction(t, w, "bob", c.ID, "10084-65039")
	if first.ID == (uuid.UUID{}) || first.CommentID != c.ID || first.ActorID != w.ids["bob"] {
		t.Fatal(first)
	}
	w.send(t, "bob", http.MethodPost, commentReactionsOf(c.ID), `{"reaction":"10084-65039"}`, 409)
	w.send(t, "alice", http.MethodDelete, commentReactionsOf(c.ID)+"/10084-65039", "", 404)
	addCommentReaction(t, w, "alice", c.ID, "10084-65039")
	w.send(t, "bob", http.MethodDelete, commentReactionsOf(c.ID)+"/10084-65039", "", 204)
	again := addCommentReaction(t, w, "bob", c.ID, "10084-65039")
	if again.ID == first.ID {
		t.Fatal("reused deleted row")
	}
	if beforeIssue != issueRow(t, w, id) || beforeComment != commentRow(t, w, c.ID) {
		t.Fatal("reaction touched parent audit")
	}
	if activities != reactionSnapshot(t, w, id, "issue_activities") || subscribers != reactionSnapshot(t, w, id, "issue_subscribers") {
		t.Fatal("reaction changed activities or subscribers")
	}
	w.adminGuest(t)
	hidden := w.send(t, "walt", http.MethodPost, commentReactionsOf(c.ID), `{"reaction":"1"}`, 404)
	missing := w.send(t, "walt", http.MethodPost, commentReactionsOf(uuid.NewV7()), `{"reaction":"1"}`, 404)
	if hidden != missing {
		t.Fatal("comment existence leaked", hidden, missing)
	}
	for _, value := range []string{"", "-1", "12345678", "1--2"} {
		req := newRequest(t, http.MethodPost, w.base+commentReactionsOf(c.ID), w.tokens["alice"], []byte(fmt.Sprintf(`{"reaction":%q}`, value)))
		res, body := send(t, req)
		w.contract.CheckResponse(t, req, res)
		if res.StatusCode != 422 {
			t.Fatal(res.StatusCode, string(body))
		}
	}
}
func TestCommentPageIncludesReactions(t *testing.T) {
	w := newIssueWorld(t)
	id := w.newWorkItem(t)
	one, two := w.newComment(t, "alice", id), w.newComment(t, "alice", id)
	off := w.newComment(t, "alice", w.newWorkItem(t))
	first := addCommentReaction(t, w, "bob", one.ID, "128077")
	second := addCommentReaction(t, w, "alice", one.ID, "128078")
	addCommentReaction(t, w, "bob", two.ID, "1")
	addCommentReaction(t, w, "bob", off.ID, "1")
	addCommentReaction(t, w, "bob", one.ID, "2")
	w.send(t, "bob", http.MethodDelete, commentReactionsOf(one.ID)+"/2", "", 204)
	// Same creation time and reverse insertion order must still use reaction id order.
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE comment_reactions SET created_at=$2 WHERE comment_id=$1", one.ID, first.CreatedAt); err != nil {
		t.Fatal(err)
	}
	var page struct {
		Data []struct {
			ID        uuid.UUID                 `json:"id"`
			Reactions []answeredCommentReaction `json:"reactions"`
		}
		NextCursor *string `json:"next_cursor"`
	}
	decodeAnswer(t, w.send(t, "alice", http.MethodGet, commentsOf(id)+"?limit=1", "", 200), &page)
	if len(page.Data) != 1 || page.Data[0].ID != one.ID || len(page.Data[0].Reactions) != 2 || page.Data[0].Reactions[0].ID != first.ID || page.Data[0].Reactions[1].ID != second.ID || page.NextCursor == nil {
		t.Fatal("embedded reactions", page)
	}
}
func TestCommentPatchIncludesReactions(t *testing.T) {
	w := newIssueWorld(t)
	id := w.newWorkItem(t)
	c := w.newComment(t, "alice", id)
	first := addCommentReaction(t, w, "bob", c.ID, "128077")
	second := addCommentReaction(t, w, "alice", c.ID, "128078")
	addCommentReaction(t, w, "bob", c.ID, "1")
	w.send(t, "bob", http.MethodDelete, commentReactionsOf(c.ID)+"/1", "", 204)
	for _, html := range []string{c.HTML, "<p>Changed</p>"} {
		beforeIssue, beforeComment := issueRow(t, w, id), commentRow(t, w, c.ID)
		var got struct {
			HTML      string                    `json:"comment_html"`
			EditedAt  *time.Time                `json:"edited_at"`
			Reactions []answeredCommentReaction `json:"reactions"`
		}
		decodeAnswer(t, w.send(t, "alice", http.MethodPatch, commentPath(c.ID), fmt.Sprintf(`{"comment_html":%q}`, html), 200), &got)
		if len(got.Reactions) != 2 || got.Reactions[0].ID != first.ID || got.Reactions[1].ID != second.ID || got.HTML != html {
			t.Fatal("PATCH lost reactions", got)
		}
		if html == c.HTML && (issueRow(t, w, id) != beforeIssue || commentRow(t, w, c.ID) != beforeComment) {
			t.Fatal("no-op changed audit")
		}
		if html != c.HTML && (got.EditedAt == nil || issueRow(t, w, id) == beforeIssue || commentRow(t, w, c.ID) == beforeComment) {
			t.Fatal("edit did not update audit")
		}
	}
}
func TestDeletingCommentReactions(t *testing.T) {
	w := newIssueWorld(t)
	id := w.newWorkItem(t)
	one, two := w.newComment(t, "bob", id), w.newComment(t, "bob", id)
	live := []answeredCommentReaction{addCommentReaction(t, w, "alice", one.ID, "1"), addCommentReaction(t, w, "bob", one.ID, "1")}
	old := addCommentReaction(t, w, "bob", one.ID, "2")
	w.send(t, "bob", http.MethodDelete, commentReactionsOf(one.ID)+"/2", "", 204)
	neighbor := addCommentReaction(t, w, "bob", two.ID, "1")
	stamps := func() map[uuid.UUID]string {
		out := map[uuid.UUID]string{}
		for _, r := range append(append([]answeredCommentReaction{}, live...), old, neighbor) {
			var s string
			if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT to_jsonb(r)::text FROM comment_reactions r WHERE id=$1", r.ID).Scan(&s); err != nil {
				t.Fatal(err)
			}
			out[r.ID] = s
		}
		return out
	}
	before := stamps()
	w.send(t, "alice", http.MethodDelete, commentPath(one.ID), "", 204)
	after := stamps()
	if before[old.ID] != after[old.ID] || before[neighbor.ID] != after[neighbor.ID] {
		t.Fatal("delete changed neighbor or old deletion")
	}
	for _, r := range live {
		var n int
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM comment_reactions r JOIN issue_comments c ON c.id=r.comment_id WHERE r.id=$1 AND r.deleted_at=c.deleted_at AND r.updated_at=c.updated_at AND r.updated_by_id=c.updated_by_id", r.ID).Scan(&n); err != nil || n != 1 {
			t.Fatal("comment reaction deletion stamp", n, err)
		}
	}
}

// reactionReadStore keeps real SQL ports while allowing a failure or a committed
// concurrent change precisely between the comment query and reaction query.
type reactionReadStore struct {
	*issuepg.Store
	afterComments func()
	readError     error
}

func (s reactionReadStore) Comments(ctx context.Context, id uuid.UUID, after *issuedomain.CommentCursor, limit int) ([]issuedomain.Comment, error) {
	rows, err := s.Store.Comments(ctx, id, after, limit)
	if err == nil && s.afterComments != nil {
		s.afterComments()
	}
	return rows, err
}
func (s reactionReadStore) CommentReactions(ctx context.Context, ids []uuid.UUID) ([]issuedomain.CommentReaction, error) {
	if s.readError != nil {
		return nil, s.readError
	}
	return s.Store.CommentReactions(ctx, ids)
}
func TestCommentReactionReadSnapshotAndRollback(t *testing.T) {
	w := newIssueWorld(t)
	id := w.newWorkItem(t)
	c := w.newComment(t, "alice", id)
	old := addCommentReaction(t, w, "bob", c.ID, "128077")
	store := issuepg.New(w.pool)
	projects := issueProjects{projects: project.Provide(w.pool).IssueProjects}
	locks := issueapp.NewLocks(projects, store, authorizerOn(w.pool), newWriteGate(testIssueConfig.WriteGate), postgres.NewTxManager(w.pool, 2*time.Second))
	ctx := shared.WithActor(pgtest.Soon(t), shared.Actor{UserID: w.ids["alice"]})
	// A separately committed change after the directory/page read is invisible
	// to the batch reaction read of this list's repeatable-read transaction.
	changed := false
	wrapped := reactionReadStore{Store: store, afterComments: func() {
		changed = true
		w.send(t, "bob", http.MethodDelete, commentReactionsOf(c.ID)+"/128077", "", 204)
		addCommentReaction(t, w, "alice", c.ID, "1")
	}}
	list := issueapp.NewListIssueComments(issueapp.ListIssueCommentsDeps{Locks: locks, Comments: wrapped, ReadTx: postgres.NewTxManager(w.pool, 2*time.Second, postgres.ReadOnlySnapshot())})
	page, err := list.Execute(ctx, id, nil, nil)
	if err != nil || !changed || len(page.Data) != 1 || len(page.Data[0].Reactions) != 1 || page.Data[0].Reactions[0].ID != old.ID {
		t.Fatal("reaction read left list snapshot", page, err)
	}
	sentinel := errors.New("reaction batch failed")
	wrapped = reactionReadStore{Store: store, readError: sentinel}
	edit := issueapp.NewUpdateIssueComment(issueapp.UpdateIssueCommentDeps{Locks: locks, Comments: wrapped, Members: projects, Sanitizer: content.NewSanitizer(), Clock: clock.System{}})
	for _, html := range []string{c.HTML, "<p>Must roll back</p>"} {
		beforeIssue, beforeComment := issueRow(t, w, id), commentRow(t, w, c.ID)
		beforeSubs := reactionSnapshot(t, w, id, "issue_subscribers")
		_, err := edit.Execute(ctx, c.ID, html)
		if err != sentinel {
			t.Fatal("read error was changed", err)
		}
		if beforeIssue != issueRow(t, w, id) || beforeComment != commentRow(t, w, c.ID) || beforeSubs != reactionSnapshot(t, w, id, "issue_subscribers") {
			t.Fatal("reaction read failure committed edit")
		}
	}
}
