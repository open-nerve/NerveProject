package postgresadapter

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// SubIssueCounts counts the undeleted, unarchived direct sub-items of each
// of the work items ids that has any, all of them and those the account
// actor created (M4 design 3.9).
func (s *Store) SubIssueCounts(ctx context.Context, actor uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]app.SubIssueCount, error) {
	rows, err := s.queries(ctx).SubIssueCounts(ctx, gen.SubIssueCountsParams{Actor: actor, Ids: ids})
	if err != nil {
		return nil, fmt.Errorf("count the sub-items of %d work items: %w", len(ids), err)
	}
	counts := make(map[uuid.UUID]app.SubIssueCount, len(rows))
	for _, r := range rows {
		counts[r.ParentID] = app.SubIssueCount{All: int(r.AllCount), Own: int(r.OwnCount)}
	}
	return counts, nil
}

// SubIssues reads the undeleted, unarchived work items directly under the
// work item parent, those the account by created when by is not nil, in
// the order of their creation (M4 design 3.9).
func (s *Store) SubIssues(ctx context.Context, parent uuid.UUID, by *uuid.UUID) ([]domain.Issue, error) {
	rows, err := s.queries(ctx).SubIssues(ctx, gen.SubIssuesParams{ParentID: parent, CreatedBy: by})
	if err != nil {
		return nil, fmt.Errorf("read the sub-items of work item %s: %w", parent, err)
	}
	issues := make([]domain.Issue, len(rows))
	for n, r := range rows {
		issues[n] = issueOf(gen.GetIssueRow(r))
	}
	return issues, nil
}

// SetParents puts the work items r names under its parent, each last
// changed at its moment and by its account (M4 design 3.9). The use case
// read each of them undeleted under the project's lock, so one not written
// is an error.
func (s *Store) SetParents(ctx context.Context, r app.ParentRows) error {
	n, err := s.queries(ctx).SetParents(ctx, gen.SetParentsParams{ParentID: r.ParentID, By: r.By, At: r.At, Ids: r.IDs})
	switch {
	case err != nil:
		return fmt.Errorf("put %d work items under %s: %w", len(r.IDs), r.ParentID, err)
	case n != int64(len(r.IDs)):
		return fmt.Errorf("put %d of %d work items under %s", n, len(r.IDs), r.ParentID)
	}
	return nil
}
