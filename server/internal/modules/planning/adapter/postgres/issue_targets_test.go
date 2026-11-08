package postgresadapter_test

import (
	"context"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// planned is a cycle or a module as the issue targets' tests write it: its project, name and day (a cycle's start
// and end, a module's target; "" for none), its status (a module's; "" for the default), archived or deleted.
type planned struct {
	id                uuid.UUID
	project           uuid.UUID
	name, day, status string
	archived, deleted bool
}

// plant writes the rows ps into table, cycles or modules.
func plant(t *testing.T, pool *pgxpool.Pool, w world, table string, ps ...planned) {
	t.Helper()
	for _, p := range ps {
		var day, archived, deleted *time.Time
		if p.day != "" {
			d, err := shared.ParseDate(p.day)
			if err != nil {
				t.Fatal(err)
			}
			at := d.Time()
			day = &at
		}
		if p.archived {
			archived = &now
		}
		if p.deleted {
			deleted = &now
		}
		status := p.status
		if status == "" {
			status = "planned"
		}
		switch table {
		case "cycles":
			exec(t, pool, `INSERT INTO cycles (id, workspace_id, project_id, name, start_date, end_date, archived_at, deleted_at)
				VALUES ($1, $2, $3, $4, $5, $5, $6, $7)`, p.id, w.workspace, p.project, p.name, day, archived, deleted)
		case "modules":
			exec(t, pool, `INSERT INTO modules (id, workspace_id, project_id, name, target_date, status, archived_at, deleted_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, p.id, w.workspace, p.project, p.name, day, status, archived, deleted)
		default:
			t.Fatalf("no planning table %s", table)
		}
	}
}

// The cycles and modules a work item's write names: of the ids asked about, the project's undeleted ones, archived
// ones too, each with its name and whether it is archived, in id order; another project's, a deleted one and an id of
// none are not among them, nor is anything when nothing is asked about.
func TestIssueTargetsOfTheProject(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newWorld(t, pool)
	for _, c := range []struct {
		table string
		read  func(context.Context, uuid.UUID, []uuid.UUID) ([]app.IssueTarget, error)
	}{
		{"cycles", s.CycleTargets},
		{"modules", s.ModuleTargets},
	} {
		t.Run(c.table, func(t *testing.T) {
			ids := sortedIDs(5)
			live, archived, deleted, other, none := ids[0], ids[1], ids[2], ids[3], ids[4]
			// Inserted out of id order, so the answer's order is the statement's.
			plant(t, pool, w, c.table,
				planned{id: archived, project: w.web, name: "Done " + c.table, archived: true},
				planned{id: live, project: w.web, name: "Live " + c.table},
				planned{id: deleted, project: w.web, name: "Gone " + c.table, deleted: true},
				planned{id: other, project: w.ops, name: "Ops " + c.table})
			got, err := c.read(ctx, w.web, []uuid.UUID{none, other, deleted, archived, live})
			if err != nil {
				t.Fatal(err)
			}
			want := []app.IssueTarget{{ID: live, Name: "Live " + c.table}, {ID: archived, Name: "Done " + c.table, Archived: true}}
			if !slices.Equal(got, want) {
				t.Errorf("targets = %+v\nwant %+v", got, want)
			}
			if got, err := c.read(ctx, w.ops, []uuid.UUID{live, other}); err != nil || !slices.Equal(got, []app.IssueTarget{{ID: other, Name: "Ops " + c.table}}) {
				t.Errorf("the other project's targets = %+v, %v; want its own alone", got, err)
			}
			if got, err := c.read(ctx, w.web, nil); err != nil || len(got) != 0 {
				t.Errorf("targets of nothing = %+v, %v; want none", got, err)
			}
		})
	}
}

// The open cycles on a day: the project's undeleted cycles without dates or ending on that day or later, archived or
// not, in id order — those whose status on the day is not completed (domain.StatusOf, the one rule of a cycle's end).
// A deleted cycle and another project's are not among them.
func TestOpenCycleIDs(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newWorld(t, pool)
	today, err := shared.ParseDate("2026-11-08")
	if err != nil {
		t.Fatal(err)
	}
	ids := sortedIDs(8)
	cycles := []planned{
		{id: ids[6], project: w.web, name: "Draft"},
		{id: ids[0], project: w.web, name: "Ended yesterday", day: "2026-11-07"},
		{id: ids[3], project: w.web, name: "Ending today", day: "2026-11-08"},
		{id: ids[5], project: w.web, name: "Upcoming", day: "2026-11-09"},
		{id: ids[1], project: w.web, name: "Archived, ending tomorrow", day: "2026-11-09", archived: true},
		{id: ids[2], project: w.web, name: "Archived, ended", day: "2026-11-01", archived: true},
		{id: ids[4], project: w.web, name: "Deleted draft", deleted: true},
		{id: ids[7], project: w.ops, name: "Ops draft"},
	}
	plant(t, pool, w, "cycles", cycles...)
	got, err := s.OpenCycleIDs(ctx, w.web, today)
	if err != nil {
		t.Fatal(err)
	}
	if want := []uuid.UUID{ids[1], ids[3], ids[5], ids[6]}; !slices.Equal(got, want) {
		t.Errorf("open cycles = %v\nwant %v (archived ending tomorrow, ending today, upcoming, draft)", got, want)
	}
	var byStatus []uuid.UUID
	for _, c := range cycles {
		var d domain.Dates
		if c.day != "" {
			day, _ := shared.ParseDate(c.day)
			d = domain.Dates{Start: &day, End: &day}
		}
		if c.project == w.web && !c.deleted && domain.StatusOf(d, today) != domain.StatusCompleted {
			byStatus = append(byStatus, c.id)
		}
	}
	slices.SortFunc(byStatus, uuid.UUID.Compare)
	if !slices.Equal(got, byStatus) {
		t.Errorf("open cycles = %v; the cycles not completed by domain.StatusOf are %v", got, byStatus)
	}
}

// The open modules on a day: the project's undeleted modules without a target day or with one on that day or later,
// in id order, archived or not and in any status (M6 design 7, as Plane). A deleted module and another project's are
// not among them.
func TestOpenModuleIDs(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newWorld(t, pool)
	today, err := shared.ParseDate("2026-11-08")
	if err != nil {
		t.Fatal(err)
	}
	ids := sortedIDs(8)
	plant(t, pool, w, "modules",
		planned{id: ids[6], project: w.web, name: "No target"},
		planned{id: ids[0], project: w.web, name: "Target yesterday", day: "2026-11-07"},
		planned{id: ids[3], project: w.web, name: "Target today", day: "2026-11-08"},
		planned{id: ids[5], project: w.web, name: "Completed, target tomorrow", day: "2026-11-09", status: "completed"},
		planned{id: ids[1], project: w.web, name: "Archived, cancelled, no target", status: "cancelled", archived: true},
		planned{id: ids[2], project: w.web, name: "Archived, target passed", day: "2026-11-01", archived: true},
		planned{id: ids[4], project: w.web, name: "Deleted", deleted: true},
		planned{id: ids[7], project: w.ops, name: "Ops"})
	got, err := s.OpenModuleIDs(ctx, w.web, today)
	if err != nil {
		t.Fatal(err)
	}
	if want := []uuid.UUID{ids[1], ids[3], ids[5], ids[6]}; !slices.Equal(got, want) {
		t.Errorf("open modules = %v\nwant %v (archived cancelled, target today, completed target tomorrow, no target)", got, want)
	}
}
