package postgresadapter

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// linkConflict maps only the exact live-URL exclusion constraint.
func linkConflict(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23P01" && pg.ConstraintName == "issue_links_issue_id_url_key" {
		return domain.ErrLinkExists
	}
	return err
}
