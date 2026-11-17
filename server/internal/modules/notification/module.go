// Package notification is the notifications module (M7 design 3.2–3.4, 4):
// it owns the notifications table, each row one account's notification of
// one change of a work item, with the snapshot of the work item and the
// change at the moment it was sent. The 60-day purge removes the deleted
// ones (Provide).
package notification

import (
	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/notification/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification/app"
)

// Provided are the adapters bootstrap builds before wiring the modules.
type Provided struct {
	Purger *app.Purger
}

// Provide builds, over pool, the notification module's purge: its one
// table, which no other table references, so the module purges before the
// issue module (M7 design 4.3).
func Provide(pool *pgxpool.Pool) Provided {
	store := postgresadapter.New(pool)
	return Provided{
		Purger: app.NewPurger([]app.PurgeStep{{Table: "notifications", Delete: store.PurgeNotifications}}),
	}
}
