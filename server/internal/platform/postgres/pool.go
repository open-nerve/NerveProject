// Package postgres connects nerve to PostgreSQL and runs its schema
// migrations.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
)

// errMalformedURL replaces pgx's parse error, which quotes the connection
// string and masks the password only on a best-effort basis (it misses the
// legal key/value form "password = secret").
var errMalformedURL = errors.New("database.url: not a valid PostgreSQL connection string (not shown, as it may contain a password)")

// NewPool creates a connection pool for database.url with at most
// database.max_conns connections. It connects lazily: an unreachable database
// shows up on first use, e.g. in the /readyz check.
func NewPool(ctx context.Context, cfg config.DatabaseConfig) (*pgxpool.Pool, error) {
	pc, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, errMalformedURL
	}
	pc.MaxConns = cfg.MaxConns
	pool, err := pgxpool.NewWithConfig(ctx, pc)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}
	return pool, nil
}
