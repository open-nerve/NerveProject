package postgresadapter

import (
	"context"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

func (s *Store) DetachAssetReferences(ctx context.Context, ids []uuid.UUID) error {
	if _, ok := postgres.Tx(ctx); !ok {
		return errors.New("image reference detachment requires a transaction")
	}
	if len(ids) > 1000 {
		return errors.New("image reference batch exceeds 1000")
	}
	if len(ids) == 0 {
		return nil
	}
	if _, err := s.queries(ctx).LockImageReferenceOwners(ctx, ids); err != nil {
		return err
	}
	return s.queries(ctx).DetachImageReferences(ctx, ids)
}
