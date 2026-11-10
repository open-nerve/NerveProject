package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// RemoveTransferredCycleIssues soft-deletes, of the cycle a transfer's
// work items leave, the undeleted memberships whose work items are the
// transfer's project's, undeleted, not archived and in one of its states,
// locked in id order, each deleted and last changed by the transfer's
// account at its moment, and answers those work items in id order (M6 P4
// spec 2.6, R8). It runs in the transfer's transaction, which ctx must
// carry: the memberships' move and planning's write commit or roll back as
// one (precedent RemoveCycleMemberships).
func (s *Store) RemoveTransferredCycleIssues(ctx context.Context, t app.CycleTransfer) ([]uuid.UUID, error) {
	if _, ok := postgres.Tx(ctx); !ok {
		return nil, errors.New("a transfer's memberships require its transaction")
	}
	issues, err := s.queries(ctx).RemoveTransferredCycleIssues(ctx, gen.RemoveTransferredCycleIssuesParams{At: t.At, By: t.By, CycleID: t.From.ID,
		ProjectID: t.ProjectID, StateIds: t.StateIDs})
	if err != nil {
		return nil, fmt.Errorf("remove the transferred memberships of cycle %s: %w", t.From.ID, err)
	}
	return issues, nil
}

// InsertTransferredCycleIssues inserts the rows j of the cycle a transfer's
// work items join (M6 P4 spec 2.6), in the transfer's transaction, which
// ctx must carry, after RemoveTransferredCycleIssues. A work item is in one
// cycle at most: a second undeleted row breaks the partial unique key on
// issue_id, a bug the removal before rules out, so an error.
func (s *Store) InsertTransferredCycleIssues(ctx context.Context, j app.CycleJoins) error {
	if _, ok := postgres.Tx(ctx); !ok {
		return errors.New("a transfer's memberships require its transaction")
	}
	err := s.queries(ctx).InsertTransferredCycleIssues(ctx, gen.InsertTransferredCycleIssuesParams{Ids: j.IDs, WorkspaceID: j.WorkspaceID,
		ProjectID: j.ProjectID, IssueIds: j.IssueIDs, CycleID: j.CycleID, By: &j.By, At: j.At})
	if err != nil {
		return fmt.Errorf("insert the transferred memberships of cycle %s: %w", j.CycleID, err)
	}
	return nil
}
