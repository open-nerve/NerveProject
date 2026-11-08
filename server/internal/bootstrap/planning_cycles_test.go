package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// cycleAnswer is a cycle as the API answers it.
type cycleAnswer struct {
	ID          uuid.UUID  `json:"id"`
	WorkspaceID uuid.UUID  `json:"workspace_id"`
	ProjectID   uuid.UUID  `json:"project_id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	StartDate   *string    `json:"start_date"`
	EndDate     *string    `json:"end_date"`
	Status      string     `json:"status"`
	OwnedByID   *uuid.UUID `json:"owned_by_id"`
	SortOrder   float64    `json:"sort_order"`
	ArchivedAt  *time.Time `json:"archived_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CreatedByID *uuid.UUID `json:"created_by_id"`
	UpdatedByID *uuid.UUID `json:"updated_by_id"`
}

// turnCyclesOn has alice turn project's cycles on through the API, its time
// zone set to Shanghai's (M6 P1 spec R7).
func (w issueWorld) turnCyclesOn(t *testing.T, project uuid.UUID) {
	t.Helper()
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+project.String(), `{"cycle_view":true,"timezone":"Asia/Shanghai"}`, http.StatusOK)
}

// createCycle has the account name create a cycle in Web with body through
// the API, checks the answer is want, and returns its body.
func (w issueWorld) createCycle(t *testing.T, name, body string, want int) string {
	t.Helper()
	return w.send(t, name, http.MethodPost, "/api/v0/projects/"+w.web.String()+"/cycles", body, want)
}

// refusedCycle has the account name create a cycle in project with body
// and checks it is refused with status and the refusal (refusalOf: the code,
// then each field's).
func (w issueWorld) refusedCycle(t *testing.T, name string, project uuid.UUID, body string, status int, code string) {
	t.Helper()
	answer := w.send(t, name, http.MethodPost, "/api/v0/projects/"+project.String()+"/cycles", body, status)
	if got := refusalOf(t, []byte(answer), false); got != code {
		t.Errorf("POST cycles as %s with %s = %s, want %s", name, body, answer, code)
	}
}

// shanghaiDay is the day at now plus days in Shanghai, as YYYY-MM-DD.
func shanghaiDay(t *testing.T, now time.Time, days int) string {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	return now.In(loc).AddDate(0, 0, days).Format(time.DateOnly)
}

func datesBody(name, start, end string) string {
	return fmt.Sprintf(`{"name":%q,"start_date":%q,"end_date":%q}`, name, start, end)
}

// The app creates a cycle as the contract says, on the business clock (two
// days ahead here) and in the project's time zone: bob, Web's member,
// creates a draft, its name trimmed, his the owner and both writers, both
// stamps one moment; alice dated ones, each first (sort order 10000 below
// the lowest), with its status on Shanghai's day. Dates sharing a day with
// another cycle, archived ones too, are refused; a deleted cycle's are free.
func TestCreateCycleThroughTheApp(t *testing.T) {
	const offset = 48 * time.Hour
	w := newIssueWorldWith(t, func(c *config.Config) { c.Clock.Offset = offset })
	w.turnCyclesOn(t, w.web)
	before := time.Now().Add(offset)
	var draft cycleAnswer
	decodeAnswer(t, w.createCycle(t, "bob", `{"name":"  Backlog  "}`, http.StatusCreated), &draft)
	after := time.Now().Add(offset)
	bob := w.ids["bob"]
	if draft.Name != "Backlog" || draft.Description != "" || draft.StartDate != nil || draft.EndDate != nil || draft.Status != "draft" ||
		draft.WorkspaceID != w.acme || draft.ProjectID != w.web || draft.SortOrder != 65535 || draft.ArchivedAt != nil ||
		draft.OwnedByID == nil || *draft.OwnedByID != bob || draft.CreatedByID == nil || *draft.CreatedByID != bob ||
		draft.UpdatedByID == nil || *draft.UpdatedByID != bob || !draft.CreatedAt.Equal(draft.UpdatedAt) ||
		draft.CreatedAt.Before(before.Add(-time.Second)) || draft.CreatedAt.After(after.Add(time.Second)) {
		t.Fatalf("draft = %+v, want bob's Backlog at %s–%s", draft, before, after)
	}
	now := time.Now().Add(offset)
	for i, tt := range []struct {
		name, start, end, status string
	}{
		{"Now", shanghaiDay(t, now, -1), shanghaiDay(t, now, 1), "current"},
		{"Later", shanghaiDay(t, now, 10), shanghaiDay(t, now, 20), "upcoming"},
		{"Before", shanghaiDay(t, now, -20), shanghaiDay(t, now, -10), "completed"},
		{"Between", shanghaiDay(t, now, 2), shanghaiDay(t, now, 9), "upcoming"},
	} {
		var got cycleAnswer
		decodeAnswer(t, w.createCycle(t, "alice", datesBody(tt.name, tt.start, tt.end), http.StatusCreated), &got)
		if got.StartDate == nil || *got.StartDate != tt.start || got.EndDate == nil || *got.EndDate != tt.end || got.Status != tt.status ||
			got.SortOrder != 65535-10000*float64(i+1) {
			t.Errorf("%s = %+v, want %s–%s %s at %v", tt.name, got, tt.start, tt.end, tt.status, 65535-10000*float64(i+1))
		}
	}
	// One shared day with Now, at either end, and a range around Between.
	for _, body := range []string{datesBody("A", shanghaiDay(t, now, 1), shanghaiDay(t, now, 1)),
		datesBody("B", shanghaiDay(t, now, -3), shanghaiDay(t, now, -1)), datesBody("C", shanghaiDay(t, now, 1), shanghaiDay(t, now, 30))} {
		w.refusedCycle(t, "alice", w.web, body, http.StatusConflict, "planning.cycle_dates_overlap")
	}
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE cycles SET archived_at=now() WHERE name='Later'"); err != nil {
		t.Fatal(err)
	}
	w.refusedCycle(t, "alice", w.web, datesBody("D", shanghaiDay(t, now, 15), shanghaiDay(t, now, 15)), http.StatusConflict,
		"planning.cycle_dates_overlap")
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE cycles SET deleted_at=now() WHERE name='Later'"); err != nil {
		t.Fatal(err)
	}
	var free cycleAnswer
	decodeAnswer(t, w.createCycle(t, "alice", datesBody("E", shanghaiDay(t, now, 15), shanghaiDay(t, now, 15)), http.StatusCreated), &free)
	// Later's sort order no longer counts: the lowest is Between's.
	if free.SortOrder != 65535-50000 {
		t.Errorf("E's sort order = %v, want %v", free.SortOrder, 65535-50000)
	}
	var rows int
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM cycles WHERE project_id=$1", w.web).Scan(&rows); err != nil || rows != 6 {
		t.Fatalf("cycles of Web = %d, %v; want 6", rows, err)
	}
}

// The refusals in their order (M6 P1 spec R2): values first (422), even
// for a caller who does not see the project or a project that is not
// there; then the project (404); then the role, a guest's and an admin of
// the workspace's who sees only his own work items in it (403); then an
// archived project (409), before cycles turned off (409).
func TestCreateCycleRefusals(t *testing.T) {
	w := newIssueWorld(t)
	nowhere := uuid.NewV7()
	reversed := datesBody("A", "2026-11-20", "2026-11-10")
	w.refusedCycle(t, "olga", w.web, `{"name":"  "}`, http.StatusUnprocessableEntity, "validation_failed name too_short")
	w.refusedCycle(t, "alice", nowhere, reversed, http.StatusUnprocessableEntity, "validation_failed end_date not_allowed")
	w.refusedCycle(t, "alice", w.web, `{"name":"A","start_date":"2026-11-20"}`, http.StatusUnprocessableEntity,
		"validation_failed end_date required")
	w.refusedCycle(t, "olga", w.web, `{"name":"A"}`, http.StatusNotFound, "project.not_found")
	w.refusedCycle(t, "alice", nowhere, `{"name":"A"}`, http.StatusNotFound, "project.not_found")
	// Cycles still off: each refusal before it holds.
	w.refusedCycle(t, "walt", w.web, `{"name":"A"}`, http.StatusForbidden, "forbidden")
	w.refusedCycle(t, "alice", w.web, `{"name":"A"}`, http.StatusConflict, "planning.cycles_disabled")
	w.adminGuest(t)
	w.refusedCycle(t, "walt", w.web, `{"name":"A"}`, http.StatusForbidden, "forbidden")
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/archive", "", http.StatusOK)
	w.refusedCycle(t, "alice", w.web, `{"name":"A"}`, http.StatusConflict, "project.archived")
	w.refusedCycle(t, "walt", w.web, `{"name":"A"}`, http.StatusForbidden, "forbidden")
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/unarchive", "", http.StatusOK)
	w.turnCyclesOn(t, w.web)
	w.refusedCycle(t, "walt", w.web, `{"name":"A"}`, http.StatusForbidden, "forbidden")
	w.createCycle(t, "bob", `{"name":"A"}`, http.StatusCreated)
	var rows int
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM cycles").Scan(&rows); err != nil || rows != 1 {
		t.Fatalf("cycles = %d, %v; want bob's one", rows, err)
	}
}

// newPlanningRoute is the planning module as bootstrap wires it, on pool,
// with tx its writes' transactions, its link crawls inserted in them, its
// deletions published in them through the app's event directory to the
// issue module's subscribers (M6 P3a spec R6).
func newPlanningRoute(t *testing.T, pool *pgxpool.Pool, tx shared.TxManager) moduleRoute {
	t.Helper()
	return newPlanningRouteObserved(t, pool, tx, nil)
}

// newPlanningRouteObserved is newPlanningRoute whose dispatcher, when observe
// is given, hands it every event of the app's directory too, after the issue
// module's subscribers.
func newPlanningRouteObserved(t *testing.T, pool *pgxpool.Pool, tx shared.TxManager, observe func(shared.Event)) moduleRoute {
	t.Helper()
	client, err := jobs.NewInserter(pool)
	if err != nil {
		t.Fatal(err)
	}
	inserter := jobInserter{client: client}
	subscribers := issue.Subscribers(issue.Provide(pool, nil), inserter)
	if observe != nil {
		subscribers = append(subscribers, shared.EventSubscriber{Name: "observer", Types: eventTypes(), Handle: func(_ context.Context, e shared.Event) error {
			observe(e)
			return nil
		}})
	}
	events, err := newDispatcher(eventTypes(), subscribers)
	if err != nil {
		t.Fatal(err)
	}
	return newRoute(t, planning.New(planning.Deps{Pool: pool, Tx: tx, ReadTx: postgres.NewTxManager(pool, 2*time.Second, postgres.ReadOnlySnapshot()), JobTx: tx, RuntimeNow: time.Now, Logger: slog.New(slog.DiscardHandler),
		LinkCrawls: inserter, Events: events,
		DeleteBatch: 1000, DeleteBudget: 45 * time.Second, WriteGate: newWriteGate(testIssueConfig.WriteGate), Clock: clock.System{},
		Authorizer: authorizerOn(pool), Projects: planningProjects{projects: project.Provide(pool, nil).PlanningProjects}}).Register)
}

// createsCycle is name's creation of a cycle in Web with the body body
// gives.
func createsCycle(name string, body func(t *testing.T, w issueWorld) string) racingWrite {
	return func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		return w.sending(e, newPlanningRoute(t, w.pool, tx), name, http.MethodPost, "/api/v0/projects/"+w.web.String()+"/cycles", body(t, w))
	}
}

// Two creations of cycles sharing a day, the second waiting on Web's row
// while the first holds it: the first is created, the second sees it under
// the lock and is refused, and neither deadlocks (no 40P01, which would
// answer 500).
func TestCreateCycleInterleaving(t *testing.T) {
	day := func(days int) func(*testing.T, issueWorld) string {
		return func(t *testing.T, w issueWorld) string {
			now := time.Now()
			return datesBody(fmt.Sprintf("Sprint %d", days), shanghaiDay(t, now, days), shanghaiDay(t, now, days+3))
		}
	}
	inBothOrdersAt(t, "projects", func(t *testing.T, w issueWorld) { w.turnCyclesOn(t, w.web) }, "alice", createsCycle("alice", day(1)),
		"bob", createsCycle("bob", day(2)), func(t *testing.T, w issueWorld, aliceFirst bool, alice, bob exchange) {
			first, second := alice, bob
			if !aliceFirst {
				first, second = bob, alice
			}
			w.answered(t, first, http.StatusCreated)
			w.refused(t, second, http.StatusConflict, "planning.cycle_dates_overlap")
			var rows int
			if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM cycles").Scan(&rows); err != nil || rows != 1 {
				t.Fatalf("cycles = %d, %v; want the first's one", rows, err)
			}
		})
}
