package app

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// DeleteState deletes a state: DELETE /api/v0/states/{state_id} (M3
// design 3.17).
type DeleteState struct {
	locks  Locks
	states StateDeleter
	tx     shared.TxManager
	clock  Clock
}

// NewDeleteState returns the use case.
func NewDeleteState(locks Locks, states StateDeleter, tx shared.TxManager, clock Clock) *DeleteState {
	return &DeleteState{locks: locks, states: states, tx: tx, clock: clock}
}

// Execute, in one transaction and in the order of M3 design 3.6: the
// state's locks (lockRowAndDecide: the state read for its project and
// workspace, the workspace FOR SHARE, the project FOR NO KEY UPDATE, the
// state read again) and the decision on state.delete; an archived
// project's states are deleted as any other's (3.19). Then, at the time the
// clock gives under the locks, the guarded deletion (3.17), which passes
// over the project's default state: the state read again answers why
// (deletedNothing). Then the states its group keeps without it
// (domain.CheckGroupKept): a refusal rolls the deletion back.
func (u *DeleteState) Execute(ctx context.Context, id uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	return u.tx.WithinTx(ctx, func(ctx context.Context) error {
		rw := stateWrite(id, domain.ActionStateDelete, u.states)
		_, s, err := lockRowAndDecide(ctx, u.locks, actor, rw)
		if err != nil {
			return err
		}
		deleted, err := u.states.DeleteState(ctx, s.ID, actor.UserID, u.clock.Now())
		switch {
		case err != nil:
			return err
		case !deleted:
			return deletedNothing(ctx, rw)
		}
		left, err := u.states.CountGroupStates(ctx, s.ProjectID, s.Group)
		if err != nil {
			return err
		}
		return domain.CheckGroupKept(left)
	})
}

// deletedNothing answers a guarded deletion of the state rw names that
// deleted nothing, from the state read again as the lock path reads it
// (rowWrite.read): project.state_default for the default,
// project.state_not_found for a state no longer there. A state there that
// is not the default is an error: the guard had no reason to pass it over.
func deletedNothing(ctx context.Context, rw rowWrite[domain.State]) error {
	s, err := rw.read(ctx)
	switch {
	case err != nil:
		return err
	case s.Default:
		return domain.ErrStateDefault
	}
	return fmt.Errorf("state %s neither deleted nor the default", s.ID)
}
