package postgresadapter_test

import (
	"context"
	"errors"
	"maps"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// A state is in use while an undeleted work item is in it, an archived one
// too; a deleted work item, and one of another state, do not count (M4
// design 3.6). A failed read is its failure, never "not in use".
func TestStateInUse(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	todo := uuid.NewV7()
	exec(t, pool, "INSERT INTO states (id, workspace_id, project_id, name, color) VALUES ($1, $2, $3, 'Todo', '#000')", todo, web.workspace, web.id)
	ctx := context.Background()
	inUse := func(want bool, with string) {
		t.Helper()
		if got, err := s.StateInUse(ctx, web.state); err != nil || got != want {
			t.Errorf("StateInUse() with %s = %v, %v; want %v", with, got, err, want)
		}
	}
	inUse(false, "no work items")
	for n, set := range []string{"deleted_at = now()", "state_id = '" + todo.String() + "'"} {
		id := uuid.NewV7()
		newIssue(t, pool, web, id, n+1, alice)
		exec(t, pool, "UPDATE issues SET "+set+" WHERE id = $1", id)
	}
	inUse(false, "a deleted work item and another state's")
	archived := uuid.NewV7()
	newIssue(t, pool, web, archived, 3, alice)
	exec(t, pool, "UPDATE issues SET archived_at = now() WHERE id = $1", archived)
	inUse(true, "an archived work item")

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if got, err := s.StateInUse(cancelled, web.state); !errors.Is(err, context.Canceled) || got {
		t.Errorf("StateInUse() = %v, %v; want context.Canceled", got, err)
	}
}

// DetachLabels soft-deletes the undeleted rows of the labels named, on
// every work item, at the moment and by the account given, and keeps every
// other column; another label's rows, and a row of a named label deleted
// before, keep every column (M4 design 3.6). A failed write is its
// failure.
func TestDetachLabels(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	bug, ui, feature := newLabel(t, pool, web, "Bug"), newLabel(t, pool, web, "UI"), newLabel(t, pool, web, "Feature")
	one, two := uuid.NewV7(), uuid.NewV7()
	newIssue(t, pool, web, one, 1, alice)
	newIssue(t, pool, web, two, 2, alice)
	ctx := context.Background()
	gone := linkRows(web, one, alice, now.Add(-time.Hour), bug)
	for _, r := range []app.LinkRows{gone, linkRows(web, one, alice, now, bug, feature), linkRows(web, two, alice, now, ui, feature)} {
		if err := s.InsertLabels(ctx, r); err != nil {
			t.Fatal(err)
		}
		if r.At.Before(now) {
			exec(t, pool, "UPDATE issue_labels SET deleted_at = updated_at WHERE id = $1", r.IDs[0])
		}
	}
	before := labelRows(t, pool)
	at := now.Add(time.Hour)
	if err := s.DetachLabels(ctx, []uuid.UUID{bug, ui}, bob, at); err != nil {
		t.Fatal(err)
	}
	after := labelRows(t, pool)
	for id, row := range before {
		if row.deleted == nil && (row.label == bug || row.label == ui) {
			row.updatedBy, row.updatedAt, row.deleted = bob, at, &at
		}
		if got := after[id]; got.String() != row.String() {
			t.Errorf("issue_labels %s = %s; want %s", id, got, row)
		}
	}
	if len(after) != len(before) {
		t.Errorf("issue_labels has %d rows, want %d", len(after), len(before))
	}

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if err := s.DetachLabels(cancelled, []uuid.UUID{feature}, bob, at); !errors.Is(err, context.Canceled) {
		t.Errorf("DetachLabels() = %v; want context.Canceled", err)
	}
	if !maps.EqualFunc(labelRows(t, pool), after, func(a, b labelRow) bool { return a.String() == b.String() }) {
		t.Error("a failed DetachLabels() changed issue_labels")
	}
}

// labelRow is a row of issue_labels as TestDetachLabels reads it: its work
// item and label, who made it and when, who changed it last and when, and
// when it was deleted, nil for never.
type labelRow struct {
	issue, label, createdBy, updatedBy uuid.UUID
	createdAt, updatedAt               time.Time
	deleted                            *time.Time
}

func (r labelRow) String() string {
	deleted := "undeleted"
	if r.deleted != nil {
		deleted = "deleted at " + r.deleted.UTC().String()
	}
	return r.issue.String() + " " + r.label.String() + " made by " + r.createdBy.String() + " at " + r.createdAt.UTC().String() +
		", changed by " + r.updatedBy.String() + " at " + r.updatedAt.UTC().String() + ", " + deleted
}

// labelRows are issue_labels' rows, by id.
func labelRows(t *testing.T, pool *pgxpool.Pool) map[uuid.UUID]labelRow {
	t.Helper()
	rows, err := pool.Query(context.Background(),
		"SELECT id, issue_id, label_id, created_by_id, updated_by_id, created_at, updated_at, deleted_at FROM issue_labels")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[uuid.UUID]labelRow{}
	for rows.Next() {
		var id uuid.UUID
		var r labelRow
		if err := rows.Scan(&id, &r.issue, &r.label, &r.createdBy, &r.updatedBy, &r.createdAt, &r.updatedAt, &r.deleted); err != nil {
			t.Fatal(err)
		}
		out[id] = r
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}
