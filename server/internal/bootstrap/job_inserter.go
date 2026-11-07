package bootstrap

import (
	"context"
	"errors"

	"github.com/riverqueue/river"

	assetriver "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/river"
	issueriver "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/river"
	planningriver "github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/river"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// This is a programming error, not a user-facing application error.
var errInserterRequiresTransaction = errors.New("job insert requires a transaction")

// jobInserter bridges the platform transaction context and River only here,
// keeping those platform packages independent of each other. It is every
// module's Inserter: a job is inserted only in the transaction of the write
// that publishes it.
type jobInserter struct{ client *jobs.Inserter }

var (
	_ issueriver.Inserter    = jobInserter{}
	_ planningriver.Inserter = jobInserter{}
	_ assetriver.Inserter    = jobInserter{}
)

func (i jobInserter) InsertTx(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) error {
	tx, ok := postgres.Tx(ctx)
	if !ok {
		return errInserterRequiresTransaction
	}
	return i.client.InsertTx(ctx, tx, args, opts)
}
