package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The interleavings of a work item's creation (M4 design 3.5, 9.3), on
// issueWorld's real database: each write through its module's route, as
// bootstrap wires it. The first write's transaction is held open past its
// writes (heldTx); the second is sent once it is held, and the gate opens
// once the second waits for Web's row of projects, which each of the two
// takes FOR NO KEY UPDATE (M4 design 3.5, M3 design 3.6). Every wait has a
// deadline.

// heldTx holds each transaction it runs open past its writes: once fn has
// done them, before the commit, it waits at its gate.
type heldTx struct {
	shared.TxManager
	gate *gate
}

func (h heldTx) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return h.TxManager.WithinTx(ctx, func(ctx context.Context) error {
		if err := fn(ctx); err != nil {
			return err
		}
		return h.gate.wait(ctx)
	})
}

// exchange is a request through a route and its answer; released, for the
// first of two writes (serialize), is the moment its gate let its
// transaction go on to its commit.
type exchange struct {
	req      *http.Request
	rec      *httptest.ResponseRecorder
	released time.Time
}

// racingWrite is a write on Web of the world through its module's route,
// built on the world's pool with tx its transactions: the sending of it
// into e, for run.
type racingWrite func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error

// sending is the sending of method path with body as the account name
// through route, into e, for run: it touches no test (moduleRoute.send).
func (w issueWorld) sending(e *exchange, route moduleRoute, name, method, path, body string) func() error {
	return func() error {
		e.req, e.rec = route.send(method, path, w.ids[name], body)
		return nil
	}
}

// createsIssue is name's creation of a work item in Web with the body body
// gives.
func createsIssue(name string, body func(t *testing.T, w issueWorld) string) racingWrite {
	return func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		return w.sending(e, newIssueRoute(t, w.pool, tx), name, http.MethodPost, "/api/v0/projects/"+w.web.String()+"/issues", body(t, w))
	}
}

// named is the body of a work item named name, with no other field.
func named(name string) func(*testing.T, issueWorld) string {
	return func(*testing.T, issueWorld) string { return `{"name":"` + name + `"}` }
}

// aliceOnWeb is alice's write method path, with body, through the
// project module's route.
func aliceOnWeb(method string, path func(t *testing.T, w issueWorld) string, body string) racingWrite {
	return func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		route := newProjectRouteOn(t, w.pool, tx, authorizerOn(w.pool), workspace.Provide(w.pool).WorkspaceMembers)
		return w.sending(e, route, "alice", method, path(t, w), body)
	}
}

// webPath is the path of Web, then rest.
func webPath(rest string) func(*testing.T, issueWorld) string {
	return func(_ *testing.T, w issueWorld) string { return "/api/v0/projects/" + w.web.String() + rest }
}

// bobsMembership is the path of bob's membership of Web.
func bobsMembership(t *testing.T, w issueWorld) string {
	return "/api/v0/project-members/" + projectMemberships(t, w.pool, w.ids["bob"], w.web)[0].String()
}

// alice's writes on Web: its archiving, its deletion, bob's removal from
// it and his demotion to a guest of it.
var (
	archivesWeb = aliceOnWeb(http.MethodPost, webPath("/archive"), "")
	deletesWeb  = aliceOnWeb(http.MethodDelete, webPath(""), "")
	removesBob  = aliceOnWeb(http.MethodDelete, bobsMembership, "")
	demotesBob  = aliceOnWeb(http.MethodPatch, bobsMembership, `{"role":5}`)
)

// serialize sends first, through a route whose transaction a gate holds
// open past its writes, then, once it is held, second; once the second
// waits for a row of projects, Web's, it opens the gate, at a's released.
// Each answers into its exchange within 10 seconds.
func (w issueWorld) serialize(t *testing.T, first, second racingWrite, a, b *exchange) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	g := newGate()
	firstWrite, secondWrite := first(t, w, heldTx{postgres.NewTxManager(w.pool, 2*time.Second), g}, a), second(t, w,
		postgres.NewTxManager(w.pool, 2*time.Second), b)
	done := run(firstWrite)
	held(t, ctx, g, done, "the first write")
	waiting := run(secondWrite)
	pgtest.WaitForLockWaitOn(t, w.pool, "projects", 5*time.Second)
	opened := time.Now().Truncate(time.Microsecond)
	close(g.open)
	if err := errors.Join(result(t, ctx, done, "the first write"), result(t, ctx, waiting, "the second write")); err != nil {
		t.Fatal(err)
	}
	a.released = opened
}

// inBothOrders runs a, then b, and b, then a, each on a world of its own
// that prepare, when given, prepares first (serialize), and checks each
// order's answers with check. aName and bName name the writes in the
// subtests' names.
func inBothOrders(t *testing.T, prepare func(t *testing.T, w issueWorld), aName string, a racingWrite, bName string, b racingWrite,
	check func(t *testing.T, w issueWorld, aFirst bool, ea, eb exchange)) {
	for _, aFirst := range []bool{true, false} {
		t.Run(map[bool]string{true: aName, false: bName}[aFirst]+" first", func(t *testing.T) {
			w := newIssueWorld(t)
			if prepare != nil {
				prepare(t, w)
			}
			var ea, eb exchange
			if aFirst {
				w.serialize(t, a, b, &ea, &eb)
			} else {
				w.serialize(t, b, a, &eb, &ea)
			}
			check(t, w, aFirst, ea, eb)
		})
	}
}

// answered checks e's answer against the contract and that it is want,
// and returns its body.
func (w issueWorld) answered(t *testing.T, e exchange, want int) string {
	t.Helper()
	w.contract.CheckResponse(t, e.req, e.rec.Result())
	if e.rec.Code != want {
		t.Errorf("%s %s = %d %s, want %d", e.req.Method, e.req.URL.Path, e.rec.Code, e.rec.Body, want)
	}
	return e.rec.Body.String()
}

// refused checks that e answered want with the problem code.
func (w issueWorld) refused(t *testing.T, e exchange, want int, code string) {
	t.Helper()
	if answer := w.answered(t, e, want); refusalOf(t, []byte(answer), false) != code {
		t.Errorf("%s %s = %s; want %s", e.req.Method, e.req.URL.Path, answer, code)
	}
}

// workItems counts Web's rows of issues, the deleted ones too.
func (w issueWorld) workItems(t *testing.T) int {
	t.Helper()
	var n int
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM issues WHERE project_id = $1", w.web).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// createdIssue is what the interleavings read of a work item created.
type createdIssue struct {
	SequenceID  int         `json:"sequence_id"`
	SortOrder   float64     `json:"sort_order"`
	AssigneeIDs []uuid.UUID `json:"assignee_ids"`
	CreatedAt   time.Time   `json:"created_at"`
}

// Two creations on Web serialize on its row (M4 design 3.5, 3.7;
// interleaving 1): alice's holds Web FOR NO KEY UPDATE and its rows past
// its writes; bob's, sent then, waits for Web's row, and once hers has
// committed takes the next number, 2, after her 1, and the sort order
// after hers in Backlog, at a moment no earlier than the gate's opening:
// he read the number, the greatest sort order and the clock under the
// lock he took once hers had committed.
func TestTwoCreationsOnAProjectSerialize(t *testing.T) {
	w := newIssueWorld(t)
	var hers, his exchange
	w.serialize(t, createsIssue("alice", named("First")), createsIssue("bob", named("Second")), &hers, &his)
	var first, second createdIssue
	decodeAnswer(t, w.answered(t, hers, http.StatusCreated), &first)
	decodeAnswer(t, w.answered(t, his, http.StatusCreated), &second)
	if first.SequenceID != 1 || second.SequenceID != 2 || second.SortOrder <= first.SortOrder || second.CreatedAt.Before(hers.released) {
		t.Errorf("created %+v, then %+v; want 1, then 2 after it in sort order, made no earlier than %v", first, second, hers.released)
	}
}

// Archiving Web and creating a work item in it serialize on Web's row (M4
// design 3.8; interleaving 8), in both orders: archived first, the
// creation, which passed the precheck, finds Web archived under its lock:
// 409 project.archived, and no work item; created first, the archiving
// waits for it and archives Web with the work item in it.
func TestArchivingAProjectAndCreatingAWorkItemSerialize(t *testing.T) {
	inBothOrders(t, nil, "archived", archivesWeb, "created", createsIssue("bob", named("Fix")), func(t *testing.T, w issueWorld,
		archivedFirst bool, archive, create exchange) {
		w.answered(t, archive, http.StatusOK)
		want := 1
		if archivedFirst {
			w.refused(t, create, http.StatusConflict, "project.archived")
			want = 0
		} else {
			w.answered(t, create, http.StatusCreated)
		}
		if n := w.workItems(t); n != want {
			t.Errorf("Web has %d work items, want %d", n, want)
		}
	})
}

// bobDefaultAssignee makes bob Web's default assignee, through the API.
func bobDefaultAssignee(t *testing.T, w issueWorld) {
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"default_assignee_id":"`+w.ids["bob"].String()+`"}`,
		http.StatusOK)
}

// Removing Web's default assignee and creating a work item without
// assignees serialize on Web's row (M4 design 3.7; interleaving 22), in
// both orders: bob, the default assignee, removed first, the creation reads
// him no member under its lock and assigns no one; created first, he is
// assigned, and his removal, which waited, leaves that row as it was.
func TestRemovingTheDefaultAssigneeAndCreatingAWorkItemSerialize(t *testing.T) {
	inBothOrders(t, bobDefaultAssignee, "removed", removesBob, "created", createsIssue("alice", named("Fix")), func(t *testing.T,
		w issueWorld, removedFirst bool, removal, create exchange) {
		w.answered(t, removal, http.StatusNoContent)
		var created createdIssue
		decodeAnswer(t, w.answered(t, create, http.StatusCreated), &created)
		want := []uuid.UUID{w.ids["bob"]}
		if removedFirst {
			want = nil
		}
		var kept int
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM issue_assignees WHERE assignee_id = $1 AND deleted_at IS NULL",
			w.ids["bob"]).Scan(&kept); err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(created.AssigneeIDs, want) || kept != len(want) {
			t.Errorf("the work item assigned to %v, %d of bob's rows kept; want %v, each kept", created.AssigneeIDs, kept, want)
		}
	})
}

// The decision under the lock outlives the precheck (M4 design 3.5;
// interleavings 4 and 5, the creation's branch): bob, a member of Web,
// made secret, creates a work item while alice's removal of him, or her
// demotion of him to a guest, holds Web past its writes. His creation
// passed the precheck, which read him a member, and waits for Web's row;
// once hers commits, the decision under the lock reads him removed, who
// no longer sees Web: 404 project.not_found; or a guest, who creates no
// work item: 403 forbidden. No work item.
func TestACallerRemovedOrDemotedWhileHisCreationWaitsIsRefused(t *testing.T) {
	for _, tt := range []struct {
		name    string
		change  racingWrite
		changed int
		want    int
		code    string
	}{
		{"removed", removesBob, http.StatusNoContent, http.StatusNotFound, "project.not_found"},
		{"made a guest", demotesBob, http.StatusOK, http.StatusForbidden, "forbidden"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := newIssueWorld(t)
			w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"network":0}`, http.StatusOK)
			var change, create exchange
			w.serialize(t, tt.change, createsIssue("bob", named("Fix")), &change, &create)
			w.answered(t, change, tt.changed)
			w.refused(t, create, tt.want, tt.code)
			if n := w.workItems(t); n != 0 {
				t.Errorf("Web has %d work items, want none", n)
			}
		})
	}
}
