package bootstrap

import (
	"context"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	planningpg "github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/postgres"
	planningapp "github.com/open-nerve/NerveProject/server/internal/modules/planning/app"
	planningdomain "github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// noWriteGate is a projects' write gate that a change of one's own display
// settings on a page never enters (M6 P3b spec R6): an entry fails the test.
type noWriteGate struct{ t *testing.T }

func (g noWriteGate) Enter(context.Context, ...uuid.UUID) (func(), error) {
	g.t.Error("a change of a page's display settings entered the write gate")
	return func() {}, nil
}

// gatedClock is the clock of a change of a page's display settings, which
// the change reads under its share: its reading waits at the gate.
type gatedClock struct{ gate *gate }

func (c gatedClock) Now() time.Time {
	_ = c.gate.wait(context.Background()) // the gate's own deadline ends a lost wait; the test fails on its own
	return time.Now()
}

// cyclePreferencesChange is the change of the caller's display settings on a
// cycle's page, the planning module's use case over the real stores, project's
// locks and the issue module's list settings, as planning.New wires it: its
// transactions tx, clock its clock, and a write gate it must not enter.
func cyclePreferencesChange(t *testing.T, pool *pgxpool.Pool, tx shared.TxManager, clock planningapp.Clock) *planningapp.UpdateCyclePreferences {
	store := planningpg.New(pool)
	locks := planningapp.NewLocks(planningProjects{projects: project.Provide(pool, nil).PlanningProjects}, store, authorizerOn(pool), noWriteGate{t}, tx)
	return planningapp.NewUpdateCyclePreferences(planningapp.UpdatePreferencesDeps{Locks: locks, Settings: issue.Provide(pool, nil).ListSettings,
		Clock: clock, Preferences: store})
}

// A change of one's own display settings on a cycle's page takes Web's row FOR SHARE (M6 P3b spec R6), no write lock
// and no place in the write gate: through the planning module's use case over the real stores, bob's change queues
// behind a transaction holding Web's row FOR NO KEY UPDATE, as a planning write or a deletion holds it, and goes on
// once it ends. While a change holds its lock, its clock read under it, Web's row is held FOR SHARE, no more and no
// less. bob's row then has both changes.
func TestAPagePreferencesChangeSharesItsProject(t *testing.T) {
	w := newIssueWorld(t)
	w.turnCyclesOn(t, w.web)
	sprint := w.newCycle(t, "alice", `{"name":"Sprint"}`)
	bob := shared.WithActor(context.Background(), shared.Actor{UserID: w.ids["bob"]})

	holder := holding(t, w.pool, "SELECT 1 FROM projects WHERE id = $1 FOR NO KEY UPDATE", w.web)
	probe := requestTx(w.pool)
	filters := planningdomain.CycleFilters{ListFilters: shared.ListFilters{Priority: []string{"high"}}}
	collect := requestBackground(t, func() error {
		_, err := cyclePreferencesChange(t, w.pool, probe, clock.System{}).Execute(bob, sprint, planningdomain.CyclePreferencesPatch{Filters: &filters})
		return err
	}, func() { requestRelease(t, holder) })
	requestBlocked(t, w.pool, probe, holder.Conn().PgConn().PID(), "projects")
	requestRelease(t, holder)
	collect()

	g, probe := newGate(), requestTx(w.pool)
	properties := shared.ListDisplayProperties{Cycle: true}
	collect = requestBackground(t, func() error {
		_, err := cyclePreferencesChange(t, w.pool, probe, gatedClock{g}).Execute(bob, sprint, planningdomain.CyclePreferencesPatch{DisplayProperties: &properties})
		return err
	}, func() { requestOpen(g) })
	receiveWithin(t, (<-chan struct{})(g.held), 5*time.Second, "the change's clock reading")
	if got := requestGateRowLock(t, w.pool, w.pool, probe, "projects WHERE id = $1", w.web); got != "FOR SHARE" {
		t.Errorf("the change holds Web's row %s while it reads its clock; want FOR SHARE", got)
	}
	requestOpen(g)
	collect()

	var stored string
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT concat_ws(' ', filters, display_properties->'cycle', display_properties->'key') FROM cycle_user_properties
		WHERE cycle_id = $1 AND user_id = $2 AND deleted_at IS NULL`, sprint, w.ids["bob"]).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if want := `{"priority": ["high"]} true false`; stored != want {
		t.Errorf("bob's settings on Sprint: %s; want %s", stored, want)
	}
}
