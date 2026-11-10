package postgresadapter_test

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// statsWorld is the work items the planning statistics read (M6 P4 spec 2.1, R2), of the cycles, or of the modules,
// Target and Other of Web and Far of Ops. In Target: a, alice's, in Web's default state, completed at now, assigned
// to carol and dave and labelled L1 and L2, with a deleted assignment of erin, a deleted label L3 and an earlier
// assignment of carol deleted before she was assigned again; b, bob's, in Done, assigned to carol; c, nobody's (its
// creator gone). Also in Target, and not counted: gone, deleted with its membership left; shelved, archived; left,
// whose membership is deleted; each assigned to erin. In Other: g, alice's, assigned to erin (and, of the modules, a
// too: a work item may be in several modules, one cycle). In Far: h, of Ops. Empty has none. Each row is written in
// the reverse of its id order, so a statement that does not order them answers them out of order.
type statsWorld struct {
	s                                  *postgresadapter.Store
	web, ops                           project
	done                               uuid.UUID
	alice, bob, carol, dave, erin      uuid.UUID
	l1, l2                             uuid.UUID
	target, other, far, empty          uuid.UUID
	a, b, c, g, h, gone, shelved, left uuid.UUID
}

func newStatsWorld(t *testing.T, table, column string, newTarget func(*testing.T, *pgxpool.Pool, project, string) uuid.UUID) statsWorld {
	t.Helper()
	s, pool := newStore(t)
	w := statsWorld{s: s}
	w.alice, w.bob = newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	w.carol, w.dave, w.erin = newAccount(t, pool, "carol@corp.com"), newAccount(t, pool, "dave@corp.com"), newAccount(t, pool, "erin@corp.com")
	acme := newWorkspace(t, pool, "acme")
	w.web, w.ops = newProject(t, pool, acme, "WEB"), newProject(t, pool, acme, "OPS")
	w.done = uuid.NewV7()
	exec(t, pool, `INSERT INTO states (id, workspace_id, project_id, name, color, "group") VALUES ($1, $2, $3, 'Done', '#46A758', 'completed')`,
		w.done, w.web.workspace, w.web.id)
	w.l1, w.l2 = newLabel(t, pool, w.web, "L1"), newLabel(t, pool, w.web, "L2")
	l3 := newLabel(t, pool, w.web, "L3")
	w.target, w.other, w.empty = newTarget(t, pool, w.web, "Target"), newTarget(t, pool, w.web, "Other"), newTarget(t, pool, w.web, "Empty")
	w.far = newTarget(t, pool, w.ops, "Far")
	ids := make([]uuid.UUID, 8)
	for n := range ids {
		ids[n] = uuid.NewV7()
	}
	w.a, w.b, w.c, w.g, w.gone, w.shelved, w.left, w.h = ids[0], ids[1], ids[2], ids[3], ids[4], ids[5], ids[6], ids[7]
	for n := len(ids) - 1; n >= 0; n-- {
		p, by := w.web, w.alice
		switch ids[n] {
		case w.h:
			p = w.ops
		case w.b:
			by = w.bob
		}
		newIssue(t, pool, p, ids[n], n+1, by)
	}
	exec(t, pool, "UPDATE issues SET completed_at = $2 WHERE id = $1", w.a, now)
	exec(t, pool, "UPDATE issues SET state_id = $2 WHERE id = $1", w.b, w.done)
	exec(t, pool, "UPDATE issues SET created_by_id = NULL WHERE id = $1", w.c)
	exec(t, pool, "UPDATE issues SET deleted_at = $2 WHERE id = $1", w.gone, now)
	exec(t, pool, "UPDATE issues SET archived_at = $2 WHERE id = $1", w.shelved, now)
	put := func(p project, table, column string, issue, value uuid.UUID, gone bool) {
		joinTarget(t, pool, p, table, column, issue, value, gone)
	}
	for _, issue := range []uuid.UUID{w.left, w.shelved, w.gone, w.c, w.b, w.a} {
		put(w.web, table, column, issue, w.target, issue == w.left)
	}
	if table == "module_issues" {
		put(w.web, table, column, w.a, w.other, false)
	}
	put(w.web, table, column, w.g, w.other, false)
	put(w.ops, table, column, w.h, w.far, false)
	put(w.web, "issue_assignees", "assignee_id", w.a, w.carol, true)
	for _, row := range []struct{ issue, value uuid.UUID }{{w.a, w.erin}, {w.a, w.dave}, {w.a, w.carol}, {w.b, w.carol}} {
		put(w.web, "issue_assignees", "assignee_id", row.issue, row.value, row.value == w.erin)
	}
	for _, issue := range []uuid.UUID{w.left, w.shelved, w.gone, w.g} {
		put(w.web, "issue_assignees", "assignee_id", issue, w.erin, false)
	}
	put(w.ops, "issue_assignees", "assignee_id", w.h, w.erin, false)
	for _, label := range []uuid.UUID{l3, w.l2, w.l1} {
		put(w.web, "issue_labels", "label_id", w.a, label, label == l3)
	}
	return w
}

// statsTargets are the cycles' and the modules' statistics, each run on a world of its own.
var statsTargets = []struct {
	table, column string
	newTarget     func(*testing.T, *pgxpool.Pool, project, string) uuid.UUID
	items         func(*postgresadapter.Store) func(context.Context, uuid.UUID, uuid.UUID, *uuid.UUID) ([]app.PlanningItem, error)
	counts        func(*postgresadapter.Store) func(context.Context, uuid.UUID, []uuid.UUID, *uuid.UUID) ([]app.PlanningStateCount, error)
}{
	{"cycle_issues", "cycle_id", newCycle,
		func(s *postgresadapter.Store) func(context.Context, uuid.UUID, uuid.UUID, *uuid.UUID) ([]app.PlanningItem, error) {
			return s.CycleItems
		},
		func(s *postgresadapter.Store) func(context.Context, uuid.UUID, []uuid.UUID, *uuid.UUID) ([]app.PlanningStateCount, error) {
			return s.CycleStateCounts
		}},
	{"module_issues", "module_id", newModule,
		func(s *postgresadapter.Store) func(context.Context, uuid.UUID, uuid.UUID, *uuid.UUID) ([]app.PlanningItem, error) {
			return s.ModuleItems
		},
		func(s *postgresadapter.Store) func(context.Context, uuid.UUID, []uuid.UUID, *uuid.UUID) ([]app.PlanningStateCount, error) {
			return s.ModuleStateCounts
		}},
}

// The work items of a cycle or a module that its progress counts (M6 P4 spec 2.1, R1, R2): those of the target asked
// about, in the project asked about, with an undeleted membership, undeleted and not archived; of an OwnOnly viewer,
// those he created, none whose creator is gone; each once, in id order, with its state, its completion and its
// undeleted assignees and labels, each once and in id order. Another project's target, a target asked about in
// another project and an empty one have none. A cancelled context is the error.
func TestPlanningItems(t *testing.T) {
	for _, tc := range statsTargets {
		t.Run(tc.table, func(t *testing.T) {
			w := newStatsWorld(t, tc.table, tc.column, tc.newTarget)
			items := tc.items(w.s)
			a := app.PlanningItem{ID: w.a, StateID: w.web.state, CompletedAt: &now, AssigneeIDs: idOrder(w.dave, w.carol), LabelIDs: idOrder(w.l2, w.l1)}
			b := app.PlanningItem{ID: w.b, StateID: w.done, AssigneeIDs: []uuid.UUID{w.carol}}
			c := app.PlanningItem{ID: w.c, StateID: w.web.state}
			g := app.PlanningItem{ID: w.g, StateID: w.web.state, AssigneeIDs: []uuid.UUID{w.erin}}
			h := app.PlanningItem{ID: w.h, StateID: w.ops.state, AssigneeIDs: []uuid.UUID{w.erin}}
			inOther := []app.PlanningItem{g}
			if tc.table == "module_issues" {
				inOther = []app.PlanningItem{a, g}
			}
			for _, q := range []struct {
				name            string
				project, target uuid.UUID
				viewer          *uuid.UUID
				want            []app.PlanningItem
			}{
				{"everyone's", w.web.id, w.target, nil, []app.PlanningItem{a, b, c}},
				{"alice's", w.web.id, w.target, &w.alice, []app.PlanningItem{a}},
				{"bob's", w.web.id, w.target, &w.bob, []app.PlanningItem{b}},
				{"carol's, who made none", w.web.id, w.target, &w.carol, nil},
				{"Other's", w.web.id, w.other, nil, inOther},
				{"Far's", w.ops.id, w.far, nil, []app.PlanningItem{h}},
				{"Far's in Web", w.web.id, w.far, nil, nil},
				{"Target's in Ops", w.ops.id, w.target, nil, nil},
				{"Empty's", w.web.id, w.empty, nil, nil},
			} {
				got, err := items(context.Background(), q.project, q.target, q.viewer)
				want := slices.Clone(q.want)
				slices.SortFunc(want, func(x, y app.PlanningItem) int { return x.ID.Compare(y.ID) })
				if err != nil || !slices.EqualFunc(got, want, sameItem) {
					t.Errorf("%s: %s, %v; want %s", q.name, itemsText(got), err, itemsText(want))
				}
			}
			cancelled, cancel := context.WithCancel(context.Background())
			cancel()
			if _, err := items(cancelled, w.web.id, w.target, nil); !errors.Is(err, context.Canceled) {
				t.Errorf("a read with a cancelled context = %v; want context.Canceled", err)
			}
		})
	}
}

// The counts of a page of cycles or modules by state (M6 P4 spec R2, R9): in one statement, of each target asked
// about that is the project's, how many of the work items its progress counts are in each state, by target and state
// in id order; another project's target has none, nor has an empty one, nor does a page of none. A work item in two
// modules counts in each.
func TestPlanningStateCounts(t *testing.T) {
	for _, tc := range statsTargets {
		t.Run(tc.table, func(t *testing.T) {
			w := newStatsWorld(t, tc.table, tc.column, tc.newTarget)
			counts := tc.counts(w.s)
			inOther := int64(1)
			if tc.table == "module_issues" {
				inOther = 2
			}
			page := []uuid.UUID{w.far, w.empty, w.other, w.target}
			for _, q := range []struct {
				name    string
				project uuid.UUID
				targets []uuid.UUID
				viewer  *uuid.UUID
				want    []app.PlanningStateCount
			}{
				{"everyone's", w.web.id, page, nil, []app.PlanningStateCount{
					{TargetID: w.target, StateID: w.web.state, Issues: 2}, {TargetID: w.target, StateID: w.done, Issues: 1},
					{TargetID: w.other, StateID: w.web.state, Issues: inOther}}},
				{"alice's", w.web.id, page, &w.alice, []app.PlanningStateCount{
					{TargetID: w.target, StateID: w.web.state, Issues: 1}, {TargetID: w.other, StateID: w.web.state, Issues: inOther}}},
				{"bob's", w.web.id, page, &w.bob, []app.PlanningStateCount{{TargetID: w.target, StateID: w.done, Issues: 1}}},
				{"carol's, who made none", w.web.id, page, &w.carol, nil},
				{"Ops'", w.ops.id, page, nil, []app.PlanningStateCount{{TargetID: w.far, StateID: w.ops.state, Issues: 1}}},
				{"an empty page", w.web.id, []uuid.UUID{}, nil, nil},
				{"no page", w.web.id, nil, nil, nil},
			} {
				got, err := counts(context.Background(), q.project, q.targets, q.viewer)
				want := slices.Clone(q.want)
				slices.SortFunc(want, func(x, y app.PlanningStateCount) int {
					return cmp.Or(x.TargetID.Compare(y.TargetID), x.StateID.Compare(y.StateID))
				})
				if err != nil || !slices.Equal(got, want) {
					t.Errorf("%s: %v, %v; want %v", q.name, got, err, want)
				}
			}
			cancelled, cancel := context.WithCancel(context.Background())
			cancel()
			if _, err := counts(cancelled, w.web.id, page, nil); !errors.Is(err, context.Canceled) {
				t.Errorf("a count with a cancelled context = %v; want context.Canceled", err)
			}
		})
	}
}

// The assignees of a page of cycles (M6 P4 spec R9): in one statement, of each cycle asked about that is the
// project's, the undeleted assignees of the work items its progress counts, each once, by cycle and assignee in id
// order; carol, assigned to a and b, once; erin, whose assignment of a is deleted, not Target's, nor are the
// assignees of the deleted, archived and left work items; another project's cycle has none, nor has an empty one,
// nor does a page of none.
func TestPlanningCycleAssignees(t *testing.T) {
	w := newStatsWorld(t, "cycle_issues", "cycle_id", newCycle)
	page := []uuid.UUID{w.far, w.empty, w.other, w.target}
	on := func(cycle, assignee uuid.UUID) app.PlanningAssignee {
		return app.PlanningAssignee{CycleID: cycle, AssigneeID: assignee}
	}
	for _, q := range []struct {
		name    string
		project uuid.UUID
		cycles  []uuid.UUID
		viewer  *uuid.UUID
		want    []app.PlanningAssignee
	}{
		{"everyone's", w.web.id, page, nil, []app.PlanningAssignee{on(w.target, w.carol), on(w.target, w.dave), on(w.other, w.erin)}},
		{"alice's", w.web.id, page, &w.alice, []app.PlanningAssignee{on(w.target, w.carol), on(w.target, w.dave), on(w.other, w.erin)}},
		{"bob's", w.web.id, page, &w.bob, []app.PlanningAssignee{on(w.target, w.carol)}},
		{"carol's, who made none", w.web.id, page, &w.carol, nil},
		{"Ops'", w.ops.id, page, nil, []app.PlanningAssignee{on(w.far, w.erin)}},
		{"an empty page", w.web.id, []uuid.UUID{}, nil, nil},
		{"no page", w.web.id, nil, nil, nil},
	} {
		got, err := w.s.CycleAssignees(context.Background(), q.project, q.cycles, q.viewer)
		want := slices.Clone(q.want)
		slices.SortFunc(want, func(x, y app.PlanningAssignee) int {
			return cmp.Or(x.CycleID.Compare(y.CycleID), x.AssigneeID.Compare(y.AssigneeID))
		})
		if err != nil || !slices.Equal(got, want) {
			t.Errorf("%s: %v, %v; want %v", q.name, got, err, want)
		}
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := w.s.CycleAssignees(cancelled, w.web.id, page, nil); !errors.Is(err, context.Canceled) {
		t.Errorf("a read with a cancelled context = %v; want context.Canceled", err)
	}
}

// idOrder is ids in id order.
func idOrder(ids ...uuid.UUID) []uuid.UUID {
	return slices.SortedFunc(slices.Values(ids), uuid.UUID.Compare)
}

// sameItem reports whether two work items are the same, every field, an empty list as none.
func sameItem(x, y app.PlanningItem) bool {
	return x.ID == y.ID && x.StateID == y.StateID && (x.CompletedAt == nil) == (y.CompletedAt == nil) &&
		(x.CompletedAt == nil || x.CompletedAt.Equal(*y.CompletedAt)) &&
		slices.Equal(x.AssigneeIDs, y.AssigneeIDs) && slices.Equal(x.LabelIDs, y.LabelIDs)
}

func itemsText(items []app.PlanningItem) string {
	out := make([]string, len(items))
	for n, i := range items {
		completed := "-"
		if i.CompletedAt != nil {
			completed = i.CompletedAt.String()
		}
		out[n] = fmt.Sprintf("{%s %s %s %v %v}", i.ID, i.StateID, completed, i.AssigneeIDs, i.LabelIDs)
	}
	return fmt.Sprint(out)
}
