package postgresadapter

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// SubIssueCounts counts the undeleted, unarchived direct sub-items of each
// of the work items ids that has any, all of them and those the account
// actor created (M4 design 3.9).
func (s *Store) SubIssueCounts(ctx context.Context, actor uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]app.SubIssueCount, error) {
	rows, err := s.queries(ctx).SubIssueCounts(ctx, gen.SubIssueCountsParams{Actor: actor, Ids: ids})
	if err != nil {
		return nil, fmt.Errorf("count the sub-items of work items %v: %w", ids, err)
	}
	counts := make(map[uuid.UUID]app.SubIssueCount, len(rows))
	for _, r := range rows {
		counts[r.ParentID] = app.SubIssueCount{All: int(r.AllCount), Own: int(r.OwnCount)}
	}
	return counts, nil
}
