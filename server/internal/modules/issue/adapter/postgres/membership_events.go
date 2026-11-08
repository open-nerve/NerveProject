package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// RemoveCycleMemberships soft-deletes the deleted cycle's undeleted
// memberships, locked in id order, each deleted and last changed by by at
// at, and answers their work items in id order (M6 P3a spec 2.5, R6). It
// runs in the deletion's transaction, which ctx must carry: the cycle's
// deletion and the memberships' removal commit or roll back as one
// (precedent RecordAttachmentChange).
func (s *Store) RemoveCycleMemberships(ctx context.Context, cycle uuid.UUID, at time.Time, by uuid.UUID) ([]uuid.UUID, error) {
	if _, ok := postgres.Tx(ctx); !ok {
		return nil, errors.New("a deleted cycle's memberships require its deletion's transaction")
	}
	issues, err := s.queries(ctx).RemoveCycleMemberships(ctx, gen.RemoveCycleMembershipsParams{CycleID: cycle, At: at, By: by})
	if err != nil {
		return nil, fmt.Errorf("remove the memberships of deleted cycle %s: %w", cycle, err)
	}
	return issues, nil
}

// RemoveModuleMemberships is RemoveCycleMemberships of a deleted module's
// memberships.
func (s *Store) RemoveModuleMemberships(ctx context.Context, module uuid.UUID, at time.Time, by uuid.UUID) ([]uuid.UUID, error) {
	if _, ok := postgres.Tx(ctx); !ok {
		return nil, errors.New("a deleted module's memberships require its deletion's transaction")
	}
	issues, err := s.queries(ctx).RemoveModuleMemberships(ctx, gen.RemoveModuleMembershipsParams{ModuleID: module, At: at, By: by})
	if err != nil {
		return nil, fmt.Errorf("remove the memberships of deleted module %s: %w", module, err)
	}
	return issues, nil
}
