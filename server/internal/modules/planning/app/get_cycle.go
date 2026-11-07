package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// GetCycleDeps are GetCycle's ports.
type GetCycleDeps struct {
	Reads Reads
	Clock Clock
}

// GetCycle answers getCycle (M6 P1 spec 2.3).
type GetCycle struct{ d GetCycleDeps }

// NewGetCycle returns the use case over d.
func NewGetCycle(d GetCycleDeps) *GetCycle { return &GetCycle{d: d} }

// Execute reads the cycle id for the caller, with its status on the clock's
// day in its project's time zone, read after the cycle.
func (u *GetCycle) Execute(ctx context.Context, id uuid.UUID) (domain.ReadCycle, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.ReadCycle{}, err
	}
	var out domain.ReadCycle
	err = u.d.Reads.run(ctx, actor, domain.ActionCycleRead, named{cycle: &id}, func(_ context.Context, h held) error {
		out, err = readCycle(h.cycle, h.project.Timezone, u.d.Clock.Now())
		return err
	})
	return out, err
}
