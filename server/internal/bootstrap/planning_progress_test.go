package bootstrap

import (
	"context"
	"maps"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	planningpg "github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/postgres"
	planningdomain "github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// cycleProgressAnswer is a cycle's progress as the API answers it, its
// progress read into the domain's values, whose JSON names are the API's
// (M6 P4 spec 2.3, 2.4).
type cycleProgressAnswer struct {
	Status   string                  `json:"status"`
	Live     planningdomain.Progress `json:"live"`
	Snapshot *planningdomain.Frozen  `json:"progress_snapshot"`
}

// cycleProgress is the account name's read of the progress of the cycle id.
func (w issueWorld) cycleProgress(t *testing.T, name string, id uuid.UUID) cycleProgressAnswer {
	t.Helper()
	var p cycleProgressAnswer
	decodeAnswer(t, w.send(t, name, http.MethodGet, "/api/v0/cycles/"+id.String()+"/progress", "", http.StatusOK), &p)
	return p
}

// moduleProgress is the account name's read of the progress of the module id.
func (w issueWorld) moduleProgress(t *testing.T, name string, id uuid.UUID) planningdomain.Progress {
	t.Helper()
	var p planningdomain.Progress
	decodeAnswer(t, w.send(t, name, http.MethodGet, "/api/v0/modules/"+id.String()+"/progress", "", http.StatusOK), &p)
	return p
}

// listTotal is the total of the account name's list of Web's work items
// filtered by query, sub-items in and archived ones out: the list engine's
// count of what a progress counts (M6 P4 spec R1).
func (w issueWorld) listTotal(t *testing.T, name, query string) int64 {
	t.Helper()
	path := "/api/v0/projects/" + w.web.String() + "/issues?" + query + "&sub_issue=true&archived=false"
	return checkListEnvelope(t, w.send(t, name, http.MethodGet, path, "", http.StatusOK), false, false).Total
}

// workItemBody is the body of a work item named name in state, with the
// assignees, labels and the rest more gives.
func workItemBody(name string, state uuid.UUID, assignees, labels []uuid.UUID, more string) string {
	return `{"name":"` + name + `","state_id":"` + state.String() + `","assignee_ids":` + idArray(assignees...) + `,"label_ids":` +
		idArray(labels...) + more + `}`
}

// burndown is a burndown from start to end, YYYY-MM-DD, of scope, whose
// days have left a number each, in order, and none after them.
func burndown(t *testing.T, start, end string, scope int64, left ...int64) *planningdomain.Burndown {
	t.Helper()
	from, to := seedDay(t, start), seedDay(t, end)
	b := &planningdomain.Burndown{StartDate: from, EndDate: to, Scope: scope, Points: []planningdomain.BurndownPoint{}}
	for d, i := from, 0; d.Compare(to) <= 0; d, i = shared.DateOf(d.Time().AddDate(0, 0, 1)), i+1 {
		point := planningdomain.BurndownPoint{Date: d}
		if i < len(left) {
			point.Remaining = &left[i]
		}
		b.Points = append(b.Points, point)
	}
	return b
}

// tally is a distribution's row: completed and pending work items.
func tally(completed, pending int64) planningdomain.Tally {
	return planningdomain.Tally{Total: completed + pending, Completed: completed, Pending: pending}
}

// byAssignee are the rows of a distribution by assignee, by id, the row of
// none last.
func byAssignee(rows map[uuid.UUID]planningdomain.Tally, none *planningdomain.Tally) []planningdomain.AssigneeDistribution {
	out := []planningdomain.AssigneeDistribution{}
	for _, id := range slices.SortedFunc(maps.Keys(rows), uuid.UUID.Compare) {
		out = append(out, planningdomain.AssigneeDistribution{AssigneeID: &id, Tally: rows[id]})
	}
	if none != nil {
		out = append(out, planningdomain.AssigneeDistribution{Tally: *none})
	}
	return out
}

// byLabel is byAssignee of the labels.
func byLabel(rows map[uuid.UUID]planningdomain.Tally, none *planningdomain.Tally) []planningdomain.LabelDistribution {
	out := []planningdomain.LabelDistribution{}
	for _, a := range byAssignee(rows, none) {
		out = append(out, planningdomain.LabelDistribution{LabelID: a.AssigneeID, Tally: a.Tally})
	}
	return out
}

// The app answers a cycle's progress as the contract says (M6 P4 spec 2.1,
// 2.3, R1, R5, R10). Sprint, in Shanghai from two days ago to two days
// ahead, has a in Todo, bob's, labelled Bug; b in Done, bob's and alice's,
// Bug and UI; c, bob's work item, in progress, UI; d, his, cancelled; e,
// walt's, in Backlog; f, walt's, in Done, alice's; g, a's sub-item, in
// Todo; and, not counted, h archived, i deleted, j moved to Other. b and f
// are done today. Its members count them all, as the list engine does; walt,
// acme's admin and Web's guest, sees only his own, e and f, and not the
// frozen progress, which is the store's; once Web shows its guests every
// work item, he counts them all and sees it. Reads go on with the cycle and
// the project archived and cycles off; a deleted cycle is not found, a
// non-member who sees Web is refused, one who does not see it is not found.
func TestReadCycleProgressThroughTheApp(t *testing.T) {
	w := newIssueWorld(t)
	w.turnCyclesOn(t, w.web)
	w.adminGuest(t)
	now := time.Now()
	start, end := shanghaiDay(t, now, -2), shanghaiDay(t, now, 2)
	sprint := w.newCycle(t, "alice", datesBody("Sprint", start, end))
	other := w.newCycle(t, "alice", datesBody("Other", shanghaiDay(t, now, 10), shanghaiDay(t, now, 12)))
	backlog, started, cancelled := stateID(t, w.pool, w.web, "Backlog"), stateID(t, w.pool, w.web, "In Progress"), stateID(t, w.pool, w.web, "Cancelled")
	var label struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/labels", `{"name":"UI"}`, http.StatusCreated), &label)
	ui, alice, bob := label.ID, w.ids["alice"], w.ids["bob"]
	inSprint := `,"cycle_id":"` + sprint.String() + `"`
	item := func(by, name string, state uuid.UUID, assignees, labels []uuid.UUID, more string) uuid.UUID {
		return w.newWorkItemBy(t, by, w.web, workItemBody(name, state, assignees, labels, inSprint+more))
	}
	a := item("alice", "A", w.todo, []uuid.UUID{bob}, []uuid.UUID{w.bug}, "")
	item("alice", "B", w.done, []uuid.UUID{bob, alice}, []uuid.UUID{w.bug, ui}, "")
	item("bob", "C", started, nil, []uuid.UUID{ui}, "")
	item("bob", "D", cancelled, []uuid.UUID{bob}, nil, "")
	item("walt", "E", backlog, nil, nil, "")
	item("walt", "F", w.done, []uuid.UUID{alice}, nil, "")
	item("alice", "G", w.todo, nil, nil, `,"parent_id":"`+a.String()+`"`)
	h, i, j := item("alice", "H", w.todo, nil, nil, ""), item("alice", "I", w.todo, nil, nil, ""), item("alice", "J", w.todo, nil, nil, "")
	listSeedSQL(t, w, "UPDATE issues SET archived_at = now() WHERE id = $1", h)
	w.send(t, "alice", http.MethodDelete, "/api/v0/issues/"+i.String(), "", http.StatusNoContent)
	w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+j.String(), `{"cycle_id":"`+other.String()+`"}`, http.StatusOK)

	everyone := planningdomain.Progress{
		Counts:    planningdomain.Counts{Total: 7, Backlog: 1, Unstarted: 2, Started: 1, Completed: 2, Cancelled: 1},
		Assignees: byAssignee(map[uuid.UUID]planningdomain.Tally{alice: tally(2, 0), bob: tally(1, 1)}, new(tally(0, 3))),
		Labels:    byLabel(map[uuid.UUID]planningdomain.Tally{w.bug: tally(1, 1), ui: tally(1, 1)}, new(tally(1, 2))),
		Burndown:  burndown(t, start, end, 6, 6, 6, 4),
	}
	walts := planningdomain.Progress{
		Counts:    planningdomain.Counts{Total: 2, Backlog: 1, Completed: 1},
		Assignees: byAssignee(map[uuid.UUID]planningdomain.Tally{alice: tally(1, 0)}, new(tally(0, 1))),
		Labels:    byLabel(nil, new(tally(1, 1))),
		Burndown:  burndown(t, start, end, 2, 2, 2, 1),
	}
	read := func(t *testing.T, name string, live planningdomain.Progress, snapshot *planningdomain.Frozen) {
		t.Helper()
		if got, want := w.cycleProgress(t, name, sprint), (cycleProgressAnswer{Status: "current", Live: live, Snapshot: snapshot}); !reflect.DeepEqual(got, want) {
			t.Errorf("%s reads\n%+v\nwant\n%+v", name, got, want)
		}
		if got := w.listTotal(t, name, "cycle_id="+sprint.String()); got != live.Total {
			t.Errorf("%s lists %d work items in Sprint; the progress counts %d", name, got, live.Total)
		}
	}
	read(t, "alice", everyone, nil)
	read(t, "bob", everyone, nil)
	read(t, "walt", walts, nil)

	// Frozen through the store: read by who sees every work item, not by walt.
	frozen := planningdomain.Frozen{Progress: walts, FrozenAt: time.Date(2026, 11, 1, 2, 3, 4, 567891000, time.UTC)}
	if ok, err := planningpg.New(w.pool).FreezeCycle(context.Background(), sprint, frozen); err != nil || !ok {
		t.Fatalf("freezing Sprint: %t, %v", ok, err)
	}
	read(t, "alice", everyone, &frozen)
	read(t, "walt", walts, nil)
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"guest_view_all_features":true}`, http.StatusOK)
	read(t, "walt", everyone, &frozen)

	// A deleted cycle, and one that never was, are not found; wanda, acme's
	// admin and none of Web's members, sees Web and may not; olga does not.
	gone := w.newCycle(t, "alice", `{"name":"Gone"}`)
	w.send(t, "alice", http.MethodDelete, "/api/v0/cycles/"+gone.String(), "", http.StatusNoContent)
	for _, tt := range []struct {
		name   string
		cycle  uuid.UUID
		status int
		code   string
	}{
		{"alice", gone, http.StatusNotFound, "planning.cycle_not_found"},
		{"alice", uuid.NewV7(), http.StatusNotFound, "planning.cycle_not_found"},
		{"wanda", sprint, http.StatusForbidden, "forbidden"},
		{"olga", sprint, http.StatusNotFound, "planning.cycle_not_found"},
	} {
		w.refusedRead(t, tt.name, "/api/v0/cycles/"+tt.cycle.String()+"/progress", tt.status, tt.code)
	}

	// The cycle archived, cycles turned off, the project archived: read as before.
	listSeedSQL(t, w, "UPDATE cycles SET archived_at = now() WHERE id = $1", sprint)
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"cycle_view":false}`, http.StatusOK)
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/archive", "", http.StatusOK)
	read(t, "bob", everyone, &frozen)
	read(t, "walt", everyone, &frozen)
}

// The app answers a module's progress as a cycle's (M6 P4 spec 2.1, 2.3,
// R10), its burndown from its start to its target day: Core, in Shanghai
// from yesterday to tomorrow, has a, alice's, in Todo; b, bob's, done today;
// c, walt's, in Backlog; and d, archived, not counted. Loose, which has a
// start alone, has a, and no burndown. walt, acme's admin and Web's guest,
// counts only his own, as the list engine does. Reads go on with the module
// and the project archived and modules off; a deleted module is not found,
// a non-member who sees Web is refused, one who does not see it is not
// found.
func TestReadModuleProgressThroughTheApp(t *testing.T) {
	w := newIssueWorld(t)
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"module_view":true,"timezone":"Asia/Shanghai"}`, http.StatusOK)
	w.adminGuest(t)
	now := time.Now()
	start, target := shanghaiDay(t, now, -1), shanghaiDay(t, now, 1)
	core := w.createModule(t, "alice", `{"name":"Core","start_date":"`+start+`","target_date":"`+target+`"}`, http.StatusCreated).ID
	loose := w.createModule(t, "alice", `{"name":"Loose","start_date":"`+start+`"}`, http.StatusCreated).ID
	backlog := stateID(t, w.pool, w.web, "Backlog")
	in := func(modules ...uuid.UUID) string { return `,"module_ids":` + idArray(modules...) }
	w.newWorkItemBy(t, "alice", w.web, workItemBody("A", w.todo, nil, nil, in(core, loose)))
	w.newWorkItemBy(t, "bob", w.web, workItemBody("B", w.done, nil, nil, in(core)))
	w.newWorkItemBy(t, "walt", w.web, workItemBody("C", backlog, nil, nil, in(core)))
	d := w.newWorkItemBy(t, "alice", w.web, workItemBody("D", w.todo, nil, nil, in(core)))
	listSeedSQL(t, w, "UPDATE issues SET archived_at = now() WHERE id = $1", d)

	everyone := planningdomain.Progress{Counts: planningdomain.Counts{Total: 3, Backlog: 1, Unstarted: 1, Completed: 1},
		Assignees: byAssignee(nil, new(tally(1, 2))), Labels: byLabel(nil, new(tally(1, 2))), Burndown: burndown(t, start, target, 3, 3, 2)}
	walts := planningdomain.Progress{Counts: planningdomain.Counts{Total: 1, Backlog: 1},
		Assignees: byAssignee(nil, new(tally(0, 1))), Labels: byLabel(nil, new(tally(0, 1))), Burndown: burndown(t, start, target, 1, 1, 1)}
	read := func(t *testing.T, name string, want planningdomain.Progress) {
		t.Helper()
		if got := w.moduleProgress(t, name, core); !reflect.DeepEqual(got, want) {
			t.Errorf("%s reads\n%+v\nwant\n%+v", name, got, want)
		}
		if got := w.listTotal(t, name, "module_id="+core.String()); got != want.Total {
			t.Errorf("%s lists %d work items in Core; the progress counts %d", name, got, want.Total)
		}
	}
	read(t, "alice", everyone)
	read(t, "bob", everyone)
	read(t, "walt", walts)
	if got := w.moduleProgress(t, "alice", loose); got.Total != 1 || got.Unstarted != 1 || got.Burndown != nil {
		t.Errorf("Loose = %+v, want a alone, without a burndown", got)
	}

	gone := w.createModule(t, "alice", `{"name":"Gone"}`, http.StatusCreated).ID
	w.send(t, "alice", http.MethodDelete, "/api/v0/modules/"+gone.String(), "", http.StatusNoContent)
	for _, tt := range []struct {
		name   string
		module uuid.UUID
		status int
		code   string
	}{
		{"alice", gone, http.StatusNotFound, "planning.module_not_found"},
		{"alice", uuid.NewV7(), http.StatusNotFound, "planning.module_not_found"},
		{"wanda", core, http.StatusForbidden, "forbidden"},
		{"olga", core, http.StatusNotFound, "planning.module_not_found"},
	} {
		w.refusedRead(t, tt.name, "/api/v0/modules/"+tt.module.String()+"/progress", tt.status, tt.code)
	}

	listSeedSQL(t, w, "UPDATE modules SET archived_at = now() WHERE id = $1", core)
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"module_view":false}`, http.StatusOK)
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/archive", "", http.StatusOK)
	read(t, "bob", everyone)
	read(t, "walt", walts)
}
