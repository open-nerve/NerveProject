package app

import (
	"context"
	"fmt"
	"math"
	"slices"
	"time"
	"uuid"
)

// PurgeBatch selects one bounded keyset of rows older than Cutoff.
type PurgeBatch struct {
	Cutoff time.Time
	After  uuid.UUID
	Limit  int
}

// PurgeStep binds a table's directory entry to its static deletion query.
type PurgeStep struct {
	Table  string
	Delete func(context.Context, PurgeBatch) ([]uuid.UUID, error)
}

// Purger deletes child tables before parents, one committed batch at a time.
type Purger struct{ steps []PurgeStep }

// NewPurger retains a private copy of the ordered deletion steps.
func NewPurger(steps []PurgeStep) *Purger { return &Purger{steps: slices.Clone(steps)} }

// Tables returns a copy of the ordered table directory.
func (p *Purger) Tables() []string {
	tables := make([]string, len(p.steps))
	for i, step := range p.steps {
		tables[i] = step.Table
	}
	return tables
}

// Purge returns only directly deleted rows from successful batches. The
// adapter commits each batch independently; errors retain prior progress.
func (p *Purger) Purge(ctx context.Context, cutoff time.Time, limit int) (int, error) {
	if limit < 1 || limit > math.MaxInt32 {
		return 0, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", limit)
	}
	count := 0
	for _, step := range p.steps {
		batch := PurgeBatch{Cutoff: cutoff, Limit: limit}
		for {
			if err := ctx.Err(); err != nil {
				return count, err
			}
			ids, err := step.Delete(ctx, batch)
			if err != nil {
				return count, err
			}
			if len(ids) == 0 {
				break
			}
			count += len(ids)
			for _, id := range ids {
				if id.Compare(batch.After) > 0 {
					batch.After = id
				}
			}
		}
	}
	return count, nil
}
