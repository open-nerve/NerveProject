package postgres_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

func TestUniqueViolation(t *testing.T) {
	const constraint = "things_name_key"
	taken := &pgconn.PgError{Code: "23505", ConstraintName: constraint}
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"the named constraint", taken, true},
		{"the named constraint, wrapped", fmt.Errorf("create thing: %w", taken), true},
		{"another unique constraint", &pgconn.PgError{Code: "23505", ConstraintName: "things_slug_key"}, false},
		{"another SQLSTATE on the constraint", &pgconn.PgError{Code: "23514", ConstraintName: constraint}, false},
		{"an error that is not PostgreSQL's", errors.New(constraint), false},
		{"no error", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := postgres.UniqueViolation(tt.err, constraint); got != tt.want {
				t.Errorf("UniqueViolation(%v, %q) = %v, want %v", tt.err, constraint, got, tt.want)
			}
		})
	}
}
