// Package postgresadapter is the issue module's repository adapter: sqlc
// queries (queries/, generated into gen/) over the transaction that the
// context carries, or the pool.
package postgresadapter

import (
	"context"
	"slices"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// Store implements the issue repository ports of app.
type Store struct {
	pool      *pgxpool.Pool
	protected []postgres.ProtectedReference
}

// New returns the store over pool.
func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// queries runs in the context's transaction when there is one.
func (s *Store) queries(ctx context.Context) *gen.Queries {
	return gen.New(postgres.DB(ctx, s.pool))
}

// NewPurgeStore fixes the accepted constraint identities at construction.
func NewPurgeStore(pool *pgxpool.Pool, references []postgres.ProtectedReference) *Store {
	return &Store{pool: pool, protected: slices.Clone(references)}
}
