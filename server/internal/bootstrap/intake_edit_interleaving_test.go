package bootstrap

import (
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The interleavings of an intake work item's content edit and deletion
// with its triage, and of the intake's counts with a submission (M7 P2b
// spec 4), on issueWorld's real database, each write through the issue
// module's route as bootstrap wires it (intake_interleaving_test.go). An
// edit and a deletion take their project's row FOR NO KEY UPDATE, as a
// triage does (Locks.run), and read under it what decides them: the work
// item and its intake row again. The second of two writes is proven to
// wait, at its own LockIssueProject, for the first's row
// (serializeAtStatement) before the first lets go of it. The counts take
// no lock, and are not serialized with a submission.

// editsIntake is name's content edit of the intake work item item with
// body, through the issue module's route, which publishes to events.
func editsIntake(name string, item uuid.UUID, body string, events shared.EventPublisher) racingWrite {
	return func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		return w.sending(e, newIssueRouteWithPublisher(t, w.pool, tx, clock.System{}, events), name, http.MethodPatch, intakeItemPath(item), body)
	}
}

// deletesIntake is name's deletion of the intake work item item, through
// the issue module's route.
func deletesIntake(name string, item uuid.UUID) racingWrite {
	return func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		return w.sending(e, newIssueRoute(t, w.pool, tx), name, http.MethodDelete, intakeItemPath(item), "")
	}
}

// An intake work item's content edit and deletion serialize with its
// triage on Web's row, and the intake's counts read beside a submission
// (M7 P2b spec 2.4, 2.5, 2.6, 4). bob submits Web's work items, Waiting
// among them, pending throughout; wanda is Web's admin; alice accepts
// Alone with nothing meanwhile, which is how an accept alone leaves a
// work item.
//
//   - alice's accept holds Web past its writes; bob's edit of his pending
//     item, which passed its precheck, waits at LockIssueProject, then
//     reads it accepted under its lock: 409 issue.intake_closed, by the
//     status stored then. The item is as an accept alone leaves it: its
//     name, its rows (workItemStamps) as Alone's, one issue.updated.
//   - bob's edit holds Web; alice's accept waits, then accepts the item as
//     edited: the accept answers its new name.
//   - alice's accept holds Web; wanda's deletion of the item, which passed
//     its precheck on it pending, waits, then reads it accepted under its
//     lock and deletes its intake row alone, as wanda's: the work item is
//     as an accept alone leaves it, and reads without its intake row.
//   - wanda's deletion of a pending item holds Web; alice's accept waits,
//     then finds no work item under its lock: 404 issue.intake_not_found,
//     a random id's answer byte for byte. The item and its intake row were
//     deleted together, by wanda, and the accept wrote nothing.
//   - bob's submission holds Web past its writes; alice's counts, through
//     the app, answer meanwhile, Web's pending as before it, then, once it
//     has committed, one more: no error, one of the two numbers.
//
// The intake turned off while an edit or a deletion waits is pinned by
// the use cases' tests ("its intake turned off while the lock waited" of
// UpdateIntakeIssue and DeleteIntakeIssue), on the lock and the toggle
// that TestTheIntakeToggleAndItsWritesSerialize proves a triage reads
// under it.
func TestIntakeEditsDeletionsAndCountsInterleave(t *testing.T) {
	w := newIssueWorld(t)
	waitForStartupRuns(t, w.pool) // 03 §5: no start-time run queues on what a write holds, nor changes the rows asserted after it.
	w.makeMember(t, "wanda", w.web, shared.RoleAdmin)
	w.intakeOn(t)
	items := map[string]uuid.UUID{}
	for _, name := range []string{"Waiting", "Alone", "Accepted, then edited", "Edited, then accepted", "Accepted, then deleted", "Deleted, then accepted"} {
		items[name] = w.submit(t, "bob", fmt.Sprintf(`{"name":%q}`, name)).ID
	}
	waitForFanouts(t, w.pool) // R10: the submissions' fan-outs, before a write holds the items' rows.
	events := newArchiveStateEvents(t)
	accepted := `{"status":"accepted"}`
	accepts := func(item uuid.UUID) racingWrite { return triages("alice", item, accepted, events) }
	var alone exchange
	if err := accepts(items["Alone"])(t, w, postgres.NewTxManager(w.pool, 2*time.Second), &alone)(); err != nil {
		t.Fatal(err)
	}
	w.answered(t, alone, http.StatusOK)
	byAcceptAlone := w.workItemStamps(t, items["Alone"])
	inBacklog := "accepted in Backlog; intake: pending → accepted by alice; state: Triage → Backlog by alice"
	nameOf := func(t *testing.T, item uuid.UUID) string {
		t.Helper()
		var name string
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT name FROM issues WHERE id = $1", item).Scan(&name); err != nil {
			t.Fatal(err)
		}
		return name
	}
	acceptedAlone := func(t *testing.T, name string) {
		t.Helper()
		item := items[name]
		if got, trail := nameOf(t, item), w.triageTrail(t, item); got != name || trail != inBacklog {
			t.Errorf("%s is named %q, its trail %q; want its name, %q", name, got, trail, inBacklog)
		}
		if got := w.workItemStamps(t, item); !slices.Equal(got, byAcceptAlone) {
			t.Errorf("%s's rows\n%q\nwant them as an accept alone leaves them\n%q", name, got, byAcceptAlone)
		}
		if n := updatesOf(events, item); n != 1 {
			t.Errorf("%s's writes published %d issue.updated; want the accept's one", name, n)
		}
	}

	t.Run("an accept, then a content edit", func(t *testing.T) {
		item := items["Accepted, then edited"]
		var accept, edit exchange
		w.serializeAtStatement(t, "LockIssueProject", accepts(item), editsIntake("bob", item, `{"name":"Mended"}`, events), &accept, &edit)
		intakeIs(t, "alice's accept", w.answered(t, accept, http.StatusOK), &answeredIntake{Status: "accepted"})
		w.refused(t, edit, http.StatusConflict, "issue.intake_closed")
		acceptedAlone(t, "Accepted, then edited")
	})
	t.Run("a content edit, then an accept", func(t *testing.T) {
		item := items["Edited, then accepted"]
		var edit, accept exchange
		w.serializeAtStatement(t, "LockIssueProject", editsIntake("bob", item, `{"name":"Mended"}`, events), accepts(item), &edit, &accept)
		intakeIs(t, "bob's edit", w.answered(t, edit, http.StatusOK), &answeredIntake{Status: "pending"})
		raw := w.answered(t, accept, http.StatusOK)
		intakeIs(t, "alice's accept", raw, &answeredIntake{Status: "accepted"})
		var answer struct {
			Name string `json:"name"`
		}
		decodeAnswer(t, raw, &answer)
		if trail := w.triageTrail(t, item); answer.Name != "Mended" || trail != inBacklog {
			t.Errorf("alice's accept answers the name %q, the item's trail %q; want Mended, bob's, and %q", answer.Name, trail, inBacklog)
		}
		if n := updatesOf(events, item); n != 2 {
			t.Errorf("the edit and the accept published %d issue.updated; want one each", n)
		}
	})
	t.Run("an accept, then a deletion", func(t *testing.T) {
		item := items["Accepted, then deleted"]
		var accept, deletion exchange
		w.serializeAtStatement(t, "LockIssueProject", accepts(item), deletesIntake("wanda", item), &accept, &deletion)
		intakeIs(t, "alice's accept", w.answered(t, accept, http.StatusOK), &answeredIntake{Status: "accepted"})
		w.answered(t, deletion, http.StatusNoContent)
		if at, by := w.intakeRowDeletion(t, item); at == nil || by != w.ids["wanda"] {
			t.Errorf("its intake row deleted at %v by %s; want deleted, by wanda", at, by)
		}
		acceptedAlone(t, "Accepted, then deleted")
		intakeIs(t, "the work item, read", w.send(t, "alice", http.MethodGet, "/api/v0/issues/"+item.String(), "", http.StatusOK), nil)
	})
	t.Run("a deletion, then an accept", func(t *testing.T) {
		item := items["Deleted, then accepted"]
		var deletion, accept exchange
		w.serializeAtStatement(t, "LockIssueProject", deletesIntake("wanda", item), accepts(item), &deletion, &accept)
		w.answered(t, deletion, http.StatusNoContent)
		got := w.answered(t, accept, http.StatusNotFound)
		none := newIssueRoute(t, w.pool, postgres.NewTxManager(w.pool, 2*time.Second)).answerWithin(t, w.contract, http.MethodPost,
			triagePath(uuid.NewV7()), w.ids["alice"], accepted, http.StatusNotFound)
		if got != none || refusalOf(t, []byte(got), false) != "issue.intake_not_found" {
			t.Errorf("the accept waiting for the deletion = %s; want 404 issue.intake_not_found, byte for byte %s", got, none)
		}
		at, by := w.intakeRowDeletion(t, item)
		var deletedAt *time.Time
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at FROM issues WHERE id = $1", item).Scan(&deletedAt); err != nil ||
			deletedAt == nil || at == nil || !at.Equal(*deletedAt) || by != w.ids["wanda"] {
			t.Errorf("deleted at %v, its intake row at %v by %s, %v; want both at one moment, by wanda", deletedAt, at, by, err)
		}
		if trail := w.triageTrail(t, item); trail != "pending in Triage" {
			t.Errorf("the item's trail %q; want pending in Triage", trail)
		}
		if n := updatesOf(events, item); n != 0 {
			t.Errorf("the accept published %d issue.updated; want none", n)
		}
	})
	t.Run("the counts, a submission holding Web", func(t *testing.T) {
		counts := func(what string, pending int64) {
			t.Helper()
			countsAre(t, "alice, "+what, w.reading("alice"), answeredCount{ProjectID: w.web, Pending: pending})
		}
		counts("before bob's submission", 1)
		probe, g := requestTx(w.pool), newGate()
		var e exchange
		var done <-chan error
		defer func() { releaseAndJoinGate(t, g, done) }()
		done = run(submits("Held")(t, w, heldTx{probe, g}, &e))
		held(t, pgtest.Soon(t), g, done, "bob's submission")
		if !pgtest.RowWaits(t, w.pool, "projects", w.web, "FOR NO KEY UPDATE") {
			t.Fatal("bob's submission, held past its writes, does not hold Web's row")
		}
		counts("while bob's submission holds Web", 1)
		requestOpen(g)
		if err := result(t, pgtest.Soon(t), done, "bob's submission"); err != nil {
			t.Fatal(err)
		}
		done = nil
		intakeIs(t, "bob's submission", w.answered(t, e, http.StatusCreated), &answeredIntake{Status: "pending"})
		counts("once bob's submission has committed", 2)
	})
}
