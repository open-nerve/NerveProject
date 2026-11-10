package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// CycleTransfer is planning's transfer of the unfinished work items of a
// cycle that has ended to another cycle of its project, as the module
// writes it (M6 P4 spec 2.6, R8): the project's and its workspace's ids;
// the cycle the work items leave and the cycle they join, each by its id
// and its name, which their activity names; the states, by id, of the work
// items that move, which planning reads the groups of; and the account
// that transfers them, at the transfer's moment.
type CycleTransfer struct {
	ProjectID, WorkspaceID uuid.UUID
	From, To               domain.Named
	StateIDs               []uuid.UUID
	By                     uuid.UUID
	At                     time.Time
}

// CycleJoins are the new rows of the work items a transfer moves into the
// cycle CycleID of the project ProjectID of the workspace WorkspaceID (M6
// P4 spec 2.6): one of each of IssueIDs, whose id is the one of IDs at its
// place, created and last changed by By at At.
type CycleJoins struct {
	IDs, IssueIDs                   []uuid.UUID
	CycleID, WorkspaceID, ProjectID uuid.UUID
	By                              uuid.UUID
	At                              time.Time
}

// CycleTransferRows are the writes of a transfer (M6 P4 spec 2.6, R8), in
// its transaction, which ctx carries and each requires, under its
// project's row lock, in the lock order of the spec: the memberships, then
// the activity, then the work items.
type CycleTransferRows interface {
	// RemoveTransferredCycleIssues soft-deletes the undeleted memberships
	// of t.From whose work items are t's project's, undeleted, not archived
	// and in one of t.StateIDs, locked in id order, each deleted and last
	// changed by t.By at t.At, and answers those work items in id order.
	RemoveTransferredCycleIssues(ctx context.Context, t CycleTransfer) ([]uuid.UUID, error)
	// InsertTransferredCycleIssues inserts the rows j of the cycle the work
	// items join.
	InsertTransferredCycleIssues(ctx context.Context, j CycleJoins) error
	InsertActivities(ctx context.Context, rows []ActivityRow) error
	MarkChanged(ctx context.Context, r ChangedRows) error
}

// CycleTransfers move, for the planning module, the unfinished work items
// of a cycle that has ended to another cycle of its project (M6 P4 spec
// 2.6, R8), in the transfer's transaction, which ctx carries, under the
// project's row lock planning holds; planning rules on the cycles and on
// which states move (R7).
type CycleTransfers struct {
	rows CycleTransferRows
}

// NewCycleTransfers returns the transfers over the memberships' store.
func NewCycleTransfers(rows CycleTransferRows) *CycleTransfers {
	return &CycleTransfers{rows: rows}
}

// TransferCycleIssues moves the work items of t and answers how many it
// moved: their memberships of t.From removed, never changed in place (P3a
// spec R4); a new one of t.To for each, in the order the removal answers
// them, each with a fresh id; one row of activity for each, its moving
// from t.From to t.To (domain.CycleMoved); then those work items last
// changed by the transfer (MarkChanged), all by t.By at t.At. It publishes
// no issue.updated (R8; precedent NewMembershipRemovals). None to move
// writes nothing past the removal and answers 0.
func (c *CycleTransfers) TransferCycleIssues(ctx context.Context, t CycleTransfer) (int, error) {
	issues, err := c.rows.RemoveTransferredCycleIssues(ctx, t)
	if err != nil || len(issues) == 0 {
		return 0, err
	}
	joins := CycleJoins{IDs: make([]uuid.UUID, len(issues)), IssueIDs: issues, CycleID: t.To.ID, WorkspaceID: t.WorkspaceID, ProjectID: t.ProjectID,
		By: t.By, At: t.At}
	for i := range joins.IDs {
		joins.IDs[i] = uuid.NewV7()
	}
	if err := c.rows.InsertTransferredCycleIssues(ctx, joins); err != nil {
		return 0, err
	}
	by, moved := t.By, domain.CycleMoved(t.From, t.To)
	activities := make([]ActivityRow, len(issues))
	for i, issue := range issues {
		activities[i] = ActivityRow{ID: uuid.NewV7(), WorkspaceID: t.WorkspaceID, ProjectID: t.ProjectID, IssueID: issue, Activity: moved,
			ActorID: &by, At: t.At}
	}
	if err := c.rows.InsertActivities(ctx, activities); err != nil {
		return 0, err
	}
	if err := c.rows.MarkChanged(ctx, ChangedRows{IDs: issues, By: t.By, At: t.At}); err != nil {
		return 0, err
	}
	return len(issues), nil
}
