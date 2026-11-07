package app

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// lockWorld is the lock path's ports as fakes that log their calls, inside or outside the write's transaction.
type lockWorld struct {
	calls    []string
	facts    map[uuid.UUID]ProjectFacts
	cycles   map[uuid.UUID]domain.Cycle
	modules  map[uuid.UUID]domain.Module
	links    map[uuid.UUID]domain.ModuleLink
	grant    shared.Grant
	refusal  error // the Authorizer's answer, nil for an allowance
	targets  []shared.Target
	onLock   func() // what commits while the lock waits
	busy     error  // the transaction's failure
	gateFail error
	now      time.Time      // the clock's reading
	created  []domain.Cycle // what CreateCycle stored
	updated  []domain.Cycle // what UpdateCycle stored
}

type inTx struct{}

func (w *lockWorld) log(ctx context.Context, s string) {
	if ctx.Value(inTx{}) == nil {
		s += " outside tx"
	}
	w.calls = append(w.calls, s)
}

func (w *lockWorld) Facts(ctx context.Context, id uuid.UUID) (ProjectFacts, bool, error) {
	w.log(ctx, "Facts")
	f, ok := w.facts[id]
	return f, ok, nil
}

func (w *lockWorld) Lock(ctx context.Context, id uuid.UUID) (ProjectFacts, bool, error) {
	if w.onLock != nil {
		w.onLock()
	}
	w.log(ctx, "Lock")
	f, ok := w.facts[id]
	return f, ok, nil
}

func (w *lockWorld) GetCycle(ctx context.Context, id uuid.UUID) (domain.Cycle, bool, error) {
	w.log(ctx, "GetCycle")
	c, ok := w.cycles[id]
	return c, ok, nil
}

func (w *lockWorld) Authorize(ctx context.Context, _ shared.Actor, action shared.Action, t shared.Target) (shared.Grant, error) {
	w.log(ctx, "Authorize "+string(action))
	w.targets = append(w.targets, t)
	return w.grant, w.refusal
}

func (w *lockWorld) Enter(ctx context.Context, projects ...uuid.UUID) (func(), error) {
	w.log(ctx, fmt.Sprint("Enter ", len(projects)))
	if w.gateFail != nil {
		return nil, w.gateFail
	}
	return func() { w.calls = append(w.calls, "Leave") }, nil
}

func (w *lockWorld) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	w.calls = append(w.calls, "Begin")
	if w.busy != nil {
		return w.busy
	}
	err := fn(context.WithValue(ctx, inTx{}, true))
	w.calls = append(w.calls, "End")
	return err
}

var (
	lockWorkspace = uuid.NewV7()
	lockProject   = uuid.NewV7()
	lockCycle     = uuid.NewV7()
	lockCreator   = uuid.NewV7()
)

func newLockWorld() *lockWorld {
	return &lockWorld{
		facts:  map[uuid.UUID]ProjectFacts{lockProject: {WorkspaceID: lockWorkspace, Timezone: "UTC", CycleView: true}},
		cycles: map[uuid.UUID]domain.Cycle{lockCycle: {ID: lockCycle, ProjectID: lockProject, WorkspaceID: lockWorkspace, CreatedByID: &lockCreator}},
		grant:  shared.Grant{ProjectRole: shared.RoleMember},
	}
}

func (w *lockWorld) locks() Locks { return NewLocks(w, w, w, w, w) }

func onProject(action shared.Action) write { return write{project: lockProject, action: action} }
func onCycle(action shared.Action) write   { c := lockCycle; return write{cycle: &c, action: action} }

// run runs w's write, its body recording what it holds.
func (w *lockWorld) run(wr write) (held, error) {
	var got held
	err := w.locks().run(context.Background(), shared.Actor{UserID: uuid.NewV7()}, wr, func(ctx context.Context, h held) error {
		w.log(ctx, "write")
		got = h
		return nil
	})
	return got, err
}

// A write's precheck reads, then decides, outside any transaction; it enters the project's write gate holding
// nothing; in its transaction it locks the project first, reads the cycle again under the lock and decides again;
// the body runs last, and the gate is left once the transaction has ended (M6 design 6).
func TestTheLockPathsOrder(t *testing.T) {
	w := newLockWorld()
	h, err := w.run(onCycle(domain.ActionCycleWrite))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"GetCycle outside tx", "Facts outside tx", "Authorize cycle.write outside tx", "Enter 1 outside tx", "Begin",
		"Lock", "GetCycle", "Authorize cycle.write", "write", "End", "Leave"}
	if !slices.Equal(w.calls, want) {
		t.Fatalf("calls\n%q\nwant\n%q", w.calls, want)
	}
	if h.projectID != lockProject || h.cycle.ID != lockCycle || h.project.WorkspaceID != lockWorkspace || h.grant != w.grant {
		t.Fatalf("held %+v", h)
	}
	// The decision knows the cycle's creator, whom cycle.delete allows (AllowCreator).
	for _, target := range w.targets {
		if target != (shared.Target{WorkspaceID: lockWorkspace, ProjectID: lockProject, CreatorID: lockCreator}) {
			t.Fatalf("target %+v", target)
		}
	}

	w = newLockWorld()
	if _, err := w.run(onProject(domain.ActionCycleWrite)); err != nil {
		t.Fatal(err)
	}
	want = []string{"Facts outside tx", "Authorize cycle.write outside tx", "Enter 1 outside tx", "Begin", "Lock", "Authorize cycle.write", "write", "End", "Leave"}
	if !slices.Equal(w.calls, want) {
		t.Fatalf("a project write's calls\n%q\nwant\n%q", w.calls, want)
	}
}

// The refusals in the order of M6 P1 spec R2: not found (or not visible) before forbidden (the role, or an OwnOnly
// grant on a write) before the archived project before the cycles turned off. Each case also has every later
// condition, so a refusal answered out of order fails here. A precheck refusal takes no gate and no transaction.
func TestTheLockPathsRefusalsInOrder(t *testing.T) {
	forbidden := shared.Forbidden()
	for _, tt := range []struct {
		name  string
		write write
		set   func(*lockWorld)
		want  error
	}{
		{"no project", onProject(domain.ActionCycleWrite), func(w *lockWorld) { delete(w.facts, lockProject) }, domain.ErrProjectNotFound},
		{"project not visible", onProject(domain.ActionCycleWrite), func(w *lockWorld) { w.refusal = shared.ErrNotVisible }, domain.ErrProjectNotFound},
		{"no cycle", onCycle(domain.ActionCycleWrite), func(w *lockWorld) { delete(w.cycles, lockCycle) }, domain.ErrCycleNotFound},
		{"cycle's project not visible", onCycle(domain.ActionCycleWrite), func(w *lockWorld) { w.refusal = shared.ErrNotVisible }, domain.ErrCycleNotFound},
		{"cycle's project gone", onCycle(domain.ActionCycleWrite), func(w *lockWorld) { delete(w.facts, lockProject) }, domain.ErrCycleNotFound},
		{"role refused", onCycle(domain.ActionCycleDelete), func(w *lockWorld) { w.refusal = forbidden }, forbidden},
		{"own only", onCycle(domain.ActionCycleWrite), func(w *lockWorld) { w.grant.OwnOnly = true }, shared.Forbidden()},
		{"project archived", onProject(domain.ActionCycleWrite), func(w *lockWorld) {}, domain.ErrProjectArchived},
		{"cycles off", onCycle(domain.ActionCycleWrite), func(w *lockWorld) { w.facts[lockProject] = ProjectFacts{WorkspaceID: lockWorkspace} }, domain.ErrCyclesDisabled},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := newLockWorld()
			// Every later condition holds too: the project archived and its cycles off.
			if tt.name != "cycles off" {
				w.facts[lockProject] = ProjectFacts{WorkspaceID: lockWorkspace, Archived: true}
			}
			tt.set(w)
			_, err := w.run(tt.write)
			if !sameRefusal(err, tt.want) {
				t.Fatalf("got %v; want %v", err, tt.want)
			}
			if slices.Contains(w.calls, "Begin") || slices.ContainsFunc(w.calls, func(c string) bool { return len(c) > 5 && c[:5] == "Enter" }) {
				t.Fatalf("a precheck refusal took the gate or a transaction: %q", w.calls)
			}
		})
	}
}

// What committed while the lock waited is decided again under the lock: the project deleted, archived, its cycles
// turned off, the caller removed or demoted, the cycle deleted. The body does not run.
func TestTheLockPathDecidesAgainUnderTheLock(t *testing.T) {
	for _, tt := range []struct {
		name   string
		write  write
		change func(*lockWorld)
		want   error
	}{
		{"project deleted", onProject(domain.ActionCycleWrite), func(w *lockWorld) { delete(w.facts, lockProject) }, domain.ErrProjectNotFound},
		{"cycle's project deleted", onCycle(domain.ActionCycleWrite), func(w *lockWorld) { delete(w.facts, lockProject) }, domain.ErrCycleNotFound},
		{"cycle deleted", onCycle(domain.ActionCycleWrite), func(w *lockWorld) { delete(w.cycles, lockCycle) }, domain.ErrCycleNotFound},
		{"caller removed", onCycle(domain.ActionCycleWrite), func(w *lockWorld) { w.refusal = shared.ErrNotVisible }, domain.ErrCycleNotFound},
		{"caller demoted", onCycle(domain.ActionCycleWrite), func(w *lockWorld) { w.refusal = shared.Forbidden() }, shared.Forbidden()},
		{"guest rule", onProject(domain.ActionCycleWrite), func(w *lockWorld) { w.grant.OwnOnly = true }, shared.Forbidden()},
		{"project archived", onCycle(domain.ActionCycleWrite), func(w *lockWorld) {
			w.facts[lockProject] = ProjectFacts{WorkspaceID: lockWorkspace, Archived: true, CycleView: true}
		}, domain.ErrProjectArchived},
		{"cycles turned off", onProject(domain.ActionCycleWrite), func(w *lockWorld) { w.facts[lockProject] = ProjectFacts{WorkspaceID: lockWorkspace} }, domain.ErrCyclesDisabled},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := newLockWorld()
			w.onLock = func() { tt.change(w) }
			_, err := w.run(tt.write)
			if !sameRefusal(err, tt.want) {
				t.Fatalf("got %v; want %v", err, tt.want)
			}
			if slices.Contains(w.calls, "write") || w.calls[len(w.calls)-1] != "Leave" {
				t.Fatalf("calls %q: the body ran or the gate was not left last", w.calls)
			}
		})
	}
}

// A refusal of the write gate, or a busy transaction (a lock timeout, 503 through bootstrap's busyTx), is the
// write's answer; the gate is left on every path after it was entered.
func TestTheLockPathPassesOnGateAndTransactionFailures(t *testing.T) {
	busy := shared.ServerBusy(time.Second)
	w := newLockWorld()
	w.gateFail = busy
	if _, err := w.run(onProject(domain.ActionCycleWrite)); !errors.Is(err, busy) || slices.Contains(w.calls, "Begin") {
		t.Fatalf("gate refusal = %v, calls %q", err, w.calls)
	}
	w = newLockWorld()
	w.busy = busy
	if _, err := w.run(onProject(domain.ActionCycleWrite)); !errors.Is(err, busy) || w.calls[len(w.calls)-1] != "Leave" {
		t.Fatalf("busy transaction = %v, calls %q", err, w.calls)
	}
}

// sameRefusal compares two shared errors by their kind and code, any other error by identity.
func sameRefusal(got, want error) bool {
	var g, w *shared.Error
	if errors.As(got, &g) && errors.As(want, &w) {
		return g.Kind == w.Kind && g.Code == w.Code
	}
	return errors.Is(got, want)
}
