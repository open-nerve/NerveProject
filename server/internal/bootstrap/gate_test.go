package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The write gate answers the gate's refusals as server_busy, to retry in a
// second (M4 design 3.5): while a write holds its project's one place, a
// second write on the project is refused at once when the queue is full,
// and when it has waited max_wait; one whose request reaches its deadline
// while it waits is server_busy too, and one whose request is cancelled
// keeps context.Canceled. A write on another project and this one is
// answered so too, and holds neither: a write on the other enters at once
// after it. A write on another project meanwhile enters at once: each
// project has places of its own (11.11). Once the first leaves, a write
// enters. Last, one on another project and this one, waiting for this one,
// holds the other's place, entered first: a write on the other is
// server_busy at its deadline.
func TestTheWriteGateAnswersItsRefusalsAsServerBusy(t *testing.T) {
	project := uuid.NewV7()
	waiting := func(limit time.Duration, cancelled bool) func() (context.Context, context.CancelFunc) {
		return func() (context.Context, context.CancelFunc) {
			if !cancelled {
				return context.WithTimeout(context.Background(), limit)
			}
			ctx, cancel := context.WithCancel(context.Background())
			time.AfterFunc(limit, cancel)
			return ctx, cancel
		}
	}
	for _, tt := range []struct {
		name  string
		gate  config.WriteGateConfig
		ctx   func() (context.Context, context.CancelFunc)
		wants func(err error) bool
	}{
		{"the queue full", config.WriteGateConfig{PerProject: 1, MaxWait: time.Minute}, waiting(time.Minute, false), isBusy},
		{"at max_wait", config.WriteGateConfig{PerProject: 1, MaxWait: 30 * time.Millisecond, MaxQueue: 1}, waiting(time.Minute, false), isBusy},
		{"at the deadline", config.WriteGateConfig{PerProject: 1, MaxWait: time.Minute, MaxQueue: 1}, waiting(30*time.Millisecond, false), isBusy},
		{"cancelled", config.WriteGateConfig{PerProject: 1, MaxWait: time.Minute, MaxQueue: 1}, waiting(30*time.Millisecond, true),
			func(err error) bool { return errors.Is(err, context.Canceled) && !isBusy(err) }},
	} {
		g := newWriteGate(tt.gate)
		leave, err := g.Enter(context.Background(), project)
		if err != nil {
			t.Fatalf("%s: the first write: %v, want it in", tt.name, err)
		}
		ctx, cancel := tt.ctx()
		if _, err := g.Enter(ctx, project); !tt.wants(err) {
			t.Errorf("%s: the second write: %v", tt.name, err)
		}
		cancel()
		across := uuid.NewV7()
		ctx, cancel = tt.ctx()
		if _, err := g.Enter(ctx, across, project); !tt.wants(err) {
			t.Errorf("%s: a write on another project and this one: %v", tt.name, err)
		}
		cancel()
		if left, err := g.Enter(context.Background(), across); err != nil {
			t.Fatalf("%s: a write on the other project after it: %v, want it in at once", tt.name, err)
		} else {
			left()
		}
		if other, err := g.Enter(context.Background(), uuid.NewV7()); err != nil {
			t.Fatalf("%s: a write on another project: %v, want it in at once", tt.name, err)
		} else {
			other()
		}
		leave()
		if leave, err := g.Enter(context.Background(), project); err != nil {
			t.Errorf("%s: a write after the first left: %v, want it in", tt.name, err)
		} else {
			leave()
		}
	}
	g := newWriteGate(config.WriteGateConfig{PerProject: 1, MaxWait: time.Minute, MaxQueue: 1})
	first, err := g.Enter(context.Background(), project)
	if err != nil {
		t.Fatal(err)
	}
	writing, stop := context.WithCancel(context.Background())
	across, done := uuid.NewV7(), make(chan struct{})
	go func() { defer close(done); _, _ = g.Enter(writing, across, project) }()
	t.Cleanup(func() { stop(); <-done; first() })
	for begun := time.Now(); ; {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		left, err := g.Enter(ctx, across)
		if cancel(); isBusy(err) {
			break
		} else if left(); time.Since(begun) > 5*time.Second {
			t.Fatal("a write on the other project entered for 5s while one on it and this one waited for this one; want it busy")
		}
	}
}

// isBusy reports whether err is shared's server_busy to retry in a second.
func isBusy(err error) bool {
	var e *shared.Error
	return errors.As(err, &e) && e.Code == shared.CodeServerBusy && e.RetryDelay == time.Second
}

// A transaction of the work-item side that a wait for a lock ended
// (lock_not_available, 55P03), however wrapped, is server_busy; one that
// ended otherwise ends as it did.
func TestABusyTransactionIsServerBusy(t *testing.T) {
	deadlock := &pgconn.PgError{Code: "40P01"}
	for _, tt := range []struct {
		name  string
		end   error
		wants func(err error) bool
	}{
		{"at its lock timeout", fmt.Errorf("lock the project: %w", &pgconn.PgError{Code: "55P03"}), isBusy},
		{"in a deadlock", deadlock, func(err error) bool { return err == deadlock }},
		{"committed", nil, func(err error) bool { return err == nil }},
	} {
		ran := false
		err := busyTx{tx: endingIn{tt.end}}.WithinTx(context.Background(), func(context.Context) error {
			ran = true
			return nil
		})
		if !ran || !tt.wants(err) {
			t.Errorf("%s: ran %v, ended in %v", tt.name, ran, err)
		}
	}
}

// endingIn is a TxManager whose transactions run their work and end in
// err.
type endingIn struct{ err error }

func (e endingIn) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if err := fn(ctx); err != nil {
		return err
	}
	return e.err
}

// A project with more writes on its work items than the app's
// issue.write_gate and issue.write_lock_timeout allow answers the writes
// beyond them 503 server_busy with Retry-After: 1 (M4 design 3.5, 11.11);
// a write its precheck refuses takes no place in the gate, and a write on
// another project does not wait for it. In an app that lets one write a
// project in at once, queues one more for a second at most, and ends a
// write's wait for a lock at three seconds: a transaction holds Web's row,
// and alice's change of a work item of Web enters the gate and waits for
// the row. Of bob's creation in Web and alice's, sent then, one waits in
// the queue and is refused at max_wait, before alice's change is at its
// lock timeout; the other before it could have waited max_wait, the queue
// full. olga's change of the work item, which she does not see, and
// walt's, who may not change it, are refused by their prechecks, 404 and
// 403, while the gate's one place is taken, before a wait at the gate
// could have ended; alice's creation in Ops, another project of acme, goes
// through before one could have ended too: Ops has places of its own.
// alice's change is refused at the lock timeout, well before the
// request's. Once the row is free, a change and a creation go through.
func TestAProjectTooBusyForAWriteAnswersServerBusy(t *testing.T) {
	w := newIssueWorldWith(t, func(cfg *config.Config) {
		cfg.Issue = config.IssueConfig{AutoArchiveInterval: 24 * time.Hour, WriteGate: config.WriteGateConfig{PerProject: 1, MaxWait: time.Second, MaxQueue: 1},
			WriteLockTimeout: 3 * time.Second}
	})
	item := w.newWorkItem(t)
	ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	path := "/api/v0/issues/" + item.String()
	change := newRequest(t, http.MethodPatch, w.base+path, w.tokens["alice"], []byte(`{"name":"Mend"}`))
	w.contract.CheckRequest(t, change)
	var creations []*http.Request
	for _, name := range []string{"bob", "alice"} {
		create := newRequest(t, http.MethodPost, w.base+"/api/v0/projects/"+w.web.String()+"/issues", w.tokens[name], []byte(`{"name":"Fix"}`))
		w.contract.CheckRequest(t, create)
		creations = append(creations, create)
	}

	held := holding(t, w.pool, "SELECT 1 FROM projects WHERE id = $1 FOR UPDATE", w.web)
	start := time.Now()
	changed := sendInBackground(change)
	pgtest.WaitForLockWaitBehind(t, w.pool, held.Conn().PgConn(), 5*time.Second)
	sent := time.Now()
	pending := []<-chan answer{sendInBackground(creations[0]), sendInBackground(creations[1])}
	var after []time.Duration
	for range pending {
		var a answer
		select {
		case a = <-pending[0]:
			pending[0] = nil
		case a = <-pending[1]:
			pending[1] = nil
		case <-time.After(3 * time.Second):
			t.Fatal("a creation did not answer within 3s")
		}
		if a.err != nil {
			t.Fatal(a.err)
		}
		after = append(after, time.Since(sent))
		w.wantBusy(t, "a creation", a.res.Request, a.res, a.body)
	}
	if after[0] >= time.Second || after[1] < time.Second || after[1] >= 3*time.Second {
		t.Errorf("the creations answered after %v; want one at once, the queue full, and the other at max_wait, 1s", after)
	}
	for _, refused := range []struct {
		name string
		want int
	}{{"olga", http.StatusNotFound}, {"walt", http.StatusForbidden}} {
		begun := time.Now()
		w.send(t, refused.name, http.MethodPatch, path, `{"name":"Mend"}`, refused.want)
		if took := time.Since(begun); took >= time.Second {
			t.Errorf("%s's change answered %d after %v; want it at once, refused by its precheck before the gate", refused.name,
				refused.want, took)
		}
	}
	begun := time.Now()
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+ops.String()+"/issues", `{"name":"Fix"}`, http.StatusCreated)
	if took := time.Since(begun); took >= time.Second {
		t.Errorf("alice's creation in Ops answered after %v; want it at once: Ops has places of its own", took)
	}
	a := receiveWithin(t, changed, 4*time.Second, "answer to alice's change")
	if a.err != nil {
		t.Fatal(a.err)
	}
	if waited := time.Since(start); waited < 3*time.Second || waited >= 4*time.Second {
		t.Errorf("alice's change answered after %v; want at the lock timeout, 3s", waited)
	}
	w.wantBusy(t, "alice's change", change, a.res, a.body)

	if err := held.Rollback(pgtest.Soon(t)); err != nil {
		t.Fatal(err)
	}
	w.send(t, "alice", http.MethodPatch, path, `{"name":"Mend"}`, http.StatusOK)
	w.send(t, "bob", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/issues", `{"name":"Fix"}`, http.StatusCreated)
}

// Each write on a work item enters its project's write gate, as a change
// does (TestAProjectTooBusyForAWriteAnswersServerBusy; M4 design 3.5,
// 11.11): while a write holds Web's one place, each of them on a work item
// of Web is refused at once, the queue full, 503 server_busy with
// Retry-After: 1, before a wait for Web's row could have ended, at the
// lock timeout at the earliest. In an app that lets one write a project in
// at once, queues none and ends a write's wait for a lock at three
// seconds: each probe holds Web's row, and a fresh change by alice
// enters the gate and waits for the row. A write across
// projects enters the gate of each project it reaches: her relation of a
// work item of Ops to one of Web is refused so too, though Ops' place is
// free. A write refused for the shape of its body takes no place at all:
// an addition of sub-items naming a work item twice, and one of relations
// naming 21 work items, are each refused 422 at once, though Web's one
// place is taken. Once the row is free, her change goes through, and so do
// the writes the gate refused. That a write across projects gives back
// each place it took is TestAWriteAcrossProjectsGivesBackEachPlaceItTook.
func TestEachWriteOnAWorkItemOfAProjectTooBusyAnswersServerBusy(t *testing.T) {
	w := newIssueWorldWith(t, func(cfg *config.Config) {
		cfg.Issue = config.IssueConfig{AutoArchiveInterval: 24 * time.Hour, WriteGate: config.WriteGateConfig{PerProject: 1, MaxWait: time.Minute},
			WriteLockTimeout: 3 * time.Second}
	})
	item, archived := w.newWorkItem(t), w.doneWorkItem(t)
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+archived.String()+"/archive", "", http.StatusOK)
	ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	writes := []struct {
		name, method, path, body string
		want                     int // its answer once Web's row is free
	}{
		{"a deletion", http.MethodDelete, "/api/v0/issues/" + w.newWorkItem(t).String(), "", http.StatusNoContent},
		{"an archiving", http.MethodPost, "/api/v0/issues/" + w.doneWorkItem(t).String() + "/archive", "", http.StatusOK},
		{"an unarchiving", http.MethodPost, "/api/v0/issues/" + archived.String() + "/unarchive", "", http.StatusOK},
		{"a comment reaction", http.MethodPost, commentReactionsOf(w.newComment(t, "alice", w.newWorkItem(t)).ID), `{"reaction":"128077"}`, http.StatusCreated},
		{"a reaction", http.MethodPost, reactionsOf(w.newWorkItem(t)), `{"reaction":"128077"}`, http.StatusCreated},
		{"a comment reaction removal", http.MethodDelete, w.newCommentReactionPath(t), "", http.StatusNoContent},
		{"a reaction removal", http.MethodDelete, w.newReactionPath(t), "", http.StatusNoContent},
		{"a comment edit", http.MethodPatch, commentPath(w.newComment(t, "alice", w.newWorkItem(t)).ID), `{"comment_html":"<p>Updated</p>"}`, http.StatusOK},
		{"a comment deletion", http.MethodDelete, commentPath(w.newComment(t, "alice", w.newWorkItem(t)).ID), "", http.StatusNoContent},
		{"a link", http.MethodPost, linksOf(w.newWorkItem(t)), `{"url":"http://169.254.169.254/gate"}`, http.StatusCreated},
		{"a comment", http.MethodPost, commentsOf(w.newWorkItem(t)), `{"comment_html":"<p>Queued comment</p>"}`, http.StatusCreated},
		{"an addition of sub-items", http.MethodPost, subIssues(w.newWorkItem(t)), subIssueIDs(w.newWorkItem(t)), http.StatusOK},
		{"an addition of relations", http.MethodPost, relationsOf(w.newWorkItem(t)), relating("relates_to", w.newWorkItem(t)),
			http.StatusCreated},
		{"an addition of relations from Ops", http.MethodPost, relationsOf(w.newWorkItemIn(t, ops, `{"name":"Far"}`)),
			relating("relates_to", w.newWorkItem(t)), http.StatusCreated},
		{"a deletion of a relation", http.MethodDelete, relationPath(listIn[answeredRelation](t, w.send(t, "alice", http.MethodPost,
			relationsOf(w.newWorkItem(t)), relating("relates_to", w.newWorkItem(t)), http.StatusCreated))[0].ID), "", http.StatusNoContent},
	}
	// Each probe owns its holder: adding cases or delaying the next case must
	// not spend the current holder's three-second lock timeout.
	changes := 0
	whileBusy := func(t *testing.T, check func()) {
		t.Helper()
		changes++
		change := newRequest(t, http.MethodPatch, w.base+"/api/v0/issues/"+item.String(), w.tokens["alice"], fmt.Appendf(nil, `{"name":"Mend %d"}`, changes))
		w.contract.CheckRequest(t, change)
		held := holding(t, w.pool, "SELECT 1 FROM projects WHERE id = $1 FOR UPDATE", w.web)
		changed := sendInBackground(change)
		defer func() {
			release(t, held)
			a := receiveWithin(t, changed, 4*time.Second, "answer to alice's change")
			if a.err != nil {
				t.Fatal(a.err)
			}
			if a.res.StatusCode != http.StatusOK {
				t.Errorf("alice's change = %d %s, want 200 once Web's row is free", a.res.StatusCode, a.body)
			}
		}()
		pgtest.WaitForLockWaitBehind(t, w.pool, held.Conn().PgConn(), 5*time.Second)
		check()
	}
	for _, write := range writes {
		t.Run(write.name, func(t *testing.T) {
			whileBusy(t, func() {
				var body []byte
				if write.body != "" {
					body = []byte(write.body)
				}
				req := newRequest(t, write.method, w.base+write.path, w.tokens["alice"], body)
				w.contract.CheckRequest(t, req)
				begun := time.Now()
				a := receiveWithin(t, sendInBackground(req), 5*time.Second, "answer to "+write.name)
				if a.err != nil {
					t.Fatal(a.err)
				}
				if took := time.Since(begun); took >= time.Second {
					t.Errorf("%s answered after %v; want at once, the queue full, before a wait for Web's row could end at 3s", write.name, took)
				}
				w.wantBusy(t, write.name, req, a.res, a.body)
			})
		})
	}
	for _, shape := range []struct{ name, path, body, want string }{
		{"an invalid link", linksOf(item), `{"url":"ftp://example.com"}`, "url not_allowed"},
		{"an empty comment", commentsOf(item), `{"comment_html":"<script>removed()</script>"}`, "comment_html required"},
		{"an addition of sub-items naming one twice", subIssues(item), subIssueIDs(archived, archived), "sub_issue_ids[1] duplicate"},
		{"an addition of relations naming 21", relationsOf(item), relating("relates_to", slices.Repeat([]uuid.UUID{archived}, 21)...),
			"issue_ids too_long"},
	} {
		t.Run(shape.name, func(t *testing.T) {
			whileBusy(t, func() {
				answer := w.send(t, "alice", http.MethodPost, shape.path, shape.body, http.StatusUnprocessableEntity)
				if got := refusalOf(t, []byte(answer), false); got != "validation_failed "+shape.want {
					t.Errorf("%s answers %s while Web's one place is taken; want %s at once, before the gate", shape.name, got, shape.want)
				}
			})
		})
	}

	for _, write := range writes {
		w.send(t, "alice", write.method, write.path, write.body, write.want)
	}
}

// A write whose request reaches its deadline while it waits at the gate is
// refused 503 server_busy then (M4 design 3.5, 11.11): its wait ends with
// its request. In an app that lets one write a project in at once, queues
// others for ten seconds, ends a request at three seconds and a write's
// wait for a lock at two seconds: a transaction holds issue_labels
// IN ACCESS EXCLUSIVE MODE, so bob's change of a work item of Web waits in
// its precheck, which reads the work item's labels. alice's creation in
// Web, sent two seconds after his change, passes its precheck, which reads
// no issue_labels, takes the project's one place and waits for Web's row,
// which another transaction holds. Once issue_labels is released, bob's
// change waits at the gate behind her creation and is refused at its
// request's deadline, three seconds after it was sent, before her
// creation, at its lock timeout, four seconds after it at the earliest and
// about a second before her own request's deadline, could leave the gate.
// Once the row is free, a change and a creation go through.
func TestAWriteWhoseRequestEndsWhileItWaitsAtTheGateIsServerBusy(t *testing.T) {
	w := newIssueWorldWith(t, func(cfg *config.Config) {
		cfg.Server.RequestTimeout = 3 * time.Second
		cfg.Issue = config.IssueConfig{AutoArchiveInterval: 24 * time.Hour, WriteGate: config.WriteGateConfig{PerProject: 1, MaxWait: 10 * time.Second, MaxQueue: 1},
			WriteLockTimeout: 2 * time.Second}
	})
	item := w.newWorkItem(t)
	path := "/api/v0/issues/" + item.String()
	change := newRequest(t, http.MethodPatch, w.base+path, w.tokens["bob"], []byte(`{"name":"Mend"}`))
	w.contract.CheckRequest(t, change)
	create := newRequest(t, http.MethodPost, w.base+"/api/v0/projects/"+w.web.String()+"/issues", w.tokens["alice"], []byte(`{"name":"Fix"}`))
	w.contract.CheckRequest(t, create)

	labels := lockedTables(t, w.pool, "issue_labels", "ACCESS EXCLUSIVE")
	sent := time.Now()
	changed := sendInBackground(change)
	pgtest.WaitForLockWaitBehind(t, w.pool, labels.Conn().PgConn(), 5*time.Second)
	time.Sleep(time.Until(sent.Add(2 * time.Second)))
	row := holding(t, w.pool, "SELECT 1 FROM projects WHERE id = $1 FOR UPDATE", w.web)
	created := sendInBackground(create)
	pgtest.WaitForLockWaitBehind(t, w.pool, row.Conn().PgConn(), 5*time.Second)
	release(t, labels)

	a := receiveWithin(t, changed, 5*time.Second, "answer to bob's change")
	if a.err != nil {
		t.Fatal(a.err)
	}
	if took := time.Since(sent); took < 3*time.Second || took >= 4*time.Second {
		t.Errorf("bob's change answered after %v; want at its request's deadline, 3s, before alice's creation left the gate", took)
	}
	w.wantBusy(t, "bob's change", change, a.res, a.body)
	a = receiveWithin(t, created, 5*time.Second, "answer to alice's creation")
	if a.err != nil {
		t.Fatal(a.err)
	}
	if took := time.Since(sent); took < 4*time.Second {
		t.Errorf("alice's creation answered %v after bob's change was sent; want at its lock timeout, 4s", took)
	}
	w.wantBusy(t, "alice's creation", create, a.res, a.body)

	release(t, row)
	w.send(t, "bob", http.MethodPatch, path, `{"name":"Mend"}`, http.StatusOK)
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/issues", `{"name":"Fix"}`, http.StatusCreated)
}

// wantBusy checks that res, the answer to req with body, is as the
// contract declares it and is 503 server_busy with Retry-After: 1.
func (w issueWorld) wantBusy(t *testing.T, what string, req *http.Request, res *http.Response, body []byte) {
	t.Helper()
	w.contract.CheckResponse(t, req, res)
	if res.StatusCode != http.StatusServiceUnavailable || problemCode(t, body) != shared.CodeServerBusy || res.Header.Get("Retry-After") != "1" {
		t.Errorf("%s = %d %s, Retry-After %q; want 503 server_busy, Retry-After 1", what, res.StatusCode, body, res.Header.Get("Retry-After"))
	}
}
