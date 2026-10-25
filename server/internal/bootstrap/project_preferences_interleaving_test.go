package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

func TestPreferenceFieldsDoNotOverwrite(t *testing.T) {
	const navigation = `{"navigation":{"default_tab":"modules","hide_in_more_menu":["views"]}}`
	const filters = `{"filters":{"label_id":[],"priority":["low"]}}`
	const display = `{"layout":"spreadsheet","group_by":"priority","sub_group_by":null,"order_by":"-updated_at","sub_issue":true,"show_empty_groups":true,"calendar":{"layout":"month","show_weekends":false}}`
	for _, pair := range []struct{ name, a, b, want string }{
		{"navigation_filters", navigation, filters, preferencesJSON("modules", `["views"]`, 17, `{"label_id":[],"priority":["low"]}`, customDisplay, customProperties)},
		{"display_filters_properties", `{"display_filters":` + display + `}`, `{"display_properties":` + defaultProperties + `}`, preferencesJSON("cycles", `["intake"]`, 17, customFilters, display, defaultProperties)},
	} {
		for _, reverse := range []bool{false, true} {
			name := pair.name + "/a_first"
			if reverse {
				name = pair.name + "/b_first"
			}
			t.Run(name, func(t *testing.T) {
				w := newIssueWorld(t)
				path := "/api/v0/me/projects/" + w.web.String() + "/preferences"
				w.send(t, "alice", http.MethodPatch, path, preferencesJSON("cycles", `["intake"]`, 17, customFilters, customDisplay, customProperties), 200)
				var row uuid.UUID
				if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT id FROM project_user_properties WHERE project_id=$1 AND user_id=$2 AND deleted_at IS NULL`, w.web, w.ids["alice"]).Scan(&row); err != nil {
					t.Fatal(err)
				}
				routePath := func(*testing.T, issueWorld) string { return path }
				a, b := aliceOnWeb(http.MethodPatch, routePath, pair.a), aliceOnWeb(http.MethodPatch, routePath, pair.b)
				if reverse {
					a, b = b, a
				}
				var ea, eb exchange
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				g := newGate()
				first := a(t, w, heldTx{postgres.NewTxManager(w.pool, 2*time.Second), g}, &ea)
				second := b(t, w, postgres.NewTxManager(w.pool, 2*time.Second), &eb)
				// ON CONFLICT waits in its unique check even when the conflict row already exists.
				// The row ID above and NOWAIT probe prove this is an existing row held by A.
				_, firstErr, secondErr := serializeWaiting(t, ctx, g, first, second, func() {
					pgtest.WaitForKeyWaitOn(t, w.pool, "project_user_properties", 5*time.Second)
					if !pgtest.RowWaits(t, w.pool, "project_user_properties", row, "FOR UPDATE") {
						t.Fatal("first write does not hold the preference row")
					}
				}, nil)
				if err := errors.Join(firstErr, secondErr); err != nil {
					t.Fatal(err)
				}
				w.answered(t, ea, 200)
				w.answered(t, eb, 200)
				samePreferencesJSON(t, w.send(t, "alice", http.MethodGet, path, "", 200), pair.want)
			})
		}
	}
}
