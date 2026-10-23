package bootstrap

import (
	"context"
	"encoding/json"
	"io"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// Interleaving 20 of M4 design 3.5 and 9.3, starvation and the connection
// pool (the regressions of the design's S5 and SP1): through the app as
// bootstrap wires it, on a pool of database.max_conns' default, 10
// connections, with the write gate of testIssueConfig, the defaults: two
// writes on a project in the database at once, 64 more waiting in the
// process, without a connection, for 2 seconds at most (a minute in the
// burst's test, which counts the gate's places and its queue's room, not
// the wait).

// newPoolWorld is the world in an app whose pool has database.max_conns'
// default, 10 connections, its configuration then changed by configure.
func newPoolWorld(t *testing.T, configure ...func(cfg *config.Config)) issueWorld {
	t.Helper()
	return newIssueWorldWith(t, func(cfg *config.Config) {
		cfg.Database.MaxConns = 10
		for _, c := range configure {
			c(cfg)
		}
	})
}

// reply is the answer to a request replyTo sent: its status, its problem's
// code, its Retry-After, how long it took and when it came, or the error
// that ended it.
type reply struct {
	status int
	code   string
	retry  string
	took   time.Duration
	at     time.Time
	err    error
}

// String is the reply as the failures name it: its status, code and
// Retry-After, those it has.
func (r reply) String() string {
	if r.err != nil {
		return r.err.Error()
	}
	return strings.TrimSpace(strconv.Itoa(r.status) + " " + r.code + " " + r.retry)
}

// replyTo sends method path with body as alice. It touches no test: the
// goroutines that send at once call it.
func (w issueWorld) replyTo(method, path, body string) reply {
	req, err := http.NewRequest(method, w.base+path, strings.NewReader(body))
	if err != nil {
		return reply{err: err}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+w.tokens["alice"])
	start := time.Now()
	res, err := client.Do(req)
	if err != nil {
		return reply{took: time.Since(start), err: err}
	}
	defer func() { _ = res.Body.Close() }()
	got, err := io.ReadAll(res.Body)
	r := reply{status: res.StatusCode, retry: res.Header.Get("Retry-After"), took: time.Since(start), at: time.Now(), err: err}
	var problem struct{ Code string }
	if r.status >= 400 && json.Unmarshal(got, &problem) == nil {
		r.code = problem.Code
	}
	return r
}

// Chained writes on the work items of a workspace's projects hold off none
// of its workspace-level writes (M4 design 3.5, decision 1's B; S5's
// regression). Sixteen clients change the work items of four projects of
// acme back to back, each sending its next change once its last has
// answered; each change puts twenty labels on its work item or takes them
// off, some forty rows, so that the writes in the database overlap at
// every moment. Once each chain has had a change answered, alice renames
// acme and makes walt an admin of
// it, each within a second: neither waits for the chains, whose writes
// lock their projects' rows alone; under a lock of acme's row that each
// write shared, the chains' overlapping locks would hold either off to the
// request's deadline. Then she deletes acme, within a second too: it waits
// on each project only for the writes in the database, at most two (the
// gate's). Every change answered 200 until acme was deleted, and each
// chain's next 404 issue.not_found.
func TestChainedWorkItemWritesHoldOffNoWorkspaceWrite(t *testing.T) {
	w := newPoolWorld(t)
	projects := []uuid.UUID{w.web}
	for _, name := range []string{"Ops", "Api", "Docs"} {
		projects = append(projects, createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", name, strings.ToUpper(name)))
	}
	items := make([]uuid.UUID, len(projects))
	labelled := make([]string, len(projects))
	for i, p := range projects {
		items[i] = w.newWorkItemIn(t, p, `{"name":"Fix"}`)
		var ids []uuid.UUID
		for n := range 20 {
			var label struct {
				ID uuid.UUID `json:"id"`
			}
			decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+p.String()+"/labels", `{"name":"L`+strconv.Itoa(n)+`"}`,
				http.StatusCreated), &label)
			ids = append(ids, label.ID)
		}
		labelled[i] = `{"label_ids":` + idArray(ids...) + `}`
	}
	walt := queryIDs(t, w.pool, "SELECT id FROM workspace_members WHERE workspace_id = $1 AND member_id = $2", w.acme, w.ids["walt"])[0]

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ends := make([]reply, 16)
	changes := make([]atomic.Int32, 16)
	var chains sync.WaitGroup
	for i := range ends {
		chains.Go(func() {
			for n := 0; ctx.Err() == nil; n++ {
				body := []string{labelled[i%len(items)], `{"label_ids":[]}`}[n%2]
				r := w.replyTo(http.MethodPatch, "/api/v0/issues/"+items[i%len(items)].String(), body)
				if r.status != http.StatusOK {
					ends[i] = r
					return
				}
				changes[i].Add(1)
			}
		})
	}
	underway := func() bool {
		for i := range changes {
			if changes[i].Load() == 0 {
				return false
			}
		}
		return true
	}
	for begun := time.Now(); !underway(); time.Sleep(10 * time.Millisecond) {
		if time.Since(begun) > 5*time.Second {
			cancel()
			chains.Wait()
			t.Fatal("not every chain had a change answered within 5s")
		}
	}
	for _, write := range []struct{ method, path, body string }{
		{http.MethodPatch, "/api/v0/workspaces/acme", `{"name":"Acme Two"}`},
		{http.MethodPatch, "/api/v0/workspace-members/" + walt.String(), `{"role":20}`},
		{http.MethodDelete, "/api/v0/workspaces/acme", ""},
	} {
		r := w.replyTo(write.method, write.path, write.body)
		t.Logf("%s %s: %v after %v", write.method, write.path, r, r.took)
		if r.err != nil || r.status >= 300 || r.took > time.Second {
			t.Errorf("%s %s = %v after %v; want it done within 1s", write.method, write.path, r, r.took)
		}
	}
	chains.Wait()
	for i, end := range ends {
		if end.status != http.StatusNotFound || end.code != "issue.not_found" || changes[i].Load() == 0 {
			t.Errorf("chain %d: %d changes, then %v; want changes, then 404 issue.not_found once acme was deleted", i, changes[i].Load(), end)
		}
	}
}

// A hot project holds up no read, of another project or of its own (M4
// design 3.5, 11.11; SP1's regression). A transaction holds Web's row while
// 200 changes of a work item of Web, a credential's burst, arrive at once:
// the write gate lets two into the database, which wait for the row, each
// holding a connection of the pool; 64 more wait in the process, without
// one, as long as the row is held; the other 134 are refused at once, the
// gate's queue full: 503 server_busy, Retry-After: 1. The row is held for a
// second at least, and until those 134 have answered. Meanwhile a read of a
// work item of Ops and one of the hot work item, every 20 ms, each take at
// most poolReadBound: without the gate the changes would hold all ten
// connections while they wait for Web's row, and the reads would queue
// behind them in the pool until it is free; a read takes no place in the
// gate. Once the row is free, the 66 changes that waited go through.
func TestAHotProjectHoldsUpNoReadOfAnother(t *testing.T) {
	w := newPoolWorld(t, func(cfg *config.Config) { cfg.Issue.WriteGate.MaxWait = time.Minute })
	hot := w.newWorkItem(t)
	ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	other := w.newWorkItemIn(t, ops, `{"name":"Fix"}`)

	held := holding(t, w.pool, "SELECT 1 FROM projects WHERE id = $1 FOR UPDATE", w.web)
	writes := make([]reply, 200)
	var answered atomic.Int32
	var burst sync.WaitGroup
	start := make(chan struct{})
	for i := range writes {
		burst.Go(func() {
			<-start
			writes[i] = w.replyTo(http.MethodPatch, "/api/v0/issues/"+hot.String(), `{"name":"Fix `+strconv.Itoa(i)+`"}`)
			answered.Add(1)
		})
	}
	close(start)
	pgtest.WaitForLockWaitBehind(t, w.pool, held.Conn().PgConn(), 5*time.Second)
	waited := time.Now()
	var reads []reply
	probing, stop := context.WithCancel(context.Background())
	var probe sync.WaitGroup
	probe.Go(func() {
		for probing.Err() == nil {
			for _, item := range []uuid.UUID{other, hot} {
				reads = append(reads, w.replyTo(http.MethodGet, "/api/v0/issues/"+item.String(), ""))
			}
			time.Sleep(20 * time.Millisecond)
		}
	})
	for time.Since(waited) < time.Second || answered.Load() < 134 {
		if time.Since(waited) > 3*time.Second {
			t.Errorf("%d changes answered within 3s of the first wait for Web's row; want the 134 beyond the gate's room", answered.Load())
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	stop()
	released := time.Now()
	if err := held.Rollback(pgtest.Soon(t)); err != nil {
		t.Fatal(err)
	}
	probe.Wait()
	burst.Wait()

	slowest := slices.MaxFunc(reads, func(a, b reply) int { return int(a.took - b.took) })
	t.Logf("%d reads while Web's row was held, the slowest after %v", len(reads), slowest.took)
	if len(reads) < 20 || slowest.took > poolReadBound {
		t.Errorf("%d reads while Web's row was held, the slowest after %v; want 20 or more, each within %v", len(reads), slowest.took,
			poolReadBound)
	}
	for _, r := range reads {
		if r.status != http.StatusOK {
			t.Errorf("a read of a work item answered %v, want 200", r)
		}
	}
	before, after := map[string]int{}, map[string]int{}
	for _, r := range writes {
		if r.at.Before(released) {
			before[r.String()]++
		} else {
			after[r.String()]++
		}
	}
	if !maps.Equal(before, map[string]int{"503 server_busy 1": 134}) || !maps.Equal(after, map[string]int{"200": 66}) {
		t.Errorf("the 200 changes answered %v before Web's row was free, %v after; want 134 503 server_busy with Retry-After 1 before, "+
			"and 66 200 after", before, after)
	}
}

// poolReadBound is how long a read of a work item may take while a project
// takes a burst of writes, whether the work item is of that project or of
// another (M4/P1b spec 3: measured, with its margin).
const poolReadBound = 500 * time.Millisecond
