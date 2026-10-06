package app

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// UpdateLabel changes a label: PATCH /api/v0/labels/{label_id} (M3 design
// 3.16).
type UpdateLabel struct {
	locks  Locks
	labels LabelUpdater
	tx     shared.TxManager
	clock  Clock
}

// NewUpdateLabel returns the use case.
func NewUpdateLabel(locks Locks, labels LabelUpdater, tx shared.TxManager, clock Clock) *UpdateLabel {
	return &UpdateLabel{locks: locks, labels: labels, tx: tx, clock: clock}
}

// Execute checks p (domain.CheckLabelPatch), then in one transaction, in
// the order of M3 design 3.6: the label's locks (lockRowAndDecide: the
// label read for its project and workspace, the workspace FOR SHARE, the
// project FOR NO KEY UPDATE, which every write of its labels takes, the
// label read again) and the decision on label.update; an archived
// project's labels change as any other's (3.19). Then, when p gives a
// parent, whether the label has labels under it and the parent
// (checkParent); a parent given as none, to the top, needs neither. Then
// the change, at the time the clock gives under the locks. The answer is
// the label as stored.
func (u *UpdateLabel) Execute(ctx context.Context, id uuid.UUID, p domain.LabelPatch) (domain.Label, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Label{}, err
	}
	if err := domain.CheckLabelPatch(p); err != nil {
		return domain.Label{}, err
	}
	var updated domain.Label
	err = u.tx.WithinTx(ctx, func(ctx context.Context) error {
		_, l, err := lockRowAndDecide(ctx, u.locks, actor, labelWrite(id, domain.ActionLabelUpdate, u.labels))
		if err != nil {
			return err
		}
		if p.SetParent && p.ParentID != nil {
			hasChildren, err := u.labels.HasChildren(ctx, l.ID)
			if err != nil {
				return err
			}
			if err := checkParent(ctx, u.labels, l.ID, l.ProjectID, *p.ParentID, hasChildren); err != nil {
				return err
			}
		}
		if updated, err = u.labels.UpdateLabel(ctx, l.ID, p, actor.UserID, u.clock.Now()); err != nil {
			return err
		}
		if updated.ID != l.ID {
			return fmt.Errorf("label %s changed as %s", l.ID, updated.ID)
		}
		return nil
	})
	if err != nil {
		return domain.Label{}, err
	}
	return updated, nil
}
