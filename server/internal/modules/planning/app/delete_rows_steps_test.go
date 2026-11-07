package app_test

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/app"
)

// planningSteps is the deletion's step order the tests hold DeletionRowSteps to: each parent before its children.
var planningSteps = []app.RowStep{"cycles", "modules", "module_members", "module_links"}

// stepRows holds two projects' undeleted rows of each planning table, by step; a batch soft-deletes the first Limit of
// its project's rows after After, stamping each with the batch's moment and actor. Each committed transaction moves
// the runtime clock on a minute.
type stepRows struct {
	live    map[app.RowStep]map[uuid.UUID][]uuid.UUID
	stamps  map[uuid.UUID]string
	steps   []app.RowStep
	batches []app.RowBatch
	probes  []app.ProjectRemainder
	commits int
	inTx    bool
	failAt  app.RowStep
	start   time.Time
}

var errStep = errors.New("step failed")

// newStepRows seeds project p with 2 cycles and 3 rows of each module table, and its neighbour q with one of each.
func newStepRows(p, q uuid.UUID) *stepRows {
	f := &stepRows{live: map[app.RowStep]map[uuid.UUID][]uuid.UUID{}, stamps: map[uuid.UUID]string{}, start: time.Now()}
	for _, step := range planningSteps {
		n := 3
		if step == "cycles" {
			n = 2
		}
		f.live[step] = map[uuid.UUID][]uuid.UUID{p: ids(n), q: ids(1)}
	}
	return f
}

func stamp(by uuid.UUID, at time.Time) string {
	return fmt.Sprintf("%s at %s", by, at.Format(time.RFC3339Nano))
}

func (f *stepRows) batch(step app.RowStep, b app.RowBatch) ([]uuid.UUID, error) {
	if !f.inTx {
		return nil, errors.New("batch outside a transaction")
	}
	f.steps = append(f.steps, step)
	f.batches = append(f.batches, b)
	if step == f.failAt {
		return nil, errStep
	}
	var out []uuid.UUID
	for _, id := range f.live[step][b.ProjectID] {
		if id.Compare(b.After) > 0 && len(out) < b.Limit {
			out = append(out, id)
		}
	}
	f.live[step][b.ProjectID] = slices.DeleteFunc(f.live[step][b.ProjectID], func(id uuid.UUID) bool { return slices.Contains(out, id) })
	for _, id := range out {
		f.stamps[id] = stamp(b.By, b.At)
	}
	return out, nil
}

func (f *stepRows) DeleteProjectCycles(_ context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	return f.batch("cycles", b)
}
func (f *stepRows) DeleteProjectModules(_ context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	return f.batch("modules", b)
}
func (f *stepRows) DeleteProjectModuleMembers(_ context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	return f.batch("module_members", b)
}
func (f *stepRows) DeleteProjectModuleLinks(_ context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	return f.batch("module_links", b)
}

// ProjectDeletionRemaining answers from the rows: the remainder's step after its cursor, every later step whole.
func (f *stepRows) ProjectDeletionRemaining(_ context.Context, r app.ProjectRemainder) (bool, error) {
	f.probes = append(f.probes, r)
	index := slices.Index(planningSteps, r.Step)
	if index < 0 {
		return false, fmt.Errorf("unknown step %q", r.Step)
	}
	after := r.After
	for _, step := range planningSteps[index:] {
		if slices.ContainsFunc(f.live[step][r.ProjectID], func(id uuid.UUID) bool { return id.Compare(after) > 0 }) {
			return true, nil
		}
		after = uuid.UUID{}
	}
	return false, nil
}

func (f *stepRows) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	f.inTx = true
	defer func() { f.inTx = false }()
	if err := fn(ctx); err != nil {
		return err
	}
	f.commits++
	return nil
}

func (f *stepRows) now() time.Time { return f.start.Add(time.Duration(f.commits) * time.Minute) }

// useCase deletes in batches of two, its budget stopping it after the given number of committed batches.
func (f *stepRows) useCase(batches int) *app.DeleteRows {
	return app.NewDeleteRows(app.DeleteRowsDeps{Rows: f, Tx: f, RuntimeNow: f.now, Batch: 2, Budget: time.Duration(batches) * time.Minute})
}

// all is every live row id of project, of every step.
func (f *stepRows) all(project uuid.UUID) []uuid.UUID {
	var out []uuid.UUID
	for _, step := range planningSteps {
		out = append(out, f.live[step][project]...)
	}
	return out
}

func cloneRows(f *stepRows) *stepRows {
	c := &stepRows{live: map[app.RowStep]map[uuid.UUID][]uuid.UUID{}}
	for step, byProject := range f.live {
		c.live[step] = map[uuid.UUID][]uuid.UUID{}
		for project, rows := range byProject {
			c.live[step][project] = slices.Clone(rows)
		}
	}
	return c
}

// The steps are the four planning tables, each parent before its children, in a slice the caller owns.
func TestDeletionRowStepsParentsFirst(t *testing.T) {
	steps := app.DeletionRowSteps()
	if !slices.Equal(steps, planningSteps) {
		t.Fatalf("steps=%v; want %v", steps, planningSteps)
	}
	steps[0] = "changed"
	if app.DeletionRowSteps()[0] != "cycles" {
		t.Fatal("the steps alias their caller")
	}
}

// The job deletes the project's rows step by step, parents first, every batch with the deletion's original moment
// and actor, each step's cursor starting over; the neighbour project's rows stay live. A retry with other audit data
// finds nothing left and changes no row.
func TestDeleteRowsRunsEveryStepWithTheOriginalMomentAndActor(t *testing.T) {
	p, q, by := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	at := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	f := newStepRows(p, q)
	seeded := cloneRows(f)
	got, err := f.useCase(100).Execute(t.Context(), p, at, by)
	if err != nil || got != (app.DeleteRowsResult{Deleted: 11, Complete: true}) {
		t.Fatalf("Execute = %+v, %v", got, err)
	}
	want := []app.RowStep{"cycles", "cycles", "modules", "modules", "modules", "module_members", "module_members", "module_members",
		"module_links", "module_links", "module_links"}
	if !slices.Equal(f.steps, want) || len(f.probes) != 0 {
		t.Fatalf("steps=%v probes=%v; want %v and none", f.steps, f.probes, want)
	}
	for i, b := range f.batches {
		rows := seeded.live[f.steps[i]][p]
		after := uuid.UUID{}
		if i > 0 && f.steps[i-1] == f.steps[i] {
			after = rows[min(2*(i-slices.Index(f.steps, f.steps[i])), len(rows))-1]
		}
		if b.ProjectID != p || b.Limit != 2 || !b.At.Equal(at) || b.By != by || b.After != after {
			t.Errorf("batch %d of %s = %+v; want after %s at %s by %s", i, f.steps[i], b, after, at, by)
		}
	}
	for _, id := range seeded.all(p) {
		if f.stamps[id] != stamp(by, at) {
			t.Errorf("row %s stamped %q; want %q", id, f.stamps[id], stamp(by, at))
		}
	}
	for _, step := range planningSteps {
		if rows := f.live[step][q]; !slices.Equal(rows, seeded.live[step][q]) || f.stamps[rows[0]] != "" {
			t.Errorf("the neighbour's %s changed: %v", step, rows)
		}
	}
	stamps := maps.Clone(f.stamps)
	again, err := f.useCase(100).Execute(t.Context(), p, at.Add(time.Hour), uuid.NewV7())
	if err != nil || again != (app.DeleteRowsResult{Complete: true}) || !maps.Equal(stamps, f.stamps) {
		t.Fatalf("retry = %+v, %v; stamps changed %t", again, err, !maps.Equal(stamps, f.stamps))
	}
}

// At its budget the job stops before the next batch and probes the step it is in from that step's committed cursor
// (a step's first batch from the start); a later run with the job's same arguments finishes, stamping every row with
// the deletion's original moment and actor.
func TestDeleteRowsProbesTheStepItStoppedIn(t *testing.T) {
	// Batches of two over 2 cycles and 3 rows of each module table: cycles 2+0, modules 2+1+0, members 2+1+0,
	// links 2+1+0.
	for _, tt := range []struct {
		batches, deleted, after int
		step                    app.RowStep
		complete                bool
	}{
		{1, 2, 2, "cycles", false},
		{2, 2, 0, "modules", false},
		{4, 5, 3, "modules", false},
		{5, 5, 0, "module_members", false},
		{9, 10, 2, "module_links", false},
		{10, 11, 3, "module_links", true},
	} {
		t.Run(fmt.Sprintf("%d batches", tt.batches), func(t *testing.T) {
			p, q, by := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
			at := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
			f := newStepRows(p, q)
			seeded := cloneRows(f)
			got, err := f.useCase(tt.batches).Execute(t.Context(), p, at, by)
			if err != nil || got != (app.DeleteRowsResult{Deleted: tt.deleted, Complete: tt.complete}) || f.commits != tt.batches {
				t.Fatalf("Execute = %+v, %v after %d batches", got, err, f.commits)
			}
			after := uuid.UUID{}
			if tt.after > 0 {
				after = seeded.live[tt.step][p][tt.after-1]
			}
			if want := (app.ProjectRemainder{ProjectID: p, Step: tt.step, After: after}); len(f.probes) != 1 || f.probes[0] != want {
				t.Fatalf("probes=%+v; want %+v", f.probes, want)
			}
			rest, err := f.useCase(100).Execute(t.Context(), p, at, by)
			if err != nil || rest != (app.DeleteRowsResult{Deleted: 11 - tt.deleted, Complete: true}) {
				t.Fatalf("resumed = %+v, %v", rest, err)
			}
			for _, id := range seeded.all(p) {
				if f.stamps[id] != stamp(by, at) {
					t.Errorf("row %s stamped %q; want %q", id, f.stamps[id], stamp(by, at))
				}
			}
			for _, id := range seeded.all(q) {
				if f.stamps[id] != "" {
					t.Errorf("the neighbour's row %s deleted", id)
				}
			}
		})
	}
}

// A step that fails stops the job with its error, the earlier steps' batches committed and counted, the later steps
// not run: River retries it.
func TestDeleteRowsStopsAtAFailedStep(t *testing.T) {
	p, q, by := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	f := newStepRows(p, q)
	f.failAt = "module_members"
	got, err := f.useCase(100).Execute(t.Context(), p, time.Now(), by)
	if !errors.Is(err, errStep) || got != (app.DeleteRowsResult{Deleted: 5}) {
		t.Fatalf("Execute = %+v, %v", got, err)
	}
	if last := f.steps[len(f.steps)-1]; last != "module_members" || len(f.live["module_links"][p]) != 3 {
		t.Fatalf("steps=%v links left=%d; want the job stopped at module_members", f.steps, len(f.live["module_links"][p]))
	}
}
