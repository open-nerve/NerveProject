package app

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// UpdateState changes a state: PATCH /api/v0/states/{state_id} (M3 design
// 3.17, 6.7).
type UpdateState struct {
	locks  Locks
	states StateUpdater
	tx     shared.TxManager
	clock  Clock
}

// NewUpdateState returns the use case.
func NewUpdateState(locks Locks, states StateUpdater, tx shared.TxManager, clock Clock) *UpdateState {
	return &UpdateState{locks: locks, states: states, tx: tx, clock: clock}
}

// Execute checks p (domain.CheckStatePatch), then in one transaction, in
// the order of M3 design 3.6 and 6.7: the state's locks (lockRowAndDecide:
// the state read for its project and workspace, the workspace FOR SHARE,
// the project FOR NO KEY UPDATE, the state read again) and the decision on
// state.update; an archived project's states change as any other's
// (3.19). Then, when p moves the state to another group, the states its
// group keeps without it (domain.CheckGroupKept); then the change, at the
// time the clock gives under the locks. The answer is the state as stored.
func (u *UpdateState) Execute(ctx context.Context, id uuid.UUID, p domain.StatePatch) (domain.State, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.State{}, err
	}
	if err := domain.CheckStatePatch(p); err != nil {
		return domain.State{}, err
	}
	var updated domain.State
	err = u.tx.WithinTx(ctx, func(ctx context.Context) error {
		_, s, err := lockRowAndDecide(ctx, u.locks, actor, rowWrite[domain.State]{id: id, action: domain.ActionStateUpdate,
			find: u.states.StateByID, notFound: domain.ErrStateNotFound})
		if err != nil {
			return err
		}
		if p.Group != nil && *p.Group != s.Group {
			n, err := u.states.CountGroupStates(ctx, s.ProjectID, s.Group)
			if err != nil {
				return err
			}
			if err := domain.CheckGroupKept(n - 1); err != nil {
				return err
			}
		}
		if updated, err = u.states.UpdateState(ctx, s.ID, p, actor.UserID, u.clock.Now()); err != nil {
			return err
		}
		if updated.ID != s.ID {
			return fmt.Errorf("state %s changed as %s", s.ID, updated.ID)
		}
		return nil
	})
	if err != nil {
		return domain.State{}, err
	}
	return updated, nil
}
