package bootstrap

import (
	"context"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// A real EXCLUSIVE table lock allows all precheck reads but rejects attempts
// to take row locks. The guard is only at the write-resource boundary.
func testDraftPrecheckResources(t *testing.T) {
	w := newIssueWorld(t)
	d := createDraftHTTP(t, w, "alice", `{"project_id":"`+w.web.String()+`","payload":{"name":"Private"}}`)
	draftExec(t, w.pool, "UPDATE workspace_members SET role=5 WHERE workspace_id=$1 AND member_id=$2", w.acme, w.ids["walt"])
	one := poolOfOne(t, w.pool.Config().ConnString())
	resources := &collaborationWriteResources{}
	route := newIssueRouteWithGate(t, one, resources, clock.System{}, discardIssueEvents{}, resources)
	before := publishSnapshot(t, w)
	holder := lockedTables(t, w.pool, issueTables, "EXCLUSIVE")
	for _, row := range []struct {
		who, method, path, body string
		want                    int
		code                    string
	}{
		{"walt", "POST", draftsPath, `{"payload":{}}`, 403, "forbidden"},
		{"walt", "PATCH", draftPath(d.ID), `{"payload":{}}`, 403, "forbidden"},
		{"walt", "DELETE", draftPath(d.ID), "", 403, "forbidden"},
		{"walt", "POST", publishPath(d.ID), "", 403, "forbidden"},
		{"bob", "PATCH", draftPath(d.ID), `{"payload":{}}`, 404, "issue.draft_not_found"},
		{"bob", "DELETE", draftPath(d.ID), "", 404, "issue.draft_not_found"},
		{"bob", "POST", publishPath(d.ID), "", 404, "issue.draft_not_found"},
		{"alice", "PATCH", draftPath(uuid.NewV7()), `{"payload":{}}`, 404, "issue.draft_not_found"},
		{"alice", "POST", draftsPath, `{"project_id":"` + uuid.NewV7().String() + `","payload":{}}`, 422, "validation_failed"},
	} {
		raw := route.answerWithin(t, w.contract, row.method, row.path, w.ids[row.who], row.body, row.want)
		if got := problemCode(t, []byte(raw)); got != row.code {
			t.Fatal(got, raw)
		}
	}
	requestRelease(t, holder)
	if resources.gates != 0 || resources.transactions != 0 || one.Stat().AcquiredConns() != 0 {
		t.Fatal("refusal acquired write resources", resources, one.Stat().AcquiredConns())
	}
	draftSameState(t, w, before)
}

// gateWaitContext announces when the real keyed gate evaluates its queued
// wait's Done arm. It is reached after enqueuing, never on immediate admission.
type gateWaitContext struct {
	context.Context
	waiting chan struct{}
	once    sync.Once
}

func (c *gateWaitContext) Done() <-chan struct{} {
	c.once.Do(func() { close(c.waiting) })
	return c.Context.Done()
}

type draftObservedGate struct {
	writeGate
	waiting chan struct{}
}

func (g draftObservedGate) Enter(ctx context.Context, ids ...uuid.UUID) (func(), error) {
	return g.writeGate.Enter(&gateWaitContext{Context: ctx, waiting: g.waiting}, ids...)
}
func testDraftGateConnection(t *testing.T) {
	w := newIssueWorld(t)
	d := createDraftHTTP(t, w, "alice", `{"project_id":"`+w.web.String()+`","payload":{"name":"Admitted"}}`)
	one := poolOfOne(t, w.pool.Config().ConnString())
	realGate := newWriteGate(config.WriteGateConfig{PerProject: 1, MaxQueue: 1, MaxWait: 2 * time.Second})
	leave, err := realGate.Enter(pgtest.Soon(t), w.web)
	if err != nil {
		t.Fatal(err)
	}
	defer leave()
	observed := draftObservedGate{realGate, make(chan struct{})}
	tx := requestTx(one)
	route := newIssueRouteWithGate(t, one, tx, clock.System{}, discardIssueEvents{}, observed)
	baseline := one.Stat().AcquiredConns()
	var e exchange
	collect := requestBackground(t, w.sending(&e, route, "alice", "POST", publishPath(d.ID), ""), leave)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	select {
	case <-observed.waiting:
	case <-ctx.Done():
		t.Fatal("request did not queue at real gate")
	}
	if got := one.Stat().AcquiredConns(); got != baseline {
		t.Errorf("gate waiter holds %d connections, baseline %d", got, baseline)
	}
	if tx.pid.Load() != 0 {
		t.Errorf("gate waiter opened request transaction PID %d", tx.pid.Load())
	}
	draftNoLock(t, w, "workspaces", w.acme)
	draftNoLock(t, w, "projects", w.web)
	draftNoLock(t, w, "draft_issues", d.ID)
	leave()
	collect()
	var got storedIssue
	decodeAnswer(t, w.answered(t, e, 201), &got)
	if got.Name != "Admitted" || got.SequenceID != 1 {
		t.Fatal(got)
	}
	var consumed time.Time
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at FROM draft_issues WHERE id=$1", d.ID).Scan(&consumed); err != nil || !consumed.Equal(got.CreatedAt) {
		t.Fatal(consumed, err)
	}
}
