package app

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CreateState creates a state in a project: POST
// /api/v0/projects/{project_id}/states (M3 design 3.17).
type CreateState struct {
	locks  Locks
	states StateCreator
	tx     shared.TxManager
	clock  Clock
}

// NewCreateState returns the use case.
func NewCreateState(locks Locks, states StateCreator, tx shared.TxManager, clock Clock) *CreateState {
	return &CreateState{locks: locks, states: states, tx: tx, clock: clock}
}

// Execute checks in (domain.CheckNewState), then in one transaction, in the
// order of M3 design 3.6: the project's locks (its workspace FOR SHARE, the
// project FOR NO KEY UPDATE, which every write of its states takes) and the
// decision on state.create; an archived project's states are created as
// any other's (3.19). Then the greatest sequence of the project's states,
// the clock under the locks, and the state, after them all
// (domain.SequenceAfter), never the default, by the caller at that time.
// The answer is the state as stored.
func (u *CreateState) Execute(ctx context.Context, projectID uuid.UUID, in domain.StateCreate) (domain.State, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.State{}, err
	}
	if err := domain.CheckNewState(in); err != nil {
		return domain.State{}, err
	}
	var created domain.State
	err = u.tx.WithinTx(ctx, func(ctx context.Context) error {
		h, err := u.locks.lockAndDecide(ctx, actor, write{project: projectID, action: domain.ActionStateCreate})
		if err != nil {
			return err
		}
		greatest, err := u.states.GreatestSequence(ctx, projectID)
		if err != nil {
			return err
		}
		row := StateRow{ID: uuid.NewV7(), WorkspaceID: h.project.WorkspaceID, ProjectID: projectID, CreatedBy: actor.UserID, Now: u.clock.Now(),
			State: domain.NewState{Name: in.Name, Color: in.Color, Sequence: domain.SequenceAfter(greatest), Group: in.Group,
				Description: in.Description}}
		if created, err = u.states.CreateState(ctx, row); err != nil {
			return err
		}
		if created.ID != row.ID {
			return fmt.Errorf("state %s created as %s", row.ID, created.ID)
		}
		return nil
	})
	if err != nil {
		return domain.State{}, err
	}
	return created, nil
}
