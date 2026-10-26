package jobs

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

// Inserter only inserts jobs in the caller's transaction. It does not own
// the pool or any worker lifecycle.
type Inserter struct{ client *river.Client[pgx.Tx] }

// NewInserter creates an insert-only client: no workers, queues or Start.
func NewInserter(pool *pgxpool.Pool) (*Inserter, error) {
	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{})
	if err != nil {
		return nil, fmt.Errorf("create the jobs inserter: %w", err)
	}
	return &Inserter{client: client}, nil
}

// InsertTx inserts a job in tx; the caller commits or rolls it back.
func (i *Inserter) InsertTx(ctx context.Context, tx pgx.Tx, args river.JobArgs, opts *river.InsertOpts) error {
	_, err := i.client.InsertTx(ctx, tx, args, opts)
	return err
}
