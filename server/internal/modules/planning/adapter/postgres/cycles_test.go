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

// The project's undeleted cycles: the unarchived ones by sort order, then creation, then id; the archived ones alone,
// the latest archived first, then by id. Another project's and deleted ones are in neither.
func TestListCycles(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newWorld(t, pool)
	at := now.Truncate(time.Microsecond)
	ids := sortedIDs(7)
	add := func(id, project uuid.UUID, sortOrder float64, created time.Time) {
		t.Helper()
		if err := s.CreateCycle(ctx, domain.Cycle{ID: id, ProjectID: project, WorkspaceID: w.workspace, Name: id.String(), SortOrder: sortOrder,
			CreatedAt: created, UpdatedAt: created}); err != nil {
			t.Fatal(err)
		}
	}
	// ids[2] and ids[1] tie on sort order and creation: by id. ids[0] ties on sort order, created later.
	add(ids[2], w.web, 100, at)
	add(ids[1], w.web, 100, at)
	add(ids[0], w.web, 100, at.Add(time.Second))
	add(ids[3], w.web, -5, at.Add(time.Hour))
	add(ids[4], w.web, 1, at)
	add(ids[5], w.web, 2, at)
	add(ids[6], w.ops, 0, at)
	deleted := uuid.NewV7()
	add(deleted, w.web, 0, at)
	exec(t, pool, "UPDATE cycles SET deleted_at = now() WHERE id = $1", deleted)
	// ids[4] and ids[5] archived at one moment, ids[1] later.
	exec(t, pool, "UPDATE cycles SET archived_at = $2 WHERE id = ANY($1)", []uuid.UUID{ids[4], ids[5]}, at)
	exec(t, pool, "UPDATE cycles SET archived_at = $2 WHERE id = $1", ids[1], at.Add(time.Minute))
	for _, tt := range []struct {
		archived bool
		want     []uuid.UUID
	}{{false, []uuid.UUID{ids[3], ids[2], ids[0]}}, {true, []uuid.UUID{ids[1], ids[4], ids[5]}}} {
		got, err := s.ListCycles(ctx, w.web, tt.archived)
		if err != nil {
			t.Fatal(err)
		}
		var gotIDs []uuid.UUID
		for _, c := range got {
			gotIDs = append(gotIDs, c.ID)
		}
		if !slices.Equal(gotIDs, tt.want) {
			t.Errorf("archived=%t: %v, want %v", tt.archived, gotIDs, tt.want)
		}
	}
	// Each row as GetCycle reads it.
	got, err := s.ListCycles(ctx, w.web, true)
	if err != nil || len(got) == 0 {
		t.Fatal(got, err)
	}
	one, _, err := s.GetCycle(ctx, got[0].ID)
	if err != nil || jsonOf(t, one) != jsonOf(t, got[0]) {
		t.Fatalf("listed %s, read %s, %v", jsonOf(t, got[0]), jsonOf(t, one), err)
	}
	if got, err := s.ListCycles(ctx, uuid.NewV7(), false); err != nil || len(got) != 0 {
		t.Fatalf("a project without cycles: %v, %v", got, err)
	}
}

// UpdateCycle writes a cycle's name, description, dates, sort order and last change; what it does not change, its
// owner, archive and creation, stays as it was.
func TestUpdateCycle(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newWorld(t, pool)
	at := now.Truncate(time.Microsecond)
	c := domain.Cycle{ID: uuid.NewV7(), ProjectID: w.web, WorkspaceID: w.workspace, Name: "Sprint", Description: "Ship",
		Dates: domain.Dates{Start: dayOf(t, "2026-11-07"), End: dayOf(t, "2026-11-20")}, OwnedByID: &w.by, SortOrder: 5,
		CreatedAt: at, UpdatedAt: at, CreatedByID: &w.by, UpdatedByID: &w.by}
	if err := s.CreateCycle(ctx, c); err != nil {
		t.Fatal(err)
	}
	editor := uuid.NewV7()
	exec(t, pool, "INSERT INTO users (id, email, password, display_name) VALUES ($1, 'eve@corp.com', 'x', 'eve')", editor)
	changed := c
	changed.Name, changed.Description, changed.Dates, changed.SortOrder = "Two", "", domain.Dates{}, -0.5
	changed.UpdatedAt, changed.UpdatedByID = at.Add(time.Minute), &editor
	// What UpdateCycle does not write, changed here, must not reach the row.
	moved := changed
	moved.OwnedByID, moved.CreatedAt, moved.CreatedByID, moved.WorkspaceID = &editor, at.Add(time.Hour), &editor, uuid.NewV7()
	if err := s.UpdateCycle(ctx, moved); err != nil {
		t.Fatal(err)
	}
	got, found, err := s.GetCycle(ctx, c.ID)
	if err != nil || !found || jsonOf(t, got) != jsonOf(t, changed) {
		t.Fatalf("after UpdateCycle: %s, %t, %v\nwant %s", jsonOf(t, got), found, err, jsonOf(t, changed))
	}
}
