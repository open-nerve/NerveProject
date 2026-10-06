package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// DeleteLabel deletes a label and the labels under it: DELETE
// /api/v0/labels/{label_id} (M3 design 3.16).
type DeleteLabel struct {
	locks  Locks
	labels LabelDeleter
	tx     shared.TxManager
	clock  Clock
}

// NewDeleteLabel returns the use case.
func NewDeleteLabel(locks Locks, labels LabelDeleter, tx shared.TxManager, clock Clock) *DeleteLabel {
	return &DeleteLabel{locks: locks, labels: labels, tx: tx, clock: clock}
}

// Execute, in one transaction, in the order of M3 design 3.6: the label's
// locks (lockRowAndDecide: the label read for its project and workspace,
// the workspace FOR SHARE, the project FOR NO KEY UPDATE, which every
// write of its labels takes, the label read again) and the decision on
// label.delete; an archived project's labels are deleted as any other's
// (3.19). Then the label and the labels under it, in one statement, by the
// caller at the time the clock gives under the locks.
func (u *DeleteLabel) Execute(ctx context.Context, id uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	return u.tx.WithinTx(ctx, func(ctx context.Context) error {
		_, l, err := lockRowAndDecide(ctx, u.locks, actor, labelWrite(id, domain.ActionLabelDelete, u.labels))
		if err != nil {
			return err
		}
		return u.labels.DeleteLabel(ctx, l.ID, actor.UserID, u.clock.Now())
	})
}
