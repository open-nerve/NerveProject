package bootstrap

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/notification"
	notificationpg "github.com/open-nerve/NerveProject/server/internal/modules/notification/adapter/postgres"
	notificationapp "github.com/open-nerve/NerveProject/server/internal/modules/notification/app"
	notificationdomain "github.com/open-nerve/NerveProject/server/internal/modules/notification/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// newTestFanout is the notification module's fan-out over pool as New builds it from notificationDeps, its
// transactions the work item's writes' (testIssueConfig's lock timeout).
func newTestFanout(pool *pgxpool.Pool) *notificationapp.Fanout {
	d := notificationDeps(pool, config.Config{Database: config.DatabaseConfig{CommitTimeout: 2 * time.Second}, Issue: testIssueConfig})
	return notificationapp.NewFanout(notificationapp.FanoutDeps{Lock: d.Issues, Issues: d.Issues, Projects: d.Projects, Readers: d.Readers,
		GuestRule: d.Issues, Rows: notificationpg.New(pool), Tx: d.JobTx, Logger: slog.New(slog.DiscardHandler)})
}

// sentNotification is a notification as the table holds it, its id aside: its receiver, sender, position and snapshot,
// and the columns every notification of one fan-out shares.
type sentNotification struct {
	Receiver                     uuid.UUID
	Sender                       string
	Seq                          int16
	Data                         notificationdomain.Data
	Workspace, Project, Issue    uuid.UUID
	Event                        uuid.UUID
	TriggeredBy, IssueCreatedBy  *uuid.UUID
	Created, Updated             time.Time
	Audit, Read, Archive, Delete bool
}

// sentNotifications reads the work item's notifications in the order of their events, receivers and positions, and
// their ids.
func sentNotifications(t *testing.T, pool *pgxpool.Pool, item uuid.UUID, events ...uuid.UUID) ([]uuid.UUID, []sentNotification) {
	t.Helper()
	rows, err := pool.Query(pgtest.Soon(t), `SELECT id, receiver_id, sender, change_seq, data, workspace_id, project_id, issue_id, event_id, triggered_by_id,
		issue_created_by_id, created_at, updated_at, created_by_id IS NOT NULL OR updated_by_id IS NOT NULL, read_at IS NOT NULL,
		archived_at IS NOT NULL OR snoozed_till IS NOT NULL, deleted_at IS NOT NULL
		FROM notifications WHERE issue_id = $1 ORDER BY array_position($2::uuid[], event_id), receiver_id, change_seq`, item, events)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var ids []uuid.UUID
	var got []sentNotification
	for rows.Next() {
		var id uuid.UUID
		var n sentNotification
		var data []byte
		if err := rows.Scan(&id, &n.Receiver, &n.Sender, &n.Seq, &data, &n.Workspace, &n.Project, &n.Issue, &n.Event, &n.TriggeredBy,
			&n.IssueCreatedBy, &n.Created, &n.Updated, &n.Audit, &n.Read, &n.Archive, &n.Delete); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, &n.Data); err != nil {
			t.Fatal(err)
		}
		n.Created, n.Updated = n.Created.UTC(), n.Updated.UTC()
		ids, got = append(ids, id), append(got, n)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return ids, got
}

// positionOf is the position of e's first change of field naming id, failing the test when there is none.
func positionOf(t *testing.T, e shared.Event, field string, id uuid.UUID) int16 {
	t.Helper()
	for n, c := range e.Changes {
		if c.Field == field && c.NewID != nil && *c.NewID == id {
			return int16(n)
		}
	}
	t.Fatalf("%s has no %s change of %s: %+v", e.Type, field, id, e.Changes)
	return 0
}

// A work item's fan-out through the ports newApp gives the module (M7 P1b spec 2.4–2.7, 2.9), of the events the issue
// module publishes of writes through the API: alice creates a work item assigned to bob whose description mentions
// walt, a member of Web, and wanda, a guest of Web who sees only the work items she created; bob comments on it,
// mentioning alice. Each event's fan-out runs while writes hold Web's and acme's rows FOR NO KEY UPDATE, and takes no
// lock they hold. Of the creation, bob, its assignee, receives its creation and his assignment, and walt his mention;
// not alice, who wrote it, nor wanda, from whom the recheck hides it. Of the comment, alice receives her mention and
// walt, a subscriber since his mention, the comment, both with its excerpt; not bob, who wrote it. Each row is sent at
// its event's moment, by its actor, of the change at its position, with the work item's snapshot completed with its
// state's and its project's names, and no audit account. Each fan-out run again sends none twice.
func TestFanoutSendsAWorkItemsNotificationsOnce(t *testing.T) {
	w := newIssueWorld(t)
	alice, bob, walt, wanda := w.ids["alice"], w.ids["bob"], w.ids["walt"], w.ids["wanda"]
	w.makeMember(t, "walt", w.web, shared.RoleMember)
	membership := queryIDs(t, w.pool, "SELECT id FROM workspace_members WHERE workspace_id=$1 AND member_id=$2", w.acme, wanda)[0]
	w.send(t, "alice", http.MethodPatch, "/api/v0/workspace-members/"+membership.String(), `{"role":15}`, http.StatusOK)
	w.makeMember(t, "wanda", w.web, shared.RoleGuest)
	at := time.Date(2044, 5, 6, 7, 8, 9, 123456000, time.UTC)
	var events []shared.Event
	var failure error
	route := issueEventRouteFor(t, w, clockAt(at), &events, &failure, []string{"issue.created", "issue.comment_created"}, nil)
	var item struct {
		ID uuid.UUID `json:"id"`
	}
	body := fmt.Sprintf(`{"name":"Fix the login","state_id":%q,"assignee_ids":[%q],"description_html":%q}`, w.todo, bob, mentionHTML(walt, wanda))
	decodeAnswer(t, route.answerWithin(t, w.contract, http.MethodPost, "/api/v0/projects/"+w.web.String()+"/issues", alice, body, 201), &item)
	var comment answeredComment
	decodeAnswer(t, route.answerWithin(t, w.contract, http.MethodPost, commentsOf(item.ID), bob, fmt.Sprintf(`{"comment_html":%q}`, mentionHTML(alice)), 201),
		&comment)
	if len(events) != 2 {
		t.Fatalf("events %+v; want the creation and the comment", events)
	}
	created, commented := events[0], events[1]
	positionOf(t, created, "mention", wanda)
	fanout := newTestFanout(w.pool)
	held := []pgx.Tx{holding(t, w.pool, "SELECT 1 FROM projects WHERE id=$1 FOR NO KEY UPDATE", w.web),
		holding(t, w.pool, "SELECT 1 FROM workspaces WHERE id=$1 FOR NO KEY UPDATE", w.acme)}
	for _, e := range events {
		if err := fanout.Execute(t.Context(), e); err != nil {
			t.Fatalf("%s's fan-out while the project and the workspace are locked: %v", e.Type, err)
		}
	}
	for _, tx := range held {
		if err := tx.Rollback(pgtest.Soon(t)); err != nil {
			t.Fatal(err)
		}
	}
	var group, excerpt string
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT (SELECT "group" FROM states WHERE id=$1), (SELECT comment_stripped FROM issue_comments WHERE id=$2)`,
		w.todo, comment.ID).Scan(&group, &excerpt); err != nil || excerpt == "" {
		t.Fatalf("the state's group %q and the comment's text %q: %v", group, excerpt, err)
	}
	snapshot := notificationdomain.IssueSnapshot{ID: item.ID, Name: "Fix the login", SequenceID: 1, Identifier: "WEB", StateName: "Todo", StateGroup: group}
	sent := func(e shared.Event, receiver uuid.UUID, sender string, seq int16, change notificationdomain.ChangeSnapshot) sentNotification {
		return sentNotification{Receiver: receiver, Sender: sender, Seq: seq, Data: notificationdomain.Data{Issue: snapshot, Change: change},
			Workspace: w.acme, Project: w.web, Issue: item.ID, Event: e.ID, TriggeredBy: e.ActorID, IssueCreatedBy: &alice, Created: at, Updated: at}
	}
	assigned := positionOf(t, created, "assignees", bob)
	want := []sentNotification{
		sent(created, bob, "assigned", -1, notificationdomain.ChangeSnapshot{Field: "issue", Verb: "created"}),
		sent(created, bob, "assigned", assigned, notificationdomain.ChangeSnapshot{Field: "assignees", Verb: "created",
			NewValue: created.Changes[assigned].New, NewID: &bob}),
		sent(created, walt, "mentioned", positionOf(t, created, "mention", walt), notificationdomain.ChangeSnapshot{Field: "description", Verb: "mentioned"}),
		sent(commented, alice, "mentioned", positionOf(t, commented, "mention", alice), notificationdomain.ChangeSnapshot{Field: "comment",
			Verb: "mentioned", CommentID: &comment.ID, Excerpt: &excerpt}),
		sent(commented, walt, "subscribed", 0, notificationdomain.ChangeSnapshot{Field: "comment", Verb: "created", CommentID: &comment.ID,
			Excerpt: &excerpt}),
	}
	if bob.Compare(walt) > 0 {
		want[0], want[1], want[2] = want[2], want[0], want[1]
	}
	if alice.Compare(walt) > 0 {
		want[3], want[4] = want[4], want[3]
	}
	if commented.Changes[0].Field != "comment_html" {
		t.Fatalf("the comment's changes %+v; want its HTML first", commented.Changes)
	}
	ids, got := sentNotifications(t, w.pool, item.ID, created.ID, commented.ID)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("notifications\n%+v\nwant\n%+v", got, want)
	}
	for _, e := range events {
		if err := fanout.Execute(t.Context(), e); err != nil {
			t.Fatal(err)
		}
	}
	if again, got := sentNotifications(t, w.pool, item.ID, created.ID, commented.ID); !slices.Equal(again, ids) || !reflect.DeepEqual(got, want) {
		t.Fatalf("each fan-out again: %v %+v; want the first copies alone %v", again, got, ids)
	}
	names := notificationProjects{projects: project.Provide(w.pool, nil).IssueProjects}
	if s, found, err := names.State(t.Context(), w.web, w.todo); err != nil || !found || s != (notification.StateName{Name: "Todo", Group: group}) {
		t.Errorf("State(Todo) = %+v, %v, %v", s, found, err)
	}
	if s, found, err := names.State(t.Context(), w.web, uuid.NewV7()); err != nil || found || s != (notification.StateName{}) {
		t.Errorf("State(none) = %+v, %v, %v; want none", s, found, err)
	}
	if id, found, err := names.Identifier(t.Context(), uuid.NewV7(), w.web); err != nil || found || id != "" {
		t.Errorf("Identifier(another workspace's) = %q, %v, %v; want none", id, found, err)
	}
}
