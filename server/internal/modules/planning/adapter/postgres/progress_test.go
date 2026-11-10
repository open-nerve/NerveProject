package postgresadapter_test

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
)

// frozenProgress is a cycle's frozen progress with started work items in progress, frozen at at: every part of the
// document, a distribution row of none and a burndown day after the freeze among them.
func frozenProgress(t *testing.T, started int64, at time.Time) domain.Frozen {
	t.Helper()
	alice, ui := uuid.NewV7(), uuid.NewV7()
	two := int64(2)
	return domain.Frozen{
		Progress: domain.Progress{
			Counts:    domain.Counts{Total: started + 3, Backlog: 1, Started: started, Completed: 1, Cancelled: 1},
			Assignees: []domain.AssigneeDistribution{{AssigneeID: &alice, Tally: domain.Tally{Total: 1, Completed: 1}}, {Tally: domain.Tally{Total: started + 1, Pending: started + 1}}},
			Labels:    []domain.LabelDistribution{{LabelID: &ui, Tally: domain.Tally{Total: 1, Pending: 1}}},
			Burndown: &domain.Burndown{StartDate: *dayOf(t, "2026-11-01"), EndDate: *dayOf(t, "2026-11-02"), Scope: started + 2,
				Points: []domain.BurndownPoint{{Date: *dayOf(t, "2026-11-01"), Remaining: &two}, {Date: *dayOf(t, "2026-11-02")}}},
		},
		FrozenAt: at.Truncate(time.Microsecond),
	}
}

// cycleRow is the cycle id's row as JSON, but its frozen progress.
func cycleRow(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) string {
	t.Helper()
	var row string
	if err := pool.QueryRow(context.Background(), "SELECT (to_jsonb(c) - 'progress_snapshot')::text FROM cycles c WHERE id=$1", id).Scan(&row); err != nil {
		t.Fatal(err)
	}
	return row
}

// storedAs reports whether the cycle id stores doc as its frozen progress, as jsonb; a doc of "" is none.
func storedAs(t *testing.T, pool *pgxpool.Pool, id uuid.UUID, doc string) bool {
	t.Helper()
	var same bool
	if err := pool.QueryRow(context.Background(), "SELECT progress_snapshot IS NOT DISTINCT FROM nullif($2, '')::jsonb FROM cycles WHERE id=$1", id, doc).Scan(&same); err != nil {
		t.Fatal(err)
	}
	return same
}

// A cycle is frozen once (M6 P4 spec R5): the first freeze stores its progress as EncodeFrozen encodes it and answers
// that it wrote; a later one answers that it did not and leaves the first. A freeze changes nothing else of the
// cycle, its last change included (spec R6). A deleted cycle and a missing one are not frozen. The progress reads
// back whole; there is none for a cycle not frozen, deleted or missing.
func TestFreezeCycle(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newWorld(t, pool)
	id, gone, missing := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	earlier := now.Add(-2 * time.Hour)
	cycle(t, pool, w, w.web, id, nil)
	cycle(t, pool, w, w.web, gone, &earlier)
	for _, c := range []uuid.UUID{id, gone, missing} {
		if f, err := s.CycleSnapshot(ctx, c); err != nil || f != nil {
			t.Fatalf("CycleSnapshot(%s) before a freeze = %v, %v; want none", c, f, err)
		}
	}
	before := cycleRow(t, pool, id)

	first := frozenProgress(t, 2, now)
	if wrote, err := s.FreezeCycle(ctx, id, first); err != nil || !wrote {
		t.Fatalf("first FreezeCycle = %t, %v; want written", wrote, err)
	}
	doc, err := domain.EncodeFrozen(first)
	if err != nil {
		t.Fatal(err)
	}
	if !storedAs(t, pool, id, string(doc)) {
		t.Fatalf("stored progress is not %s", doc)
	}
	if after := cycleRow(t, pool, id); after != before {
		t.Fatalf("the freeze changed the cycle:\n%s\nwant\n%s", after, before)
	}
	got, err := s.CycleSnapshot(ctx, id)
	if err != nil || got == nil || jsonOf(t, *got) != jsonOf(t, first) {
		t.Fatalf("CycleSnapshot = %v, %v\nwant %s", got, err, jsonOf(t, first))
	}

	second := frozenProgress(t, 5, now.Add(time.Hour))
	if wrote, err := s.FreezeCycle(ctx, id, second); err != nil || wrote {
		t.Fatalf("second FreezeCycle = %t, %v; want not written", wrote, err)
	}
	if got, err := s.CycleSnapshot(ctx, id); err != nil || got == nil || jsonOf(t, *got) != jsonOf(t, first) {
		t.Fatalf("CycleSnapshot after a second freeze = %v, %v\nwant the first %s", got, err, jsonOf(t, first))
	}
	if after := cycleRow(t, pool, id); after != before {
		t.Fatalf("the second freeze changed the cycle:\n%s\nwant\n%s", after, before)
	}

	for _, c := range []uuid.UUID{gone, missing} {
		if wrote, err := s.FreezeCycle(ctx, c, first); err != nil || wrote {
			t.Fatalf("FreezeCycle(%s) = %t, %v; want not written", c, wrote, err)
		}
	}
	if !storedAs(t, pool, gone, "") {
		t.Fatal("the deleted cycle was frozen")
	}
	// A cycle frozen and then deleted has no progress to read.
	exec(t, pool, "UPDATE cycles SET progress_snapshot=$2 WHERE id=$1", gone, doc)
	if f, err := s.CycleSnapshot(ctx, gone); err != nil || f != nil {
		t.Fatalf("CycleSnapshot of a deleted cycle = %v, %v; want none", f, err)
	}
}

// The list rows' frozen counts (spec R5): of the cycles ids, the undeleted frozen ones, archived or not, each answer
// the six counts and the moment of their progress, read from those seven keys alone (DecodeFrozenCounts refuses the
// whole document); a cycle not frozen, deleted or missing has none, and no ids none at all. A stored progress that is
// not EncodeFrozen's is refused, by either read.
func TestCycleSnapshotCounts(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newWorld(t, pool)
	ids := sortedIDs(5) // ids[4] is missing
	frozen, archived, open, deleted := ids[0], ids[1], ids[2], ids[3]
	plant(t, pool, w, "cycles",
		planned{id: frozen, project: w.web, name: "frozen", day: "2026-11-01"},
		planned{id: archived, project: w.web, name: "archived", day: "2026-10-01", archived: true},
		planned{id: open, project: w.web, name: "open", day: "2026-11-20"},
		planned{id: deleted, project: w.web, name: "deleted", day: "2026-11-01", deleted: true})
	progress := map[uuid.UUID]domain.Frozen{frozen: frozenProgress(t, 2, now), archived: frozenProgress(t, 7, now.Add(-time.Hour))}
	for id, f := range progress {
		if wrote, err := s.FreezeCycle(ctx, id, f); err != nil || !wrote {
			t.Fatalf("FreezeCycle(%s) = %t, %v", id, wrote, err)
		}
	}
	doc, err := domain.EncodeFrozen(progress[frozen])
	if err != nil {
		t.Fatal(err)
	}
	exec(t, pool, "UPDATE cycles SET progress_snapshot=$2 WHERE id=$1", deleted, doc)

	got, err := s.CycleSnapshotCounts(ctx, ids)
	if err != nil {
		t.Fatal(err)
	}
	want := map[uuid.UUID]domain.FrozenCounts{}
	for id, f := range progress {
		want[id] = domain.FrozenCounts{Counts: f.Counts, FrozenAt: f.FrozenAt}
	}
	if len(got) != len(want) || jsonOf(t, got) != jsonOf(t, want) {
		t.Fatalf("CycleSnapshotCounts = %s\nwant %s", jsonOf(t, got), jsonOf(t, want))
	}
	if got, err := s.CycleSnapshotCounts(ctx, nil); err != nil || len(got) != 0 {
		t.Fatalf("CycleSnapshotCounts of no ids = %v, %v; want none", got, err)
	}

	// A frozen progress that EncodeFrozen did not write is a defect, not a cycle without one.
	exec(t, pool, `UPDATE cycles SET progress_snapshot='{"total_issues":1}' WHERE id=$1`, open)
	if got, err := s.CycleSnapshotCounts(ctx, []uuid.UUID{frozen, open}); err == nil {
		t.Fatalf("CycleSnapshotCounts read a stray document: %v", got)
	}
	if got, err := s.CycleSnapshot(ctx, open); err == nil {
		t.Fatalf("CycleSnapshot read a stray document: %v", got)
	}
}

// The freeze job's candidates (spec 2.5): the projects, in id order and each once, that have an undeleted cycle not
// frozen whose end is on the UTC day or before it, archived or not. A cycle ending after it, a draft, a deleted one
// and a frozen one make no project a candidate.
func TestFreezeDueProjects(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newWorld(t, pool)
	projects := sortedIDs(6)
	for i, p := range projects {
		exec(t, pool, "INSERT INTO projects (id, workspace_id, name, identifier) VALUES ($1, $2, $3, $4)", p, w.workspace,
			fmt.Sprintf("P%d", i), fmt.Sprintf("P%d", i))
	}
	plant(t, pool, w, "cycles",
		// Ends on the UTC day: due.
		planned{id: uuid.NewV7(), project: projects[1], name: "today", day: "2026-11-10"},
		// Two ended, one archived: due, once.
		planned{id: uuid.NewV7(), project: projects[0], name: "archived", day: "2026-11-09", archived: true},
		planned{id: uuid.NewV7(), project: projects[0], name: "ended", day: "2026-11-01"},
		// Not due: ends tomorrow, a draft.
		planned{id: uuid.NewV7(), project: projects[2], name: "tomorrow", day: "2026-11-11"},
		planned{id: uuid.NewV7(), project: projects[2], name: "draft"},
		// Not due: deleted.
		planned{id: uuid.NewV7(), project: projects[3], name: "deleted", day: "2026-11-01", deleted: true})
	frozen := uuid.NewV7()
	plant(t, pool, w, "cycles", planned{id: frozen, project: projects[4], name: "frozen", day: "2026-11-01"})
	exec(t, pool, "UPDATE cycles SET progress_snapshot='{}' WHERE id=$1", frozen)

	got, err := s.FreezeDueProjects(ctx, *dayOf(t, "2026-11-10"))
	if err != nil || !slices.Equal(got, projects[:2]) {
		t.Fatalf("FreezeDueProjects = %v, %v; want %v", got, err, projects[:2])
	}
	if got, err := s.FreezeDueProjects(ctx, *dayOf(t, "2026-10-31")); err != nil || len(got) != 0 {
		t.Fatalf("FreezeDueProjects before every end = %v, %v; want none", got, err)
	}
}

// The cycles the job freezes in a project on its day (spec 2.5): the project's undeleted cycles not frozen that
// ended before today, archived ones too, in id order, as GetCycle reads them. One ending today has not ended; one
// ending later, a draft, a deleted one, a frozen one and another project's are not among them.
func TestCyclesToFreeze(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newWorld(t, pool)
	ids := sortedIDs(8)
	plant(t, pool, w, "cycles",
		planned{id: ids[7], project: w.ops, name: "other project", day: "2026-11-01"},
		planned{id: ids[6], project: w.web, name: "frozen", day: "2026-11-01"},
		planned{id: ids[5], project: w.web, name: "deleted", day: "2026-11-01", deleted: true},
		planned{id: ids[4], project: w.web, name: "draft"},
		planned{id: ids[3], project: w.web, name: "later", day: "2026-11-20"},
		planned{id: ids[2], project: w.web, name: "today", day: "2026-11-10"},
		planned{id: ids[1], project: w.web, name: "archived", day: "2026-11-01", archived: true},
		planned{id: ids[0], project: w.web, name: "yesterday", day: "2026-11-09"})
	exec(t, pool, "UPDATE cycles SET progress_snapshot='{}' WHERE id=$1", ids[6])

	got, err := s.CyclesToFreeze(ctx, w.web, *dayOf(t, "2026-11-10"))
	if err != nil {
		t.Fatal(err)
	}
	var want []domain.Cycle
	for _, id := range ids[:2] {
		c, found, err := s.GetCycle(ctx, id)
		if err != nil || !found {
			t.Fatalf("GetCycle(%s) = %t, %v", id, found, err)
		}
		want = append(want, c)
	}
	if jsonOf(t, got) != jsonOf(t, want) {
		t.Fatalf("CyclesToFreeze = %s\nwant %s", jsonOf(t, got), jsonOf(t, want))
	}
	if got, err := s.CyclesToFreeze(ctx, w.web, *dayOf(t, "2026-11-01")); err != nil || len(got) != 0 {
		t.Fatalf("CyclesToFreeze on the first end day = %v, %v; want none", got, err)
	}
}
