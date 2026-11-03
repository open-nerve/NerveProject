package postgresadapter

import (
	"context"
	"time"
	"uuid"
)

// These operations are bound only to this owner's generated queries.
type protectedPurgeQueries struct {
	candidates func(context.Context, time.Time, uuid.UUID, int) ([]uuid.UUID, error)
	remove     func(context.Context, time.Time, []uuid.UUID) ([]uuid.UUID, error)
}

func (q protectedPurgeQueries) Candidates(ctx context.Context, cutoff time.Time, after uuid.UUID, limit int) ([]uuid.UUID, error) {
	return q.candidates(ctx, cutoff, after, limit)
}
func (q protectedPurgeQueries) Delete(ctx context.Context, cutoff time.Time, ids []uuid.UUID) ([]uuid.UUID, error) {
	return q.remove(ctx, cutoff, ids)
}
