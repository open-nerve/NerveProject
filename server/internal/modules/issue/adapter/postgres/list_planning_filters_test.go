package postgresadapter_test

import (
	"context"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// The three list statements filter by cycle and by module alike (M6 P3b spec
// 2.1): the alternatives of one field are joined, none meaning no undeleted
// membership, and different fields intersect. Membership rows left (deleted)
// never count, archived cycles and modules do. i0, labelled l, is in Sprint
// and Core; i1 in Sprint, Core and UI, archived; i2 in Old, archived, having
// left Sprint and Core; i3 in UI, having left Core; i4 in nothing, having
// left Sprint and UI. They are listed in the reverse of their id order, and
// their priorities split them into groups whose totals count only the rows
// the filters admit.
func TestIssueListPlanningFilters(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	a, b := newAccount(t, pool, "planning-filter-a"), newAccount(t, pool, "planning-filter-b")
	p := newProject(t, pool, newWorkspace(t, pool, "planning-filter"), "PLAN")
	l := newLabel(t, pool, p, "L")
	sprint, old := newCycle(t, pool, p, "Sprint"), newCycle(t, pool, p, "Old")
	core, ui := newModule(t, pool, p, "Core"), newModule(t, pool, p, "UI")
	exec(t, pool, "UPDATE cycles SET archived_at = $2 WHERE id = $1", old, now)
	exec(t, pool, "UPDATE modules SET archived_at = $2 WHERE id = $1", ui, now)
	ids := make([]uuid.UUID, 5)
	priorities := map[uuid.UUID]string{}
	for n, priority := range []string{"high", "none", "high", "low", "none"} {
		ids[n] = uuid.NewV7()
		by := a
		if n%2 == 1 {
			by = b
		}
		newIssue(t, pool, p, ids[n], n+1, by)
		exec(t, pool, "UPDATE issues SET sort_order = $2, priority = $3 WHERE id = $1", ids[n], (5-n)*10, priority)
		priorities[ids[n]] = priority
	}
	i0, i1, i2, i3, i4 := ids[0], ids[1], ids[2], ids[3], ids[4]
	exec(t, pool, "INSERT INTO issue_labels (id, workspace_id, project_id, issue_id, label_id) VALUES ($1, $2, $3, $4, $5)", uuid.NewV7(), p.workspace, p.id, i0, l)
	for _, row := range []struct {
		table, column string
		issue, target uuid.UUID
		gone          bool
	}{
		{"cycle_issues", "cycle_id", i0, sprint, false}, {"module_issues", "module_id", i0, core, false},
		{"cycle_issues", "cycle_id", i1, sprint, false}, {"module_issues", "module_id", i1, core, false}, {"module_issues", "module_id", i1, ui, false},
		{"cycle_issues", "cycle_id", i2, sprint, true}, {"cycle_issues", "cycle_id", i2, old, false}, {"module_issues", "module_id", i2, core, true},
		{"module_issues", "module_id", i3, core, true}, {"module_issues", "module_id", i3, ui, false},
		{"cycle_issues", "cycle_id", i4, sprint, true}, {"module_issues", "module_id", i4, ui, true},
	} {
		joinTarget(t, pool, p, row.table, row.column, row.issue, row.target, row.gone)
	}
	in := func(f domain.ListFilters) app.ListRead {
		f.SubIssues = true
		return app.ListRead{ActorID: a, Directories: app.ListDirectories{Projects: []app.VisibleProject{{ID: p.id}},
			States: []app.ListState{{ID: p.state, Group: "backlog", Key: "backlog"}}}, Query: domain.ListQuery{Order: "sort_order", Limit: 100, Filters: f}}
	}
	ofIDs := func(ids ...uuid.UUID) domain.IDFilter { return domain.IDFilter{IDs: ids} }
	withNone := func(ids ...uuid.UUID) domain.IDFilter { return domain.IDFilter{IDs: ids, None: true} }
	for _, tc := range []struct {
		name    string
		filters domain.ListFilters
		want    []uuid.UUID
	}{
		{"cycle", domain.ListFilters{Cycles: ofIDs(sprint)}, []uuid.UUID{i1, i0}},
		{"archived cycle", domain.ListFilters{Cycles: ofIDs(old)}, []uuid.UUID{i2}},
		{"no cycle", domain.ListFilters{Cycles: withNone()}, []uuid.UUID{i4, i3}},
		{"no cycle or a cycle", domain.ListFilters{Cycles: withNone(sprint)}, []uuid.UUID{i4, i3, i1, i0}},
		{"two cycles", domain.ListFilters{Cycles: ofIDs(old, sprint)}, []uuid.UUID{i2, i1, i0}},
		{"module", domain.ListFilters{Modules: ofIDs(core)}, []uuid.UUID{i1, i0}},
		{"archived module", domain.ListFilters{Modules: ofIDs(ui)}, []uuid.UUID{i3, i1}},
		{"no module", domain.ListFilters{Modules: withNone()}, []uuid.UUID{i4, i2}},
		{"no module or a module", domain.ListFilters{Modules: withNone(core)}, []uuid.UUID{i4, i2, i1, i0}},
		{"two modules", domain.ListFilters{Modules: ofIDs(core, ui)}, []uuid.UUID{i3, i1, i0}},
		{"cycle and module", domain.ListFilters{Cycles: ofIDs(sprint), Modules: ofIDs(ui)}, []uuid.UUID{i1}},
		{"cycle and label", domain.ListFilters{Cycles: ofIDs(sprint), Labels: ofIDs(l)}, []uuid.UUID{i0}},
		{"no cycle and no module", domain.ListFilters{Cycles: withNone(), Modules: withNone()}, []uuid.UUID{i4}},
		{"unknown cycle", domain.ListFilters{Cycles: ofIDs(uuid.NewV7())}, nil},
		{"unknown module", domain.ListFilters{Modules: ofIDs(uuid.NewV7())}, nil},
	} {
		for _, form := range []struct {
			name            string
			read            func(context.Context, app.ListRead) ([]app.ListRecord, error)
			group, subgroup domain.ListGroup
		}{{"flat", s.Flat, "", ""}, {"groups", s.Groups, "priority", ""}, {"subgroups", s.Subgroups, "priority", "state_group"}} {
			t.Run(tc.name+"/"+form.name, func(t *testing.T) {
				read := in(tc.filters)
				read.Query.GroupBy, read.Query.SubGroupBy = form.group, form.subgroup
				rows, err := form.read(ctx, read)
				if err != nil {
					t.Fatal(err)
				}
				got := make([]uuid.UUID, len(rows))
				for n, r := range rows {
					got[n] = r.Issue.ID
				}
				want := slices.Clone(tc.want)
				if form.group != "" {
					slices.SortFunc(got, uuid.UUID.Compare)
					slices.SortFunc(want, uuid.UUID.Compare)
				}
				if !slices.Equal(got, want) {
					t.Fatalf("listed %v; want %v", got, want)
				}
				inGroup := map[string]int64{}
				for _, id := range tc.want {
					inGroup[priorities[id]]++
				}
				for _, r := range rows {
					if r.Total != int64(len(tc.want)) {
						t.Fatalf("total %d; want %d", r.Total, len(tc.want))
					}
					if form.group != "" && (r.GroupKey != priorities[r.Issue.ID] || r.GroupTotal != inGroup[r.GroupKey]) {
						t.Fatalf("%s in group %s of %d; want %s of %d", r.Issue.ID, r.GroupKey, r.GroupTotal, priorities[r.Issue.ID], inGroup[priorities[r.Issue.ID]])
					}
					if form.subgroup != "" && (r.SubGroupKey != "backlog" || r.SubgroupTotal != int64(len(tc.want)) || r.CellTotal != inGroup[r.GroupKey]) {
						t.Fatalf("%s in cell %s/%s of %d, lane of %d", r.Issue.ID, r.GroupKey, r.SubGroupKey, r.CellTotal, r.SubgroupTotal)
					}
				}
			})
		}
	}
}
