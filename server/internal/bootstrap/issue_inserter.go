package bootstrap

import (
	"context"
	"errors"

	"github.com/riverqueue/river"

	issueriver "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/river"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// This is a programming error, not a user-facing application error.
var errInserterRequiresTransaction = errors.New("issue job insert requires a transaction")

// issueInserter bridges the platform transaction context and River only
// here, keeping those platform packages independent of each other.
type issueInserter struct{ client *jobs.Inserter }

var _ issueriver.Inserter = issueInserter{}

func (i issueInserter) InsertTx(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) error {
	tx, ok := postgres.Tx(ctx)
	if !ok {
		return errInserterRequiresTransaction
	}
	return i.client.InsertTx(ctx, tx, args, opts)
}
