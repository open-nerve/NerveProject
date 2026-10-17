package postgresadapter

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// The batches of a deleted project's work-item side (app.ProjectRowsDeleter,
// delete_rows.sql): each soft-deletes the batch's rows of one table and
// returns their ids.

// batch is b as the statement takes it.
func batch(b app.RowBatch) gen.DeleteProjectIssueRowsParams {
	return gen.DeleteProjectIssueRowsParams{At: b.At, DeletedBy: b.By, ProjectID: b.ProjectID, After: b.After, Batch: int32(b.Limit)}
}

// deleted returns the batch's ids, or its failure named by what.
func deleted(ids []uuid.UUID, err error, what string) ([]uuid.UUID, error) {
	if err != nil {
		return nil, fmt.Errorf("delete a batch of the project's %s: %w", what, err)
	}
	return ids, nil
}

// DeleteProjectIssueRows soft-deletes a batch of the project's work items,
// archived ones too.
func (s *Store) DeleteProjectIssueRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).DeleteProjectIssueRows(ctx, batch(b))
	return deleted(ids, err, "work items")
}
