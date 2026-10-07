package app

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CreateLabel creates a label in a project: POST
// /api/v0/projects/{project_id}/labels (M3 design 3.16).
type CreateLabel struct {
	locks  Locks
	labels LabelCreator
	tx     shared.TxManager
	clock  Clock
}

// NewCreateLabel returns the use case.
func NewCreateLabel(locks Locks, labels LabelCreator, tx shared.TxManager, clock Clock) *CreateLabel {
	return &CreateLabel{locks: locks, labels: labels, tx: tx, clock: clock}
}

// Execute checks in (domain.CheckNewLabel), then in one transaction, in the
// order of M3 design 3.6: the project's locks (its workspace FOR SHARE, the
// project FOR NO KEY UPDATE, which every write of its labels takes) and the
// decision on label.create; an archived project's labels are created as
// any other's (3.19). Then, under the locks, the parent given
// (checkParent); the sort order after the greatest of the project's
// labels (domain.SortOrderAfter; 4.10); the clock; and the label, by the
// caller at that time. The answer is the label as stored.
func (u *CreateLabel) Execute(ctx context.Context, projectID uuid.UUID, in domain.LabelCreate) (domain.Label, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Label{}, err
	}
	if err := domain.CheckNewLabel(in); err != nil {
		return domain.Label{}, err
	}
	var created domain.Label
	err = u.tx.WithinTx(ctx, func(ctx context.Context) error {
		h, err := u.locks.lockAndDecide(ctx, actor, write{project: projectID, action: domain.ActionLabelCreate})
		if err != nil {
			return err
		}
		if in.ParentID != nil {
			if err := checkParent(ctx, u.labels, uuid.Nil(), projectID, *in.ParentID, false); err != nil {
				return err
			}
		}
		greatest, err := u.labels.GreatestSortOrder(ctx, projectID)
		if err != nil {
			return err
		}
		row := LabelRow{ID: uuid.NewV7(), WorkspaceID: h.project.WorkspaceID, ProjectID: projectID, ParentID: in.ParentID, Name: in.Name,
			Color: in.Color, SortOrder: domain.SortOrderAfter(greatest), CreatedBy: actor.UserID, Now: u.clock.Now()}
		if created, err = u.labels.CreateLabel(ctx, row); err != nil {
			return err
		}
		if created.ID != row.ID {
			return fmt.Errorf("label %s created as %s", row.ID, created.ID)
		}
		return nil
	})
	if err != nil {
		return domain.Label{}, err
	}
	return created, nil
}
