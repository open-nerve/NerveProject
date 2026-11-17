package postgresadapter_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// A project's work items remain while one of them is undeleted, archived or not, and a sub-item too; deleted ones,
// another project's and a project without any do not count (M7 P1b spec 2.6, 2.8). It reads what is committed.
func TestProjectIssuesRemain(t *testing.T) {
	s, pool := newStore(t)
	by := newAccount(t, pool, "remain@corp.com")
	acme := newWorkspace(t, pool, "remain")
	web, ops, empty := newProject(t, pool, acme, "WEB"), newProject(t, pool, acme, "OPS"), newProject(t, pool, acme, "EMP")
	parent, child, shelved, far := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	newIssue(t, pool, web, parent, 1, by)
	newIssue(t, pool, web, child, 2, by)
	newIssue(t, pool, web, shelved, 3, by)
	newIssue(t, pool, ops, far, 1, by)
	exec(t, pool, "UPDATE issues SET parent_id = $1 WHERE id = $2", parent, child)
	exec(t, pool, "UPDATE issues SET archived_at = now() WHERE id = $1", shelved)
	remain := func(p project, want bool) {
		t.Helper()
		if got, err := s.ProjectIssuesRemain(context.Background(), p.id); err != nil || got != want {
			t.Fatalf("project %s remain=%t,%v; want %t", p.id, got, err, want)
		}
	}
	remain(web, true)
	remain(empty, false)
	for _, id := range []uuid.UUID{parent, child} {
		exec(t, pool, "UPDATE issues SET deleted_at = now() WHERE id = $1", id)
		remain(web, true)
	}
	exec(t, pool, "UPDATE issues SET deleted_at = now() WHERE id = $1", shelved)
	remain(web, false)
	remain(ops, true)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.ProjectIssuesRemain(cancelled, ops.id); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled read=%v", err)
	}
}

// A work item's subscribers are the accounts of its undeleted rows, each once, in id order (M7 P1b spec 2.6): a row
// deleted by an unsubscription does not count, an account that subscribed again counts once, another work item's
// subscribers do not count, and a work item without any has none. It reads what is committed, without a lock.
func TestSubscriberIDs(t *testing.T) {
	s, pool := newStore(t)
	by := newAccount(t, pool, "subscribers@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "subscribers"), "WEB")
	item, other, quiet := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	for n, i := range []uuid.UUID{item, other, quiet} {
		newIssue(t, pool, web, i, n+1, by)
	}
	first, second, again, gone, elsewhere := newAccount(t, pool, "first@corp.com"), newAccount(t, pool, "second@corp.com"),
		newAccount(t, pool, "again@corp.com"), newAccount(t, pool, "gone@corp.com"), newAccount(t, pool, "elsewhere@corp.com")
	subscribe := func(issue, account uuid.UUID, deleted bool) {
		exec(t, pool, `INSERT INTO issue_subscribers (id, workspace_id, project_id, issue_id, subscriber_id, deleted_at)
			VALUES ($1, $2, $3, $4, $5, CASE WHEN $6 THEN now() END)`, uuid.NewV7(), web.workspace, web.id, issue, account, deleted)
	}
	subscribe(item, again, true)
	subscribe(item, second, false)
	subscribe(item, gone, true)
	subscribe(item, first, false)
	subscribe(item, again, false)
	subscribe(other, elsewhere, false)
	for _, tt := range []struct {
		issue uuid.UUID
		want  []uuid.UUID
	}{{item, []uuid.UUID{first, second, again}}, {other, []uuid.UUID{elsewhere}}, {quiet, nil}} {
		if got, err := s.SubscriberIDs(context.Background(), tt.issue); err != nil || !slices.Equal(got, tt.want) {
			t.Errorf("SubscriberIDs(%s) = %v, %v; want %v", tt.issue, got, err, tt.want)
		}
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.SubscriberIDs(cancelled, item); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled read=%v", err)
	}
}

// A work item as the notification module reads it, its own columns (M7 P1b spec 2.6): its name, its number, its state,
// its creator, nil when not known, and its current assignees, the accounts of its undeleted assignees' rows, each once,
// in id order: one removed does not count, one removed and assigned again counts once, another work item's do not. An
// archived work item is read; a deleted one, or none, is not found. It reads what is committed, without a lock.
func TestNotificationIssue(t *testing.T) {
	s, pool := newStore(t)
	by := newAccount(t, pool, "facts@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "facts"), "WEB")
	item, other, quiet := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	for n, i := range []uuid.UUID{item, other, quiet} {
		newIssue(t, pool, web, i, n+7, by)
	}
	exec(t, pool, "UPDATE issues SET name = 'Fix the login', archived_at = now() WHERE id = $1", item)
	exec(t, pool, "UPDATE issues SET created_by_id = NULL WHERE id = $1", quiet)
	first, second, again, removed, elsewhere := newAccount(t, pool, "first@corp.com"), newAccount(t, pool, "second@corp.com"),
		newAccount(t, pool, "again@corp.com"), newAccount(t, pool, "removed@corp.com"), newAccount(t, pool, "elsewhere@corp.com")
	assign := func(issue, account uuid.UUID, deleted bool) {
		exec(t, pool, `INSERT INTO issue_assignees (id, workspace_id, project_id, issue_id, assignee_id, deleted_at)
			VALUES ($1, $2, $3, $4, $5, CASE WHEN $6 THEN now() END)`, uuid.NewV7(), web.workspace, web.id, issue, account, deleted)
	}
	assign(item, again, true)
	assign(item, second, false)
	assign(item, removed, true)
	assign(item, first, false)
	assign(item, again, false)
	assign(other, elsewhere, false)
	for _, want := range []app.NotificationIssue{
		{Name: "Fix the login", SequenceID: 7, StateID: web.state, CreatedByID: &by, AssigneeIDs: []uuid.UUID{first, second, again}},
		{Name: "Work", SequenceID: 8, StateID: web.state, CreatedByID: &by, AssigneeIDs: []uuid.UUID{elsewhere}},
		{Name: "Work", SequenceID: 9, StateID: web.state},
	} {
		id := map[int]uuid.UUID{7: item, 8: other, 9: quiet}[want.SequenceID]
		got, found, err := s.NotificationIssue(context.Background(), id)
		if err != nil || !found || got.Name != want.Name || got.SequenceID != want.SequenceID || got.StateID != want.StateID ||
			!reflect.DeepEqual(got.CreatedByID, want.CreatedByID) || !slices.Equal(got.AssigneeIDs, want.AssigneeIDs) {
			t.Errorf("NotificationIssue(%s) = %+v, %v, %v; want %+v", id, got, found, err, want)
		}
	}
	exec(t, pool, "UPDATE issues SET deleted_at = now() WHERE id = $1", item)
	for _, id := range []uuid.UUID{item, uuid.NewV7()} {
		if got, found, err := s.NotificationIssue(context.Background(), id); err != nil || found || !reflect.DeepEqual(got, app.NotificationIssue{}) {
			t.Errorf("NotificationIssue(%s) = %+v, %v, %v; want none", id, got, found, err)
		}
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := s.NotificationIssue(cancelled, other); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled read=%v", err)
	}
}

// A comment's text is the plain text it stores, comment_stripped, empty too (M7 P1b spec 2.5, 2.6, R6); a deleted
// comment, or none, is not found. It reads what is committed, without a lock.
func TestCommentText(t *testing.T) {
	s, pool := newStore(t)
	by := newAccount(t, pool, "text@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "text"), "WEB")
	item := uuid.NewV7()
	newIssue(t, pool, web, item, 1, by)
	said, blank := uuid.NewV7(), uuid.NewV7()
	for id, stripped := range map[uuid.UUID]string{said: "Hi there", blank: ""} {
		exec(t, pool, `INSERT INTO issue_comments (id, workspace_id, project_id, issue_id, actor_id, comment_html, comment_stripped)
			VALUES ($1, $2, $3, $4, $5, '<p>Hi <strong>there</strong></p>', $6)`, id, web.workspace, web.id, item, by, stripped)
	}
	for id, want := range map[uuid.UUID]string{said: "Hi there", blank: ""} {
		if got, found, err := s.CommentText(context.Background(), id); err != nil || !found || got != want {
			t.Errorf("CommentText(%s) = %q, %v, %v; want %q", id, got, found, err, want)
		}
	}
	exec(t, pool, "UPDATE issue_comments SET deleted_at = now() WHERE id = $1", said)
	for _, id := range []uuid.UUID{said, uuid.NewV7()} {
		if got, found, err := s.CommentText(context.Background(), id); err != nil || found || got != "" {
			t.Errorf("CommentText(%s) = %q, %v, %v; want none", id, got, found, err)
		}
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := s.CommentText(cancelled, blank); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled read=%v", err)
	}
}
