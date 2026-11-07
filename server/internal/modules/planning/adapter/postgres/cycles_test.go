package postgresadapter_test

import (
	"context"
	"encoding/json"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func jsonOf(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func dayOf(t *testing.T, s string) *shared.Date {
	t.Helper()
	d, err := shared.ParseDate(s)
	if err != nil {
		t.Fatal(err)
	}
	return &d
}

// A created cycle reads back as written, its dates as days, its owner and writers; one deleted is not read.
func TestCreateAndGetCycle(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newWorld(t, pool)
	at := now.Truncate(time.Microsecond)
	want := domain.Cycle{ID: uuid.NewV7(), ProjectID: w.web, WorkspaceID: w.workspace, Name: "Sprint", Description: "Ship",
		Dates: domain.Dates{Start: dayOf(t, "2026-11-07"), End: dayOf(t, "2026-11-20")}, OwnedByID: &w.by, SortOrder: -9900.5,
		CreatedAt: at, UpdatedAt: at, CreatedByID: &w.by, UpdatedByID: &w.by}
	if err := s.CreateCycle(ctx, want); err != nil {
		t.Fatal(err)
	}
	draft := domain.Cycle{ID: uuid.NewV7(), ProjectID: w.web, WorkspaceID: w.workspace, Name: "Draft", SortOrder: 65535, CreatedAt: at, UpdatedAt: at}
	if err := s.CreateCycle(ctx, draft); err != nil {
		t.Fatal(err)
	}
	for _, c := range []domain.Cycle{want, draft} {
		got, found, err := s.GetCycle(ctx, c.ID)
		if err != nil || !found || jsonOf(t, got) != jsonOf(t, c) {
			t.Fatalf("GetCycle = %s, %t, %v\nwant %s", jsonOf(t, got), found, err, jsonOf(t, c))
		}
	}
	exec(t, pool, "UPDATE cycles SET deleted_at = now() WHERE id = $1", draft.ID)
	if _, found, err := s.GetCycle(ctx, draft.ID); err != nil || found {
		t.Fatalf("a deleted cycle read: %t, %v", found, err)
	}
	if _, found, err := s.GetCycle(ctx, uuid.NewV7()); err != nil || found {
		t.Fatalf("a missing cycle read: %t, %v", found, err)
	}
}

// The project's undeleted cycles, archived ones too, give their dates and the lowest sort order; deleted ones and
// another project's do not. A project without cycles has no lowest.
func TestProjectCycleDatesAndLowestSortOrder(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newWorld(t, pool)
	if lowest, err := s.LowestCycleSortOrder(ctx, w.web); err != nil || lowest != nil {
		t.Fatalf("lowest of none = %v, %v", lowest, err)
	}
	insert := func(project uuid.UUID, sort float64, start, end string, archived, deleted bool) uuid.UUID {
		id := uuid.NewV7()
		var s, e *time.Time
		if start != "" {
			a, b := dayOf(t, start).Time(), dayOf(t, end).Time()
			s, e = &a, &b
		}
		exec(t, pool, `INSERT INTO cycles (id, workspace_id, project_id, name, start_date, end_date, sort_order, archived_at, deleted_at)
			VALUES ($1, $2, $3, 'C', $4, $5, $6, CASE WHEN $7 THEN now() END, CASE WHEN $8 THEN now() END)`, id, w.workspace, project, s, e, sort, archived, deleted)
		return id
	}
	live := insert(w.web, 10, "2026-11-01", "2026-11-05", false, false)
	archived := insert(w.web, -50, "2026-10-01", "2026-10-05", true, false)
	draft := insert(w.web, 20, "", "", false, false)
	insert(w.web, -500, "2026-12-01", "2026-12-05", false, true)
	insert(w.ops, -900, "2026-11-02", "2026-11-03", false, false)
	if lowest, err := s.LowestCycleSortOrder(ctx, w.web); err != nil || lowest == nil || *lowest != -50 {
		t.Fatalf("lowest = %v, %v; want -50, the archived one's", lowest, err)
	}
	dates, err := s.ProjectCycleDates(ctx, w.web)
	if err != nil {
		t.Fatal(err)
	}
	slices.SortFunc(dates, func(a, b domain.DatedCycle) int { return a.ID.Compare(b.ID) })
	want := []domain.DatedCycle{
		{ID: live, Dates: domain.Dates{Start: dayOf(t, "2026-11-01"), End: dayOf(t, "2026-11-05")}},
		{ID: archived, Dates: domain.Dates{Start: dayOf(t, "2026-10-01"), End: dayOf(t, "2026-10-05")}},
		{ID: draft},
	}
	slices.SortFunc(want, func(a, b domain.DatedCycle) int { return a.ID.Compare(b.ID) })
	if jsonOf(t, dates) != jsonOf(t, want) {
		t.Fatalf("dates %s\nwant %s", jsonOf(t, dates), jsonOf(t, want))
	}
}
