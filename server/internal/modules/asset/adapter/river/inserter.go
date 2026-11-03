// Package riveradapter adapts asset recovery work to River.
package riveradapter

import (
	"context"
	"time"
	"uuid"

	"github.com/riverqueue/river"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

const DeleteScopeKind = "asset.delete_scope"

type DeleteScopeArgs struct {
	Scope domain.ScopeKind `json:"scope"`
	ID    uuid.UUID        `json:"id"`
	At    time.Time        `json:"at"`
	By    *uuid.UUID       `json:"by"`
}

func (DeleteScopeArgs) Kind() string { return DeleteScopeKind }

type Inserter interface {
	InsertTx(context.Context, river.JobArgs, *river.InsertOpts) error
}
type ScopeDeletions struct{ inserter Inserter }

func NewScopeDeletions(inserter Inserter) *ScopeDeletions { return &ScopeDeletions{inserter: inserter} }
func (d *ScopeDeletions) Enqueue(ctx context.Context, kind domain.ScopeKind, id uuid.UUID, stamp app.DeletionStamp) error {
	return d.inserter.InsertTx(ctx, DeleteScopeArgs{Scope: kind, ID: id, At: stamp.At, By: stamp.By}, &river.InsertOpts{Queue: river.QueueDefault})
}
