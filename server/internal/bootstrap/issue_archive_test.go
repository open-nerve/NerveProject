package bootstrap

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// archiveIssue and unarchiveIssue as bootstrap wires them write the work
// item and their activity's row at one moment within their request, as
// their caller (M4 design 3.8, 3.13): alice makes a work item in Web's
// Done; bob archives it, then archives it again; alice unarchives it, then
// unarchives it again. The first archiving archives it, changed last by
// bob at one moment within his request, and writes his row 14 then:
// archived_at, updated, from NULL to manual_archive. The first
// unarchiving unarchives it, changed last by alice at one moment within
// her request, and writes her row 16 then: archive to restore. Each second
// one writes nothing; the creation's rows stay as they were. A work item
// in Todo, made at the start, is not archived (409 issue.not_archivable),
// and nothing of it is written, neither by its refused archiving nor by
// the other work item's archivings; moved to Cancelled, bob archives it.
// Cancelled moved to the started group since (Dropped, a second cancelled
// state, keeps that group in Web), alice restores it: a restoring asks no
// state group.
func TestArchivingAWorkItemStampsItsRows(t *testing.T) {
	w := newIssueWorld(t)
	item, todo := w.doneWorkItem(t), w.newWorkItem(t)
	path := "/api/v0/issues/" + item.String()
	created, todoStamps := w.workItemStamps(t, item), w.workItemStamps(t, todo)

	var moments []time.Time
	for _, step := range []struct {
		name, verb string
		archived   bool
	}{{"bob", "archive", true}, {"alice", "unarchive", false}} {
		before := time.Now().Truncate(time.Microsecond)
		w.send(t, step.name, http.MethodPost, path+"/"+step.verb, "", http.StatusOK)
		after := time.Now()
		archivedAt, changed, by := archivalOf(t, w, item)
		if changed.Before(before) || changed.After(after) || by != w.ids[step.name] || (archivedAt != nil) != step.archived ||
			archivedAt != nil && !archivedAt.Equal(changed) {
			t.Fatalf("after %s's %s: archived at %v, changed last by %s at %v; want archived %v, at one moment within %v to %v", step.name,
				step.verb, archivedAt, by, changed, step.archived, before, after)
		}
		stamps := w.workItemStamps(t, item)
		w.send(t, step.name, http.MethodPost, path+"/"+step.verb, "", http.StatusOK)
		if again := w.workItemStamps(t, item); !slices.Equal(again, stamps) {
			t.Errorf("%s's second %s wrote:\n%q\nwas\n%q", step.name, step.verb, again, stamps)
		}
		moments = append(moments, changed)
	}

	want := []string{
		fmt.Sprintf("updated archived_at <nil> → manual_archive, archived the work item, by %s at %s", w.ids["bob"], moments[0]),
		fmt.Sprintf("updated archived_at archive → restore, restored the work item, by %s at %s", w.ids["alice"], moments[1]),
	}
	if got := archivalRows(t, w, item); !slices.Equal(got, want) {
		t.Errorf("the work item's archived_at rows:\n%q\nwant\n%q", got, want)
	}
	stamps := w.workItemStamps(t, item)
	for _, line := range created {
		if !slices.Contains(stamps, line) && !strings.HasPrefix(line, "issues ") {
			t.Errorf("the creation's row %q changed: the rows are\n%q", line, stamps)
		}
	}

	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+todo.String()+"/archive", "", http.StatusConflict)
	if after := w.workItemStamps(t, todo); !slices.Equal(after, todoStamps) {
		t.Errorf("the work item in Todo, after the other's archivings and its own refused one:\n%q\nwas\n%q", after, todoStamps)
	}
	cancelled := stateID(t, w.pool, w.web, "Cancelled")
	w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+todo.String(), `{"state_id":"`+cancelled.String()+`"}`, http.StatusOK)
	w.send(t, "bob", http.MethodPost, "/api/v0/issues/"+todo.String()+"/archive", "", http.StatusOK)
	if archivedAt, _, by := archivalOf(t, w, todo); archivedAt == nil || by != w.ids["bob"] {
		t.Errorf("the work item moved to Cancelled: archived at %v, changed last by %s; want archived by bob", archivedAt, by)
	}
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/states", `{"name":"Dropped","color":"#9AA4BC","group":"cancelled"}`,
		http.StatusCreated)
	w.send(t, "alice", http.MethodPatch, "/api/v0/states/"+cancelled.String(), `{"group":"started"}`, http.StatusOK)
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+todo.String()+"/unarchive", "", http.StatusOK)
	if archivedAt, _, by := archivalOf(t, w, todo); archivedAt != nil || by != w.ids["alice"] {
		t.Errorf("the work item in Cancelled, of the started group since: archived at %v, changed last by %s; want it restored by alice",
			archivedAt, by)
	}
}

// archivalOf is the work item id's archived_at, nil for none, and its last
// change's moment and account.
func archivalOf(t *testing.T, w issueWorld, id uuid.UUID) (*time.Time, time.Time, uuid.UUID) {
	t.Helper()
	var archivedAt *time.Time
	var changed time.Time
	var by uuid.UUID
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT archived_at, updated_at, updated_by_id FROM issues WHERE id = $1", id).Scan(&archivedAt,
		&changed, &by); err != nil {
		t.Fatal(err)
	}
	return archivedAt, changed, by
}

// archivalRows are the work item id's activity rows on archived_at, in
// their ids' order, each as "<verb> archived_at <old> → <new>, <comment>,
// by <actor> at <moment>", each of whose made and last-changed stamps is
// its actor's and its moment's; one that is not fails the test.
func archivalRows(t *testing.T, w issueWorld, id uuid.UUID) []string {
	t.Helper()
	rows, err := w.pool.Query(pgtest.Soon(t), `SELECT verb, old_value, new_value, comment, actor_id, created_at, created_by_id, updated_by_id,
		updated_at FROM issue_activities WHERE issue_id = $1 AND field = 'archived_at' ORDER BY id`, id)
	if err != nil {
		t.Fatal(err)
	}
	lines, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (string, error) {
		var verb, comment string
		var oldValue, newValue *string
		var actor, madeBy, changedBy uuid.UUID
		var at, changedAt time.Time
		if err := row.Scan(&verb, &oldValue, &newValue, &comment, &actor, &at, &madeBy, &changedBy, &changedAt); err != nil {
			return "", err
		}
		if madeBy != actor || changedBy != actor || !changedAt.Equal(at) {
			t.Errorf("an archived_at row by %s at %v was made by %s, changed by %s at %v", actor, at, madeBy, changedBy, changedAt)
		}
		value := func(v *string) string {
			if v == nil {
				return "<nil>"
			}
			return *v
		}
		return fmt.Sprintf("%s archived_at %s → %s, %s, by %s at %s", verb, value(oldValue), value(newValue), comment, actor, at), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return lines
}
