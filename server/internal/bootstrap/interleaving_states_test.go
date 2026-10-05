package bootstrap

import (
	"context"
	"maps"
	"testing"
	"time"
	"uuid"

	projectpg "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres"
	projectapp "github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	projectdomain "github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Interleaving 10, the last state of a group and a state's sequence (M3
// design 9.3, 3.17), on memberWorld's real database, in both orders:
// project's CreateState, UpdateState, DeleteState and MarkDefaultState over
// projectLocks, with the Authorizer as bootstrap wires them, the first held
// at a gate past its write. Every wait has a deadline.

// stateStore is what a write on a state reads and writes through: project's
// store, or one held at a gate.
type stateStore interface {
	projectapp.StateCreator
	projectapp.StateUpdater
	projectapp.StateDeleter
	projectapp.DefaultMarker
}

// stateWrittenHolding stops a write on a state past its write: once it
// holds the workspace FOR SHARE, the project FOR NO KEY UPDATE and the rows
// it wrote, before its commit; a deletion before it counts the group.
type stateWrittenHolding struct {
	*projectpg.Store
	gate *gate
}

func (s stateWrittenHolding) CreateState(ctx context.Context, r projectapp.StateRow) (projectdomain.State, error) {
	created, err := s.Store.CreateState(ctx, r)
	if err != nil {
		return projectdomain.State{}, err
	}
	return created, s.gate.wait(ctx)
}

func (s stateWrittenHolding) UpdateState(ctx context.Context, id uuid.UUID, p projectdomain.StatePatch, by uuid.UUID,
	now time.Time) (projectdomain.State, error) {
	changed, err := s.Store.UpdateState(ctx, id, p, by, now)
	if err != nil {
		return projectdomain.State{}, err
	}
	return changed, s.gate.wait(ctx)
}

func (s stateWrittenHolding) DeleteState(ctx context.Context, id, by uuid.UUID, now time.Time) (bool, error) {
	deleted, err := s.Store.DeleteState(ctx, id, by, now)
	if err != nil {
		return false, err
	}
	return deleted, s.gate.wait(ctx)
}

func (s stateWrittenHolding) MarkDefaultState(ctx context.Context, projectID, id, by uuid.UUID, now time.Time) (bool, error) {
	marked, err := s.Store.MarkDefaultState(ctx, projectID, id, by, now)
	if err != nil {
		return false, err
	}
	return marked, s.gate.wait(ctx)
}

// projectState is a state of a project as statesOf reads it.
type projectState struct {
	Group    projectdomain.StateGroup
	Sequence float64
	Default  bool
}

// statesOf is project's undeleted states, its triage state too, by name.
func (w memberWorld) statesOf(t *testing.T, project uuid.UUID) map[string]projectState {
	t.Helper()
	rows, err := w.pool.Query(pgtest.Soon(t), `SELECT name, "group", sequence, "default" FROM states WHERE project_id = $1 AND deleted_at IS NULL`,
		project)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	states := map[string]projectState{}
	for rows.Next() {
		var name string
		var s projectState
		if err := rows.Scan(&name, &s.Group, &s.Sequence, &s.Default); err != nil {
			t.Fatal(err)
		}
		states[name] = s
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return states
}

// stateWrite is bob's write on Web's states, through project's use case
// over a store, with projectLocks, as project.New wires it, given the ids
// of Web's states by name; apply is what it does to Web's statesOf.
type stateWrite struct {
	name  string
	run   func(ctx context.Context, w memberWorld, states stateStore, ids map[string]uuid.UUID) error
	apply func(states map[string]projectState)
}

// asBob is ctx with bob, Web's admin, its caller.
func (w memberWorld) asBob(ctx context.Context) context.Context {
	return shared.WithActor(ctx, shared.Actor{UserID: w.ids["bob"]})
}

// creates is the creation of the state name in group: after the greatest
// sequence of Web's undeleted states but its triage state: not Ops's,
// whose Cancelled comes after them all, nor Web's deleted Retired.
func creates(name string, group projectdomain.StateGroup) stateWrite {
	return stateWrite{"creates " + name, func(ctx context.Context, w memberWorld, states stateStore, _ map[string]uuid.UUID) error {
		_, err := projectapp.NewCreateState(w.projectLocks(), states, w.tx(), clock.System{}).Execute(w.asBob(ctx), w.web, projectdomain.StateCreate{Name: name, Color: "#0EA5E9", Group: group})
		return err
	}, func(states map[string]projectState) {
		greatest := 0.0
		for _, s := range states {
			if s.Group != projectdomain.GroupTriage {
				greatest = max(greatest, s.Sequence)
			}
		}
		states[name] = projectState{Group: group, Sequence: greatest + 15000}
	}}
}

func deletes(name string) stateWrite {
	return stateWrite{"deletes " + name, func(ctx context.Context, w memberWorld, states stateStore, ids map[string]uuid.UUID) error {
		return projectapp.NewDeleteState(w.projectLocks(), states, w.tx(), clock.System{}).Execute(w.asBob(ctx), ids[name])
	}, func(states map[string]projectState) { delete(states, name) }}
}

func moves(name string, group projectdomain.StateGroup) stateWrite {
	return stateWrite{"moves " + name + " to " + string(group), func(ctx context.Context, w memberWorld, states stateStore,
		ids map[string]uuid.UUID) error {
		_, err := projectapp.NewUpdateState(w.projectLocks(), states, w.tx(), clock.System{}).Execute(w.asBob(ctx), ids[name], projectdomain.StatePatch{Group: &group})
		return err
	}, func(states map[string]projectState) {
		s := states[name]
		s.Group = group
		states[name] = s
	}}
}

func marksDefault(name string) stateWrite {
	return stateWrite{"marks " + name + " the default", func(ctx context.Context, w memberWorld, states stateStore, ids map[string]uuid.UUID) error {
		return projectapp.NewMarkDefaultState(w.projectLocks(), states, w.tx(), clock.System{}).Execute(w.asBob(ctx), ids[name])
	}, func(states map[string]projectState) {
		for n, s := range states {
			s.Default = n == name
			states[n] = s
		}
	}}
}

// Two writes on Web's states at once serialize on Web's row (M3 design 3.6,
// 3.17; 9.3, interleaving 10), in both orders. The first holds acme FOR
// SHARE, Web FOR NO KEY UPDATE and the rows it wrote, and waits at its gate
// past its write: a deletion before it counts the group, a move after its
// count, a creation after its read of the greatest sequence. The second
// shares acme and waits for Web's row: neither writes a row of projects, so
// only that lock's wait satisfies the probe (the order of the second's
// locks is TestEachWriteOnAProjectSharesItsWorkspaceFirst's). Once the
// first has committed, the second decides on what it committed:
//   - Done and QA, the completed group's two states: one deleted or moved to
//     another group, the other's deletion or move is 409
//     project.state_last_in_group: the group keeps one, whichever goes
//     first;
//   - QA made the default and deleted (interleaving 10): made the default
//     first, its deletion is 409 project.state_default; deleted first,
//     making it the default is 404 project.state_not_found; Web keeps one
//     default either way;
//   - two states created: each after the greatest sequence it reads, the
//     second's after the first's, at a moment no earlier than the gate's
//     opening: it read the clock under the locks it took once the first
//     had committed (M3 design 3.3).
//
// Web's states are then as the first left them, and the second's too when
// it succeeded, each read with its group, sequence and default; Ops's, in
// acme too, and Lab's, in beta, as they were.
func TestStateWritesOnOneProjectSerialize(t *testing.T) {
	last := projectdomain.ErrStateLastInGroup
	for _, tt := range []struct {
		a, b               stateWrite
		ifAFirst, ifBFirst error // the second's answer
	}{
		{deletes("Done"), deletes("QA"), last, last},
		{deletes("Done"), moves("QA", projectdomain.GroupStarted), last, last},
		{moves("Done", projectdomain.GroupStarted), moves("QA", projectdomain.GroupBacklog), last, last},
		{marksDefault("QA"), deletes("QA"), projectdomain.ErrStateDefault, projectdomain.ErrStateNotFound},
		{creates("Checked", projectdomain.GroupCompleted), creates("Shipped", projectdomain.GroupCompleted), nil, nil},
	} {
		for _, aFirst := range []bool{true, false} {
			first, second, want := tt.a, tt.b, tt.ifAFirst
			if !aFirst {
				first, second, want = tt.b, tt.a, tt.ifBFirst
			}
			t.Run(first.name+", then "+second.name, func(t *testing.T) {
				w := newMemberWorld(t)
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				states, ops, lab := w.statesOf(t, w.web), w.statesOf(t, w.ops), w.statesOf(t, w.lab)
				ids := map[string]uuid.UUID{}
				for name := range states {
					ids[name] = stateID(t, w.pool, w.web, name)
				}
				g := newGate()
				done := run(func() error { return first.run(ctx, w, stateWrittenHolding{projectpg.New(w.pool), g}, ids) })
				held(t, ctx, g, done, "the first write")
				answered := run(func() error { return second.run(ctx, w, projectpg.New(w.pool), ids) })
				pgtest.WaitForLockWaitOn(t, w.pool, "projects", 5*time.Second)
				opened := time.Now().Truncate(time.Microsecond)
				close(g.open)

				if err := result(t, ctx, done, "the first write"); err != nil {
					t.Errorf("%s = %v, want it done", first.name, err)
				}
				if err := result(t, ctx, answered, "the second write"); !sameOutcome(err, want) {
					t.Errorf("%s = %v, want %v as its first problem", second.name, err, want)
				}
				first.apply(states)
				if want == nil {
					second.apply(states)
					var latest time.Time
					err := w.pool.QueryRow(pgtest.Soon(t), "SELECT max(updated_at) FROM states WHERE project_id = $1", w.web).Scan(&latest)
					if err != nil || latest.Before(opened) {
						t.Errorf("Web's last write at %v (%v); want one no earlier than the gate's opening, %v", latest, err, opened)
					}
				}
				if got := w.statesOf(t, w.web); !maps.Equal(got, states) {
					t.Errorf("Web's states after both: %v; want %v", got, states)
				}
				if got, gotLab := w.statesOf(t, w.ops), w.statesOf(t, w.lab); !maps.Equal(got, ops) || !maps.Equal(gotLab, lab) {
					t.Errorf("Ops's and Lab's states after both: %v, %v; want them as they were, %v, %v", got, gotLab, ops, lab)
				}
			})
		}
	}
}
