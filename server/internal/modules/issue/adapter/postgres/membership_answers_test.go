package postgresadapter_test

import (
	"context"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// membership is a work item's cycle and modules as a read answers them.
type membership struct {
	cycle   *uuid.UUID
	modules []uuid.UUID
}

// joinTarget writes the row of table that puts the work item issue of p in
// the cycle or module target (column its id's), by nobody at now; deleted
// at now when gone is true. planning owns the targets and the membership
// writes come with the work item's fields (M6 P3a spec 2.5): the test
// writes the rows directly, as a fixture.
func joinTarget(t *testing.T, pool *pgxpool.Pool, p project, table, column string, issue, target uuid.UUID, gone bool) {
	t.Helper()
	var deleted *time.Time
	if gone {
		deleted = &now
	}
	exec(t, pool, `INSERT INTO `+table+` (id, workspace_id, project_id, `+column+`, issue_id, created_at, updated_at, deleted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $6, $7)`, uuid.NewV7(), p.workspace, p.id, target, issue, now, deleted)
}

// Each read of work items answers each one's cycle and modules (M6 P3a spec
// 2.3): its cycle the cycle of its undeleted cycle_issues row, an archived
// one too, none without one; its modules those of its undeleted
// module_issues rows, archived ones too, in their id order, whatever the
// order the rows are stored in. GetIssue, GetIssues, SubIssues, Search and
// the flat, grouped and subgrouped lists answer them alike (spec 6: each
// has GetIssue's columns). a left Sprint for Old, an archived cycle, and
// left M3; it is in M2 and in M1, archived, stored in that order (M1 the
// first of the three by id). b left
// Sprint and M1 and is in none; c never was in any; p, their parent, is in
// Sprint alone.
func TestEveryReadAnswersTheMemberships(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	alice := newAccount(t, pool, "alice@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	sprint, old := newCycle(t, pool, web, "Sprint"), newCycle(t, pool, web, "Old")
	modules := []uuid.UUID{newModule(t, pool, web, "M1"), newModule(t, pool, web, "M2"), newModule(t, pool, web, "M3")}
	slices.SortFunc(modules, uuid.UUID.Compare)
	m1, m2, m3 := modules[0], modules[1], modules[2]
	exec(t, pool, "UPDATE cycles SET archived_at = $2 WHERE id = $1", old, now)
	exec(t, pool, "UPDATE modules SET archived_at = $2 WHERE id = $1", m1, now)
	p, a, b, c := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	for n, id := range []uuid.UUID{p, a, b, c} {
		newIssue(t, pool, web, id, n+1, alice)
	}
	exec(t, pool, "UPDATE issues SET parent_id = $2 WHERE id = ANY ($1::uuid[])", []uuid.UUID{a, b, c}, p)
	for _, row := range []struct {
		table, column string
		issue, target uuid.UUID
		gone          bool
	}{
		{"cycle_issues", "cycle_id", a, sprint, true}, {"cycle_issues", "cycle_id", a, old, false},
		{"cycle_issues", "cycle_id", b, sprint, true}, {"cycle_issues", "cycle_id", p, sprint, false},
		{"module_issues", "module_id", a, m3, true}, {"module_issues", "module_id", a, m2, false},
		{"module_issues", "module_id", a, m1, false}, {"module_issues", "module_id", b, m1, true},
	} {
		joinTarget(t, pool, web, row.table, row.column, row.issue, row.target, row.gone)
	}
	// With the tables' sizes known, the reads scan their few rows in the
	// order they are stored, M2's first, as they would through the key on
	// issue_id alone; through the key on (issue_id, module_id) they would come
	// in id order whether a read sorts them or not.
	exec(t, pool, "ANALYZE cycle_issues, module_issues")
	want := map[uuid.UUID]membership{p: {cycle: &sprint}, a: {cycle: &old, modules: []uuid.UUID{m1, m2}}, b: {}, c: {}}

	check := func(t *testing.T, read string, issues []domain.Issue, n int) {
		t.Helper()
		if len(issues) != n {
			t.Fatalf("%s read %d work items, want %d", read, len(issues), n)
		}
		for _, i := range issues {
			w, ok := want[i.ID]
			if !ok {
				t.Fatalf("%s read the work item %s, not one of the fixture's", read, i.ID)
			}
			if !reflect.DeepEqual(i.CycleID, w.cycle) || !slices.Equal(i.ModuleIDs, w.modules) {
				t.Errorf("%s: %s is in cycle %v and modules %v; want %v and %v", read, i.ID, cycleOf(i.CycleID), i.ModuleIDs,
					cycleOf(w.cycle), w.modules)
			}
		}
	}
	for _, id := range []uuid.UUID{p, a, b, c} {
		i, found, err := s.GetIssue(ctx, id)
		if err != nil || !found {
			t.Fatalf("GetIssue(%s) = %v, %v", id, found, err)
		}
		check(t, "GetIssue", []domain.Issue{i}, 1)
	}
	issues, err := s.GetIssues(ctx, []uuid.UUID{c, b, a, p})
	if err != nil {
		t.Fatal(err)
	}
	check(t, "GetIssues", issues, 4)
	if issues, err = s.SubIssues(ctx, p, nil); err != nil {
		t.Fatal(err)
	}
	check(t, "SubIssues", issues, 3)
	q, err := domain.ParseSearchQuery(map[string][]string{})
	if err != nil {
		t.Fatal(err)
	}
	scope := []app.VisibleProject{{ID: web.id}}
	if issues, err = s.Search(ctx, app.SearchRead{ActorID: alice, Projects: scope, Query: q}); err != nil {
		t.Fatal(err)
	}
	check(t, "Search", issues, 4)
	dirs := app.ListDirectories{Projects: scope, States: []app.ListState{{ID: web.state, Group: "backlog", Key: "backlog"}}}
	for _, list := range []struct {
		name            string
		read            func(context.Context, app.ListRead) ([]app.ListRecord, error)
		group, subgroup domain.ListGroup
	}{{"Flat", s.Flat, "", ""}, {"Groups", s.Groups, "priority", ""}, {"Subgroups", s.Subgroups, "priority", "state_group"}} {
		rows, err := list.read(ctx, app.ListRead{ActorID: alice, Directories: dirs, Query: domain.ListQuery{Order: "sort_order", Limit: 100,
			GroupBy: list.group, SubGroupBy: list.subgroup, Filters: domain.ListFilters{SubIssues: true}}})
		if err != nil {
			t.Fatalf("%s: %v", list.name, err)
		}
		issues := make([]domain.Issue, len(rows))
		for n, r := range rows {
			issues[n] = r.Issue
		}
		check(t, list.name, issues, 4)
	}
}

// cycleOf is the cycle id shows, none for nil.
func cycleOf(id *uuid.UUID) string {
	if id == nil {
		return "none"
	}
	return id.String()
}
