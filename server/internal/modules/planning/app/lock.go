package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CycleReader reads the undeleted cycle id, archived or not, in the
// transaction ctx carries; found is false when there is none.
type CycleReader interface {
	GetCycle(ctx context.Context, id uuid.UUID) (c domain.Cycle, found bool, err error)
}

// ModuleReader reads the undeleted module id, archived or not, with its
// undeleted members' accounts in id order, in the transaction ctx carries;
// found is false when there is none.
type ModuleReader interface {
	GetModule(ctx context.Context, id uuid.UUID) (m domain.Module, found bool, err error)
}

// ModuleLinkReader reads the undeleted module link id, in the transaction
// ctx carries; found is false when there is none. Its module is read apart
// (ModuleReader).
type ModuleLinkReader interface {
	GetModuleLink(ctx context.Context, id uuid.UUID) (l domain.ModuleLink, found bool, err error)
}

// NamedReader reads what a planning call names below its project (M6 P2
// spec R1): a cycle, a module, a module link.
type NamedReader interface {
	CycleReader
	ModuleReader
	ModuleLinkReader
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
// after another; what the write names read again under it — a cycle, a
// module, a module link and its module (M6 P2 spec R1); and the decision
// again, which is final: what committed while the write waited is seen. A
// change of one's own display settings on a page takes its own, lighter way
// (share).
type Locks struct {
	projects Projects
	objects  NamedReader
	auth     shared.Authorizer
	gate     WriteGate
	tx       shared.TxManager
}

// NewLocks returns the locks over project's facts and lock, the reads of
// what a write names, the Authorizer, the write gate and write
// transactions.
func NewLocks(projects Projects, objects NamedReader, auth shared.Authorizer, gate WriteGate, tx shared.TxManager) Locks {
	return Locks{projects: projects, objects: objects, auth: auth, gate: gate, tx: tx}
}

// write is a planning write as Locks takes its lock: what it names, the
// project, or a cycle, a module or a module link, whose project it is then on
// (named); and the action it decides.
type write struct {
	named
	action shared.Action
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

// share takes, in one transaction, the lock of a change of the caller's own
// display settings on a page (M6 P3b spec R6), which is no planning write,
// and runs fn in it with what it holds. What n names and its project's
// facts, read without a lock, and the decision on action (see), which
// answers a refusal before any lock; then the project's row FOR SHARE
// (Projects.Share), what n names read again under it, and the decision
// again, which is final. No write gate, and none of a write's refusals
// (decideOn): not an OwnOnly grant, an archived project nor a toggle turned
// off. The share waits for the FOR NO KEY UPDATE of a cycle's, a module's or
// the project's deletion, and they for it, so no row is written under a page
// being deleted.
func (l Locks) share(ctx context.Context, actor shared.Actor, action shared.Action, n named, fn func(ctx context.Context, h held) error) error {
	return l.tx.WithinTx(ctx, func(ctx context.Context) error {
		pre, err := see(ctx, l.objects, l.auth, actor, action, n, uuid.UUID{}, l.projects.Facts)
		if err != nil {
			return err
		}
		h, err := see(ctx, l.objects, l.auth, actor, action, n, pre.projectID, l.projects.Share)
		if err != nil {
			return err
		}
		return fn(ctx, h)
	})
}

// decideOn reads what w is on and decides w's action on it for actor (M6 P1
// spec R2, P2 spec R2), through facts: the project's facts without a lock for
// the precheck, or its lock, in the transaction ctx carries (see). Then what
// a write is refused and a read is not: an OwnOnly grant, which no planning
// write takes (R6), is 403; then an archived project, 409; then the toggle of
// what the action writes turned off (R7, turnedOff), 409.
func (l Locks) decideOn(ctx context.Context, actor shared.Actor, w write, locked uuid.UUID,
	facts func(context.Context, uuid.UUID) (ProjectFacts, bool, error)) (held, error) {
	h, err := see(ctx, l.objects, l.auth, actor, w.action, w.named, locked, facts)
	switch {
	case err != nil:
		return held{}, err
	case h.grant.OwnOnly:
		return held{}, shared.Forbidden()
	case h.project.Archived:
		return held{}, domain.ErrProjectArchived
	}
	if err := turnedOff(w.action, h.project); err != nil {
		return held{}, err
	}
	return h, nil
}

// turnedOff is the refusal of a write of action on a project of facts f
// whose toggle of what action writes is off (M6 design 3.8, P2 spec R1): a
// module action's, planning.modules_disabled when its modules are off; any
// other's, planning.cycles_disabled when its cycles are. nil when it is on.
func turnedOff(action shared.Action, f ProjectFacts) error {
	switch {
	case domain.ModuleAction(action):
		if !f.ModuleView {
			return domain.ErrModulesDisabled
		}
	case !f.CycleView:
		return domain.ErrCyclesDisabled
	}
	return nil
}
