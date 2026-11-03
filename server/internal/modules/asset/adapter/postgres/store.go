// Package postgresadapter owns persistence for file assets.
package postgresadapter

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

type Store struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Store                       { return &Store{pool: pool} }
func (s *Store) queries(ctx context.Context) *gen.Queries { return gen.New(postgres.DB(ctx, s.pool)) }
