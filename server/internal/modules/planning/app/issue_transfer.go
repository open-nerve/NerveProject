package app

import (
	"context"
	"time"
	"uuid"
)

// IssueTransfer moves, of the issue module, the unfinished work items of a
// cycle that has ended to another cycle of its project (M6 P4 spec 2.6, R2,
// R8), in the transfer's write, which ctx carries, under the project's row
// lock: the memberships, their activity and the work items are the issue
// module's, written in the same transaction as the cycle's freeze, so all
// commit or roll back as one. The module rules on the cycles and on which
// states move (R7); the issue module only moves. issue.Provide's
// PlanningTransfer, converted (bootstrap).
type IssueTransfer interface {
	// TransferCycleIssues moves the work items with an undeleted membership
	// of t.From that are t's project's, undeleted, not archived and in one
	// of t.StateIDs to t.To, each with one row of activity, its moving from
	// t.From to t.To by their names and ids, and each last changed by t.By
	// at t.At; it publishes no event (R8). It answers how many it moved;
	// none writes nothing.
	TransferCycleIssues(ctx context.Context, t Transfer) (int, error)
}

// Transfer is a transfer's move of work items as IssueTransfer takes it:
// the project's and its workspace's ids, the cycle they leave and the
// cycle they join, the states, by id, of the work items that move (those
// of the backlog, unstarted and started groups, read under the lock), and
// the account that transfers them at the write's moment.
type Transfer struct {
	ProjectID, WorkspaceID uuid.UUID
	From, To               NamedCycle
	StateIDs               []uuid.UUID
	By                     uuid.UUID
	At                     time.Time
}

// NamedCycle is a cycle as a transfer's activity names it: its id and its
// name.
type NamedCycle struct {
	ID   uuid.UUID
	Name string
}
