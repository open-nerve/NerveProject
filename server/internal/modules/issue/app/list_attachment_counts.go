package app

import (
	"context"
	"fmt"
	"slices"
	"uuid"
)

// Read the complete nonzero directory in this list's snapshot before SQL can
// rank any issue. A directory page is not a public issue page: all its entries
// must survive, including hidden or filtered rows that SQL will exclude.
func (l *Listing) attachmentCounts(ctx context.Context, projects []VisibleProject) ([]AttachmentCount, error) {
	ids := make([]uuid.UUID, len(projects))
	for n, project := range projects {
		ids[n] = project.ID
	}
	slices.SortFunc(ids, func(a, b uuid.UUID) int { return a.Compare(b) })
	ids = slices.Compact(ids)
	const size = 1000
	var after uuid.UUID
	var counts []AttachmentCount
	for {
		page, err := l.deps.AttachmentCatalog.ProjectAttachmentCounts(ctx, ids, after, size)
		if err != nil {
			return nil, err
		}
		if len(page) > size {
			return nil, fmt.Errorf("attachment count directory exceeds page size")
		}
		for _, row := range page {
			if row.IssueID.Compare(after) <= 0 || row.Count <= 0 {
				return nil, fmt.Errorf("invalid attachment count directory")
			}
			after = row.IssueID
			counts = append(counts, row)
		}
		if len(page) < size {
			return counts, nil
		}
	}
}
