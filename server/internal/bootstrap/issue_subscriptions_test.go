package bootstrap

import (
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The compositions of subscribeIssue and unsubscribeIssue (M7 P1a spec 2.3,
// 2.4, R3, R7), through the API and the issue module's routes as bootstrap
// wires them. Observed elsewhere, and not here: the shape of their lock (no
// project lock, no place in the gate, the work item's row FOR SHARE, no
// lock for a refused caller) by the write-lock directory (checkShareLocks);
// their answers while a project or a workspace is being deleted by
// TestIssueDeletionWindowAnswersAreStable; is_subscribed by
// TestEachDetailAnswerSaysWhetherTheCallerSubscribes; each kind of caller's
// answer by the permission matrix.

// subscriptionOf is the path of the work item id's subscribeIssue, verb
// "subscribe", or unsubscribeIssue, verb "unsubscribe".
func subscriptionOf(id uuid.UUID, verb string) string {
	return "/api/v0/issues/" + id.String() + "/" + verb
}

// subscriptionRow is a row of a work item's subscribers as the database
// keeps it: who made it and wrote it last, when, and when it was deleted,
// if it was.
type subscriptionRow struct {
	MadeBy, ChangedBy uuid.UUID
	MadeAt, ChangedAt time.Time
	DeletedAt         *time.Time
}

// equal reports whether r and o are the same row as kept.
func (r subscriptionRow) equal(o subscriptionRow) bool {
	return r.MadeBy == o.MadeBy && r.ChangedBy == o.ChangedBy && r.MadeAt.Equal(o.MadeAt) && r.ChangedAt.Equal(o.ChangedAt) &&
		(r.DeletedAt == nil) == (o.DeletedAt == nil) && (r.DeletedAt == nil || r.DeletedAt.Equal(*o.DeletedAt))
}

// subscriptionRows are the rows of the account name among the work item
// id's subscribers, the deleted ones too, in the order they were made.
func (w issueWorld) subscriptionRows(t *testing.T, id uuid.UUID, name string) []subscriptionRow {
	t.Helper()
	rows, err := w.pool.Query(pgtest.Soon(t), `SELECT created_by_id, updated_by_id, created_at, updated_at, deleted_at
		FROM issue_subscribers WHERE issue_id = $1 AND subscriber_id = $2 ORDER BY created_at, id`, id, w.ids[name])
	if err != nil {
		t.Fatal(err)
	}
	got, err := pgx.CollectRows(rows, pgx.RowToStructByPos[subscriptionRow])
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// subscribing has the account name send verb on the work item id twice,
// each answered 204 without a body, and returns his rows among its
// subscribers, which the second changes in nothing, with the moments just
// before the first was sent and just after the second was answered.
func (w issueWorld) subscribing(t *testing.T, name string, id uuid.UUID, verb string) (rows []subscriptionRow, from, to time.Time) {
	t.Helper()
	from = time.Now().Truncate(time.Microsecond)
	for n := range 2 {
		if body := w.send(t, name, http.MethodPost, subscriptionOf(id, verb), "", http.StatusNoContent); body != "" {
			t.Errorf("%s by %s answered a body: %s", verb, name, body)
		}
		got := w.subscriptionRows(t, id, name)
		if n == 1 && !slices.EqualFunc(got, rows, subscriptionRow.equal) {
			t.Errorf("%s by %s again changed his rows: %+v, then %+v", verb, name, rows, got)
		}
		rows = got
	}
	return rows, from, time.Now()
}

// roundTrip has the account name subscribe to the work item id, then, once
// between, when given, has run, unsubscribe from it (subscribing, each
// twice): the subscription a new row of his, made and last written by him
// at a moment of its requests; the unsubscription that row deleted, by him,
// at a moment of its requests, nothing else of it changed. His rows from
// before stay as they were.
func (w issueWorld) roundTrip(t *testing.T, name string, id uuid.UUID, between func()) {
	t.Helper()
	before := w.subscriptionRows(t, id, name)
	account := w.ids[name]
	rows, from, to := w.subscribing(t, name, id, "subscribe")
	made := subscriptionRow{MadeBy: account, ChangedBy: account}
	if len(rows) > len(before) {
		made.MadeAt, made.ChangedAt = rows[len(before)].MadeAt, rows[len(before)].MadeAt
	}
	if !slices.EqualFunc(rows, append(slices.Clone(before), made), subscriptionRow.equal) || made.MadeAt.Before(from) || made.MadeAt.After(to) {
		t.Fatalf("%s's rows of %s once subscribed: %+v, want those of before, %+v, and one live, his, made in [%s, %s]", name, id, rows,
			before, from, to)
	}
	if between != nil {
		between()
	}
	rows, from, to = w.subscribing(t, name, id, "unsubscribe")
	gone := made
	if len(rows) > len(before) && rows[len(before)].DeletedAt != nil {
		gone.ChangedAt, gone.DeletedAt = *rows[len(before)].DeletedAt, rows[len(before)].DeletedAt
	}
	if !slices.EqualFunc(rows, append(slices.Clone(before), gone), subscriptionRow.equal) || gone.DeletedAt == nil || gone.ChangedAt.Before(from) ||
		gone.ChangedAt.After(to) {
		t.Fatalf("%s's rows of %s once unsubscribed: %+v, want those of before, %+v, and %+v deleted by him in [%s, %s]", name, id, rows,
			before, made, from, to)
	}
}

// Subscribing and unsubscribing through the API (M7 P1a spec 2.3, 2.4, R3,
// R7). bob, Web's member, subscribes to alice's Fix, and again, then
// unsubscribes, and again (roundTrip): the list of Web's work items he
// subscribes to (subscriber_id) and his own list of them
// (relation=subscribed) hold Fix while he does, and nothing before or
// after. Once he has unsubscribed, his comment, his assignment and his new
// mention, in Fix's description or in a comment, each subscribe him again,
// a new row, from which he unsubscribes; a save of the description, and an
// edit of the comment, that still mention him do not (R7). A work item
// archived, and one of an archived project, are subscribed to and
// unsubscribed from as any other (R3).
func TestSubscribingAndUnsubscribingThroughTheAPI(t *testing.T) {
	w := newIssueWorld(t)
	fix := w.newWorkItemIn(t, w.web, `{"name":"Fix"}`)
	listed := func(want ...uuid.UUID) {
		t.Helper()
		page := checkListEnvelope(t, w.send(t, "bob", http.MethodGet, "/api/v0/projects/"+w.web.String()+"/issues?subscriber_id="+
			w.ids["bob"].String(), "", http.StatusOK), false, false)
		got := make([]uuid.UUID, len(page.Data))
		for n, row := range page.Data {
			got[n] = row.ID
		}
		if !slices.Equal(got, want) {
			t.Fatalf("Web's work items bob subscribes to: %v, want %v", got, want)
		}
		memberListIDs(t, w, "bob", "bob", "subscribed", "", want...)
	}
	listed()
	w.roundTrip(t, "bob", fix, func() { listed(fix) })
	listed()

	path := "/api/v0/issues/" + fix.String()
	mentioning := func(html string) string { return fmt.Sprintf("%q", html+mentionHTML(w.ids["bob"])) }
	again := func(what string, want bool) {
		t.Helper()
		rows := w.subscriptionRows(t, fix, "bob")
		if live := rows[len(rows)-1].DeletedAt == nil; live != want {
			t.Fatalf("after %s, bob's last subscription to Fix live=%v, want %v: %+v", what, live, want, rows)
		}
		if want {
			listed(fix)
			w.send(t, "bob", http.MethodPost, subscriptionOf(fix, "unsubscribe"), "", http.StatusNoContent)
			listed()
		}
	}
	w.send(t, "bob", http.MethodPost, commentsOf(fix), `{"comment_html":"<p>Mine</p>"}`, http.StatusCreated)
	again("his comment", true)
	w.send(t, "alice", http.MethodPatch, path, `{"assignee_ids":["`+w.ids["bob"].String()+`"]}`, http.StatusOK)
	again("his assignment", true)
	w.send(t, "alice", http.MethodPatch, path, `{"description_html":`+mentioning("<p>Ask</p>")+`}`, http.StatusOK)
	again("his new mention in the description", true)
	w.send(t, "alice", http.MethodPatch, path, `{"description_html":`+mentioning("<p>Ask again</p>")+`}`, http.StatusOK)
	again("a save of the description that still mentions him", false)
	var c answeredComment
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, commentsOf(fix), `{"comment_html":`+mentioning("<p>Look</p>")+`}`, http.StatusCreated), &c)
	again("his mention in a new comment", true)
	w.send(t, "alice", http.MethodPatch, commentPath(c.ID), `{"comment_html":`+mentioning("<p>Look again</p>")+`}`, http.StatusOK)
	again("an edit of the comment that still mentions him", false)
	if rows := w.subscriptionRows(t, fix, "bob"); len(rows) != 5 {
		t.Errorf("bob's rows of Fix: %+v, want five, each deleted: his subscription and four made again", rows)
	}

	done := w.doneWorkItem(t)
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+done.String()+"/archive", "", http.StatusOK)
	w.roundTrip(t, "bob", done, nil)
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/archive", "", http.StatusOK)
	w.roundTrip(t, "bob", fix, nil)
}

// Who may not read a work item is refused subscribing to it and
// unsubscribing from it as he is a work item that does not exist (404
// issue.not_found), and no row of his is written (M7 P1a spec 2.3): walt,
// Web's guest, who sees only his own, on alice's Fix; bob, whom alice
// removed from Ops, a private project, on its Plan; and walt, who left Ops,
// on Plan.
func TestWhoMayNotReadAWorkItemIsRefusedItsSubscription(t *testing.T) {
	w := newIssueWorld(t)
	fix := w.newWorkItemIn(t, w.web, `{"name":"Fix"}`)
	w.makeMember(t, "walt", w.web, shared.RoleGuest)
	ops := createdProjectFrom(t, w.contract, w.base, w.tokens["alice"], "acme", `{"name":"Ops","identifier":"OPS","network":0}`)
	for _, name := range []string{"bob", "walt"} {
		w.makeMember(t, name, ops, shared.RoleMember)
	}
	plan := w.newWorkItemIn(t, ops, `{"name":"Plan"}`)
	w.send(t, "alice", http.MethodDelete, "/api/v0/project-members/"+projectMemberships(t, w.pool, w.ids["bob"], ops)[0].String(), "",
		http.StatusNoContent)
	w.send(t, "walt", http.MethodPost, "/api/v0/projects/"+ops.String()+"/leave", "", http.StatusNoContent)
	for _, c := range []struct {
		name, item string
		id         uuid.UUID
	}{{"walt", "Fix", fix}, {"bob", "Plan", plan}, {"walt", "Plan", plan}} {
		for _, verb := range []string{"subscribe", "unsubscribe"} {
			refused := w.send(t, c.name, http.MethodPost, subscriptionOf(c.id, verb), "", http.StatusNotFound)
			if missing := w.send(t, c.name, http.MethodPost, subscriptionOf(uuid.NewV7(), verb), "", http.StatusNotFound); refused != missing {
				t.Errorf("%s by %s on %s answered %s, on no work item %s", verb, c.name, c.item, refused, missing)
			}
		}
		if rows := w.subscriptionRows(t, c.id, c.name); len(rows) != 0 {
			t.Errorf("%s's rows of %s: %+v, want none", c.name, c.item, rows)
		}
	}
}

// subscribes is the account name's verb on Web's one work item (workItem):
// subscribe or unsubscribe.
func subscribes(name, verb string) racingWrite {
	return func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		return w.sending(e, newIssueRoute(t, w.pool, tx), name, http.MethodPost, subscriptionOf(w.workItem(t), verb), "")
	}
}

// A subscription and its work item's deletion serialize on the work item's
// row (M7 P1a spec 2.4), in both orders: bob subscribes to alice's Fix
// (aWorkItem), and she deletes it. Subscribed first, holding Fix's row FOR
// SHARE, his row written: the deletion, which takes Web's lock meanwhile,
// waits for Fix's row, then deletes Fix and its rows, bob's subscription
// among them, four, by alice, at its one moment. Deleted first: the
// subscription, which passed its precheck on Fix as it was, finds no Fix
// under its lock: 404 issue.not_found, and bob has no row; the three rows
// are deleted. The deletion reads its moment before it waits for the row,
// so it is not bound by the subscription's commit.
func TestSubscribingAndDeletingTheWorkItemSerialize(t *testing.T) {
	inBothOrdersAt(t, "issues", aWorkItem, "subscribed", subscribes("bob", "subscribe"), "deleted", deletesIssue("alice", nil),
		func(t *testing.T, w issueWorld, subscribedFirst bool, subscribed, deleted exchange) {
			w.answered(t, deleted, http.StatusNoContent)
			rows, his := 3, 0
			if subscribedFirst {
				w.answered(t, subscribed, http.StatusNoContent)
				rows, his = 4, 1
			} else {
				w.refused(t, subscribed, http.StatusNotFound, "issue.not_found")
			}
			if got := w.subscriptionRows(t, w.workItem(t), "bob"); len(got) != his {
				t.Errorf("bob's rows of Fix: %+v, want %d", got, his)
			}
			w.deletedWith(t, "issues", w.workItem(t), time.Time{}, rows)
		})
}

// A caller whose access ends while his subscription waits for the work
// item's row is refused under it (M7 P1a spec 2.4: the decision again on
// the row as locked), as he is a work item that does not exist, and writes
// nothing. Another transaction holds alice's Fix's row FOR NO KEY UPDATE, as
// a write of it does; bob's request passes its precheck and waits for the
// row; meanwhile alice ends his access, which his request, holding no lock
// of Web, does not hold up; once the row is let go, it answers 404
// issue.not_found. Removed from Web, made private, bob's subscription to Fix
// writes no row; demoted to Web's guest, who sees only his own, his
// unsubscription leaves his row of Fix as it was.
func TestAnAccessEndedWhileASubscriptionWaitsIsRefused(t *testing.T) {
	for _, c := range []struct {
		name, verb, method, body string
		status                   int
	}{
		{"removed from private Web, subscribing", "subscribe", http.MethodDelete, "", http.StatusNoContent},
		{"demoted to Web's guest, unsubscribing", "unsubscribe", http.MethodPatch, `{"role":5}`, http.StatusOK},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := newIssueWorld(t)
			if c.method == http.MethodDelete {
				w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"network":0}`, http.StatusOK)
			}
			fix := w.newWorkItemIn(t, w.web, `{"name":"Fix"}`)
			if c.verb == "unsubscribe" {
				w.send(t, "bob", http.MethodPost, subscriptionOf(fix, "subscribe"), "", http.StatusNoContent)
			}
			before := w.subscriptionRows(t, fix, "bob")
			row := holding(t, w.pool, "SELECT 1 FROM issues WHERE id = $1 FOR NO KEY UPDATE", fix)
			probe := requestTx(w.pool)
			var e exchange
			collect := requestBackground(t, w.sending(&e, newIssueRoute(t, w.pool, probe), "bob", http.MethodPost, subscriptionOf(fix, c.verb), ""),
				func() { requestRelease(t, row) })
			requestBlocked(t, w.pool, probe, row.Conn().PgConn().PID(), "issues")
			w.send(t, "alice", c.method, bobsMembership(t, w), c.body, c.status)
			requestRelease(t, row)
			collect()
			w.refused(t, e, http.StatusNotFound, "issue.not_found")
			req, rec := newIssueRoute(t, w.pool, nil).send(http.MethodPost, subscriptionOf(uuid.NewV7(), c.verb), w.ids["bob"], "")
			w.contract.CheckResponse(t, req, rec.Result())
			if got, missing := wholeAnswerOf(e.rec.Result(), e.rec.Body.Bytes()), wholeAnswerOf(rec.Result(), rec.Body.Bytes()); got != missing {
				t.Errorf("%s of Fix answered %v, of no work item %v", c.verb, got, missing)
			}
			if after := w.subscriptionRows(t, fix, "bob"); !slices.EqualFunc(after, before, subscriptionRow.equal) {
				t.Errorf("bob's rows of Fix: %+v, then %+v", before, after)
			}
		})
	}
}

// A subscription that holds its work item's row when the work item's
// project is deleted writes its row after the deletion has committed, and
// the deletion's job, which deletes the project's work items in batches,
// then their subscribers, deletes it with them (M7 P1a spec 2.4). bob's
// subscription to alice's Fix holds Fix's row FOR SHARE, his row written
// and not committed; alice deletes Web meanwhile, which waits for no lock
// of his; its job, running, waits for Fix's row, and no one sees his row
// yet; his subscription is answered 204; the job completes, and his row is
// deleted at Web's deletion's moment, by alice.
func TestALateSubscriptionIsDeletedWithItsProject(t *testing.T) {
	w := newIssueWorld(t)
	fix := w.newWorkItemIn(t, w.web, `{"name":"Fix"}`)
	probe, g := requestTx(w.pool), newGate()
	var e exchange
	var done <-chan error
	defer func() { releaseAndJoinGate(t, g, done) }()
	done = run(w.sending(&e, newIssueRoute(t, w.pool, heldTx{probe, g}), "bob", http.MethodPost, subscriptionOf(fix, "subscribe"), ""))
	held(t, pgtest.Soon(t), g, done, "bob's subscription")
	w.send(t, "alice", http.MethodDelete, "/api/v0/projects/"+w.web.String(), "", http.StatusNoContent)
	job := projectDeletionJob(t, w.pool, w.web)
	deletionWorkerBlockedBy(t, w.pool, requestPID(t, probe), job, "issues")
	if rows := w.subscriptionRows(t, fix, "bob"); len(rows) != 0 {
		t.Fatalf("bob's rows of Fix before his subscription commits: %+v", rows)
	}
	requestOpen(g)
	if err := result(t, pgtest.Soon(t), done, "bob's subscription"); err != nil {
		t.Fatal(err)
	}
	done = nil
	w.answered(t, e, http.StatusNoContent)
	waitDeletionJob(t, w.pool, job, "completed")
	rows := w.subscriptionRows(t, fix, "bob")
	if len(rows) != 1 || rows[0].DeletedAt == nil || !rows[0].DeletedAt.Equal(job.At) || !rows[0].ChangedAt.Equal(job.At) ||
		rows[0].ChangedBy != w.ids["alice"] || job.By != w.ids["alice"] {
		t.Errorf("bob's rows of Fix: %+v, want one, deleted by alice at Web's deletion's moment, %s (job %+v)", rows, job.At, job)
	}
}

// A subscription's wait for its work item's row ends at the work-item
// side's write lock timeout (issue.write_lock_timeout, a second here): 503
// server_busy, Retry-After 1, nothing written (M7 P1a spec 2.3). Another
// transaction holds alice's Fix's row FOR NO KEY UPDATE, as a write of it
// does; bob's subscription to it and alice's unsubscription from it, sent
// together, each answer so a second or more after they were sent; bob has
// no row of Fix, and alice's is as it was. Once the row is let go, each is
// answered 204.
func TestASubscriptionWaitingPastTheWriteLockTimeoutIsBusy(t *testing.T) {
	w := newIssueWorldWith(t, func(cfg *config.Config) { cfg.Issue.WriteLockTimeout = time.Second })
	fix := w.newWorkItemIn(t, w.web, `{"name":"Fix"}`)
	hers := w.subscriptionRows(t, fix, "alice")
	row := holding(t, w.pool, "SELECT 1 FROM issues WHERE id = $1 FOR NO KEY UPDATE", fix)
	sends := []struct{ name, verb string }{{"bob", "subscribe"}, {"alice", "unsubscribe"}}
	reqs, answers := make([]*http.Request, len(sends)), make([]<-chan answer, len(sends))
	sent := time.Now()
	for n, s := range sends {
		reqs[n] = newRequest(t, http.MethodPost, w.base+subscriptionOf(fix, s.verb), w.tokens[s.name], nil)
		w.contract.CheckRequest(t, reqs[n])
		answers[n] = sendInBackground(reqs[n])
	}
	for n, s := range sends {
		a := receiveWithin(t, answers[n], 4*time.Second, s.verb+" by "+s.name)
		if a.err != nil {
			t.Fatal(a.err)
		}
		if took := time.Since(sent); took < time.Second {
			t.Errorf("%s by %s answered %v after it was sent, before the lock timeout, 1s", s.verb, s.name, took)
		}
		w.wantBusy(t, s.verb+" by "+s.name, reqs[n], a.res, a.body)
	}
	release(t, row)
	if rows := w.subscriptionRows(t, fix, "bob"); len(rows) != 0 {
		t.Errorf("bob's rows of Fix: %+v, want none", rows)
	}
	if rows := w.subscriptionRows(t, fix, "alice"); !slices.EqualFunc(rows, hers, subscriptionRow.equal) {
		t.Errorf("alice's rows of Fix: %+v, then %+v", hers, rows)
	}
	for _, s := range sends {
		w.send(t, s.name, http.MethodPost, subscriptionOf(fix, s.verb), "", http.StatusNoContent)
	}
}
