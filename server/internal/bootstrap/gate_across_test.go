package bootstrap

import (
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
)

// A write across projects waits max_wait in all at their gates, not
// max_wait at each (M4 design 3.5, 11.11; config's rule that max_wait and
// write_lock_timeout together stay under request_timeout counts one). In
// an app that lets one write a project in at once, queues one more, ends a
// wait at the gate at two seconds and a write's wait for a lock at five:
// transactions hold Web's row and Ops', and alice's change of a work item
// of each enters its gate and waits for its row. Her relation of the work
// item of Web to the one of Ops waits at Web's gate first, Web's id the
// lower. Web's row is freed a second and a half after the relation was
// sent: her change there goes through, and the relation enters Web's gate
// and waits at Ops'. It is refused 503 server_busy at max_wait in all, two
// seconds after it was sent, where a wait of max_wait at Ops' gate alone
// would end at three and a half: the two are a second and a half apart,
// and the answer must come at max_wait at the earliest and before halfway
// between them. Once Ops' row is free, her change there goes through, and
// so does her relation.
func TestAWriteAcrossProjectsWaitsMaxWaitInAll(t *testing.T) {
	const maxWait, freed = 2 * time.Second, 1500 * time.Millisecond
	w := newObservedIssueWorldWith(t, func(cfg *config.Config) {
		cfg.Issue = config.IssueConfig{AutoArchiveInterval: 24 * time.Hour, TrimVersionsInterval: 24 * time.Hour, WriteGate: config.WriteGateConfig{PerProject: 1, MaxWait: maxWait, MaxQueue: 1},
			WriteLockTimeout: 5 * time.Second}
	})
	ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	near, far := w.newWorkItem(t), w.newWorkItemIn(t, ops, `{"name":"Far"}`)
	var rows []pgx.Tx
	var changed []<-chan answer
	for _, held := range []struct{ project, item uuid.UUID }{{w.web, near}, {ops, far}} {
		change := newRequest(t, http.MethodPatch, w.base+"/api/v0/issues/"+held.item.String(), w.tokens["alice"], []byte(`{"name":"Mend"}`))
		w.contract.CheckRequest(t, change)
		row := holding(t, w.pool, "SELECT 1 FROM projects WHERE id = $1 FOR UPDATE", held.project)
		probe := w.observe(change)
		changed = append(changed, sendInBackground(change))
		requestBlocked(t, w.pool, probe, row.Conn().PgConn().PID(), "projects")
		rows = append(rows, row)
	}
	relate := newRequest(t, http.MethodPost, w.base+relationsOf(near), w.tokens["alice"], []byte(relating("relates_to", far)))
	w.contract.CheckRequest(t, relate)

	sent := time.Now()
	related := sendInBackground(relate)
	time.Sleep(time.Until(sent.Add(freed)))
	release(t, rows[0])
	if a := receiveWithin(t, changed[0], 4*time.Second, "answer to her change in Web"); a.err != nil || a.res.StatusCode != http.StatusOK {
		t.Fatalf("her change in Web = %v %s, want 200 once Web's row is free", a.err, a.body)
	}
	a := receiveWithin(t, related, 4*time.Second, "answer to her relation")
	if a.err != nil {
		t.Fatal(a.err)
	}
	if took := time.Since(sent); took < maxWait || took >= maxWait+freed/2 {
		t.Errorf("her relation answered after %v; want at max_wait in all, %v, before %v", took, maxWait, maxWait+freed/2)
	}
	w.wantBusy(t, "her relation", relate, a.res, a.body)

	release(t, rows[1])
	if a = receiveWithin(t, changed[1], 4*time.Second, "answer to her change in Ops"); a.err != nil || a.res.StatusCode != http.StatusOK {
		t.Fatalf("her change in Ops = %v %s, want 200 once Ops' row is free", a.err, a.body)
	}
	w.send(t, "alice", http.MethodPost, relationsOf(near), relating("relates_to", far), http.StatusCreated)
}

// A write across projects gives back each place it took at their gates,
// gone through or refused (M4 design 3.5, 11.11). In an app that lets one
// write a project in at once, queues none and ends a write's wait for a
// lock at three seconds: once alice's relation of a work item of Ops to
// one of Web has gone through, a work item, Far, is made in Ops at once.
// While her change of Far waits for Ops' row, which a transaction holds,
// and holds Ops' one place, her relation of the work item of Web to Far
// takes Web's place first, Web's id the lower, and is refused at once at
// Ops' gate: her change of the work item of Web then goes through. Once
// Ops' row is free, her change of Far goes through.
func TestAWriteAcrossProjectsGivesBackEachPlaceItTook(t *testing.T) {
	w := newObservedIssueWorldWith(t, func(cfg *config.Config) {
		cfg.Issue = config.IssueConfig{AutoArchiveInterval: 24 * time.Hour, TrimVersionsInterval: 24 * time.Hour, WriteGate: config.WriteGateConfig{PerProject: 1, MaxWait: time.Minute},
			WriteLockTimeout: 3 * time.Second}
	})
	ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	item := w.newWorkItem(t)
	w.send(t, "alice", http.MethodPost, relationsOf(w.newWorkItemIn(t, ops, `{"name":"First"}`)), relating("relates_to", item),
		http.StatusCreated)
	far := w.newWorkItemIn(t, ops, `{"name":"Far"}`)
	mend := newRequest(t, http.MethodPatch, w.base+"/api/v0/issues/"+far.String(), w.tokens["alice"], []byte(`{"name":"Mend"}`))
	w.contract.CheckRequest(t, mend)
	across := newRequest(t, http.MethodPost, w.base+relationsOf(item), w.tokens["alice"], []byte(relating("relates_to", far)))
	w.contract.CheckRequest(t, across)
	held := holding(t, w.pool, "SELECT 1 FROM projects WHERE id = $1 FOR UPDATE", ops)
	probe := w.observe(mend)
	mended := sendInBackground(mend)
	requestBlocked(t, w.pool, probe, held.Conn().PgConn().PID(), "projects")
	begun := time.Now()
	a := receiveWithin(t, sendInBackground(across), 5*time.Second, "answer to her relation of the work item of Web to Far")
	if a.err != nil {
		t.Fatal(a.err)
	}
	if took := time.Since(begun); took >= time.Second {
		t.Errorf("her relation of the work item of Web to Far answered after %v; want at once, refused at Ops' gate", took)
	}
	w.wantBusy(t, "her relation of the work item of Web to Far", across, a.res, a.body)
	w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+item.String(), `{"name":"Mended"}`, http.StatusOK)
	release(t, held)
	if a = receiveWithin(t, mended, 4*time.Second, "answer to her change of Far"); a.err != nil {
		t.Fatal(a.err)
	}
	if a.res.StatusCode != http.StatusOK {
		t.Errorf("her change of Far = %d %s, want 200 once Ops' row is free", a.res.StatusCode, a.body)
	}
}
