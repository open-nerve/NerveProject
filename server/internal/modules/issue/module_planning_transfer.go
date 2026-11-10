package issue

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// PlanningTransfer is what the module does for the planning module when a
// transfer moves the unfinished work items of a cycle that has ended to
// another cycle of the project (M6 P4 spec 2.6, R2, R8), in the transfer's
// transaction, which ctx carries and which it requires, under the
// project's row lock planning holds. issue.Provide builds it over the pool;
// bootstrap converts it into planning's IssueTransfer.
type PlanningTransfer interface {
	// TransferCycleIssues moves the work items with an undeleted membership
	// of t.From that are t's project's, undeleted, not archived and in one
	// of t.StateIDs to t.To: each one's membership of t.From soft-deleted
	// and a new one of t.To inserted, never changed in place; one row of
	// activity for each, its moving from t.From to t.To; and each one last
	// changed by the transfer, all by t.By at t.At. It publishes no event
	// and answers how many it moved; none writes nothing.
	TransferCycleIssues(ctx context.Context, t CycleTransfer) (int, error)
}

// CycleTransfer is the value PlanningTransfer takes, and Named its cycles,
// each by its id and its name: bootstrap converts planning's into them.
type (
	CycleTransfer = app.CycleTransfer
	Named         = domain.Named
)
