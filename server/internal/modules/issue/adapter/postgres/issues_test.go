package postgresadapter_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// linkRows are the rows of a link table under the work item issue of p,
// one per target, each with a fresh id, by by at at.
func linkRows(p project, issue uuid.UUID, by uuid.UUID, at time.Time, targets ...uuid.UUID) app.LinkRows {
	r := app.LinkRows{WorkspaceID: p.workspace, ProjectID: p.id, IssueID: issue, Targets: targets, By: by, At: at}
	for range targets {
		r.IDs = append(r.IDs, uuid.NewV7())
	}
	return r
}

// textRows are the answer of query, whose one column is text, in its order.
func textRows(t *testing.T, pool *pgxpool.Pool, query string, args ...any) []string {
	t.Helper()
	rows, err := pool.Query(context.Background(), query, args...)
	if err != nil {
		t.Fatal(err)
	}
	texts, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	return texts
}

// linkTable is every row of table, in id order, as text: its id, its work
// item's, its target in column, its audit columns, whether it is deleted.
func linkTable(t *testing.T, pool *pgxpool.Pool, table, column string) []string {
	t.Helper()
	return textRows(t, pool, `SELECT format('%s %s/%s/%s %s by %s/%s at %s/%s deleted %s', id, workspace_id, project_id, issue_id, `+
		column+`, created_by_id, updated_by_id, created_at, updated_at, deleted_at IS NOT NULL) FROM `+table+` ORDER BY id`)
}

// linkLine is linkTable's line of a row id of table under the work item
// issue of p, of the target, by by at at, undeleted.
func linkLine(t *testing.T, pool *pgxpool.Pool, id uuid.UUID, p project, issue, target, by uuid.UUID, at time.Time) string {
	t.Helper()
	var stamp string
	if err := pool.QueryRow(context.Background(), "SELECT $1::timestamptz::text", at).Scan(&stamp); err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%s %s/%s/%s %s by %s/%s at %s/%s deleted f", id, p.workspace, p.id, issue, target, by, by, stamp, stamp)
}

// createWebIssue inserts web's work item 7 through the store, by alice at
// now, high, in web's default state, from the 12th to the 19th of October,
// at 75535, completed at an hour before: every column createIssue writes,
// none at its default, but its parent, which TestAWorkItemsParentIsWritten
// writes.
func createWebIssue(t *testing.T, s *postgresadapter.Store, web project, id, alice uuid.UUID) app.IssueRow {
	t.Helper()
	done := now.Add(-time.Hour)
	start, target := day(12), day(19)
	r := app.IssueRow{ID: id, WorkspaceID: web.workspace, ProjectID: web.id, SequenceID: 7, Name: "Fix the login", StateID: web.state,
		Priority: domain.PriorityHigh, StartDate: &start, TargetDate: &target, SortOrder: 75535, CompletedAt: &done, By: alice, At: now}
	if err := s.InsertIssue(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	return r
}

// day is the day d of October 2026, a work item's date.
func day(d int) shared.Date {
	return shared.DateOf(time.Date(2026, time.October, d, 0, 0, 0, 0, time.UTC))
}

// createIssue's rows through the store (M4 design 3.5): the work item with
// every value given, its assignees' and labels' rows, one per target in
// the order given, its subscribers', one per account however often named,
// and its activity's, each row's own; all of them by alice at now, under
// the work item, in its project and workspace. GetIssue reads the work
// item back with its undeleted assignees and labels in their id order:
// not its deleted assignee's or label's row, nor another work item's
// assignee or label.
func TestCreatingAWorkItemWritesItsRows(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	bug, ui, feature := newLabel(t, pool, web, "Bug"), newLabel(t, pool, web, "UI"), newLabel(t, pool, web, "Feature")
	id, other := uuid.NewV7(), uuid.NewV7()
	r := createWebIssue(t, s, web, id, alice)
	newIssue(t, pool, web, other, 8, alice)

	assignees, labels := linkRows(web, id, alice, now, carol, bob), linkRows(web, id, alice, now, ui, bug)
	subscribers := linkRows(web, id, alice, now, alice, bob, alice)
	for _, step := range []struct {
		insert func(context.Context, app.LinkRows) error
		rows   app.LinkRows
	}{{s.InsertAssignees, assignees}, {s.InsertLabels, labels}, {s.InsertSubscribers, subscribers},
		{s.InsertAssignees, linkRows(web, other, alice, now, alice)}, {s.InsertLabels, linkRows(web, other, alice, now, feature)}} {
		if err := step.insert(ctx, step.rows); err != nil {
			t.Fatal(err)
		}
	}
	for _, deleted := range []struct {
		table, column string
		target        uuid.UUID
	}{{"issue_assignees", "assignee_id", alice}, {"issue_labels", "label_id", feature}} {
		exec(t, pool, `INSERT INTO `+deleted.table+` (id, workspace_id, project_id, issue_id, `+deleted.column+`, deleted_at) VALUES ($1, $2,
			$3, $4, $5, $6)`, uuid.NewV7(), web.workspace, web.id, id, deleted.target, now)
	}
	created, assigned := uuid.NewV7(), uuid.NewV7()
	rows := []app.ActivityRow{
		{ID: created, WorkspaceID: web.workspace, ProjectID: web.id, IssueID: id, ActorID: alice, At: now,
			Activity: domain.CreationActivity(nil)[0]},
		{ID: assigned, WorkspaceID: web.workspace, ProjectID: web.id, IssueID: id, ActorID: alice, At: now,
			Activity: domain.CreationActivity([]domain.Assignee{{ID: bob, DisplayName: "Bob"}})[1]},
	}
	if err := s.InsertActivities(ctx, rows); err != nil {
		t.Fatal(err)
	}

	got, found, err := s.GetIssue(ctx, id)
	want := domain.Issue{ID: id, WorkspaceID: web.workspace, ProjectID: web.id, SequenceID: 7, Name: "Fix the login", StateID: web.state,
		Priority: domain.PriorityHigh, StartDate: r.StartDate, TargetDate: r.TargetDate, SortOrder: 75535, CompletedAt: r.CompletedAt,
		CreatedByID: &alice, UpdatedByID: &alice, CreatedAt: now, UpdatedAt: now,
		AssigneeIDs: slices.SortedFunc(slices.Values([]uuid.UUID{bob, carol}), uuid.UUID.Compare),
		LabelIDs:    slices.SortedFunc(slices.Values([]uuid.UUID{bug, ui}), uuid.UUID.Compare)}
	if err != nil || !found || !reflect.DeepEqual(got, want) {
		t.Errorf("GetIssue() = %+v, %v, %v; want %+v", got, found, err, want)
	}

	for _, tt := range []struct {
		table, column string
		rows          app.LinkRows
		// lines: of the rows, those written, first in the table; all: the
		// table's rows, the other work item's and the deleted one too
		lines, all int
	}{{"issue_assignees", "assignee_id", assignees, 2, 4}, {"issue_labels", "label_id", labels, 2, 4},
		{"issue_subscribers", "subscriber_id", subscribers, 2, 2}} {
		var want []string
		for i := range tt.lines {
			want = append(want, linkLine(t, pool, tt.rows.IDs[i], web, id, tt.rows.Targets[i], alice, now))
		}
		if got := linkTable(t, pool, tt.table, tt.column); len(got) != tt.all || !slices.Equal(got[:tt.lines], want) {
			t.Errorf("%s:\n%v\nwant %d rows, first\n%v", tt.table, got, tt.all, want)
		}
	}
	activity := textRows(t, pool, `SELECT format('%s %s/%s/%s %s %L %L→%L %L→%L %L %s/%s/%s %s/%s %L', id, workspace_id, project_id, issue_id,
		verb, field, old_value, new_value, old_identifier, new_identifier, comment, actor_id, created_by_id, updated_by_id, created_at = $1,
		updated_at = $1, deleted_at) FROM issue_activities ORDER BY id`, now)
	place := fmt.Sprintf("%s/%s/%s", web.workspace, web.id, id)
	by := fmt.Sprintf("%s/%s/%s", alice, alice, alice)
	wantActivity := []string{
		fmt.Sprintf("%s %s created NULL NULL→NULL NULL→NULL 'created the work item' %s t/t NULL", created, place, by),
		fmt.Sprintf("%s %s updated 'assignees' ''→'Bob' NULL→'%s' 'added assignee ' %s t/t NULL", assigned, place, bob, by),
	}
	if !slices.Equal(activity, wantActivity) {
		t.Errorf("issue_activities:\n%v\nwant\n%v", activity, wantActivity)
	}
}

// An account that subscribes to the work item already keeps his row and is
// not subscribed again; one named twice is subscribed once, in the first
// row given; one whose subscription was deleted is subscribed anew beside
// it (M4 design 3.12).
func TestInsertSubscribersSubscribesEachAccountOnce(t *testing.T) {
	s, pool := newStore(t)
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	id := uuid.NewV7()
	newIssue(t, pool, web, id, 1, alice)
	earlier := now.Add(-time.Hour)
	before := linkRows(web, id, alice, earlier, bob, carol)
	if err := s.InsertSubscribers(context.Background(), before); err != nil {
		t.Fatal(err)
	}
	exec(t, pool, "UPDATE issue_subscribers SET deleted_at = $2 WHERE id = $1", before.IDs[1], earlier)
	again := linkRows(web, id, alice, now, carol, bob, carol)
	if err := s.InsertSubscribers(context.Background(), again); err != nil {
		t.Fatal(err)
	}
	got := linkTable(t, pool, "issue_subscribers", "subscriber_id")
	deleted := linkLine(t, pool, before.IDs[1], web, id, carol, alice, earlier)
	want := []string{linkLine(t, pool, before.IDs[0], web, id, bob, alice, earlier), deleted[:len(deleted)-1] + "t",
		linkLine(t, pool, again.IDs[0], web, id, carol, alice, now)}
	if !slices.Equal(got, want) {
		t.Errorf("issue_subscribers:\n%v\nwant\n%v", got, want)
	}
}

// The greatest sort order of a state's undeleted work items, an archived
// one's too, is where a new one goes after (M4 design 3.7); none for a
// state without work items. A deleted work item, another state's and
// another project's, though in the state, do not count.
func TestGreatestSortOrder(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, ops := newProject(t, pool, acme, "WEB"), newProject(t, pool, acme, "OPS")
	other := uuid.NewV7()
	exec(t, pool, "INSERT INTO states (id, workspace_id, project_id, name, color) VALUES ($1, $2, $3, 'Todo', '#000')", other, acme, web.id)
	ctx := context.Background()
	if greatest, err := s.GreatestSortOrder(ctx, web.id, web.state); err != nil || greatest != nil {
		t.Errorf("GreatestSortOrder() of a state without work items = %v, %v; want none", greatest, err)
	}
	for n, w := range []struct {
		p     project
		state uuid.UUID
		sort  float64
		set   string
	}{{web, web.state, 100, ""}, {web, web.state, 300, "archived_at = now()"}, {web, web.state, 900, "deleted_at = now()"},
		{web, other, 500, ""}, {ops, web.state, 700, ""}} {
		id := uuid.NewV7()
		newIssue(t, pool, w.p, id, n+1, alice)
		exec(t, pool, "UPDATE issues SET state_id = $2, sort_order = $3 WHERE id = $1", id, w.state, w.sort)
		if w.set != "" {
			exec(t, pool, "UPDATE issues SET "+w.set+" WHERE id = $1", id)
		}
	}
	if greatest, err := s.GreatestSortOrder(ctx, web.id, web.state); err != nil || greatest == nil || *greatest != 300 {
		t.Errorf("GreatestSortOrder() = %v, %v; want 300", greatest, err)
	}
}

// A work item is found by its number in its project, archived or not
// (M4 design 3.7): web's 1 and 2, and ops's 1, each its own; not a
// deleted one, nor a number of no work item of the project, though
// another project's work item has it.
func TestIssueByNumber(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, ops := newProject(t, pool, acme, "WEB"), newProject(t, pool, acme, "OPS")
	one, two, gone, opsOne, opsThree := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	newIssue(t, pool, web, one, 1, alice)
	newIssue(t, pool, web, two, 2, alice)
	newIssue(t, pool, web, gone, 3, alice)
	newIssue(t, pool, ops, opsOne, 1, alice)
	newIssue(t, pool, ops, opsThree, 3, alice)
	exec(t, pool, "UPDATE issues SET archived_at = now() WHERE id = $1", two)
	exec(t, pool, "UPDATE issues SET deleted_at = now() WHERE id = $1", gone)
	for _, tt := range []struct {
		p      project
		number int
		want   uuid.UUID
		found  bool
	}{{web, 1, one, true}, {web, 2, two, true}, {ops, 1, opsOne, true}, {web, 3, uuid.UUID{}, false}, {web, 4, uuid.UUID{}, false},
		{ops, 2, uuid.UUID{}, false}} {
		if id, found, err := s.IssueByNumber(context.Background(), tt.p.id, tt.number); err != nil || id != tt.want || found != tt.found {
			t.Errorf("IssueByNumber(%s, %d) = %s, %v, %v; want %s, %v", tt.p.id, tt.number, id, found, err, tt.want, tt.found)
		}
	}
}

// A work item without assignees or labels reads with neither, nil and not
// empty; a deleted work item, and one that is not there, read as none. A
// read or a write that fails answers its error as itself: never a work
// item, nor "none", nor a sort order, where the right answers are the work
// item and its sort order, 65535 (M4 design 3.5).
func TestAFailedIssueStatementIsAnError(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	id := uuid.NewV7()
	newIssue(t, pool, web, id, 1, alice)
	ctx := context.Background()
	if got, found, err := s.GetIssue(ctx, id); err != nil || !found || got.ID != id || got.AssigneeIDs != nil || got.LabelIDs != nil {
		t.Errorf("GetIssue() of a work item without assignees or labels = %+v, %v, %v; want it, with nil for both", got, found, err)
	}

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if got, found, err := s.GetIssue(cancelled, id); !errors.Is(err, context.Canceled) || found || got.ID != (uuid.UUID{}) {
		t.Errorf("GetIssue() = %+v, %v, %v; want context.Canceled", got, found, err)
	}
	if greatest, err := s.GreatestSortOrder(cancelled, web.id, web.state); !errors.Is(err, context.Canceled) || greatest != nil {
		t.Errorf("GreatestSortOrder() = %v, %v; want context.Canceled", greatest, err)
	}
	if got, found, err := s.IssueByNumber(cancelled, web.id, 1); !errors.Is(err, context.Canceled) || found || got != (uuid.UUID{}) {
		t.Errorf("IssueByNumber() = %s, %v, %v; want context.Canceled", got, found, err)
	}

	if got, found, err := s.GetIssue(ctx, uuid.NewV7()); err != nil || found || got.ID != (uuid.UUID{}) {
		t.Errorf("GetIssue() of no work item = %+v, %v, %v; want none", got, found, err)
	}
	exec(t, pool, "UPDATE issues SET deleted_at = now() WHERE id = $1", id)
	if got, found, err := s.GetIssue(ctx, id); err != nil || found || got.ID != (uuid.UUID{}) {
		t.Errorf("GetIssue() of a deleted work item = %+v, %v, %v; want none", got, found, err)
	}

	if err := s.InsertIssue(cancelled, app.IssueRow{ID: uuid.NewV7(), WorkspaceID: web.workspace, ProjectID: web.id, SequenceID: 2, Name: "a",
		StateID: web.state, Priority: domain.PriorityNone, By: alice, At: now}); !errors.Is(err, context.Canceled) {
		t.Errorf("InsertIssue() = %v; want context.Canceled", err)
	}
	links := linkRows(web, id, alice, now, alice)
	for name, insert := range map[string]func(context.Context, app.LinkRows) error{"InsertAssignees": s.InsertAssignees,
		"InsertLabels": s.InsertLabels, "InsertSubscribers": s.InsertSubscribers} {
		if err := insert(cancelled, links); !errors.Is(err, context.Canceled) {
			t.Errorf("%s() = %v; want context.Canceled", name, err)
		}
	}
	if err := s.InsertActivities(cancelled, []app.ActivityRow{{ID: uuid.NewV7(), WorkspaceID: web.workspace, ProjectID: web.id, IssueID: id,
		ActorID: alice, At: now, Activity: domain.CreationActivity(nil)[0]}}); !errors.Is(err, context.Canceled) {
		t.Errorf("InsertActivities() = %v; want context.Canceled", err)
	}
}
