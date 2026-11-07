package riveradapter

import (
	"context"
	"time"
	"uuid"
)

// RowDeletions adapts deletion events to transactional River inserts
// (app.RowDeletions).
type RowDeletions struct{ inserter Inserter }

func NewRowDeletions(i Inserter) *RowDeletions { return &RowDeletions{inserter: i} }

func (r *RowDeletions) Enqueue(ctx context.Context, project uuid.UUID, at time.Time, by uuid.UUID) error {
	args := DeleteRowsArgs{ProjectID: project, At: at, By: by}
	opts := args.InsertOpts()
	return r.inserter.InsertTx(ctx, args, &opts)
}
