package bootstrap

import (
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// joinedItem is a work item as a write answers it, with its cycle.
type joinedItem struct {
	ID        uuid.UUID  `json:"id"`
	CycleID   *uuid.UUID `json:"cycle_id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// cycleNames names the world's accounts and the cycles given, and the
// moments given, for the tests' readable rows.
type cycleNames struct {
	w       issueWorld
	cycles  map[uuid.UUID]string
	moments map[string]string
}

func (n cycleNames) of(id *uuid.UUID) string {
	if id == nil {
		return "-"
	}
	for name, account := range n.w.ids {
		if account == *id {
			return name
		}
	}
	if name, ok := n.cycles[*id]; ok {
		return name
	}
	return id.String()
}

func (n cycleNames) at(at time.Time) string {
	if name, ok := n.moments[at.UTC().Format(time.RFC3339Nano)]; ok {
		return name
	}
	return at.UTC().Format(time.RFC3339Nano)
}

// moment names at name in n's rows.
func (n cycleNames) moment(name string, at time.Time) {
	n.moments[at.UTC().Format(time.RFC3339Nano)] = name
}

// cycleRows are the work item's cycle_issues rows, in the order made: its
// cycle, made by whom and when, and, for a row left, by whom and when,
// each row of the work item's workspace and project.
func (n cycleNames) cycleRows(t *testing.T, item uuid.UUID) []string {
	t.Helper()
	rows, err := n.w.pool.Query(pgtest.Soon(t), `SELECT c.cycle_id, c.created_by_id, c.created_at, c.updated_by_id, c.updated_at, c.deleted_at,
		c.workspace_id = i.workspace_id AND c.project_id = i.project_id FROM cycle_issues c JOIN issues i ON i.id = c.issue_id
		WHERE c.issue_id = $1 ORDER BY c.created_at, c.id`, item)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var cycle uuid.UUID
		var made, changed *uuid.UUID
		var at, updated time.Time
		var deleted *time.Time
		var under bool
		if err := rows.Scan(&cycle, &made, &at, &changed, &updated, &deleted, &under); err != nil {
			t.Fatal(err)
		}
		row := fmt.Sprintf("%s made by %s at %s", n.of(&cycle), n.of(made), n.at(at))
		switch {
		case !under:
			row += " outside the work item's project"
		case deleted != nil:
			row += fmt.Sprintf(", left by %s at %s, written at %s", n.of(changed), n.at(*deleted), n.at(updated))
		case !updated.Equal(at) || n.of(changed) != n.of(made):
			row += fmt.Sprintf(", changed by %s at %s", n.of(changed), n.at(updated))
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// cycleActivity is the work item's activity, in its order: each row's
// field, verb, values, identifiers and sentence, by whom and when.
func (n cycleNames) cycleActivity(t *testing.T, item uuid.UUID) []string {
	t.Helper()
	rows, err := n.w.pool.Query(pgtest.Soon(t), `SELECT coalesce(field, ''), verb, coalesce(old_value, '-'), coalesce(new_value, '-'),
		old_identifier, new_identifier, comment, actor_id, created_at FROM issue_activities WHERE issue_id = $1 ORDER BY id`, item)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var field, verb, old, value, comment string
		var oldID, newID, actor *uuid.UUID
		var at time.Time
		if err := rows.Scan(&field, &verb, &old, &value, &oldID, &newID, &comment, &actor, &at); err != nil {
			t.Fatal(err)
		}
		out = append(out, fmt.Sprintf("%s %s %q→%q %s→%s %q by %s at %s", field, verb, old, value, n.of(oldID), n.of(newID), comment, n.of(actor), n.at(at)))
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// audit is the work item's last writer and moment, as stored.
func (n cycleNames) audit(t *testing.T, item uuid.UUID) string {
	t.Helper()
	var by uuid.UUID
	var at time.Time
	if err := n.w.pool.QueryRow(pgtest.Soon(t), "SELECT updated_by_id, updated_at FROM issues WHERE id = $1", item).Scan(&by, &at); err != nil {
		t.Fatal(err)
	}
	return "by " + n.of(&by) + " at " + n.at(at)
}

// wantRows checks got against want, line by line.
func wantRows(t *testing.T, what string, got, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Errorf("%s:\n%s\nwant\n%s", what, strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// joinWrite is name's write of method path with body on a work item,
// answered want, as the item it answers.
func (w issueWorld) joinWrite(t *testing.T, name, method, path, body string, want int) joinedItem {
	t.Helper()
	var item joinedItem
	decodeAnswer(t, w.send(t, name, method, path, body, want), &item)
	return item
}

// The app puts work items in Web's cycles through their own field (M6 P3a
// spec 2.3, 2.4, R4, R5): bob creates one in Sprint, its row and its
// activity's, the creation's then the cycle's, his at the creation's
// moment; he moves it to Next, the row of Sprint left and one of Next made
// at the change's moment, the work item last changed by him then, one
// updated row of activity naming both; Next again changes nothing; null
// takes it out of Next, one deleted row; null again changes nothing. Out
// of a cycle that has since ended, or ended and been archived, a work item
// moves and is taken (R3): only the cycle joined is checked.
func TestWorkItemsJoinMoveAndLeaveCycles(t *testing.T) {
	w := newIssueWorld(t)
	w.turnCyclesOn(t, w.web)
	now := time.Now()
	day := func(days int) string { return shanghaiDay(t, now, days) }
	sprint := w.newCycle(t, "alice", datesBody("Sprint", day(-1), day(1)))
	next := w.newCycle(t, "alice", datesBody("Next", day(2), day(5)))
	n := cycleNames{w: w, cycles: map[uuid.UUID]string{sprint: "Sprint", next: "Next"}, moments: map[string]string{}}
	path := "/api/v0/projects/" + w.web.String() + "/issues"
	item := w.joinWrite(t, "bob", http.MethodPost, path, `{"name":"Fix","cycle_id":"`+sprint.String()+`"}`, http.StatusCreated)
	if item.CycleID == nil || *item.CycleID != sprint {
		t.Fatalf("created in %v, want Sprint", item.CycleID)
	}
	n.moment("made", item.CreatedAt)
	wantRows(t, "rows after the creation", n.cycleRows(t, item.ID), []string{"Sprint made by bob at made"})
	created := []string{` created "-"→"-" -→- "created the work item" by bob at made`,
		`cycles created ""→"Sprint" -→Sprint "added cycle " by bob at made`}
	wantRows(t, "activity after the creation", n.cycleActivity(t, item.ID), created)

	itemPath := "/api/v0/issues/" + item.ID.String()
	moved := w.joinWrite(t, "bob", http.MethodPatch, itemPath, `{"cycle_id":"`+next.String()+`"}`, http.StatusOK)
	n.moment("moved", moved.UpdatedAt)
	if moved.CycleID == nil || *moved.CycleID != next || n.audit(t, item.ID) != "by bob at moved" {
		t.Fatalf("moved into %v, last changed %s; want Next, by bob at the move", moved.CycleID, n.audit(t, item.ID))
	}
	rows := []string{"Sprint made by bob at made, left by bob at moved, written at moved", "Next made by bob at moved"}
	wantRows(t, "rows after the move", n.cycleRows(t, item.ID), rows)
	activity := append(created, `cycles updated "Sprint"→"Next" Sprint→Next "updated cycle from " by bob at moved`)
	wantRows(t, "activity after the move", n.cycleActivity(t, item.ID), activity)

	w.joinWrite(t, "bob", http.MethodPatch, itemPath, `{"cycle_id":"`+next.String()+`"}`, http.StatusOK)
	wantRows(t, "rows after Next again", n.cycleRows(t, item.ID), rows)
	wantRows(t, "activity after Next again", n.cycleActivity(t, item.ID), activity)
	left := w.joinWrite(t, "alice", http.MethodPatch, itemPath, `{"cycle_id":null}`, http.StatusOK)
	n.moment("left", left.UpdatedAt)
	if left.CycleID != nil || n.audit(t, item.ID) != "by alice at left" {
		t.Fatalf("left into %v, last changed %s; want none, by alice", left.CycleID, n.audit(t, item.ID))
	}
	rows[1] = "Next made by bob at moved, left by alice at left, written at left"
	wantRows(t, "rows after leaving", n.cycleRows(t, item.ID), rows)
	activity = append(activity, `cycles deleted "Next"→"" Next→- "removed this work item from cycle " by alice at left`)
	wantRows(t, "activity after leaving", n.cycleActivity(t, item.ID), activity)
	w.joinWrite(t, "alice", http.MethodPatch, itemPath, `{"cycle_id":null}`, http.StatusOK)
	wantRows(t, "rows after null again", n.cycleRows(t, item.ID), rows)
	wantRows(t, "activity after null again", n.cycleActivity(t, item.ID), activity)
	if n.audit(t, item.ID) != "by alice at left" {
		t.Errorf("a change of nothing changed the work item: %s", n.audit(t, item.ID))
	}

	// Ended, and ended and archived: joined while open (the business clock
	// three weeks back), then left and moved out of on the clock of today.
	earlyAt := now.AddDate(0, 0, -21).Truncate(time.Microsecond)
	n.moment("early", earlyAt)
	early := newIssueRouteWithClock(t, w.pool, postgres.NewTxManager(w.pool, 2*time.Second), clockAt(earlyAt))
	ended := w.newCycle(t, "alice", datesBody("Ended", day(-25), day(-20)))
	shelved := w.newCycle(t, "alice", datesBody("Shelved", day(-19), day(-15)))
	n.cycles[ended], n.cycles[shelved] = "Ended", "Shelved"
	for _, out := range []struct {
		cycle uuid.UUID
		body  string
		want  *uuid.UUID
	}{{ended, `{"cycle_id":null}`, nil}, {shelved, `{"cycle_id":null}`, nil}, {ended, `{"cycle_id":"` + next.String() + `"}`, &next}} {
		id := w.newWorkItemIn(t, w.web, `{"name":"Old"}`)
		early.answerWithin(t, w.contract, http.MethodPatch, "/api/v0/issues/"+id.String(), w.ids["bob"], `{"cycle_id":"`+out.cycle.String()+`"}`,
			http.StatusOK)
		if out.cycle == shelved {
			w.send(t, "alice", http.MethodPost, "/api/v0/cycles/"+shelved.String()+"/archive", "", http.StatusOK)
		}
		got := w.joinWrite(t, "bob", http.MethodPatch, "/api/v0/issues/"+id.String(), out.body, http.StatusOK)
		n.moment("then", got.UpdatedAt)
		rows := []string{n.of(&out.cycle) + " made by bob at early, left by bob at then, written at then"}
		if out.want != nil {
			rows = append(rows, "Next made by bob at then")
		}
		if !reflect.DeepEqual(got.CycleID, out.want) {
			t.Errorf("out of %s with %s: in %v, want %v", n.of(&out.cycle), out.body, got.CycleID, out.want)
		}
		wantRows(t, "rows out of "+n.of(&out.cycle), n.cycleRows(t, id), rows)
	}
}

// issue.created ends its changes with the cycle the work item is created
// in, as with its labels; issue.updated's are the cycle's activity row,
// moved or left (M6 P3a spec 2.4), through the dispatcher in the write's
// transaction.
func TestIssueEventsCarryTheCycle(t *testing.T) {
	w := newIssueWorld(t)
	w.turnCyclesOn(t, w.web)
	now := time.Now()
	day := func(days int) string { return shanghaiDay(t, now, days) }
	sprint := w.newCycle(t, "alice", datesBody("Sprint", day(-1), day(1)))
	next := w.newCycle(t, "alice", datesBody("Next", day(2), day(5)))
	var received []shared.Event
	var failure error
	route := issueEventRoute(t, w, clockAt(now), &received, &failure)
	var item joinedItem
	decodeAnswer(t, route.answerWithin(t, w.contract, http.MethodPost, "/api/v0/projects/"+w.web.String()+"/issues", w.ids["alice"],
		`{"name":"Fix","cycle_id":"`+sprint.String()+`"}`, http.StatusCreated), &item)
	path := "/api/v0/issues/" + item.ID.String()
	route.answerWithin(t, w.contract, http.MethodPatch, path, w.ids["alice"], `{"cycle_id":"`+next.String()+`"}`, http.StatusOK)
	route.answerWithin(t, w.contract, http.MethodPatch, path, w.ids["alice"], `{"cycle_id":null}`, http.StatusOK)
	if len(received) != 3 {
		t.Fatalf("events %+v, want the creation and two changes", received)
	}
	created := received[0].Changes
	if last := created[len(created)-1]; !reflect.DeepEqual(last, shared.Change{Field: "cycles", New: eventValue("Sprint"), NewID: &sprint}) {
		t.Errorf("issue.created's last change %+v, want Sprint", last)
	}
	for i, want := range []shared.Change{{Field: "cycles", Old: eventValue("Sprint"), New: eventValue("Next"), OldID: &sprint, NewID: &next},
		{Field: "cycles", Old: eventValue("Next"), New: eventValue(""), OldID: &next}} {
		if e := received[i+1]; e.Type != "issue.updated" || !reflect.DeepEqual(e.Changes, []shared.Change{want}) {
			t.Errorf("event %d: %s %+v, want issue.updated %+v", i+1, e.Type, e.Changes, want)
		}
	}
}

// A work item's cycle is its own write's (M6 P3a spec R7): walt, acme's
// admin and Web's guest who sees only his own work items (adminGuest),
// creates one in Sprint and takes it out, as any change of his own work
// item, while planning's writes are forbidden to him; alice's work item is
// hidden from him: issue.not_found, as for any field, not forbidden.
func TestOwnOnlyCallersChangeTheCycleOfTheirOwnWorkItems(t *testing.T) {
	w := newIssueWorld(t)
	w.turnCyclesOn(t, w.web)
	sprint := w.newCycle(t, "alice", `{"name":"Sprint"}`)
	hers := w.newWorkItemIn(t, w.web, `{"name":"Hers"}`)
	w.adminGuest(t)
	his := w.joinWrite(t, "walt", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/issues", `{"name":"His","cycle_id":"`+sprint.String()+`"}`,
		http.StatusCreated)
	if his.CycleID == nil || *his.CycleID != sprint {
		t.Fatalf("his work item in %v, want Sprint", his.CycleID)
	}
	if left := w.joinWrite(t, "walt", http.MethodPatch, "/api/v0/issues/"+his.ID.String(), `{"cycle_id":null}`, http.StatusOK); left.CycleID != nil {
		t.Errorf("his work item left into %v, want none", left.CycleID)
	}
	w.refusedCycle(t, "walt", w.web, `{"name":"His own"}`, http.StatusForbidden, "forbidden")
	answer := w.send(t, "walt", http.MethodPatch, "/api/v0/issues/"+hers.String(), `{"cycle_id":"`+sprint.String()+`"}`, http.StatusNotFound)
	if code := refusalOf(t, []byte(answer), false); code != "issue.not_found" {
		t.Errorf("walt's change of her work item's cycle = %s, want issue.not_found", answer)
	}
}
