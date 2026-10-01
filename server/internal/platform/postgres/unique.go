package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// UniqueViolation reports whether err, or an error it wraps, broke the
// unique constraint named constraint (SQLSTATE 23505). Repositories map a
// taken value to their domain's error by the constraint's name.
func UniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == constraint
}
