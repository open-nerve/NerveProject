package app_test

import (
	"context"
	"uuid"
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
