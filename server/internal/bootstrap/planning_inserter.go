package bootstrap

import (
	"context"
	"errors"

	"github.com/riverqueue/river"

	planningriver "github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/river"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// planningInserter bridges the platform transaction context and River for the
// planning module's jobs, inserted only in the publishing transaction.
type planningInserter struct{ client *jobs.Inserter }

var _ planningriver.Inserter = planningInserter{}

func (i planningInserter) InsertTx(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) error {
	tx, ok := postgres.Tx(ctx)
	if !ok {
		return errors.New("planning job insert requires a transaction")
	}
	return i.client.InsertTx(ctx, tx, args, opts)
}
