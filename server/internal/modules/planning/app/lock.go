package app

import (
	"context"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CycleReader reads the undeleted cycle id, archived or not, in the
// transaction ctx carries; found is false when there is none.
type CycleReader interface {
	GetCycle(ctx context.Context, id uuid.UUID) (c domain.Cycle, found bool, err error)
}

// WriteGate is the projects' write gate (platform/gate), the one the work
// items' writes enter too: Enter returns once a write on the project may
// begin its transaction, with the function that leaves the gate, which the
// write calls once its transaction has ended. It waits holding no
// connection and no lock.
type WriteGate interface {
	Enter(ctx context.Context, projects ...uuid.UUID) (leave func(), err error)
}

// Locks is the one way a planning write takes its lock and its decision
// (M6 design 6, as M4's for the work items). Before any lock and outside the
// write's transaction, the precheck: the cycle the write names and its
// project, or the project, read without a lock, and the decision on them,
// which answers every refusal the lock would, so a refused caller takes no
// lock nor a place in the write gate. Then the gate. Then, in the write's
// transaction, the project's row FOR NO KEY UPDATE, the first lock of every
// write on the project's work items, cycles and modules, so they run one
// after another; the cycle read again under it; and the decision again,
// which is final: what committed while the write waited is seen.
type Locks struct {
	projects Projects
	cycles   CycleReader
	auth     shared.Authorizer
	gate     WriteGate
	tx       shared.TxManager
}

// NewLocks returns the locks over project's facts and lock, the cycles'
// reads, the Authorizer, the write gate and write transactions.
func NewLocks(projects Projects, cycles CycleReader, auth shared.Authorizer, gate WriteGate, tx shared.TxManager) Locks {
	return Locks{projects: projects, cycles: cycles, auth: auth, gate: gate, tx: tx}
}

// write is a planning write as Locks takes its lock: on the project project,
// or on the cycle cycle, whose project it is then on; and the action it
// decides.
type write struct {
	project uuid.UUID
	cycle   *uuid.UUID
	action  shared.Action
}

// held is what a write holds under its lock: its project's id and facts,
// locked; the cycle it names, read under the lock, zero for a write on a
// project; and the caller's grant, as decided under the lock.
type held struct {
	projectID uuid.UUID
	project   ProjectFacts
	cycle     domain.Cycle
	grant     shared.Grant
}

// run takes w's precheck, then its project's write gate, then in one
// transaction its lock and decision, and runs fn in it with what they hold:
// w's whole write. It leaves the gate once the transaction has ended, on
// every path. A refusal of the precheck, the gate or the decision is its
// answer, and fn does not run.
func (l Locks) run(ctx context.Context, actor shared.Actor, w write, fn func(ctx context.Context, h held) error) error {
	pre, err := l.decideOn(ctx, actor, w, uuid.UUID{}, l.projects.Facts)
	if err != nil {
		return err
	}
	leave, err := l.gate.Enter(ctx, pre.projectID)
	if err != nil {
		return err
	}
	defer leave()
	return l.tx.WithinTx(ctx, func(ctx context.Context) error {
		h, err := l.decideOn(ctx, actor, w, pre.projectID, l.projects.Lock)
		if err != nil {
			return err
		}
		return fn(ctx, h)
	})
}

// decideOn reads what w is on and decides w's action on it for actor (M6 P1
// spec R2), through facts: the project's facts without a lock for the
// precheck, or its lock, in the transaction ctx carries. Under the lock,
// locked is the project the precheck found, locked before the cycle is
// read again. A cycle that is not there, or no longer in that project, and
// a project that is not there or that actor does not see, is the 404 of
// what the caller named; then a role the rule does not allow, and an
// OwnOnly grant, which no planning write takes (R6), is 403; then an
// archived project, then a project whose cycles are turned off, 409.
func (l Locks) decideOn(ctx context.Context, actor shared.Actor, w write, locked uuid.UUID,
	facts func(context.Context, uuid.UUID) (ProjectFacts, bool, error)) (held, error) {
	notFound := domain.ErrProjectNotFound
	h := held{projectID: w.project}
	switch {
	case w.cycle != nil && locked == (uuid.UUID{}):
		// The precheck of a cycle write: its project is the cycle's.
		notFound = domain.ErrCycleNotFound
		c, err := l.readCycle(ctx, *w.cycle, uuid.UUID{})
		if err != nil {
			return held{}, err
		}
		h.cycle, h.projectID = c, c.ProjectID
	case w.cycle != nil:
		// Under the lock: the project first, then the cycle read again.
		notFound, h.projectID = domain.ErrCycleNotFound, locked
	}
	f, found, err := facts(ctx, h.projectID)
	switch {
	case err != nil:
		return held{}, err
	case !found:
		return held{}, notFound
	}
	h.project = f
	if w.cycle != nil && locked != (uuid.UUID{}) {
		if h.cycle, err = l.readCycle(ctx, *w.cycle, locked); err != nil {
			return held{}, err
		}
	}
	target := shared.Target{WorkspaceID: f.WorkspaceID, ProjectID: h.projectID}
	if h.cycle.CreatedByID != nil {
		target.CreatorID = *h.cycle.CreatedByID
	}
	h.grant, err = l.auth.Authorize(ctx, actor, w.action, target)
	switch {
	case errors.Is(err, shared.ErrNotVisible):
		return held{}, notFound
	case err != nil:
		return held{}, err
	case h.grant.OwnOnly:
		return held{}, shared.Forbidden()
	case f.Archived:
		return held{}, domain.ErrProjectArchived
	case !f.CycleView:
		return held{}, domain.ErrCyclesDisabled
	}
	return h, nil
}

// readCycle reads the cycle id; one that is not there, or, when project is
// not zero, not in project, is planning.cycle_not_found.
func (l Locks) readCycle(ctx context.Context, id, project uuid.UUID) (domain.Cycle, error) {
	c, found, err := l.cycles.GetCycle(ctx, id)
	switch {
	case err != nil:
		return domain.Cycle{}, err
	case !found || (project != (uuid.UUID{}) && c.ProjectID != project):
		return domain.Cycle{}, domain.ErrCycleNotFound
	}
	return c, nil
}
