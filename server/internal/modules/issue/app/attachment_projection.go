package app

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// projectAttachments fills an already authorized output, in the caller's
// transaction. Repeated group cells share one count; lookahead rows never enter.
func projectAttachments(ctx context.Context, counter AttachmentCounter, items []domain.Issue) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(items))
	for n, item := range items {
		ids[n] = item.ID
	}
	ids = ascending(ids)
	rows, err := counter.CountIssueAttachments(ctx, ids)
	if err != nil {
		return err
	}
	counts := make(map[uuid.UUID]int64, len(ids))
	for _, id := range ids {
		counts[id] = -1
	}
	for _, row := range rows {
		previous, asked := counts[row.IssueID]
		if !asked || previous >= 0 || row.Count < 0 {
			return fmt.Errorf("invalid attachment count projection")
		}
		counts[row.IssueID] = row.Count
	}
	for _, id := range ids {
		if counts[id] < 0 {
			return fmt.Errorf("incomplete attachment count projection")
		}
	}
	for n := range items {
		items[n].AttachmentCount = counts[items[n].ID]
	}
	return nil
}
