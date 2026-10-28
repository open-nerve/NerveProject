package app

import (
	"context"
	"time"
	"uuid"
)

// StateLabelUses are the work items' uses of a project's states and
// labels, as the project module's deletions of them ask about them (M4
// design 3.6), each in the transaction ctx carries.
type StateLabelUses interface {
	// StateInUse reports whether an undeleted work item, archived or not,
	// is in the state.
	StateInUse(ctx context.Context, stateID uuid.UUID) (bool, error)
	// DetachLabels soft-deletes the undeleted rows that put the labels on
	// work items, at now by by.
	DetachLabels(ctx context.Context, labelIDs []uuid.UUID, by uuid.UUID, now time.Time) error
}

// Cascade supplies the project module's state and label operations.
type Cascade struct{ uses StateLabelUses }

// NewCascade returns the state and label boundary.
func NewCascade(uses StateLabelUses) *Cascade { return &Cascade{uses: uses} }

// StateInUse reports whether an undeleted work item, archived or not, is
// in the state: deleteState's check.
func (c *Cascade) StateInUse(ctx context.Context, stateID uuid.UUID) (bool, error) {
	return c.uses.StateInUse(ctx, stateID)
}

// DetachLabel soft-deletes the rows that put the labels on work items, at
// now by by: deleteLabel's last step.
func (c *Cascade) DetachLabel(ctx context.Context, labelIDs []uuid.UUID, by uuid.UUID, now time.Time) error {
	return c.uses.DetachLabels(ctx, labelIDs, by, now)
}
