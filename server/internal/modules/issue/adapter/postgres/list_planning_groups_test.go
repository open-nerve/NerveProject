package postgresadapter_test

import (
	"context"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// planningGroupRow is where one listed work item (id, as text) stands: its
// group, lane and the totals of its group, lane, cell and list.
type planningGroupRow struct {
	id, group, lane         string
	groupTotal, laneTotal   int64
	cellTotal, overallTotal int64
}

// The group statements group a project's list by cycle and by module (M6
// P3b spec 2.1): a work item is in its cycle's group, and in the group of
// each of its modules; one with none is in the none group, last, after the
// others in their keys' order. Memberships left never count, an archived
// cycle or module does, and one with no member has no group. Each group's
// total counts its rows, the outer total each work item once, and for a
// guest who sees only his own work items (OwnOnly) only those. i0, b's, is
// in Sprint, Core and UI, archived; i1, a's, in Sprint and Core; i2, a's,
// in Old, archived, having left Sprint and Core; i3, b's, in UI, having
// left Gone; i4, a's, in nothing, having left Idle and UI. They are created
// and ordered in their index order.
func TestIssueListPlanningGroups(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	a, b := newAccount(t, pool, "planning-group-a"), newAccount(t, pool, "planning-group-b")
	p := newProject(t, pool, newWorkspace(t, pool, "planning-group"), "PGROUP")
	sprint, old, gone := newCycle(t, pool, p, "Sprint"), newCycle(t, pool, p, "Old"), newCycle(t, pool, p, "Gone")
	core, ui, idle := newModule(t, pool, p, "Core"), newModule(t, pool, p, "UI"), newModule(t, pool, p, "Idle")
	exec(t, pool, "UPDATE cycles SET archived_at = $2 WHERE id = $1", old, now)
	exec(t, pool, "UPDATE modules SET archived_at = $2 WHERE id = $1", ui, now)
	ids := make([]uuid.UUID, 5)
	for n, by := range []uuid.UUID{b, a, a, b, a} {
		ids[n] = uuid.NewV7()
		newIssue(t, pool, p, ids[n], n+1, by)
		at := now.Add(time.Duration(n) * time.Second)
		exec(t, pool, "UPDATE issues SET sort_order = $2, created_at = $3, updated_at = $3 WHERE id = $1", ids[n], n*10, at)
	}
	i0, i1, i2, i3, i4 := ids[0], ids[1], ids[2], ids[3], ids[4]
	for _, row := range []struct {
		table, column string
		issue, target uuid.UUID
		gone          bool
	}{
		{"cycle_issues", "cycle_id", i0, sprint, false}, {"module_issues", "module_id", i0, core, false}, {"module_issues", "module_id", i0, ui, false},
		{"cycle_issues", "cycle_id", i1, sprint, false}, {"module_issues", "module_id", i1, core, false},
		{"cycle_issues", "cycle_id", i2, sprint, true}, {"cycle_issues", "cycle_id", i2, old, false}, {"module_issues", "module_id", i2, core, true},
		{"cycle_issues", "cycle_id", i3, gone, true}, {"module_issues", "module_id", i3, ui, false},
		{"module_issues", "module_id", i4, idle, true}, {"module_issues", "module_id", i4, ui, true},
	} {
		joinTarget(t, pool, p, row.table, row.column, row.issue, row.target, row.gone)
	}
	// Written independently of the statements: each group's work items in
	// their order. Gone and Idle have none.
	members := map[domain.ListGroup]map[string][]uuid.UUID{
		"cycle_id":  {sprint.String(): {i0, i1}, old.String(): {i2}, "none": {i3, i4}},
		"module_id": {core.String(): {i0, i1}, ui.String(): {i0, i3}, "none": {i2, i4}},
	}
	seen := func(field domain.ListGroup, key string, visible []uuid.UUID) []uuid.UUID {
		return slices.DeleteFunc(slices.Clone(members[field][key]), func(id uuid.UUID) bool { return !slices.Contains(visible, id) })
	}
	keys := func(field domain.ListGroup, visible []uuid.UUID) []string {
		var out []string
		for key := range members[field] {
			if key != "none" && len(seen(field, key, visible)) > 0 {
				out = append(out, key)
			}
		}
		slices.Sort(out)
		if len(seen(field, "none", visible)) > 0 {
			out = append(out, "none")
		}
		return out
	}
	read := func(own bool, order domain.ListOrder, group, lane domain.ListGroup) app.ListRead {
		return app.ListRead{ActorID: a, Directories: app.ListDirectories{Projects: []app.VisibleProject{{ID: p.id, OwnOnly: own}},
			States: []app.ListState{{ID: p.state, Group: "backlog", Key: "backlog"}}},
			Query: domain.ListQuery{Order: order, Limit: 100, GroupBy: group, SubGroupBy: lane, Filters: domain.ListFilters{SubIssues: true}}}
	}
	viewers := []struct {
		name    string
		own     bool
		visible []uuid.UUID
	}{{"everyone's", false, ids}, {"own only", true, []uuid.UUID{i1, i2, i4}}}
	fields := []domain.ListGroup{"cycle_id", "module_id"}
	for _, viewer := range viewers {
		for _, field := range fields {
			t.Run("groups/"+string(field)+"/"+viewer.name, func(t *testing.T) {
				var want []planningGroupRow
				for _, key := range keys(field, viewer.visible) {
					in := seen(field, key, viewer.visible)
					for _, id := range in {
						want = append(want, planningGroupRow{id: id.String(), group: key, groupTotal: int64(len(in)), overallTotal: int64(len(viewer.visible))})
					}
				}
				rows, err := s.Groups(ctx, read(viewer.own, "sort_order", field, ""))
				if err != nil {
					t.Fatal(err)
				}
				got := make([]planningGroupRow, len(rows))
				for n, r := range rows {
					got[n] = planningGroupRow{id: r.Issue.ID.String(), group: r.GroupKey, groupTotal: r.GroupTotal, overallTotal: r.Total}
				}
				if !slices.Equal(got, want) {
					t.Fatalf("listed %+v;\nwant %+v", got, want)
				}
			})
			for _, lane := range fields {
				if lane == field {
					continue
				}
				t.Run("subgroups/"+string(field)+"/"+string(lane)+"/"+viewer.name, func(t *testing.T) {
					var want []planningGroupRow
					for _, key := range keys(field, viewer.visible) {
						group := seen(field, key, viewer.visible)
						for _, laneKey := range keys(lane, viewer.visible) {
							cell := slices.DeleteFunc(slices.Clone(group), func(id uuid.UUID) bool { return !slices.Contains(members[lane][laneKey], id) })
							for _, id := range cell {
								want = append(want, planningGroupRow{id: id.String(), group: key, lane: laneKey, groupTotal: int64(len(group)),
									laneTotal: int64(len(seen(lane, laneKey, viewer.visible))), cellTotal: int64(len(cell)), overallTotal: int64(len(viewer.visible))})
							}
						}
					}
					rows, err := s.Subgroups(ctx, read(viewer.own, "sort_order", field, lane))
					if err != nil {
						t.Fatal(err)
					}
					got := make([]planningGroupRow, len(rows))
					for n, r := range rows {
						got[n] = planningGroupRow{id: r.Issue.ID.String(), group: r.GroupKey, lane: r.SubGroupKey, groupTotal: r.GroupTotal,
							laneTotal: r.SubgroupTotal, cellTotal: r.CellTotal, overallTotal: r.Total}
					}
					if !slices.Equal(got, want) {
						t.Fatalf("listed %+v;\nwant %+v", got, want)
					}
				})
			}
		}
	}
	// A group and a cell continue page by page in either direction; a
	// cycle or module without members selects nothing.
	for _, order := range []domain.ListOrder{"created_at", "-created_at"} {
		for _, field := range fields {
			t.Run("pages/"+string(field)+"/"+string(order), func(t *testing.T) {
				for _, key := range append(keys(field, ids), gone.String(), idle.String()) {
					want := seen(field, key, ids)
					if order == "-created_at" {
						slices.Reverse(want)
					}
					in := read(false, order, field, "")
					in.Query.GroupKey = &key
					if len(want) == 0 {
						rows, err := s.Groups(ctx, in)
						if err != nil || len(rows) != 0 {
							t.Fatalf("group %s without members: %+v %v", key, rows, err)
						}
						continue
					}
					checkGroupPages(t, s, in, want, int64(len(want)))
				}
			})
		}
		t.Run("cells/"+string(order), func(t *testing.T) {
			for _, cell := range []struct {
				group, lane domain.ListGroup
				key, at     string
				want        []uuid.UUID
			}{
				{"cycle_id", "module_id", sprint.String(), core.String(), []uuid.UUID{i0, i1}},
				{"cycle_id", "module_id", "none", ui.String(), []uuid.UUID{i3}},
				{"module_id", "cycle_id", ui.String(), sprint.String(), []uuid.UUID{i0}},
				{"module_id", "cycle_id", "none", "none", []uuid.UUID{i4}},
				{"module_id", "cycle_id", "none", old.String(), []uuid.UUID{i2}},
				{"cycle_id", "module_id", sprint.String(), idle.String(), nil},
			} {
				want := slices.Clone(cell.want)
				if order == "-created_at" {
					slices.Reverse(want)
				}
				in := read(false, order, cell.group, cell.lane)
				in.Query.GroupKey, in.Query.SubGroupKey = &cell.key, &cell.at
				checkCellPages(t, s, in, want)
			}
		})
	}
}
