// Package postgresadapter is the navigation module's repository adapter:
// sqlc queries (queries/, generated into gen/) over the pool.
package postgresadapter

import "github.com/jackc/pgx/v5/pgxpool"

// Store implements the navigation repository ports of app.
type Store struct{ pool *pgxpool.Pool }

// New returns the store over pool.
func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }
