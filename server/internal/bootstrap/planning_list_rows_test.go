package bootstrap

import (
	"context"
	"log/slog"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning"
	planningpg "github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/postgres"
	planningdomain "github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// cycleRowAnswer is a cycle's list row as the API answers it: the cycle,
// its counts, read into the domain's, whose JSON names are the API's, its
// assignees and its frozen counts (M6 P4 spec 2.3).
type cycleRowAnswer struct {
	cycleAnswer
	planningdomain.Counts
	AssigneeIDs []uuid.UUID                  `json:"assignee_ids"`
	Snapshot    *planningdomain.FrozenCounts `json:"progress_snapshot"`
}

// moduleRowAnswer is a module's list row as the API answers it: the module
// and its counts.
type moduleRowAnswer struct {
	moduleAnswer
	planningdomain.Counts
}

// cycleRows are the rows of the account name's list of Web's cycles with
// query, by cycle.
func (w issueWorld) cycleRows(t *testing.T, name, query string) map[uuid.UUID]cycleRowAnswer {
	t.Helper()
	rows := map[uuid.UUID]cycleRowAnswer{}
	for _, r := range listIn[cycleRowAnswer](t, w.send(t, name, http.MethodGet, "/api/v0/projects/"+w.web.String()+"/cycles"+query, "", http.StatusOK)) {
		rows[r.ID] = r
	}
	return rows
}

// moduleRows are cycleRows of Web's modules.
func (w issueWorld) moduleRows(t *testing.T, name, query string) map[uuid.UUID]moduleRowAnswer {
	t.Helper()
	rows := map[uuid.UUID]moduleRowAnswer{}
	for _, r := range listIn[moduleRowAnswer](t, w.send(t, name, http.MethodGet, "/api/v0/projects/"+w.web.String()+"/modules"+query, "", http.StatusOK)) {
		rows[r.ID] = r
	}
	return rows
}

// cycleRow is a cycle's expected numbers in a list row: its counts, its
// assignees, and its frozen counts, nil for none.
type cycleRow struct {
	counts    planningdomain.Counts
	assignees []uuid.UUID
	frozen    *planningdomain.Frozen
}

// frozenAs is a frozen progress of counts, at, without assignees, labels or
// burndown, as the store keeps one.
func frozenAs(counts planningdomain.Counts, at time.Time) *planningdomain.Frozen {
	return &planningdomain.Frozen{Progress: planningdomain.Progress{Counts: counts, Assignees: []planningdomain.AssigneeDistribution{},
		Labels: []planningdomain.LabelDistribution{}}, FrozenAt: at}
}

// The app answers a cycle's list rows as the contract says (M6 P4 spec 2.1,
// 2.3, R1, R5, R9, R10), in Web's list and its archived catalog. Sprint, in
// Shanghai from two days ago to two days ahead, has a in Todo, bob's; b in
// Done, bob's and alice's; c, bob's work item, in progress; d, his,
// cancelled, his; e, walt's, in Backlog; f, walt's, in Done, alice's; g,
// a's sub-item, in Todo; k in Todo, which wanda was assigned to and taken
// off. Not counted, each assigned to wanda: h archived, i deleted, j moved
// to Other. Quiet, a draft, has none; Old, archived, has l in progress,
// wanda's, and m, walt's, in Done. Its members count them all, each assignee
// once, wanda only where she is on a counted work item; walt, acme's admin
// and Web's guest, counts only his own, and is not answered the frozen
// counts; once Web shows its guests every work item, he counts them all and
// is answered them. Each row's numbers are those of the cycle's progress
// for the same caller.
func TestListCyclesWithCountsThroughTheApp(t *testing.T) {
	w, now := newNoonWorld(t)
	w.turnCyclesOn(t, w.web)
	w.adminGuest(t)
	w.makeMember(t, "wanda", w.web, shared.RoleAdmin)
	sprint := w.newCycle(t, "alice", datesBody("Sprint", shanghaiDay(t, now, -2), shanghaiDay(t, now, 2)))
	other := w.newCycle(t, "alice", datesBody("Other", shanghaiDay(t, now, 10), shanghaiDay(t, now, 12)))
	quiet := w.newCycle(t, "alice", `{"name":"Quiet"}`)
	old := w.newCycle(t, "alice", datesBody("Old", shanghaiDay(t, now, 20), shanghaiDay(t, now, 22)))
	backlog, started, cancelled := stateID(t, w.pool, w.web, "Backlog"), stateID(t, w.pool, w.web, "In Progress"), stateID(t, w.pool, w.web, "Cancelled")
	alice, bob, wanda := w.ids["alice"], w.ids["bob"], w.ids["wanda"]
	item := func(by, name string, cycle, state uuid.UUID, assignees []uuid.UUID, more string) uuid.UUID {
		return w.newWorkItemBy(t, by, w.web, workItemBody(name, state, assignees, nil, `,"cycle_id":"`+cycle.String()+`"`+more))
	}
	a := item("alice", "A", sprint, w.todo, []uuid.UUID{bob}, "")
	item("alice", "B", sprint, w.done, []uuid.UUID{bob, alice}, "")
	item("bob", "C", sprint, started, nil, "")
	item("bob", "D", sprint, cancelled, []uuid.UUID{bob}, "")
	item("walt", "E", sprint, backlog, nil, "")
	item("walt", "F", sprint, w.done, []uuid.UUID{alice}, "")
	item("alice", "G", sprint, w.todo, nil, `,"parent_id":"`+a.String()+`"`)
	h, i := item("alice", "H", sprint, w.todo, []uuid.UUID{wanda}, ""), item("alice", "I", sprint, w.todo, []uuid.UUID{wanda}, "")
	j, k := item("alice", "J", sprint, w.todo, []uuid.UUID{wanda}, ""), item("alice", "K", sprint, w.todo, []uuid.UUID{wanda}, "")
	item("alice", "L", old, started, []uuid.UUID{wanda}, "")
	item("walt", "M", old, w.done, nil, "")
	listSeedSQL(t, w, "UPDATE issues SET archived_at = now() WHERE id = $1", h)
	w.send(t, "alice", http.MethodDelete, "/api/v0/issues/"+i.String(), "", http.StatusNoContent)
	w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+j.String(), `{"cycle_id":"`+other.String()+`"}`, http.StatusOK)
	w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+k.String(), `{"assignee_ids":[]}`, http.StatusOK)
	listSeedSQL(t, w, "UPDATE cycles SET archived_at = now() WHERE id = $1", old)
	people := []uuid.UUID{alice, bob}
	slices.SortFunc(people, uuid.UUID.Compare)

	everyone := map[uuid.UUID]cycleRow{
		sprint: {counts: planningdomain.Counts{Total: 8, Backlog: 1, Unstarted: 3, Started: 1, Completed: 2, Cancelled: 1}, assignees: people},
		other:  {counts: planningdomain.Counts{Total: 1, Unstarted: 1}, assignees: []uuid.UUID{wanda}},
		quiet:  {assignees: []uuid.UUID{}},
		old:    {counts: planningdomain.Counts{Total: 2, Started: 1, Completed: 1}, assignees: []uuid.UUID{wanda}},
	}
	walts := map[uuid.UUID]cycleRow{
		sprint: {counts: planningdomain.Counts{Total: 2, Backlog: 1, Completed: 1}, assignees: []uuid.UUID{alice}},
		other:  {assignees: []uuid.UUID{}},
		quiet:  {assignees: []uuid.UUID{}},
		old:    {counts: planningdomain.Counts{Total: 1, Completed: 1}, assignees: []uuid.UUID{}},
	}
	lists := func(t *testing.T, name string, want map[uuid.UUID]cycleRow) {
		t.Helper()
		for query, ids := range map[string][]uuid.UUID{"": {sprint, other, quiet}, "?archived=true": {old}} {
			rows := w.cycleRows(t, name, query)
			if len(rows) != len(ids) {
				t.Fatalf("%s lists %d cycles with %q, want %v", name, len(rows), query, ids)
			}
			for _, id := range ids {
				got, row := rows[id], want[id]
				var snapshot *planningdomain.FrozenCounts
				if row.frozen != nil {
					snapshot = &planningdomain.FrozenCounts{Counts: row.frozen.Counts, FrozenAt: row.frozen.FrozenAt}
				}
				if got.Name == "" || got.Counts != row.counts || got.AssigneeIDs == nil || !slices.Equal(got.AssigneeIDs, row.assignees) ||
					!reflect.DeepEqual(got.Snapshot, snapshot) {
					t.Errorf("%s lists %s (%q) with %+v, %v, %+v; want %+v, %v, %+v", name, got.Name, query, got.Counts, got.AssigneeIDs,
						got.Snapshot, row.counts, row.assignees, snapshot)
				}
				// The row's numbers are the cycle's progress's, for the same caller.
				p := w.cycleProgress(t, name, id)
				if p.Live.Counts != got.Counts || (p.Snapshot == nil) != (got.Snapshot == nil) ||
					(p.Snapshot != nil && (p.Snapshot.Counts != got.Snapshot.Counts || !p.Snapshot.FrozenAt.Equal(got.Snapshot.FrozenAt))) {
					t.Errorf("%s lists %s with %+v, %+v; its progress is %+v, %+v", name, got.Name, got.Counts, got.Snapshot, p.Live.Counts, p.Snapshot)
				}
			}
		}
	}
	lists(t, "alice", everyone)
	lists(t, "bob", everyone)
	lists(t, "walt", walts)

	// Sprint and Old frozen through the store: their frozen counts and
	// moments are answered to who sees every work item, not to walt.
	store := planningpg.New(w.pool)
	for id, f := range map[uuid.UUID]*planningdomain.Frozen{
		sprint: frozenAs(planningdomain.Counts{Total: 5, Started: 2, Completed: 3}, time.Date(2026, 11, 1, 2, 3, 4, 567891000, time.UTC)),
		old:    frozenAs(planningdomain.Counts{Total: 4, Cancelled: 4}, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)),
	} {
		if ok, err := store.FreezeCycle(context.Background(), id, *f); err != nil || !ok {
			t.Fatalf("freezing %s: %t, %v", id, ok, err)
		}
		row := everyone[id]
		row.frozen = f
		everyone[id] = row
	}
	lists(t, "alice", everyone)
	lists(t, "walt", walts)
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"guest_view_all_features":true}`, http.StatusOK)
	lists(t, "walt", everyone)
}

// The app answers a module's list rows as the contract says (M6 P4 spec
// 2.1, 2.3, R1, R9, R10), in Web's list and its archived catalog: Core has
// a, alice's, in Todo; b, bob's, in Done; c, walt's, in Backlog; and, not
// counted, d archived, e deleted, f taken out of it. Loose has a too, which
// counts in each; Idle has none; Shelved, archived, has g, walt's, in
// progress. Its members count them all; walt, acme's admin and Web's
// guest, counts only his own. Each row's counts are those of the module's
// progress for the same caller.
func TestListModulesWithCountsThroughTheApp(t *testing.T) {
	w := newIssueWorld(t)
	w.turnModulesOn(t, w.web)
	w.adminGuest(t)
	module := func(name string) uuid.UUID {
		return w.createModule(t, "alice", `{"name":"`+name+`"}`, http.StatusCreated).ID
	}
	core, loose, idle, shelved := module("Core"), module("Loose"), module("Idle"), module("Shelved")
	backlog, started := stateID(t, w.pool, w.web, "Backlog"), stateID(t, w.pool, w.web, "In Progress")
	item := func(by, name string, state uuid.UUID, modules ...uuid.UUID) uuid.UUID {
		return w.newWorkItemBy(t, by, w.web, workItemBody(name, state, nil, nil, `,"module_ids":`+idArray(modules...)))
	}
	item("alice", "A", w.todo, core, loose)
	item("bob", "B", w.done, core)
	item("walt", "C", backlog, core)
	d, e, f := item("alice", "D", w.todo, core), item("alice", "E", w.todo, core), item("alice", "F", w.todo, core)
	item("walt", "G", started, shelved)
	listSeedSQL(t, w, "UPDATE issues SET archived_at = now() WHERE id = $1", d)
	w.send(t, "alice", http.MethodDelete, "/api/v0/issues/"+e.String(), "", http.StatusNoContent)
	w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+f.String(), `{"module_ids":[]}`, http.StatusOK)
	listSeedSQL(t, w, "UPDATE modules SET archived_at = now() WHERE id = $1", shelved)

	everyone := map[uuid.UUID]planningdomain.Counts{core: {Total: 3, Backlog: 1, Unstarted: 1, Completed: 1}, loose: {Total: 1, Unstarted: 1},
		idle: {}, shelved: {Total: 1, Started: 1}}
	walts := map[uuid.UUID]planningdomain.Counts{core: {Total: 1, Backlog: 1}, loose: {}, idle: {}, shelved: {Total: 1, Started: 1}}
	for name, want := range map[string]map[uuid.UUID]planningdomain.Counts{"alice": everyone, "bob": everyone, "walt": walts} {
		for query, ids := range map[string][]uuid.UUID{"": {core, loose, idle}, "?archived=true": {shelved}} {
			rows := w.moduleRows(t, name, query)
			if len(rows) != len(ids) {
				t.Fatalf("%s lists %d modules with %q, want %v", name, len(rows), query, ids)
			}
			for _, id := range ids {
				if got := rows[id]; got.Name == "" || got.Counts != want[id] {
					t.Errorf("%s lists %s (%q) with %+v, want %+v", name, got.Name, query, got.Counts, want[id])
				}
				if p := w.moduleProgress(t, name, id); p.Counts != rows[id].Counts {
					t.Errorf("%s lists %s with %+v; its progress counts %+v", name, rows[id].Name, rows[id].Counts, p.Counts)
				}
			}
		}
	}
}

func (d probedStats) CycleCounts(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID, viewer *uuid.UUID) ([]planning.StateCount, error) {
	counts, err := d.planningIssues.CycleCounts(ctx, projectID, ids, viewer)
	if err == nil {
		err = d.p.saw(ctx, "counts")
	}
	return counts, err
}

func (d probedStats) ModuleCounts(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID, viewer *uuid.UUID) ([]planning.StateCount, error) {
	counts, err := d.planningIssues.ModuleCounts(ctx, projectID, ids, viewer)
	if err == nil {
		err = d.p.saw(ctx, "counts")
	}
	return counts, err
}

func (d probedStats) CycleAssignees(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID, viewer *uuid.UUID) ([]planning.CycleAssignee, error) {
	assignees, err := d.planningIssues.CycleAssignees(ctx, projectID, ids, viewer)
	if err == nil {
		err = d.p.saw(ctx, "assignees")
	}
	return assignees, err
}

// A list of three cycles, and one of three modules, reads each of the
// issue module's numbers once for its page, not once per row (M6 P4 spec
// A5), in its one read-only snapshot with the project's facts and its
// states' groups.
func TestPlanningListsReadEachPageOnceInOneSnapshot(t *testing.T) {
	w := newIssueWorld(t)
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"cycle_view":true,"module_view":true}`, http.StatusOK)
	for _, name := range []string{"One", "Two", "Three"} {
		cycle := w.newCycle(t, "alice", `{"name":"`+name+`"}`)
		module := w.createModule(t, "alice", `{"name":"`+name+`"}`, http.StatusCreated).ID
		w.newWorkItemBy(t, "alice", w.web, `{"name":"A","assignee_ids":["`+w.ids["bob"].String()+`"],"cycle_id":"`+cycle.String()+
			`","module_ids":["`+module.String()+`"]}`)
	}
	probe := &progressProbe{pool: w.pool}
	route := newRoute(t, planning.New(planning.Deps{Pool: w.pool, Tx: postgres.NewTxManager(w.pool, 2*time.Second),
		ReadTx: postgres.NewTxManager(w.pool, 2*time.Second, postgres.ReadOnlySnapshot()), JobTx: postgres.NewTxManager(w.pool, 2*time.Second),
		RuntimeNow: time.Now, Logger: slog.New(slog.DiscardHandler), DeleteBatch: 1000, DeleteBudget: 45 * time.Second,
		WriteGate: newWriteGate(testIssueConfig.WriteGate), Clock: clock.System{}, Authorizer: authorizerOn(w.pool),
		Projects:      probedProjects{planningProjects: planningProjects{projects: project.Provide(w.pool, nil).PlanningProjects}, p: probe},
		IssueStats:    probedStats{planningIssues: planningIssues{stats: issue.Provide(w.pool, nil).PlanningStats}, p: probe},
		IssueTransfer: planningTransfer{transfer: issue.Provide(w.pool, nil).PlanningTransfer},
		ListSettings:  issue.Provide(w.pool, nil).ListSettings}).Register)
	for path, want := range map[string][]string{
		"/api/v0/projects/" + w.web.String() + "/cycles":  {"facts", "groups", "counts", "assignees"},
		"/api/v0/projects/" + w.web.String() + "/modules": {"facts", "groups", "counts"},
	} {
		probe.reads = nil
		_, rec := route.send(http.MethodGet, path, w.ids["alice"], "")
		if rec.Code != http.StatusOK || len(listIn[map[string]any](t, rec.Body.String())) != 3 {
			t.Fatalf("GET %s = %d %s, want three rows", path, rec.Code, rec.Body)
		}
		var names []string
		for _, r := range probe.reads {
			names = append(names, r.read)
			first := probe.reads[0]
			if r.isolation != "repeatable read" || r.readOnly != "on" || r.backend != first.backend || !r.began.Equal(first.began) {
				t.Errorf("GET %s: %+v, read in another transaction than %+v, or not in a read-only snapshot", path, r, first)
			}
		}
		if !slices.Equal(names, want) {
			t.Errorf("GET %s read %q, want %q", path, names, want)
		}
	}
}
