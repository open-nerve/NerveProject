// Package navigation is the saved views module (M7 design 3.7, 4): it owns
// the issue_views table, each row a saved view, a project's or, without a
// project, a workspace's: its name, description, icon and the list
// settings it shows its work items with, and its owner. The 60-day purge
// removes the deleted ones (Provide).
package navigation

import (
	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/navigation/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Provided are the adapters bootstrap builds before wiring the modules.
type Provided struct {
	Purger *app.Purger
}

// Provide builds, over pool, the navigation module's purge: its one table,
// which no other table references, so the module purges before the issue
// module (M7 design 4.3, P3b spec 2.7).
func Provide(pool *pgxpool.Pool) Provided {
	store := postgresadapter.New(pool)
	return Provided{
		Purger: app.NewPurger([]app.PurgeStep{{Table: "issue_views", Delete: store.PurgeIssueViews}}),
	}
}

// Actions lists the module's actions: bootstrap's test holds the union of
// every module's actions equal to access's rule table (M3 design 3.4).
func Actions() []shared.Action {
	return domain.Actions()
}
