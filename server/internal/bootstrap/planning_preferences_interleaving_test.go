package bootstrap

import (
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// changesPageSettings is bob's change of his display settings on the page of
// Web's cycle or module named name, table cycles or modules, through the
// planning module's route: a filter of the priorities, his first change.
func changesPageSettings(table, name string) racingWrite {
	return func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		path := "/api/v0/me/" + table + "/" + w.targetNamed(t, table, name).String() + "/preferences"
		return w.sending(e, newPlanningRoute(t, w.pool, tx), "bob", http.MethodPatch, path, `{"filters":{"priority":["high"]}}`)
	}
}

// A change of one's display settings on a cycle's or a module's page and
// the target's deletion serialize on Web's row, which the change takes FOR
// SHARE and the deletion FOR NO KEY UPDATE (M6 P3b spec R6), in both orders.
// Deleted first, the change, waiting, decides under the lock it then takes
// and finds no page: planning.cycle_not_found or planning.module_not_found,
// and no row written. Changed first, the deletion, waiting, then deletes
// bob's row, made at his change, with the target: at the deletion's moment,
// by alice (spec 2.5).
func TestPageSettingsAndTheirTargetsDeletionInterleave(t *testing.T) {
	for _, tc := range []struct {
		table, name, notFound string
		page                  func(uuid.UUID) pagePreferences
	}{
		{"cycles", "Sprint", "planning.cycle_not_found", cyclePreferences},
		{"modules", "Auth", "planning.module_not_found", modulePreferences},
	} {
		t.Run(tc.table, func(t *testing.T) {
			prepare := func(t *testing.T, w issueWorld) {
				if tc.table == "cycles" {
					w.turnCyclesOn(t, w.web)
					w.newCycle(t, "alice", `{"name":"Sprint"}`)
				} else {
					w.turnModulesOn(t, w.web)
					w.newModule(t, `{"name":"Auth"}`)
				}
			}
			inBothOrdersAt(t, "projects", prepare, "change", changesPageSettings(tc.table, tc.name), "deletion", deletesTarget(tc.table, tc.name),
				func(t *testing.T, w issueWorld, changeFirst bool, change, deletion exchange) {
					w.answered(t, deletion, http.StatusNoContent)
					target := w.targetNamed(t, tc.table, tc.name)
					page := tc.page(target)
					if !changeFirst {
						w.refused(t, change, http.StatusNotFound, tc.notFound)
						if rows := page.stamps(t, w); len(rows) != 0 {
							t.Errorf("%s of the deleted %s after the refused change: %q; want none", page.table, tc.name, rows)
						}
						return
					}
					samePreferencesJSON(t, w.answered(t, change, http.StatusOK), pageSettingsJSON(`{"priority":["high"]}`, defaultDisplay, defaultProperties))
					var deletedAt time.Time
					if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at FROM "+tc.table+" WHERE id = $1", target).Scan(&deletedAt); err != nil {
						t.Fatal(err)
					}
					page.wantDeletedWith(t, w, []string{"live"}, deletedAt, w.ids["alice"])
					var by uuid.UUID
					var filters string
					if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT user_id, filters::text FROM "+page.table+" WHERE "+page.column+" = $1", target).
						Scan(&by, &filters); err != nil || by != w.ids["bob"] || filters != `{"priority": ["high"]}` {
						t.Errorf("the deleted row: bob's %t, filters %s, %v; want bob's change", by == w.ids["bob"], filters, err)
					}
				})
		})
	}
}
