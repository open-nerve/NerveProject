package app

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The lock world's cycle store and clock, for the use cases.

func (w *lockWorld) ProjectCycleDates(ctx context.Context, project uuid.UUID) ([]domain.DatedCycle, error) {
	w.log(ctx, "ProjectCycleDates")
	var out []domain.DatedCycle
	for _, c := range w.cycles {
		if c.ProjectID == project {
			out = append(out, domain.DatedCycle{ID: c.ID, Dates: c.Dates})
		}
	}
	return out, nil
}

func (w *lockWorld) LowestCycleSortOrder(ctx context.Context, project uuid.UUID) (*float64, error) {
	w.log(ctx, "LowestCycleSortOrder")
	var lowest *float64
	for _, c := range w.cycles {
		if c.ProjectID == project && (lowest == nil || c.SortOrder < *lowest) {
			v := c.SortOrder
			lowest = &v
		}
	}
	return lowest, nil
}

func (w *lockWorld) CreateCycle(ctx context.Context, c domain.Cycle) error {
	w.log(ctx, "CreateCycle")
	w.cycles[c.ID] = c
	w.created = append(w.created, c)
	return nil
}

func (w *lockWorld) Now() time.Time {
	w.calls = append(w.calls, "Now")
	return w.now
}

func date(t *testing.T, s string) *shared.Date {
	t.Helper()
	d, err := shared.ParseDate(s)
	if err != nil {
		t.Fatal(err)
	}
	return &d
}

func (w *lockWorld) create(t *testing.T, in domain.NewCycle) (domain.ReadCycle, error) {
	t.Helper()
	uc := NewCreateCycle(CreateCycleDeps{Locks: w.locks(), Cycles: w, Clock: w})
	return uc.Execute(shared.WithActor(context.Background(), shared.Actor{UserID: lockCreator}), lockProject, in)
}

// A draft is created first among the project's cycles, owned, created and last changed by the caller at the moment
// read after the project's lock, its name without surrounding white space; with no cycle yet, at 65535. A draft
// reads no other cycle's dates.
func TestCreateCycleDraft(t *testing.T) {
	w := newLockWorld()
	w.cycles = map[uuid.UUID]domain.Cycle{}
	w.now = time.Date(2026, 11, 7, 1, 0, 0, 0, time.UTC)
	got, err := w.create(t, domain.NewCycle{Name: "  Sprint 1 ", Description: "Ship it"})
	if err != nil {
		t.Fatal(err)
	}
	c := got.Cycle
	if c.ID == (uuid.UUID{}) || c.ProjectID != lockProject || c.WorkspaceID != lockWorkspace || c.Name != "Sprint 1" || c.Description != "Ship it" ||
		c.Dates.Dated() || c.SortOrder != 65535 || c.ArchivedAt != nil || got.Status != domain.StatusDraft {
		t.Fatalf("created %+v status %s", c, got.Status)
	}
	for _, id := range []*uuid.UUID{c.OwnedByID, c.CreatedByID, c.UpdatedByID} {
		if id == nil || *id != lockCreator {
			t.Fatalf("owner/creator/writer %v; want the caller", id)
		}
	}
	if !c.CreatedAt.Equal(w.now) || !c.UpdatedAt.Equal(w.now) || len(w.created) != 1 || w.created[0] != c {
		t.Fatalf("stored %+v at %s/%s", w.created, c.CreatedAt, c.UpdatedAt)
	}
	lock, now := slices.Index(w.calls, "Lock"), slices.Index(w.calls, "Now")
	if lock < 0 || now < lock || slices.Contains(w.calls, "ProjectCycleDates") || w.calls[len(w.calls)-1] != "Leave" {
		t.Fatalf("calls %q: want the clock read after the lock, no dates read for a draft", w.calls)
	}
}

// A dated cycle is checked against the project's other cycles under the lock: one sharing a day is refused and
// nothing is written; one apart is created before the lowest, its status read on the project's day.
func TestCreateCycleWithDates(t *testing.T) {
	w := newLockWorld()
	w.facts[lockProject] = ProjectFacts{WorkspaceID: lockWorkspace, Timezone: "Asia/Shanghai", CycleView: true}
	w.cycles[lockCycle] = domain.Cycle{ID: lockCycle, ProjectID: lockProject, SortOrder: 100,
		Dates: domain.Dates{Start: date(t, "2026-11-01"), End: date(t, "2026-11-07")}}
	// 16:00 UTC on November 6 is midnight of November 7 in Shanghai.
	w.now = time.Date(2026, 11, 6, 16, 0, 0, 0, time.UTC)
	if _, err := w.create(t, domain.NewCycle{Name: "Overlap", Dates: domain.Dates{Start: date(t, "2026-11-07"), End: date(t, "2026-11-10")}}); !errors.Is(err, domain.ErrDatesOverlap) || len(w.created) != 0 {
		t.Fatalf("overlap = %v, created %d", err, len(w.created))
	}
	read, lock := slices.Index(w.calls, "ProjectCycleDates"), slices.Index(w.calls, "Lock")
	if read < lock || lock < 0 {
		t.Fatalf("calls %q: want the dates read under the lock", w.calls)
	}
	w.calls = nil
	got, err := w.create(t, domain.NewCycle{Name: "Next", Dates: domain.Dates{Start: date(t, "2026-11-08"), End: date(t, "2026-11-10")}})
	if err != nil {
		t.Fatal(err)
	}
	if got.SortOrder != -9900 || got.Status != domain.StatusUpcoming || got.Dates.Start.String() != "2026-11-08" {
		t.Fatalf("created %+v status %s; want sort order -9900, upcoming on Shanghai's November 7", got.Cycle, got.Status)
	}
}

// The values are checked before anything is read: a bad name, a date alone or dates out of order answer
// validation_failed with every problem, and the project is not looked at.
func TestCreateCycleChecksValuesFirst(t *testing.T) {
	w := newLockWorld()
	delete(w.facts, lockProject) // a read would answer project.not_found
	_, err := w.create(t, domain.NewCycle{Name: " ", Description: "a\x00b", Dates: domain.Dates{Start: date(t, "2026-11-07")}})
	var e *shared.Error
	if !errors.As(err, &e) || e.Code != shared.CodeValidationFailed || len(e.Fields) != 3 {
		t.Fatalf("got %v; want validation_failed on name, description and end_date", err)
	}
	if len(w.calls) != 0 {
		t.Fatalf("calls %q; want none", w.calls)
	}
}
