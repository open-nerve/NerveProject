package app_test

import (
	"context"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// Ancestry logs the read and answers the work item id's ancestry as the
// fake is told it, or the work item alone (fakeIssues' ancestry).
func (f *fakeIssues) Ancestry(ctx context.Context, id uuid.UUID) ([]uuid.UUID, error) {
	if err := f.log.add(ctx, "Ancestry", id); err != nil {
		return nil, err
	}
	if a, ok := f.ancestry[id]; ok {
		return a, nil
	}
	return []uuid.UUID{id}, nil
}

// SubIssueCounts logs the count and answers the numbers of sub-items the
// fake is told of the work items ids, and none of the others (fakeIssues'
// counts).
func (f *fakeIssues) SubIssueCounts(ctx context.Context, actor uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]app.SubIssueCount, error) {
	if err := f.log.add(ctx, "SubIssueCounts", actor, ids); err != nil {
		return nil, err
	}
	counts := map[uuid.UUID]app.SubIssueCount{}
	for _, id := range ids {
		if c, ok := f.counts[id]; ok {
			counts[id] = c
		}
	}
	return counts, nil
}

// GetIssues logs the read and answers the work items among ids that the
// fake has (read), in id order.
func (f *fakeIssues) GetIssues(ctx context.Context, ids []uuid.UUID) ([]domain.Issue, error) {
	if err := f.log.add(ctx, "GetIssues", ids); err != nil {
		return nil, err
	}
	var out []domain.Issue
	for _, id := range slices.SortedFunc(slices.Values(ids), uuid.UUID.Compare) {
		if i, found := f.read(id); found {
			out = append(out, i)
		}
	}
	return out, nil
}
