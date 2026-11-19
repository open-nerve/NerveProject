package bootstrap

import (
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	notificationdomain "github.com/open-nerve/NerveProject/server/internal/modules/notification/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// activityLine is an activity as the tests of a triage compare it: its
// verb, field, values and identifiers.
func activityLine(a answeredActivity) string {
	text := func(s *string) string {
		if s == nil {
			return "null"
		}
		return *s
	}
	id := func(u *uuid.UUID) string {
		if u == nil {
			return "null"
		}
		return u.String()
	}
	return fmt.Sprintf("%s %s: %s → %s (%s → %s)", a.Verb, text(a.Field), text(a.OldValue), text(a.NewValue), id(a.OldIdentifier),
		id(a.NewIdentifier))
}

// H10 through the app (M7 P2a spec 2.5, 4). In Web, its intake turned on,
// bob submits work items assigned to wanda, acme's admin and so Web's,
// and walt, acme's admin and a guest of Web, whom Web shows only his own
// work items (adminGuest), submits his own. A triage of a work item that
// never was, alice's of Ordinary, which has no intake row, and walt's of
// bob's, which the guest rule hides from him, answer the one 404
// issue.intake_not_found, byte for byte; bob's, Web's member, of
// Ordinary, 403. alice snoozes Snoozed, snoozes it again and ends its
// snooze, and declines Declined: each answered and recorded in its
// activity, from its status to the new one; Other, pending, is not moved
// to pending, nor Declined, then Duplicate, once a duplicate, anywhere:
// 409. A duplicate's target of another project, in the intake, the work
// item itself, one that never was, deleted, or, to walt, hidden from him,
// is refused with the one 422, byte for byte; an archived one is taken. A
// snooze without a time, or till an hour ago, is refused 422; through the
// issue module wired as bootstrap wires it on a clock standing still, till
// now too, and a microsecond on is taken; once the clock is there, the
// snooze is over: Expired moves as pending, not to pending, and its
// decline is recorded from pending. None of those triages enqueues a
// fan-out or sends anything. With Done made Web's default, alice accepts
// Accepted: in Done, completed, its intake accepted, listed and searched;
// its activity, the triage's then the state's, at the one moment the work
// item and its row are changed at; one issue.updated, whose fan-out sends
// bob, who submitted it, and wanda, its assignee, the change of its state.
// Accepting it again is 409.
func TestTriagingThroughTheApp(t *testing.T) {
	w := newIssueWorld(t)
	alice, bob, wanda := w.ids["alice"], w.ids["bob"], w.ids["wanda"]
	w.makeMember(t, "wanda", w.web, shared.RoleAdmin)
	w.adminGuest(t)
	far := w.newWorkItemIn(t, createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS"), `{"name":"Far"}`)
	ordinary, archived, deleted := w.newWorkItemIn(t, w.web, `{"name":"Ordinary"}`), w.doneWorkItem(t), w.newWorkItemIn(t, w.web, `{"name":"Gone"}`)
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+archived.String()+"/archive", "", http.StatusOK)
	w.send(t, "alice", http.MethodDelete, "/api/v0/issues/"+deleted.String(), "", http.StatusNoContent)
	w.intakeOn(t)
	items := map[string]submission{}
	for _, name := range []string{"Snoozed", "Declined", "Duplicate", "Other", "Expired", "Accepted"} {
		items[name] = w.submit(t, "bob", fmt.Sprintf(`{"name":%q,"assignee_ids":[%q]}`, name, wanda))
	}
	walts := w.submit(t, "walt", `{"name":"Walt's"}`).ID
	waitForFanouts(t, w.pool)
	sent := map[string][]uuid.UUID{}
	for name, s := range items {
		sent[name], _ = sentNotifications(t, w.pool, s.ID)
	}
	fanouts := fanoutCount(t, w.pool)
	declined := `{"status":"declined"}`

	t.Run("refused before the intake row", func(t *testing.T) {
		none := w.triage(t, "alice", uuid.NewV7(), declined, http.StatusNotFound)
		if got := refusalOf(t, []byte(none), false); got != "issue.intake_not_found" {
			t.Fatalf("a triage of a work item that never was = %s, want issue.intake_not_found", got)
		}
		w.refusesAs(t, "bob", http.MethodPost, triagePath(ordinary), declined, http.StatusForbidden, "forbidden")
		for name, item := range map[string]uuid.UUID{"alice": ordinary, "walt": items["Other"].ID} {
			if got := w.triage(t, name, item, declined, http.StatusNotFound); got != none {
				t.Errorf("%s's triage of %s = %s, want %s, byte for byte", name, item, got, none)
			}
		}
	})
	t.Run("transitions", func(t *testing.T) {
		till, later := time.Now().Add(time.Hour).UTC().Truncate(time.Microsecond), time.Now().Add(2*time.Hour).UTC().Truncate(time.Microsecond)
		snoozed := items["Snoozed"].ID
		intakeIs(t, "a snooze", w.triage(t, "alice", snoozed, snoozedTill(till), http.StatusOK), &answeredIntake{Status: "snoozed", SnoozedTill: &till})
		intakeIs(t, "a snooze again", w.triage(t, "alice", snoozed, snoozedTill(later), http.StatusOK),
			&answeredIntake{Status: "snoozed", SnoozedTill: &later})
		intakeIs(t, "the end of a snooze", w.triage(t, "alice", snoozed, `{"status":"pending"}`, http.StatusOK), &answeredIntake{Status: "pending"})
		intakeIs(t, "a decline", w.triage(t, "alice", items["Declined"].ID, declined, http.StatusOK), &answeredIntake{Status: "declined"})
		for _, refused := range []struct {
			item, body string
		}{{"Other", `{"status":"pending"}`}, {"Declined", `{"status":"accepted"}`}, {"Declined", `{"status":"pending"}`}} {
			w.refusesAs(t, "alice", http.MethodPost, triagePath(items[refused.item].ID), refused.body, http.StatusConflict,
				"issue.intake_transition_not_allowed")
		}
		for name, want := range map[string][]string{
			"Snoozed":  {"intake: pending → snoozed", "intake: snoozed → snoozed", "intake: snoozed → pending"},
			"Declined": {"intake: pending → declined"}, "Other": nil,
		} {
			if got := w.activityBy(t, items[name].ID, alice); !slices.Equal(got, want) {
				t.Errorf("%s's activity by alice %q, want %q", name, got, want)
			}
		}
	})
	t.Run("a duplicate's target", func(t *testing.T) {
		duplicate := items["Duplicate"].ID
		var refusal string
		for _, c := range []struct {
			why, name    string
			item, target uuid.UUID
		}{
			{"of another project", "alice", duplicate, far}, {"in the intake", "alice", duplicate, items["Other"].ID},
			{"declined, in the intake", "alice", duplicate, items["Declined"].ID}, {"itself", "alice", duplicate, duplicate},
			{"that never was", "alice", duplicate, uuid.NewV7()}, {"deleted", "alice", duplicate, deleted},
			{"hidden from him", "walt", walts, ordinary},
		} {
			got := w.triage(t, c.name, c.item, duplicateOf(c.target), http.StatusUnprocessableEntity)
			if refusal == "" {
				refusal = got
				if r := refusalOf(t, []byte(got), false); r != "validation_failed duplicate_to_id not_allowed" {
					t.Fatalf("a duplicate of a target %s = %s, want validation_failed duplicate_to_id not_allowed", c.why, r)
				}
			}
			if got != refusal {
				t.Errorf("%s's duplicate of a target %s = %s, want %s, byte for byte", c.name, c.why, got, refusal)
			}
		}
		intakeIs(t, "a duplicate of an archived target", w.triage(t, "alice", duplicate, duplicateOf(archived), http.StatusOK),
			&answeredIntake{Status: "duplicate", DuplicateToID: &archived})
		w.refusesAs(t, "alice", http.MethodPost, triagePath(duplicate), declined, http.StatusConflict, "issue.intake_transition_not_allowed")
		if got, want := w.activityBy(t, duplicate, alice), []string{"intake: pending → duplicate"}; !slices.Equal(got, want) {
			t.Errorf("Duplicate's activity by alice %q, want %q", got, want)
		}
	})
	t.Run("a snooze's time", func(t *testing.T) {
		expired := items["Expired"].ID
		w.refusesAs(t, "alice", http.MethodPost, triagePath(expired), `{"status":"snoozed"}`, http.StatusUnprocessableEntity,
			"validation_failed snoozed_till required")
		w.refusesAs(t, "alice", http.MethodPost, triagePath(expired), snoozedTill(time.Now().Add(-time.Hour)), http.StatusUnprocessableEntity,
			"validation_failed snoozed_till must_be_future")
		c := clockAt(time.Now().Truncate(time.Microsecond))
		route := newIssueRouteWithClock(t, w.pool, postgres.NewTxManager(w.pool, 2*time.Second), c)
		triage := func(body string, want int) string {
			return route.answerWithin(t, w.contract, http.MethodPost, triagePath(expired), alice, body, want)
		}
		if got := refusalOf(t, []byte(triage(snoozedTill(c.Now()), http.StatusUnprocessableEntity)), false); got !=
			"validation_failed snoozed_till must_be_future" {
			t.Errorf("a snooze till now = %s, want validation_failed snoozed_till must_be_future", got)
		}
		till := c.Now().Add(time.Microsecond)
		intakeIs(t, "a snooze till a microsecond on", triage(snoozedTill(till), http.StatusOK), &answeredIntake{Status: "snoozed", SnoozedTill: &till})
		c.advance(time.Microsecond)
		if got := refusalOf(t, []byte(triage(`{"status":"pending"}`, http.StatusConflict)), false); got != "issue.intake_transition_not_allowed" {
			t.Errorf("the end of a snooze over = %s, want issue.intake_transition_not_allowed", got)
		}
		intakeIs(t, "the decline of a snooze over", triage(declined, http.StatusOK), &answeredIntake{Status: "declined"})
		if got, want := w.activityBy(t, expired, alice), []string{"intake: pending → snoozed", "intake: pending → declined"}; !slices.Equal(got, want) {
			t.Errorf("Expired's activity by alice %q, want %q", got, want)
		}
	})
	t.Run("none of them notifies", func(t *testing.T) {
		if got := fanoutCount(t, w.pool); got != fanouts {
			t.Errorf("the triages enqueued %d fan-outs; want none", got-fanouts)
		}
		waitForFanouts(t, w.pool)
		for name, before := range sent {
			if after, _ := sentNotifications(t, w.pool, items[name].ID); !slices.Equal(after, before) {
				t.Errorf("%s's notifications %v, want its submission's alone, %v", name, after, before)
			}
		}
	})
	t.Run("an acceptance", func(t *testing.T) {
		accepted := items["Accepted"]
		triage := stateID(t, w.pool, w.web, "Triage")
		w.send(t, "alice", http.MethodPost, "/api/v0/states/"+w.done.String()+"/mark-default", "", http.StatusNoContent)
		raw := w.triage(t, "alice", accepted.ID, `{"status":"accepted"}`, http.StatusOK)
		var answer struct {
			StateID     uuid.UUID  `json:"state_id"`
			CompletedAt *time.Time `json:"completed_at"`
		}
		decodeAnswer(t, raw, &answer)
		intakeIs(t, "an acceptance", raw, &answeredIntake{Status: "accepted"})
		var at, row time.Time
		if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT i.updated_at, ii.updated_at FROM issues i JOIN intake_issues ii ON ii.issue_id = i.id
			WHERE i.id = $1`, accepted.ID).Scan(&at, &row); err != nil || !row.Equal(at) {
			t.Errorf("the work item changed at %s, its row at %s, %v; want one moment", at, row, err)
		}
		if answer.StateID != w.done || answer.CompletedAt == nil || !answer.CompletedAt.Equal(at) {
			t.Errorf("the acceptance answers state %s completed at %v; want Done, Web's default, at %s", answer.StateID, answer.CompletedAt, at)
		}
		projectListIDs(t, w, "alice", ordinary, accepted.ID)
		searchHTTPIDs(t, w, "alice", "project_id="+w.web.String(), ordinary, accepted.ID)
		rows := activityPage(t, w, "alice", activitiesOf(accepted.ID)+"?limit=100").Data
		lines := make([]string, len(rows))
		for n, row := range rows {
			lines[n] = activityLine(row) + " at " + row.CreatedAt.UTC().Format(time.RFC3339Nano)
		}
		when := at.UTC().Format(time.RFC3339Nano)
		want := []string{"accepted intake: pending → accepted (null → null) at " + when,
			fmt.Sprintf("updated state: Triage → Done (%s → %s) at %s", triage, w.done, when)}
		if n := len(lines); n < 2 || !slices.Equal(lines[n-2:], want) {
			t.Errorf("the activity %q; want it to end %q", lines, want)
		}
		waitForFanouts(t, w.pool)
		created, updated := fanoutOf(t, w.pool, "issue.created", accepted.ID), fanoutOf(t, w.pool, "issue.updated", accepted.ID)
		wandaName := displayNames(t, w, wanda)[0]
		shown := notificationdomain.IssueSnapshot{ID: accepted.ID, Name: "Accepted", SequenceID: accepted.SequenceID, Identifier: "WEB",
			StateName: "Triage", StateGroup: "triage"}
		submitted := w.sentOf(created, &bob, &bob, shown, sending{wanda, "assigned", -1, madeIt},
			sending{wanda, "assigned", assignedAt(t, created, wanda), assigning(&wandaName, &wanda)})
		shown.StateName, shown.StateGroup = "Done", "completed"
		state := changed("state", eventValue("Triage"), eventValue("Done"), &triage, &w.done)
		assertSent(t, w.pool, accepted.ID, []uuid.UUID{created.EventID, updated.EventID},
			slices.Concat(submitted, w.sentOf(updated, &alice, &bob, shown, sending{bob, "created", 0, state}, sending{wanda, "assigned", 0, state})))
		w.refusesAs(t, "alice", http.MethodPost, triagePath(accepted.ID), `{"status":"accepted"}`, http.StatusConflict,
			"issue.intake_transition_not_allowed")
	})
}
