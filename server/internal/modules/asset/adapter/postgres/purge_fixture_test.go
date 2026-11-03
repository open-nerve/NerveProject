package postgresadapter

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

type testPurgeResult struct{ Deleted int }

// Existing recovery fixtures have no reverse owner references. Exercise only
// the asset repository here; owner-aware purge is tested in bootstrap.
func (s *Store) purgeUnreferencedTestAssets(ctx context.Context, cutoff time.Time, after uuid.UUID, limit int) (testPurgeResult, error) {
	ids, err := s.PurgeAssetCandidates(ctx, cutoff, after, limit)
	if err != nil {
		return testPurgeResult{}, err
	}
	var deleted []uuid.UUID
	err = postgres.NewTxManager(s.pool, time.Second).WithinTx(ctx, func(ctx context.Context) error {
		var err error
		deleted, err = s.DeletePurgeAssets(ctx, cutoff, ids)
		return err
	})
	return testPurgeResult{Deleted: len(deleted)}, err
}
