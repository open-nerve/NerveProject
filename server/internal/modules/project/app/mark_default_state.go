package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// MarkDefaultState makes a state its project's default: POST
// /api/v0/states/{state_id}/mark-default (M3 design 3.17).
type MarkDefaultState struct {
	locks  Locks
	states DefaultMarker
	tx     shared.TxManager
	clock  Clock
}

// NewMarkDefaultState returns the use case.
func NewMarkDefaultState(locks Locks, states DefaultMarker, tx shared.TxManager, clock Clock) *MarkDefaultState {
	return &MarkDefaultState{locks: locks, states: states, tx: tx, clock: clock}
}

// Execute, in one transaction and in the order of M3 design 3.6: the
// state's locks (lockRowAndDecide) and the decision on state.mark_default;
// an archived project's default changes as any other's (3.19). Then, at
// the time the clock gives under the locks, the two statements of 3.17:
// the project's default the default no longer, then the state the default.
// When the second writes no row the state is project.state_not_found, and
// the transaction rolls the first back: no project is left without a
// default.
func (u *MarkDefaultState) Execute(ctx context.Context, id uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	return u.tx.WithinTx(ctx, func(ctx context.Context) error {
		_, s, err := lockRowAndDecide(ctx, u.locks, actor, stateWrite(id, domain.ActionStateMarkDefault, u.states))
		if err != nil {
			return err
		}
		marked, err := u.states.MarkDefaultState(ctx, s.ProjectID, s.ID, actor.UserID, u.clock.Now())
		switch {
		case err != nil:
			return err
		case !marked:
			return domain.ErrStateNotFound
		}
		return nil
	})
}
