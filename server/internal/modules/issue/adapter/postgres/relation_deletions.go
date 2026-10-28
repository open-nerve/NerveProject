package postgresadapter

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

func (s *Store) LockProjectRelationRows(ctx context.Context, b app.RowBatch) ([]app.RelationDeletionRow, error) {
	rows, err := s.queries(ctx).LockProjectRelationRows(ctx, gen.LockProjectRelationRowsParams{ProjectID: b.ProjectID, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return nil, err
	}
	out := make([]app.RelationDeletionRow, len(rows))
	for i, row := range rows {
		out[i] = app.RelationDeletionRow(row)
	}
	return out, nil
}

func (s *Store) LockIssueRelationRows(ctx context.Context, d app.IssueDeletion) ([]app.RelationDeletionRow, error) {
	rows, err := s.queries(ctx).LockIssueRelationRows(ctx, d.IssueIDs)
	if err != nil {
		return nil, err
	}
	out := make([]app.RelationDeletionRow, len(rows))
	for i, row := range rows {
		out[i] = app.RelationDeletionRow(row)
	}
	return out, nil
}

func (s *Store) DeleteLockedRelationRows(ctx context.Context, ids []uuid.UUID, by uuid.UUID, at time.Time) (int, error) {
	n, err := s.queries(ctx).DeleteLockedRelationRows(ctx, gen.DeleteLockedRelationRowsParams{Ids: ids, DeletedBy: by, At: at})
	return int(n), err
}
