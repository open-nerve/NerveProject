// Package riveradapter adapts the issue module's jobs to River.
package riveradapter

import (
	"context"

	"github.com/riverqueue/river"
)

// Inserter inserts in the transaction already carried by ctx. Bootstrap
// supplies the bridge to the platform's transaction and jobs client.
type Inserter interface {
	InsertTx(context.Context, river.JobArgs, *river.InsertOpts) error
}
