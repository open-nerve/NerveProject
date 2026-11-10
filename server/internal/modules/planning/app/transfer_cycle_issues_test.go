package app

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// transferWorld is the lock world of a transfer: its issue module answers that it moved moved, and records what it was
// asked to move; its store freezes a cycle once, as the store's FreezeCycle.
type transferWorld struct {
	*lockWorld
	moved     int
	transfers []Transfer
}

func (w *transferWorld) FreezeCycle(ctx context.Context, id uuid.UUID, f domain.Frozen) (bool, error) {
	w.log(ctx, "FreezeCycle")
	if w.snapshots[id] != nil {
		return false, nil
	}
	w.snapshots[id] = &f
	return true, nil
}

func (w *transferWorld) TransferCycleIssues(ctx context.Context, t Transfer) (int, error) {
	w.log(ctx, "TransferCycleIssues")
	w.transfers = append(w.transfers, t)
	return w.moved, nil
}

var (
	transferNext, transferElsewhere = uuid.NewV7(), uuid.NewV7()
	// The project's states: one of each group, and a triage one.
	transferBacklog, transferTodo, transferDoing    = uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	transferDone, transferCancelled, transferTriage = uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
)

// newTransferWorld is Sprint's world on the 14th of November in Shanghai (completedSprint), the day after its last:
// Sprint has a work item in each state of the five groups, the one completed at 10:00 UTC on the 13th; Next, from the
// 16th to the 20th, is of its project, Elsewhere of another. The issue module moves three.
func newTransferWorld(t *testing.T) *transferWorld {
	w := &transferWorld{lockWorld: sprintWorld(t), moved: 3}
	completedSprint(w.lockWorld)
	w.cycles[transferNext] = domain.Cycle{ID: transferNext, ProjectID: lockProject, WorkspaceID: lockWorkspace, Name: "Next",
		Dates: domain.Dates{Start: date(t, "2026-11-16"), End: date(t, "2026-11-20")}}
	w.cycles[transferElsewhere] = domain.Cycle{ID: transferElsewhere, ProjectID: uuid.NewV7(), WorkspaceID: lockWorkspace, Name: "Elsewhere",
		Dates: domain.Dates{Start: date(t, "2026-11-16"), End: date(t, "2026-11-20")}}
	w.groups = map[uuid.UUID]domain.StateGroup{transferBacklog: domain.GroupBacklog, transferTodo: domain.GroupUnstarted,
		transferDoing: domain.GroupStarted, transferDone: domain.GroupCompleted, transferCancelled: domain.GroupCancelled, transferTriage: "triage"}
	completed := time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC)
	w.items = map[uuid.UUID][]domain.WorkItem{lockCycle: {{ID: uuid.NewV7(), StateID: transferBacklog}, {ID: uuid.NewV7(), StateID: transferTodo},
		{ID: uuid.NewV7(), StateID: transferDoing}, {ID: uuid.NewV7(), StateID: transferDone, CompletedAt: &completed},
		{ID: uuid.NewV7(), StateID: transferCancelled}}}
	w.snapshots = map[uuid.UUID]*domain.Frozen{}
	return w
}

// transfer is the editor's transfer of Sprint's work items to the cycle target.
func (w *transferWorld) transfer(target uuid.UUID) (int, error) {
	uc := NewTransferCycleIssues(TransferCycleIssuesDeps{Locks: w.locks(), Cycles: w, Groups: w, Stats: w, Issues: w, Clock: w})
	return uc.Execute(shared.WithActor(context.Background(), shared.Actor{UserID: lockEditor}), lockCycle, target)
}

// sprintFrozen is Sprint's progress as the transfer world freezes it, at its clock's reading: every work item, on the
// 14th in Shanghai, its burndown of four (the cancelled one out) from the 2nd to the 12th, three on the 13th, the day of
// its completion there.
func (w *transferWorld) sprintFrozen(t *testing.T) domain.Frozen {
	t.Helper()
	sprint := w.cycles[lockCycle]
	p, err := domain.ProgressOf(domain.ProgressFacts{Items: w.items[lockCycle], Groups: w.groups, Start: sprint.Dates.Start,
		End: sprint.Dates.End, Timezone: "Asia/Shanghai", Today: *date(t, "2026-11-14")})
	if err != nil {
		t.Fatal(err)
	}
	if left := p.Burndown.Points; *left[0].Remaining != 4 || *left[10].Remaining != 4 || *left[11].Remaining != 3 {
		t.Fatalf("Sprint's burndown %+v: not the world's", p.Burndown)
	}
	return domain.Frozen{Progress: p, FrozenAt: w.now}
}

// A transfer takes the lock path on its cycle (Locks, cycle.write); under the lock, its target read again, then the
// clock, last; its project's states' groups; whether the cycle is frozen; its freeze, of every work item's progress
// before the move, at that reading; then the move of the work items in the backlog, unstarted and started states, by
// their ids, from Sprint to Next, by the caller at that reading. The answer is how many the issue module moved: none
// too, after the same freeze, and nothing more is written.
func TestTransferCycleIssues(t *testing.T) {
	for _, moved := range []int{3, 0} {
		w := newTransferWorld(t)
		w.moved = moved
		got, err := w.transfer(transferNext)
		if err != nil || got != moved {
			t.Fatalf("moved %d, %v; want the issue module's %d", got, err, moved)
		}
		want := []string{"GetCycle outside tx", "Facts outside tx", "Authorize cycle.write outside tx", "Enter 1 outside tx", "Begin", "Lock",
			"GetCycle", "Authorize cycle.write", "GetCycle", "Now", "StateGroups", "CycleSnapshotCounts", "CycleItems", "FreezeCycle",
			"TransferCycleIssues", "End", "Leave"}
		if !slices.Equal(w.calls, want) {
			t.Fatalf("calls\n%q\nwant\n%q", w.calls, want)
		}
		if frozen := w.snapshots[lockCycle]; frozen == nil || !reflect.DeepEqual(*frozen, w.sprintFrozen(t)) {
			t.Errorf("Sprint frozen as %+v; want every work item's progress before the move, at the clock's reading", frozen)
		}
		if len(w.viewers) != 1 || w.viewers[0] != nil || !reflect.DeepEqual(w.pages, [][]uuid.UUID{{lockCycle}}) {
			t.Errorf("viewers %v, pages %v; want Sprint's every work item, and Sprint's freeze asked for", w.viewers, w.pages)
		}
		wantMove := []Transfer{{ProjectID: lockProject, WorkspaceID: lockWorkspace, From: NamedCycle{ID: lockCycle, Name: "Sprint"},
			To: NamedCycle{ID: transferNext, Name: "Next"}, StateIDs: domain.PendingStates(w.groups), By: lockEditor, At: w.now}}
		if !reflect.DeepEqual(w.transfers, wantMove) || len(wantMove[0].StateIDs) != 3 {
			t.Errorf("moved\n%+v\nwant\n%+v", w.transfers, wantMove)
		}
	}
}

// A cycle frozen already, by the job or a transfer before, keeps its freeze: the transfer neither reads its work items
// for another nor writes one (M6 P4 spec R5, R7).
func TestTransferCycleIssuesKeepsAFreeze(t *testing.T) {
	w := newTransferWorld(t)
	w.snapshots[lockCycle] = &progressFrozen
	if got, err := w.transfer(transferNext); err != nil || got != 3 {
		t.Fatalf("moved %d, %v; want 3", got, err)
	}
	want := []string{"GetCycle outside tx", "Facts outside tx", "Authorize cycle.write outside tx", "Enter 1 outside tx", "Begin", "Lock",
		"GetCycle", "Authorize cycle.write", "GetCycle", "Now", "StateGroups", "CycleSnapshotCounts", "TransferCycleIssues", "End", "Leave"}
	if !slices.Equal(w.calls, want) {
		t.Fatalf("calls\n%q\nwant\n%q", w.calls, want)
	}
	if w.snapshots[lockCycle] != &progressFrozen || len(w.transfers) != 1 {
		t.Errorf("Sprint's freeze %+v, moves %d; want the one it had, and the move", w.snapshots[lockCycle], len(w.transfers))
	}
}

// The refusals in the order of M6 P4 spec 2.3: the lock path's, then, under the lock, an archived cycle; a target that
// is not another cycle of the project (validation_failed at target_cycle_id, no 404); an archived target; then, on the
// clock read after them, a cycle that is not completed on its project's day, and a target that is. Each case also has
// every later condition, so a refusal answered out of order fails here. Nothing is frozen or moved; the clock is read
// for the last two alone.
func TestTransferCycleIssuesRefusesInOrder(t *testing.T) {
	archived := time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)
	set := func(id uuid.UUID, change func(c *domain.Cycle)) func(*transferWorld) {
		return func(w *transferWorld) { c := w.cycles[id]; change(&c); w.cycles[id] = c }
	}
	dates := func(start, end string) func(c *domain.Cycle) {
		return func(c *domain.Cycle) { c.Dates = domain.Dates{Start: date(t, start), End: date(t, end)} }
	}
	sourceArchived := set(lockCycle, func(c *domain.Cycle) { c.ArchivedAt = &archived })
	sourceOpen := set(lockCycle, dates("2026-11-02", "2026-11-30"))
	sourceDraft := set(lockCycle, func(c *domain.Cycle) { c.Dates = domain.Dates{} })
	targetArchived := set(transferNext, func(c *domain.Cycle) { c.ArchivedAt = &archived })
	targetCompleted := set(transferNext, dates("2026-10-01", "2026-10-14"))
	// Next's last day is the 13th: completed in Shanghai, where it is the 14th; not in UTC, where it is still the 13th.
	targetEndedThere := set(transferNext, dates("2026-11-10", "2026-11-13"))
	inLosAngeles := func(w *transferWorld) {
		// The 13th there: Sprint's last day, so it is current; in UTC, the 14th.
		w.facts[lockProject] = ProjectFacts{WorkspaceID: lockWorkspace, Timezone: "America/Los_Angeles", CycleView: true}
		w.now = time.Date(2026, 11, 14, 5, 0, 0, 0, time.UTC)
	}
	targetGone := func(w *transferWorld) { delete(w.cycles, transferNext) }
	invalid := shared.Invalid(domain.TransferTargetNotAllowed())
	for _, tt := range []struct {
		name   string
		target uuid.UUID
		set    []func(*transferWorld)
		want   error
		clock  bool // read: the refusal is decided on its reading
	}{
		{"own only", transferNext, []func(*transferWorld){func(w *transferWorld) { w.grant.OwnOnly = true }, sourceArchived, targetArchived,
			sourceOpen, targetCompleted}, shared.Forbidden(), false},
		{"source archived", transferNext, []func(*transferWorld){sourceArchived, targetGone, sourceOpen}, domain.ErrArchived, false},
		{"source archived, target archived", transferNext, []func(*transferWorld){sourceArchived, targetArchived, sourceOpen, targetCompleted},
			domain.ErrArchived, false},
		{"target missing", transferNext, []func(*transferWorld){targetGone, sourceOpen}, invalid, false},
		{"target of another project", transferElsewhere, []func(*transferWorld){set(transferElsewhere, func(c *domain.Cycle) {
			c.ArchivedAt = &archived
		}), sourceOpen}, invalid, false},
		{"target is the source", lockCycle, []func(*transferWorld){sourceOpen}, invalid, false},
		{"target archived", transferNext, []func(*transferWorld){targetArchived, sourceOpen, targetCompleted}, domain.ErrArchived, false},
		{"source current", transferNext, []func(*transferWorld){sourceOpen, targetCompleted}, domain.ErrTransferFromOpenCycle, true},
		{"source current in Los Angeles", transferNext, []func(*transferWorld){inLosAngeles, targetCompleted}, domain.ErrTransferFromOpenCycle, true},
		{"source a draft", transferNext, []func(*transferWorld){sourceDraft, targetCompleted}, domain.ErrTransferFromOpenCycle, true},
		{"target completed", transferNext, []func(*transferWorld){targetCompleted}, domain.ErrTransferToCompletedCycle, true},
		{"target completed in Shanghai", transferNext, []func(*transferWorld){targetEndedThere}, domain.ErrTransferToCompletedCycle, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := newTransferWorld(t)
			for _, set := range tt.set {
				set(w)
			}
			_, err := w.transfer(tt.target)
			if !sameRefusal(err, tt.want) || !sameMessage(err, tt.want) {
				t.Fatalf("got %v; want %v", err, tt.want)
			}
			var problem *shared.Error
			if errors.As(tt.want, &problem) && problem.Code == shared.CodeValidationFailed {
				var got *shared.Error
				if !errors.As(err, &got) || !slices.Equal(got.Fields, problem.Fields) {
					t.Fatalf("got %v; want its one error at target_cycle_id", err)
				}
			}
			if slices.Contains(w.calls, "Now") != tt.clock {
				t.Errorf("calls %q: the clock read %t, want %t", w.calls, !tt.clock, tt.clock)
			}
			if len(w.snapshots) != 0 || len(w.transfers) != 0 || slices.Contains(w.calls, "CycleItems") {
				t.Errorf("calls %q: a refusal froze %v or moved %v", w.calls, w.snapshots, w.transfers)
			}
		})
	}
}

// sameMessage compares two shared errors' details: a transfer's refusals share codes with other writes' but not
// their messages (M6 P4 spec 2.3).
func sameMessage(got, want error) bool {
	var g, w *shared.Error
	if errors.As(got, &g) && errors.As(want, &w) {
		return g.Detail == w.Detail
	}
	return true
}
