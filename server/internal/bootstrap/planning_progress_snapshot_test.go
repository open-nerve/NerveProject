package bootstrap

import (
	"context"
	"log/slog"
	"net/http"
	"slices"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// progressRead is a read of a progress as progressProbe saw it: what it
// read, and the transaction it read in, by its backend and its start, and
// whether that is a read-only snapshot.
type progressRead struct {
	read, isolation, readOnly string
	backend                   int32
	began                     time.Time
}

// progressProbe records the reads of the ports it wraps.
type progressProbe struct {
	pool  *pgxpool.Pool
	mu    sync.Mutex
	reads []progressRead
}

func (p *progressProbe) saw(ctx context.Context, read string) error {
	r := progressRead{read: read}
	if err := postgres.DB(ctx, p.pool).QueryRow(ctx, "SELECT pg_backend_pid(), now(), current_setting('transaction_isolation'), "+
		"current_setting('transaction_read_only')").Scan(&r.backend, &r.began, &r.isolation, &r.readOnly); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.reads = append(p.reads, r)
	return nil
}

// probedProjects are planningProjects whose facts and states' groups p sees
// read.
type probedProjects struct {
	planningProjects
	p *progressProbe
}

func (d probedProjects) Facts(ctx context.Context, id uuid.UUID) (planning.ProjectFacts, bool, error) {
	f, found, err := d.planningProjects.Facts(ctx, id)
	if err == nil {
		err = d.p.saw(ctx, "facts")
	}
	return f, found, err
}

func (d probedProjects) StateGroups(ctx context.Context, id uuid.UUID) (map[uuid.UUID]planning.StateGroup, error) {
	g, err := d.planningProjects.StateGroups(ctx, id)
	if err == nil {
		err = d.p.saw(ctx, "groups")
	}
	return g, err
}

// probedStats are planningIssues whose work items p sees read.
type probedStats struct {
	planningIssues
	p *progressProbe
}

func (d probedStats) CycleItems(ctx context.Context, projectID, cycleID uuid.UUID, viewer *uuid.UUID) ([]planning.WorkItem, error) {
	items, err := d.planningIssues.CycleItems(ctx, projectID, cycleID, viewer)
	if err == nil {
		err = d.p.saw(ctx, "items")
	}
	return items, err
}

func (d probedStats) ModuleItems(ctx context.Context, projectID, moduleID uuid.UUID, viewer *uuid.UUID) ([]planning.WorkItem, error) {
	items, err := d.planningIssues.ModuleItems(ctx, projectID, moduleID, viewer)
	if err == nil {
		err = d.p.saw(ctx, "items")
	}
	return items, err
}

// A cycle's and a module's progress read everything in one read-only
// snapshot (M6 P4 spec 2.3): the project's facts, its states' groups and
// the work items in one transaction, repeatable read and read only, the
// read transactions' (ReadTx), not the writes' (Tx).
func TestPlanningProgressReadsInOneSnapshot(t *testing.T) {
	w := newIssueWorld(t)
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"cycle_view":true,"module_view":true}`, http.StatusOK)
	cycle := w.newCycle(t, "alice", `{"name":"Sprint"}`)
	module := w.createModule(t, "alice", `{"name":"Core"}`, http.StatusCreated).ID
	w.newWorkItemBy(t, "alice", w.web, `{"name":"A","cycle_id":"`+cycle.String()+`","module_ids":["`+module.String()+`"]}`)
	probe := &progressProbe{pool: w.pool}
	route := newRoute(t, planning.New(planning.Deps{Pool: w.pool, Tx: postgres.NewTxManager(w.pool, 2*time.Second),
		ReadTx: postgres.NewTxManager(w.pool, 2*time.Second, postgres.ReadOnlySnapshot()), JobTx: postgres.NewTxManager(w.pool, 2*time.Second),
		RuntimeNow: time.Now, Logger: slog.New(slog.DiscardHandler), DeleteBatch: 1000, DeleteBudget: 45 * time.Second,
		WriteGate: newWriteGate(testIssueConfig.WriteGate), Clock: clock.System{}, Authorizer: authorizerOn(w.pool),
		Projects:     probedProjects{planningProjects: planningProjects{projects: project.Provide(w.pool, nil).PlanningProjects}, p: probe},
		IssueStats:   probedStats{planningIssues: planningIssues{stats: issue.Provide(w.pool, nil).PlanningStats}, p: probe},
		ListSettings: issue.Provide(w.pool, nil).ListSettings}).Register)
	for _, path := range []string{"/api/v0/cycles/" + cycle.String() + "/progress", "/api/v0/modules/" + module.String() + "/progress"} {
		probe.reads = nil
		if _, rec := route.send(http.MethodGet, path, w.ids["alice"], ""); rec.Code != http.StatusOK {
			t.Fatalf("GET %s = %d %s", path, rec.Code, rec.Body)
		}
		var names []string
		for _, r := range probe.reads {
			names = append(names, r.read)
			first := probe.reads[0]
			if r.isolation != "repeatable read" || r.readOnly != "on" || r.backend != first.backend || !r.began.Equal(first.began) {
				t.Errorf("GET %s: %+v, read in another transaction than %+v, or not in a read-only snapshot", path, r, first)
			}
		}
		if want := []string{"facts", "groups", "items"}; !slices.Equal(names, want) {
			t.Errorf("GET %s read %q, want %q", path, names, want)
		}
	}
}
