package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Reads is the one way a planning read takes its decision: in one read-only
// snapshot transaction of its own, what it names and its project's facts,
// read without a lock, and the decision on them (see), which is a write's
// but for what a read is not refused (M6 P1 spec R6, R7): an OwnOnly grant,
// an archived project, cycles turned off. No gate and no lock: a read waits
// for no write.
type Reads struct {
	projects Projects
	cycles   CycleReader
	auth     shared.Authorizer
	tx       shared.TxManager
}

// NewReads returns the reads over project's facts, the cycles' reads, the
// Authorizer and read-only snapshot transactions.
func NewReads(projects Projects, cycles CycleReader, auth shared.Authorizer, tx shared.TxManager) Reads {
	return Reads{projects: projects, cycles: cycles, auth: auth, tx: tx}
}

// run decides cycle.read on the cycle cycle, or on the project project when
// cycle is nil, for actor, and runs fn with what it holds, all in one
// snapshot.
func (r Reads) run(ctx context.Context, actor shared.Actor, project uuid.UUID, cycle *uuid.UUID, fn func(ctx context.Context, h held) error) error {
	return r.tx.WithinTx(ctx, func(ctx context.Context) error {
		h, err := see(ctx, r.cycles, r.auth, actor, domain.ActionCycleRead, project, cycle, uuid.UUID{}, r.projects.Facts)
		if err != nil {
			return err
		}
		return fn(ctx, h)
	})
}
