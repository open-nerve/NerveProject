package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CycleLister lists a project's undeleted cycles: the unarchived ones by
// sort_order, created_at, id, or, when archived, the archived ones alone by
// archived_at DESC, id (M6 P1 spec 2.3).
type CycleLister interface {
	ListCycles(ctx context.Context, project uuid.UUID, archived bool) ([]domain.Cycle, error)
}

// ListCyclesDeps are ListCycles's ports.
type ListCyclesDeps struct {
	Reads  Reads
	Cycles CycleLister
	Clock  Clock
}

// ListCycles answers listCycles (M6 P1 spec 2.3, R9).
type ListCycles struct{ d ListCyclesDeps }

// NewListCycles returns the use case over d.
func NewListCycles(d ListCyclesDeps) *ListCycles { return &ListCycles{d: d} }

// Execute lists the project's cycles, archived or not as asked, in the
// store's order, each with its status on the clock's one reading, after the
// list; an empty list for none.
func (u *ListCycles) Execute(ctx context.Context, projectID uuid.UUID, archived bool) ([]domain.ReadCycle, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	out := []domain.ReadCycle{}
	err = u.d.Reads.run(ctx, actor, projectID, nil, func(ctx context.Context, h held) error {
		cycles, err := u.d.Cycles.ListCycles(ctx, projectID, archived)
		if err != nil {
			return err
		}
		now := u.d.Clock.Now()
		for _, c := range cycles {
			r, err := readCycle(c, h.project.Timezone, now)
			if err != nil {
				return err
			}
			out = append(out, r)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
