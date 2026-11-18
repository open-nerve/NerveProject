package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The interleavings of the intake's writes (M7 P2a spec 2.4, 2.5, 4), on
// issueWorld's real database, each write through its module's route as
// bootstrap wires it (issue_interleaving_test.go). A submission and a
// triage take their project's row FOR NO KEY UPDATE, as every write of the
// project's work items, of its states and of the project itself does, and
// read under it what decides them: the work item and its intake row again,
// the intake's toggle, and an accept's default state. The second of two
// writes is proven to wait, at its own lock statement, for the first's row
// (serializeAtStatement) before the first lets go of it.

// triages is name's triage of the work item item with body, through the
// issue module's route, which publishes to events.
func triages(name string, item uuid.UUID, body string, events shared.EventPublisher) racingWrite {
	return func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		return w.sending(e, newIssueRouteWithPublisher(t, w.pool, tx, clock.System{}, events), name, http.MethodPost, triagePath(item), body)
	}
}

// submits is bob's submission of a work item named name to Web's intake,
// through the issue module's route.
func submits(name string) racingWrite {
	return func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		return w.sending(e, newIssueRoute(t, w.pool, tx), "bob", http.MethodPost, intakePath(w.web), fmt.Sprintf(`{"name":%q}`, name))
	}
}

// turnsIntakeOff is alice's turning Web's intake off, through the project
// module's route.
var turnsIntakeOff = aliceOnWeb(http.MethodPatch, webPath(""), `{"intake_view":false}`)

// marksWebDefault is alice's making Web's state name its default, through
// the project module's route.
func marksWebDefault(name string) racingWrite {
	return aliceOnWeb(http.MethodPost, func(t *testing.T, w issueWorld) string {
		return "/api/v0/states/" + stateID(t, w.pool, w.web, name).String() + "/mark-default"
	}, "")
}

// serializeAtStatement is serialize, the second write's wait for the
// first's row of projects pinned to the second's own statement, the sqlc
// query named statement: LockIssueProject for a write on the work-item
// side, LockProject for one on the project or its states.
func (w issueWorld) serializeAtStatement(t *testing.T, statement string, first, second racingWrite, a, b *exchange) {
	t.Helper()
	w.serializeAtObserved(t, "projects", first, second, a, b, func(_, waiter *requestProbe) {
		sample, waiting := readRequestLock(t, pgtest.Soon(t), w.pool, waiter)
		if f := strings.Fields(sample.query); !waiting || len(f) < 3 || f[0] != "--" || f[1] != "name:" || f[2] != statement {
			t.Fatalf("the second write waits (%v) running %q; want it waiting at %s", waiting, sample.query, statement)
		}
	})
}

// pastItsPrecheck sends write with its transaction held once begun: after
// the write's precheck and its place in the write gate, before its first
// lock (requestProbe.before). It runs meanwhile, then lets the write go on,
// and answers its exchange.
func (w issueWorld) pastItsPrecheck(t *testing.T, write racingWrite, meanwhile func()) exchange {
	t.Helper()
	g, tx := newGate(), requestTx(w.pool)
	tx.before = g
	var e exchange
	collect := requestBackground(t, write(t, w, tx, &e), func() { requestOpen(g) })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	select {
	case <-g.held:
	case <-ctx.Done():
		t.Fatal("the write did not pass its precheck within 5s")
	}
	meanwhile()
	requestOpen(g)
	collect()
	return e
}

// triageTrail is what triages left of the work item item: its intake row's
// status and its state's name, then its activities of the intake and of
// the state, deleted ones too, each with the account it names as its
// actor, the intake's first.
func (w issueWorld) triageTrail(t *testing.T, item uuid.UUID) string {
	t.Helper()
	var trail string
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT ii.status || ' in ' || s.name || coalesce((
			SELECT string_agg(format('; %s: %s → %s by %s', a.field, a.old_value, a.new_value, split_part(u.email, '@', 1)), '' ORDER BY a.field, a.id)
			FROM issue_activities a JOIN users u ON u.id = a.actor_id WHERE a.issue_id = i.id AND a.field IN ('intake', 'state')), '')
		FROM issues i JOIN intake_issues ii ON ii.issue_id = i.id JOIN states s ON s.id = i.state_id WHERE i.id = $1`, item).Scan(&trail); err != nil {
		t.Fatal(err)
	}
	return trail
}

// updatesOf counts the issue.updated events of the work item item that
// events, the recorder the triages' routes publish to
// (newArchiveStateEvents), recorded. A triage refused under its lock
// publishes nothing: its refusal comes before its first write.
func updatesOf(events *archiveRaceEvents, item uuid.UUID) int {
	n := 0
	for _, e := range events.snapshot() {
		if e.Entity.ID == item {
			n++
		}
	}
	return n
}

// Two triages of one work item of Web's intake, both from pending,
// serialize on Web's row (M7 P2a spec 2.5, 4): alice's holds it past its
// writes; wanda's, Web's admin too, sent then, passes her precheck on the
// item as pending and waits at LockIssueProject. Once alice's has
// committed, wanda's reads the item and its intake row again under her
// lock, finds them moved, and is refused: 409
// issue.intake_transition_not_allowed, and she writes nothing. Whatever
// the two ask, a decline each, or an accept and a decline either way
// round, only alice's moves the item: one intake activity, the item as
// she left it, and an issue.updated only if hers is the accept.
func TestTwoTriagesOfAnIntakeItemSerialize(t *testing.T) {
	w := newIssueWorld(t)
	w.makeMember(t, "wanda", w.web, shared.RoleAdmin)
	w.intakeOn(t)
	cases := []struct {
		name, hers, wandas, trail string
		updates                   int
	}{
		{"two declines", "declined", "declined", "declined in Triage; intake: pending → declined by alice", 0},
		{"an accept, then a decline", "accepted", "declined",
			"accepted in Backlog; intake: pending → accepted by alice; state: Triage → Backlog by alice", 1},
		{"a decline, then an accept", "declined", "accepted", "declined in Triage; intake: pending → declined by alice", 0},
	}
	items := make([]uuid.UUID, len(cases))
	for n, c := range cases {
		items[n] = w.submit(t, "bob", fmt.Sprintf(`{"name":%q}`, c.name)).ID
	}
	waitForFanouts(t, w.pool) // R10: the submissions' fan-outs, before a write holds the items' rows.
	events := newArchiveStateEvents(t)
	for n, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			item := items[n]
			var hers, wandas exchange
			w.serializeAtStatement(t, "LockIssueProject", triages("alice", item, `{"status":"`+c.hers+`"}`, events),
				triages("wanda", item, `{"status":"`+c.wandas+`"}`, events), &hers, &wandas)
			intakeIs(t, "alice's triage", w.answered(t, hers, http.StatusOK), &answeredIntake{Status: c.hers})
			w.refused(t, wandas, http.StatusConflict, "issue.intake_transition_not_allowed")
			if got := w.triageTrail(t, item); got != c.trail {
				t.Errorf("the item's trail %q; want %q", got, c.trail)
			}
			if got := updatesOf(events, item); got != c.updates {
				t.Errorf("the triages published %d issue.updated; want %d", got, c.updates)
			}
		})
	}
}

// Turning Web's intake off and a triage, or a submission, serialize on
// Web's row (M7 P2a spec 2.4, 2.5, 4: the toggle is read under the lock
// alone), in both orders. Turned off first, holding Web FOR NO KEY UPDATE:
// alice's triage of a pending item, or bob's submission, which passed its
// precheck, waits at LockIssueProject; once the change has committed, it
// reads the toggle off under its lock and is refused, 409
// issue.intake_disabled: the triage wrote nothing, the submission no work
// item. Written first: the change waits at LockProject, and the triage or
// the submission went through. Turned off after the write's precheck and
// before its first lock, with nothing held between them: the write reads
// the toggle under its lock, not at its precheck, and is refused all the
// same.
func TestTheIntakeToggleAndItsWritesSerialize(t *testing.T) {
	w := newIssueWorld(t)
	w.intakeOn(t)
	items := map[string]uuid.UUID{}
	for _, name := range []string{"Off first", "Triaged first", "Off past its precheck"} {
		items[name] = w.submit(t, "bob", fmt.Sprintf(`{"name":%q}`, name)).ID
	}
	waitForFanouts(t, w.pool) // R10: the submissions' fan-outs, before a write holds the items' rows.
	declines := func(name string) racingWrite {
		return triages("alice", items[name], `{"status":"declined"}`, discardIssueEvents{})
	}
	off := func(t *testing.T) func() {
		return func() {
			w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"intake_view":false}`, http.StatusOK)
		}
	}
	pending, declined := "pending in Triage", "declined in Triage; intake: pending → declined by alice"
	named := func(t *testing.T, name string) int {
		t.Helper()
		var n int
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM issues WHERE project_id = $1 AND name = $2", w.web, name).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	t.Run("a triage, turned off first", func(t *testing.T) {
		w.intakeOn(t)
		var turned, triaged exchange
		w.serializeAtStatement(t, "LockIssueProject", turnsIntakeOff, declines("Off first"), &turned, &triaged)
		w.answered(t, turned, http.StatusOK)
		w.refused(t, triaged, http.StatusConflict, "issue.intake_disabled")
		if got := w.triageTrail(t, items["Off first"]); got != pending {
			t.Errorf("the item's trail %q; want %q", got, pending)
		}
	})
	t.Run("a triage first", func(t *testing.T) {
		w.intakeOn(t)
		var triaged, turned exchange
		w.serializeAtStatement(t, "LockProject", declines("Triaged first"), turnsIntakeOff, &triaged, &turned)
		intakeIs(t, "the triage", w.answered(t, triaged, http.StatusOK), &answeredIntake{Status: "declined"})
		w.answered(t, turned, http.StatusOK)
		if got := w.triageTrail(t, items["Triaged first"]); got != declined {
			t.Errorf("the item's trail %q; want %q", got, declined)
		}
	})
	t.Run("a triage, turned off past its precheck", func(t *testing.T) {
		w.intakeOn(t)
		w.refused(t, w.pastItsPrecheck(t, declines("Off past its precheck"), off(t)), http.StatusConflict, "issue.intake_disabled")
		if got := w.triageTrail(t, items["Off past its precheck"]); got != pending {
			t.Errorf("the item's trail %q; want %q", got, pending)
		}
	})
	t.Run("a submission, turned off first", func(t *testing.T) {
		w.intakeOn(t)
		var turned, submitted exchange
		w.serializeAtStatement(t, "LockIssueProject", turnsIntakeOff, submits("Late"), &turned, &submitted)
		w.answered(t, turned, http.StatusOK)
		w.refused(t, submitted, http.StatusConflict, "issue.intake_disabled")
		if n := named(t, "Late"); n != 0 {
			t.Errorf("Web has %d work items named Late; want none", n)
		}
	})
	t.Run("a submission first", func(t *testing.T) {
		w.intakeOn(t)
		var submitted, turned exchange
		w.serializeAtStatement(t, "LockProject", submits("Early"), turnsIntakeOff, &submitted, &turned)
		intakeIs(t, "the submission", w.answered(t, submitted, http.StatusCreated), &answeredIntake{Status: "pending"})
		w.answered(t, turned, http.StatusOK)
		if n := named(t, "Early"); n != 1 {
			t.Errorf("Web has %d work items named Early; want the one submitted", n)
		}
	})
	t.Run("a submission, turned off past its precheck", func(t *testing.T) {
		w.intakeOn(t)
		w.refused(t, w.pastItsPrecheck(t, submits("Past"), off(t)), http.StatusConflict, "issue.intake_disabled")
		if n := named(t, "Past"); n != 0 {
			t.Errorf("Web has %d work items named Past; want none", n)
		}
	})
}

// Making another of Web's states its default and accepting a work item of
// its intake serialize on Web's row (M7 P2a spec 2.5, 4: the default state
// is read under the lock; M3 design 3.6: every write of a project's states
// takes the project's row FOR NO KEY UPDATE, LockProject), in both orders.
// Done marked first, holding Web: alice's accept, which passed its
// precheck while Backlog was the default, waits at LockIssueProject, and
// once the marking has committed moves the item into Done, the default
// under its lock. Accepted first: alice's marking of Todo waits at
// LockProject; the item is in Done, the default the accept read, and Todo
// is the default after. Each accept publishes one issue.updated.
func TestAnAcceptanceAndTheDefaultStateSerialize(t *testing.T) {
	w := newIssueWorld(t)
	w.intakeOn(t)
	first, second := w.submit(t, "bob", `{"name":"Marked first"}`).ID, w.submit(t, "bob", `{"name":"Accepted first"}`).ID
	waitForFanouts(t, w.pool) // R10: the submissions' fan-outs, before a write holds the items' rows.
	events := newArchiveStateEvents(t)
	accepts := func(item uuid.UUID) racingWrite { return triages("alice", item, `{"status":"accepted"}`, events) }
	inDone := "accepted in Done; intake: pending → accepted by alice; state: Triage → Done by alice"
	accepted := func(t *testing.T, e exchange, item uuid.UUID) {
		t.Helper()
		var answer struct {
			StateID uuid.UUID `json:"state_id"`
		}
		decodeAnswer(t, w.answered(t, e, http.StatusOK), &answer)
		if got := w.triageTrail(t, item); answer.StateID != w.done || got != inDone {
			t.Errorf("the accept answers state %s, the item's trail %q; want Done, %s, and %q", answer.StateID, got, w.done, inDone)
		}
		if n := updatesOf(events, item); n != 1 {
			t.Errorf("the accept published %d issue.updated; want one", n)
		}
	}
	defaultState := func(t *testing.T) string {
		t.Helper()
		var name string
		if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT name FROM states WHERE project_id = $1 AND "default" AND deleted_at IS NULL`, w.web).Scan(&name); err != nil {
			t.Fatal(err)
		}
		return name
	}

	t.Run("marked first", func(t *testing.T) {
		var marked, accept exchange
		w.serializeAtStatement(t, "LockIssueProject", marksWebDefault("Done"), accepts(first), &marked, &accept)
		w.answered(t, marked, http.StatusNoContent)
		accepted(t, accept, first)
	})
	t.Run("accepted first", func(t *testing.T) {
		var accept, marked exchange
		w.serializeAtStatement(t, "LockProject", accepts(second), marksWebDefault("Todo"), &accept, &marked)
		accepted(t, accept, second)
		w.answered(t, marked, http.StatusNoContent)
		if got := defaultState(t); got != "Todo" {
			t.Errorf("Web's default state is %s; want Todo", got)
		}
	})
}

// Deleting Web and triaging a work item of its intake serialize on Web's
// row (M7 P2a spec 2.5, 2.10, 4): alice's deletion holds it past its
// writes; her triage, which passed its precheck on the item, pending, waits
// at LockIssueProject, and once the deletion has committed finds no Web
// under its lock: 404 issue.intake_not_found, the answer to a work item
// that never was, byte for byte, as the deletion window has it
// (TestIssueDeletionWindowAnswersAreStable). It wrote nothing: the item's
// intake row is pending, with no intake activity.
func TestATriageWaitingForItsProjectsDeletion(t *testing.T) {
	w := newIssueWorld(t)
	w.intakeOn(t)
	item := w.submit(t, "bob", `{"name":"Login breaks"}`).ID
	waitForFanouts(t, w.pool) // R10: the submission's fan-out, before a write holds the item's row.
	declined := `{"status":"declined"}`
	var deleted, triaged exchange
	w.serializeAtStatement(t, "LockIssueProject", deletesWeb, triages("alice", item, declined, discardIssueEvents{}), &deleted, &triaged)
	w.answered(t, deleted, http.StatusNoContent)
	got := w.answered(t, triaged, http.StatusNotFound)
	none := newIssueRoute(t, w.pool, postgres.NewTxManager(w.pool, 2*time.Second)).answerWithin(t, w.contract, http.MethodPost,
		triagePath(uuid.NewV7()), w.ids["alice"], declined, http.StatusNotFound)
	if got != none || refusalOf(t, []byte(got), false) != "issue.intake_not_found" {
		t.Errorf("the triage waiting for Web's deletion = %s; want 404 issue.intake_not_found, byte for byte %s", got, none)
	}
	if trail := w.triageTrail(t, item); trail != "pending in Triage" {
		t.Errorf("the item's trail %q; want pending in Triage", trail)
	}
}
