package bootstrap

import (
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// membershipSnapshot is what a work item reads as about its memberships: its
// cycle's or its modules' rows, its activity and its last change.
type membershipSnapshot struct {
	rows, activity []string
	audit          string
}

// snapshot is item's membershipSnapshot, its rows of table, cycle_issues or
// module_issues.
func (n cycleNames) snapshot(t *testing.T, table string, item uuid.UUID) membershipSnapshot {
	t.Helper()
	rows := n.cycleRows(t, item)
	if table == "module_issues" {
		rows = n.moduleRows(t, item)
	}
	return membershipSnapshot{rows: rows, activity: n.cycleActivity(t, item), audit: n.audit(t, item)}
}

// wantSnapshot checks item's membershipSnapshot against want.
func (n cycleNames) wantSnapshot(t *testing.T, what, table string, item uuid.UUID, want membershipSnapshot) {
	t.Helper()
	got := n.snapshot(t, table, item)
	wantRows(t, what+": rows", got.rows, want.rows)
	wantRows(t, what+": activity", got.activity, want.activity)
	if got.audit != want.audit {
		t.Errorf("%s: last changed %s, want %s", what, got.audit, want.audit)
	}
}

// deletedThrough has alice delete the cycle or the module path names through
// a planning route that observes the events its deletion publishes, and
// answers them and the moment table's row of id was deleted at.
func (w issueWorld) deletedThrough(t *testing.T, path, table string, id uuid.UUID) ([]shared.Event, time.Time) {
	t.Helper()
	var received []shared.Event
	route := newPlanningRouteObserved(t, w.pool, postgres.NewTxManager(w.pool, 2*time.Second), func(e shared.Event) { received = append(received, e) })
	route.answerWithin(t, w.contract, http.MethodDelete, path, w.ids["alice"], "", http.StatusNoContent)
	var at time.Time
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at FROM "+table+" WHERE id = $1", id).Scan(&at); err != nil {
		t.Fatal(err)
	}
	return received, at
}

// wantDeletionEvent checks that received is the one deletion of kind,
// "cycle" or "module", of the target id named name in Web, by alice at at:
// planning's event, no other (no issue.updated, M6 P3a spec 2.5).
func (w issueWorld) wantDeletionEvent(t *testing.T, received []shared.Event, kind string, id uuid.UUID, name string, at time.Time) {
	t.Helper()
	if len(received) != 1 {
		t.Fatalf("events %+v, want the one deletion of the %s", received, kind)
	}
	alice, got := w.ids["alice"], received[0]
	if !got.OccurredAt.Equal(at) {
		t.Errorf("event at %s, want the deletion's moment %s", got.OccurredAt, at)
	}
	got.OccurredAt = at
	want := shared.Event{ID: got.ID, Type: "planning." + kind + "_deleted", OccurredAt: at, WorkspaceID: w.acme, ProjectID: w.web,
		ActorID: &alice, Entity: shared.EntityRef{Kind: kind, ID: id}, Changes: []shared.Change{{Field: "name", Old: &name}}}
	if got.ID == uuid.Nil() || !reflect.DeepEqual(got, want) {
		t.Errorf("event %+v, want %+v", got, want)
	}
}

// The app takes the work items out of a cycle that is deleted (M6 P3a spec
// 2.5, R6): alice deletes Sprint, which bob's Fix and the archived Done are
// in, Other is in Next, and Gone was in before its own deletion. planning
// publishes planning.cycle_deleted, the cycle, Acme, Web, alice and the
// moment of the cycle's deletion, its name the one change, and nothing else
// is published (no issue.updated). Fix's and Done's rows of Sprint are
// deleted at that moment by alice, each a row of activity naming Sprint, by
// her then, and each work item last changed by her then; Other's and Gone's
// rows, activity and last change are as they were. Deleting Empty, no work
// item's, writes nothing but the cycle. Fix, out of Sprint, changes its
// cycle as any work item in none (T6's 500 before the subscriber): null
// changes nothing, Next joins it.
func TestDeletingACycleTakesItsWorkItemsOut(t *testing.T) {
	w := newIssueWorld(t)
	w.turnCyclesOn(t, w.web)
	sprint, next, empty := w.newCycle(t, "alice", `{"name":"Sprint"}`), w.newCycle(t, "alice", `{"name":"Next"}`), w.newCycle(t, "alice", `{"name":"Empty"}`)
	n := cycleNames{w: w, cycles: map[uuid.UUID]string{sprint: "Sprint", next: "Next"}, moments: map[string]string{}}
	create := "/api/v0/projects/" + w.web.String() + "/issues"
	in := func(cycle uuid.UUID, more string) string {
		return `{"name":"Work","cycle_id":"` + cycle.String() + `"` + more + `}`
	}
	fix := w.joinWrite(t, "bob", http.MethodPost, create, in(sprint, ""), http.StatusCreated).ID
	done := w.joinWrite(t, "bob", http.MethodPost, create, in(sprint, `,"state_id":"`+w.done.String()+`"`), http.StatusCreated).ID
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+done.String()+"/archive", "", http.StatusOK)
	other := w.joinWrite(t, "bob", http.MethodPost, create, in(next, ""), http.StatusCreated).ID
	gone := w.joinWrite(t, "bob", http.MethodPost, create, in(sprint, ""), http.StatusCreated).ID
	w.send(t, "alice", http.MethodDelete, "/api/v0/issues/"+gone.String(), "", http.StatusNoContent)
	before := map[uuid.UUID]membershipSnapshot{}
	for _, item := range []uuid.UUID{fix, done, other, gone} {
		before[item] = n.snapshot(t, "cycle_issues", item)
	}

	received, at := w.deletedThrough(t, "/api/v0/cycles/"+sprint.String(), "cycles", sprint)
	n.moment("deleted", at)
	w.wantDeletionEvent(t, received, "cycle", sprint, "Sprint", at)
	for name, item := range map[string]uuid.UUID{"Fix": fix, "Done": done} {
		want := before[item]
		want.rows = []string{want.rows[0] + ", left by alice at deleted, written at deleted"}
		want.activity = append(slices.Clone(want.activity), `cycles deleted "Sprint"→"" Sprint→- "removed this work item from cycle " by alice at deleted`)
		want.audit = "by alice at deleted"
		n.wantSnapshot(t, name+" after Sprint's deletion", "cycle_issues", item, want)
	}
	for name, item := range map[string]uuid.UUID{"Other": other, "Gone": gone} {
		n.wantSnapshot(t, name+" after Sprint's deletion", "cycle_issues", item, before[item])
	}
	written := w.cycleWritten(t)
	if received, _ := w.deletedThrough(t, "/api/v0/cycles/"+empty.String(), "cycles", empty); len(received) != 1 || w.cycleWritten(t) != written {
		t.Errorf("Empty's deletion published %+v and wrote %s, was %s; want its deletion, writing nothing more", received, w.cycleWritten(t), written)
	}

	path := "/api/v0/issues/" + fix.String()
	if left := w.joinWrite(t, "bob", http.MethodPatch, path, `{"cycle_id":null}`, http.StatusOK); left.CycleID != nil || w.cycleWritten(t) != written {
		t.Errorf("Fix out of a deleted Sprint left into %v, writing %s; want none, writing nothing", left.CycleID, w.cycleWritten(t))
	}
	joined := w.joinWrite(t, "bob", http.MethodPatch, path, `{"cycle_id":"`+next.String()+`"}`, http.StatusOK)
	n.moment("joined", joined.UpdatedAt)
	if joined.CycleID == nil || *joined.CycleID != next {
		t.Errorf("Fix joined %v, want Next", joined.CycleID)
	}
	want := n.snapshot(t, "cycle_issues", fix)
	wantRows(t, "Fix's rows after joining Next", want.rows[1:], []string{"Next made by bob at joined"})
	wantRows(t, "Fix's activity after joining Next", want.activity[len(want.activity)-1:], []string{`cycles created ""→"Next" -→Next "added cycle " by bob at joined`})
}

// The app takes the work items out of a module that is deleted (M6 P3a spec
// 2.5, R6), as out of a cycle: alice deletes Auth, which bob's Fix, in
// Billing too, and Test are in, Other is in Billing alone, and Gone was in
// before its own deletion. planning publishes planning.module_deleted and
// nothing else is published; Fix's and Test's rows of Auth are deleted at
// that moment by alice, each a row of activity naming Auth, by her then, and
// each work item last changed by her then; Fix's row of Billing, and Other's
// and Gone's rows, activity and last change, are as they were. Deleting
// Empty, no work item's, writes nothing but the module. Fix changes its
// modules as any work item (T7's 500 before the subscriber): Billing alone
// changes nothing, none takes it out of Billing.
func TestDeletingAModuleTakesItsWorkItemsOut(t *testing.T) {
	w := newIssueWorld(t)
	w.turnModulesOn(t, w.web)
	auth, billing, empty := w.newModule(t, `{"name":"Auth"}`), w.newModule(t, `{"name":"Billing"}`), w.newModule(t, `{"name":"Empty"}`)
	n := cycleNames{w: w, cycles: map[uuid.UUID]string{auth: "Auth", billing: "Billing"}, moments: map[string]string{}}
	create := "/api/v0/projects/" + w.web.String() + "/issues"
	in := func(modules ...uuid.UUID) string { return `{"name":"Work",` + modulesBody(modules...)[1:] }
	fix := w.joinWrite(t, "bob", http.MethodPost, create, in(auth, billing), http.StatusCreated).ID
	test := w.joinWrite(t, "bob", http.MethodPost, create, in(auth), http.StatusCreated).ID
	other := w.joinWrite(t, "bob", http.MethodPost, create, in(billing), http.StatusCreated).ID
	gone := w.joinWrite(t, "bob", http.MethodPost, create, in(auth), http.StatusCreated).ID
	w.send(t, "alice", http.MethodDelete, "/api/v0/issues/"+gone.String(), "", http.StatusNoContent)
	before := map[uuid.UUID]membershipSnapshot{}
	for _, item := range []uuid.UUID{fix, test, other, gone} {
		before[item] = n.snapshot(t, "module_issues", item)
	}

	received, at := w.deletedThrough(t, "/api/v0/modules/"+auth.String(), "modules", auth)
	n.moment("deleted", at)
	w.wantDeletionEvent(t, received, "module", auth, "Auth", at)
	for name, item := range map[string]uuid.UUID{"Fix": fix, "Test": test} {
		want := before[item]
		want.rows = slices.Clone(want.rows)
		want.rows[0] += ", left by alice at deleted, written at deleted"
		want.activity = append(slices.Clone(want.activity), `modules deleted "Auth"→"" Auth→- "removed this work item from module " by alice at deleted`)
		want.audit = "by alice at deleted"
		n.wantSnapshot(t, name+" after Auth's deletion", "module_issues", item, want)
	}
	for name, item := range map[string]uuid.UUID{"Other": other, "Gone": gone} {
		n.wantSnapshot(t, name+" after Auth's deletion", "module_issues", item, before[item])
	}
	written := w.cycleWritten(t)
	if received, _ := w.deletedThrough(t, "/api/v0/modules/"+empty.String(), "modules", empty); len(received) != 1 || w.cycleWritten(t) != written {
		t.Errorf("Empty's deletion published %+v and wrote %s, was %s; want its deletion, writing nothing more", received, w.cycleWritten(t), written)
	}

	path := "/api/v0/issues/" + fix.String()
	if kept := w.joinWrite(t, "bob", http.MethodPatch, path, modulesBody(billing), http.StatusOK); !slices.Equal(kept.ModuleIDs, []uuid.UUID{billing}) ||
		w.cycleWritten(t) != written {
		t.Errorf("Fix in Billing alone answered %v, writing %s; want Billing, writing nothing", kept.ModuleIDs, w.cycleWritten(t))
	}
	left := w.joinWrite(t, "bob", http.MethodPatch, path, modulesBody(), http.StatusOK)
	n.moment("left", left.UpdatedAt)
	if len(left.ModuleIDs) != 0 {
		t.Errorf("Fix left into %v, want none", left.ModuleIDs)
	}
	got := n.snapshot(t, "module_issues", fix)
	wantRows(t, "Fix's activity after leaving Billing", got.activity[len(got.activity)-1:],
		[]string{`modules deleted "Billing"→"" Billing→- "removed this work item from module " by bob at left`})
}
