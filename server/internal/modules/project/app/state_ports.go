package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

// The project store's states, as the state operations read and write them
// (M3 design 3.17). The triage state is never one of them.

// StateCreator is createState's repository. Each method runs in the
// transaction ctx carries, under the project's FOR NO KEY UPDATE.
type StateCreator interface {
	// GreatestSequence is the greatest sequence of projectID's undeleted
	// states but its triage state, nil when it has none.
	GreatestSequence(ctx context.Context, projectID uuid.UUID) (*float64, error)
	// CreateState inserts r and answers it as stored. A name another
	// undeleted state of the project has is domain.ErrStateNameTaken.
	CreateState(ctx context.Context, r StateRow) (domain.State, error)
}
