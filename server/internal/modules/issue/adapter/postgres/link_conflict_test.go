package postgresadapter

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// A broad SQLSTATE-only mapping would hide unrelated constraint failures.
func TestLinkConflictMapping(t *testing.T) {
	for _, tc := range []struct {
		name, code, constraint string
		match                  bool
	}{
		{"exact", "23P01", "issue_links_issue_id_url_key", true},
		{"other exclusion", "23P01", "other_constraint", false},
		{"same name unique", "23505", "issue_links_issue_id_url_key", false},
		{"other unique", "23505", "other_constraint", false},
		{"same name other code", "23514", "issue_links_issue_id_url_key", false},
		{"non PG", "", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cause := error(errors.New("plain storage failure"))
			if tc.code != "" {
				cause = &pgconn.PgError{Code: tc.code, ConstraintName: tc.constraint}
			}
			wrapped := fmt.Errorf("wrapped: %w", cause)
			got := linkConflict(wrapped)
			if tc.match {
				if !errors.Is(got, domain.ErrLinkExists) {
					t.Fatalf("mapping=%v want link_exists", got)
				}
			} else if got != wrapped || !errors.Is(got, cause) {
				t.Fatalf("unrelated chain replaced: %v", got)
			}
		})
	}
}
