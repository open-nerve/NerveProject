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

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// moduleRows are the work item's module_issues rows, in the order made,
// then by module id: its module, made by whom and when, and, for a row
// left, by whom and when, each row of the work item's workspace and
// project.
func (n cycleNames) moduleRows(t *testing.T, item uuid.UUID) []string {
	t.Helper()
	rows, err := n.w.pool.Query(pgtest.Soon(t), `SELECT m.module_id, m.created_by_id, m.created_at, m.updated_by_id, m.updated_at, m.deleted_at,
		m.workspace_id = i.workspace_id AND m.project_id = i.project_id FROM module_issues m JOIN issues i ON i.id = m.issue_id
		WHERE m.issue_id = $1 ORDER BY m.created_at, m.module_id`, item)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var module uuid.UUID
		var made, changed *uuid.UUID
		var at, updated time.Time
		var deleted *time.Time
		var under bool
		if err := rows.Scan(&module, &made, &at, &changed, &updated, &deleted, &under); err != nil {
			t.Fatal(err)
		}
		row := fmt.Sprintf("%s made by %s at %s", n.of(&module), n.of(made), n.at(at))
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

// newModule is the id of the module alice creates in Web with body.
func (w issueWorld) newModule(t *testing.T, body string) uuid.UUID {
	t.Helper()
	return w.createModule(t, "alice", body, http.StatusCreated).ID
}

// modulesBody is a body naming the modules ids, in their order.
func modulesBody(ids ...uuid.UUID) string {
	quoted := make([]string, len(ids))
	for i, id := range ids {
		quoted[i] = fmt.Sprintf("%q", id)
	}
	return fmt.Sprintf(`{"module_ids":[%s]}`, strings.Join(quoted, ","))
}

// The app puts work items in Web's modules through their own field, a new
// set replacing the whole (M6 P3a spec 2.3, 2.4, R3, R4, R5): bob creates
// one in Billing and Auth, given in that order, answered and written in id
// order, its activity's rows the creation's then each module's, his at the
// creation's moment; he adds Shelved, which alice then archives; he gives
// Core and Shelved: Auth and Billing left and Core made at the change's
// moment, Shelved, archived, kept untouched and not checked, the work item
// last changed by him then, one row of activity for Core joined, then one
// each for Auth and Billing left; the same set in another order changes
// nothing; alice empties it, Core and the archived Shelved left; empty
// again changes nothing.
func TestWorkItemsJoinAndLeaveModules(t *testing.T) {
	w := newIssueWorld(t)
	w.turnModulesOn(t, w.web)
	auth, billing, core := w.newModule(t, `{"name":"Auth"}`), w.newModule(t, `{"name":"Billing"}`), w.newModule(t, `{"name":"Core"}`)
	shelved := w.newModule(t, `{"name":"Shelved","status":"completed"}`)
	n := cycleNames{w: w, cycles: map[uuid.UUID]string{auth: "Auth", billing: "Billing", core: "Core", shelved: "Shelved"}, moments: map[string]string{}}
	item := w.joinWrite(t, "bob", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/issues",
		`{"name":"Fix","module_ids":["`+billing.String()+`","`+auth.String()+`"]}`, http.StatusCreated)
	if !slices.Equal(item.ModuleIDs, []uuid.UUID{auth, billing}) {
		t.Fatalf("created in %v, want Auth and Billing", item.ModuleIDs)
	}
	n.moment("made", item.CreatedAt)
	wantRows(t, "rows after the creation", n.moduleRows(t, item.ID), []string{"Auth made by bob at made", "Billing made by bob at made"})
	activity := []string{` created "-"→"-" -→- "created the work item" by bob at made`,
		`modules created ""→"Auth" -→Auth "added module " by bob at made`, `modules created ""→"Billing" -→Billing "added module " by bob at made`}
	wantRows(t, "activity after the creation", n.cycleActivity(t, item.ID), activity)

	path := "/api/v0/issues/" + item.ID.String()
	added := w.joinWrite(t, "bob", http.MethodPatch, path, modulesBody(auth, shelved, billing), http.StatusOK)
	n.moment("added", added.UpdatedAt)
	w.send(t, "alice", http.MethodPost, "/api/v0/modules/"+shelved.String()+"/archive", "", http.StatusOK)
	activity = append(activity, `modules created ""→"Shelved" -→Shelved "added module " by bob at added`)
	replaced := w.joinWrite(t, "bob", http.MethodPatch, path, modulesBody(shelved, core), http.StatusOK)
	n.moment("replaced", replaced.UpdatedAt)
	if !slices.Equal(replaced.ModuleIDs, []uuid.UUID{core, shelved}) || n.audit(t, item.ID) != "by bob at replaced" {
		t.Fatalf("replaced into %v, last changed %s; want Core and Shelved, by bob at the change", replaced.ModuleIDs, n.audit(t, item.ID))
	}
	rows := []string{"Auth made by bob at made, left by bob at replaced, written at replaced",
		"Billing made by bob at made, left by bob at replaced, written at replaced", "Shelved made by bob at added", "Core made by bob at replaced"}
	wantRows(t, "rows after the replacement", n.moduleRows(t, item.ID), rows)
	activity = append(activity, `modules created ""→"Core" -→Core "added module " by bob at replaced`,
		`modules deleted "Auth"→"" Auth→- "removed this work item from module " by bob at replaced`,
		`modules deleted "Billing"→"" Billing→- "removed this work item from module " by bob at replaced`)
	wantRows(t, "activity after the replacement", n.cycleActivity(t, item.ID), activity)

	w.joinWrite(t, "bob", http.MethodPatch, path, modulesBody(core, shelved), http.StatusOK)
	wantRows(t, "rows after the same set", n.moduleRows(t, item.ID), rows)
	wantRows(t, "activity after the same set", n.cycleActivity(t, item.ID), activity)
	emptied := w.joinWrite(t, "alice", http.MethodPatch, path, `{"module_ids":[]}`, http.StatusOK)
	n.moment("emptied", emptied.UpdatedAt)
	if len(emptied.ModuleIDs) != 0 || emptied.ModuleIDs == nil || n.audit(t, item.ID) != "by alice at emptied" {
		t.Fatalf("emptied into %v, last changed %s; want [], by alice", emptied.ModuleIDs, n.audit(t, item.ID))
	}
	rows[2] = "Shelved made by bob at added, left by alice at emptied, written at emptied"
	rows[3] = "Core made by bob at replaced, left by alice at emptied, written at emptied"
	wantRows(t, "rows after emptying", n.moduleRows(t, item.ID), rows)
	activity = append(activity, `modules deleted "Core"→"" Core→- "removed this work item from module " by alice at emptied`,
		`modules deleted "Shelved"→"" Shelved→- "removed this work item from module " by alice at emptied`)
	wantRows(t, "activity after emptying", n.cycleActivity(t, item.ID), activity)
	w.joinWrite(t, "alice", http.MethodPatch, path, `{"module_ids":[]}`, http.StatusOK)
	wantRows(t, "rows after empty again", n.moduleRows(t, item.ID), rows)
	wantRows(t, "activity after empty again", n.cycleActivity(t, item.ID), activity)
	if n.audit(t, item.ID) != "by alice at emptied" {
		t.Errorf("a change of nothing changed the work item: %s", n.audit(t, item.ID))
	}
}

// One write changes the cycle and the modules together (M6 P3a spec 2.4,
// R4, R5): created in Sprint and Auth, its activity is the creation's row,
// the cycle's, then the module's; moved to Next and Billing, every row of
// both at the change's moment, its activity the cycle's row, then Billing
// joined, then Auth left, which issue.updated's changes are, in that order.
// Each write is on a business clock of its own, a minute apart.
func TestTheCycleAndTheModulesChangeInOneWrite(t *testing.T) {
	w := newIssueWorld(t)
	w.turnCyclesOn(t, w.web)
	w.turnModulesOn(t, w.web)
	sprint, next := w.newCycle(t, "alice", `{"name":"Sprint"}`), w.newCycle(t, "alice", `{"name":"Next"}`)
	auth, billing := w.newModule(t, `{"name":"Auth"}`), w.newModule(t, `{"name":"Billing"}`)
	n := cycleNames{w: w, cycles: map[uuid.UUID]string{sprint: "Sprint", next: "Next", auth: "Auth", billing: "Billing"}, moments: map[string]string{}}
	var received []shared.Event
	var failure error
	now := time.Now()
	route, later := issueEventRoute(t, w, clockAt(now), &received, &failure), issueEventRoute(t, w, clockAt(now.Add(time.Minute)), &received, &failure)
	var item joinedItem
	decodeAnswer(t, route.answerWithin(t, w.contract, http.MethodPost, "/api/v0/projects/"+w.web.String()+"/issues", w.ids["alice"],
		`{"name":"Fix","cycle_id":"`+sprint.String()+`","module_ids":["`+auth.String()+`"]}`, http.StatusCreated), &item)
	n.moment("made", item.CreatedAt)
	activity := []string{` created "-"→"-" -→- "created the work item" by alice at made`,
		`cycles created ""→"Sprint" -→Sprint "added cycle " by alice at made`, `modules created ""→"Auth" -→Auth "added module " by alice at made`}
	wantRows(t, "activity after the creation", n.cycleActivity(t, item.ID), activity)
	var moved joinedItem
	decodeAnswer(t, later.answerWithin(t, w.contract, http.MethodPatch, "/api/v0/issues/"+item.ID.String(), w.ids["alice"],
		`{"cycle_id":"`+next.String()+`","module_ids":["`+billing.String()+`"]}`, http.StatusOK), &moved)
	n.moment("moved", moved.UpdatedAt)
	if moved.CycleID == nil || *moved.CycleID != next || !slices.Equal(moved.ModuleIDs, []uuid.UUID{billing}) || n.audit(t, item.ID) != "by alice at moved" {
		t.Fatalf("moved into %v and %v, last changed %s; want Next and Billing, by alice then", moved.CycleID, moved.ModuleIDs, n.audit(t, item.ID))
	}
	wantRows(t, "cycle rows", n.cycleRows(t, item.ID), []string{"Sprint made by alice at made, left by alice at moved, written at moved",
		"Next made by alice at moved"})
	wantRows(t, "module rows", n.moduleRows(t, item.ID), []string{"Auth made by alice at made, left by alice at moved, written at moved",
		"Billing made by alice at moved"})
	activity = append(activity, `cycles updated "Sprint"→"Next" Sprint→Next "updated cycle from " by alice at moved`,
		`modules created ""→"Billing" -→Billing "added module " by alice at moved`,
		`modules deleted "Auth"→"" Auth→- "removed this work item from module " by alice at moved`)
	wantRows(t, "activity after the change", n.cycleActivity(t, item.ID), activity)
	if len(received) != 2 {
		t.Fatalf("events %+v, want the creation and the change", received)
	}
	created := received[0].Changes
	want := []shared.Change{{Field: "cycles", New: eventValue("Sprint"), NewID: &sprint}, {Field: "modules", New: eventValue("Auth"), NewID: &auth}}
	if last := created[len(created)-2:]; !reflect.DeepEqual(last, want) {
		t.Errorf("issue.created's last changes %+v, want %+v", last, want)
	}
	want = []shared.Change{{Field: "cycles", Old: eventValue("Sprint"), New: eventValue("Next"), OldID: &sprint, NewID: &next},
		{Field: "modules", Old: eventValue(""), New: eventValue("Billing"), NewID: &billing},
		{Field: "modules", Old: eventValue("Auth"), New: eventValue(""), OldID: &auth}}
	if e := received[1]; e.Type != "issue.updated" || !reflect.DeepEqual(e.Changes, want) {
		t.Errorf("event %s %+v, want issue.updated %+v", e.Type, e.Changes, want)
	}
}

// A work item's modules are its own write's (M6 P3a spec R7): walt, acme's
// admin and Web's guest who sees only his own work items (adminGuest),
// creates one in Auth and empties its modules, as any change of his own
// work item, while planning's writes are forbidden to him; alice's work
// item is hidden from him: issue.not_found, as for any field.
func TestOwnOnlyCallersChangeTheModulesOfTheirOwnWorkItems(t *testing.T) {
	w := newIssueWorld(t)
	w.turnModulesOn(t, w.web)
	auth := w.newModule(t, `{"name":"Auth"}`)
	hers := w.newWorkItemIn(t, w.web, `{"name":"Hers"}`)
	w.adminGuest(t)
	his := w.joinWrite(t, "walt", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/issues", `{"name":"His","module_ids":["`+auth.String()+`"]}`,
		http.StatusCreated)
	if !slices.Equal(his.ModuleIDs, []uuid.UUID{auth}) {
		t.Fatalf("his work item in %v, want Auth", his.ModuleIDs)
	}
	if left := w.joinWrite(t, "walt", http.MethodPatch, "/api/v0/issues/"+his.ID.String(), `{"module_ids":[]}`, http.StatusOK); len(left.ModuleIDs) != 0 {
		t.Errorf("his work item left into %v, want none", left.ModuleIDs)
	}
	w.refusedModule(t, "walt", w.web, `{"name":"His own"}`, http.StatusForbidden, "forbidden")
	answer := w.send(t, "walt", http.MethodPatch, "/api/v0/issues/"+hers.String(), modulesBody(auth), http.StatusNotFound)
	if code := refusalOf(t, []byte(answer), false); code != "issue.not_found" {
		t.Errorf("walt's change of her work item's modules = %s, want issue.not_found", answer)
	}
}
