package postgresadapter_test

import (
	"context"
	"errors"
	"maps"
	"slices"
	"testing"
	"time"
	"uuid"

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
	item := uuid.NewV7()
	newIssue(t, pool, web, item, 3, alice)
	inUse(true, "a work item")
	exec(t, pool, "UPDATE issues SET archived_at = now() WHERE id = $1", item)
	inUse(true, "the work item archived")

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if got, err := s.StateInUse(cancelled, web.state); !errors.Is(err, context.Canceled) || got {
		t.Errorf("StateInUse() = %v, %v; want context.Canceled", got, err)
	}
}

// DetachLabels, asked for Bug and UI, soft-deletes their undeleted rows on
// every work item, Bug's on one and UI's on two, at the moment and by the
// account given, and keeps every other column of them; Feature's rows, and
// Bug's row on one deleted before, keep every column (M4 design 3.6). A
// failed write is its failure and changes nothing.
func TestDetachLabels(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	bug, ui, feature := newLabel(t, pool, web, "Bug"), newLabel(t, pool, web, "UI"), newLabel(t, pool, web, "Feature")
	one, two := uuid.NewV7(), uuid.NewV7()
	newIssue(t, pool, web, one, 1, alice)
	newIssue(t, pool, web, two, 2, alice)
	ctx := context.Background()
	gone, onOne, onTwo := linkRows(web, one, alice, now.Add(-time.Hour), bug), linkRows(web, one, alice, now, bug, feature),
		linkRows(web, two, alice, now, ui, feature)
	for _, r := range []app.LinkRows{gone, onOne, onTwo} {
		if err := s.InsertLabels(ctx, r); err != nil {
			t.Fatal(err)
		}
		if r.At.Before(now) { // gone, deleted before Bug is put on one again
			exec(t, pool, "UPDATE issue_labels SET deleted_at = updated_at WHERE id = $1", r.IDs[0])
		}
	}
	rows, auditsBefore := rowsAsText(t, pool, "issue_labels"), audits(t, pool, "issue_labels")
	at := now.Add(time.Hour)
	if err := s.DetachLabels(ctx, []uuid.UUID{bug, ui}, bob, at); err != nil {
		t.Fatal(err)
	}
	detached := []uuid.UUID{onOne.IDs[0], onTwo.IDs[0]} // Bug's row on one, UI's on two
	auditsAfter := audits(t, pool, "issue_labels")
	for id, was := range auditsBefore {
		want := was
		if slices.Contains(detached, id) {
			want = deletion(at, bob)
		}
		if auditsAfter[id] != want {
			t.Errorf("issue_labels %s: %s, want %s", id, auditsAfter[id], want)
		}
	}
	if after := rowsAsText(t, pool, "issue_labels"); !maps.Equal(after, rows) {
		t.Errorf("issue_labels but their audit columns:\n%v\nwant them as they were:\n%v", after, rows)
	}

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if err := s.DetachLabels(cancelled, []uuid.UUID{feature}, bob, at); !errors.Is(err, context.Canceled) {
		t.Errorf("DetachLabels() = %v; want context.Canceled", err)
	}
	if !maps.Equal(rowsAsText(t, pool, "issue_labels"), rows) || !maps.Equal(audits(t, pool, "issue_labels"), auditsAfter) {
		t.Error("a failed DetachLabels() changed issue_labels")
	}
}
