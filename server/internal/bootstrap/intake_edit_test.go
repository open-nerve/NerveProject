package bootstrap

import (
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	notificationriver "github.com/open-nerve/NerveProject/server/internal/modules/notification/adapter/river"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// intakeItemPath is the path the intake work item item is changed and
// deleted at.
func intakeItemPath(item uuid.UUID) string { return "/api/v0/intake-issues/" + item.String() }

// updateEffects are the rows an update of a work item writes besides the
// work item's own, as the tests compare them: the activity of its last
// change, each row "<field>: <old> → <new> (<old id> → <new id>)" in id
// order; the accounts it mentions and its subscribers, by id; its number
// of description versions; and the changes of its one issue.updated
// fan-out.
type updateEffects struct {
	activity               []string
	mentioned, subscribers []uuid.UUID
	versions               int
	changes                []notificationriver.FanoutChange
}

func (w issueWorld) effectsOf(t *testing.T, item uuid.UUID) updateEffects {
	t.Helper()
	rows, err := w.pool.Query(pgtest.Soon(t), `SELECT format('%s: %s → %s (%s → %s)', coalesce(a.field, 'null'), coalesce(a.old_value, 'null'),
		coalesce(a.new_value, 'null'), coalesce(a.old_identifier::text, 'null'), coalesce(a.new_identifier::text, 'null'))
		FROM issue_activities a JOIN issues i ON i.id = a.issue_id AND a.created_at = i.updated_at WHERE a.issue_id = $1 ORDER BY a.id`, item)
	if err != nil {
		t.Fatal(err)
	}
	e := updateEffects{changes: fanoutOf(t, w.pool, "issue.updated", item).Changes}
	if e.activity, err = pgx.CollectRows(rows, pgx.RowTo[string]); err != nil {
		t.Fatal(err)
	}
	e.mentioned = queryIDs(t, w.pool, "SELECT mention_id FROM issue_mentions WHERE issue_id = $1 AND deleted_at IS NULL ORDER BY mention_id", item)
	e.subscribers = queryIDs(t, w.pool, "SELECT subscriber_id FROM issue_subscribers WHERE issue_id = $1 AND deleted_at IS NULL ORDER BY subscriber_id", item)
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM issue_description_versions WHERE issue_id = $1 AND deleted_at IS NULL",
		item).Scan(&e.versions); err != nil {
		t.Fatal(err)
	}
	return e
}

// intakeRowDeletion is when the intake row of item was deleted, nil while
// it is not, and who changed it last.
func (w issueWorld) intakeRowDeletion(t *testing.T, item uuid.UUID) (*time.Time, uuid.UUID) {
	t.Helper()
	var at *time.Time
	var by uuid.UUID
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at, updated_by_id FROM intake_issues WHERE issue_id = $1", item).Scan(&at, &by); err != nil {
		t.Fatal(err)
	}
	return at, by
}

// refusedUnwritten has the account name send method path with body,
// checks it is refused with status and refusal (refusesAs), and that the
// work item item's rows and its intake row are as they were.
func (w issueWorld) refusedUnwritten(t *testing.T, item uuid.UUID, name, method, path, body string, status int, refusal string) {
	t.Helper()
	stamps := w.workItemStamps(t, item)
	w.refusesAs(t, name, method, path, body, status, refusal)
	if after := w.workItemStamps(t, item); !slices.Equal(after, stamps) {
		t.Errorf("%s %s as %s, refused, changed its rows:\n%q\nwant\n%q", method, path, name, after, stamps)
	}
	if at, _ := w.intakeRowDeletion(t, item); at != nil {
		t.Errorf("%s %s as %s, refused, deleted its intake row", method, path, name)
	}
}

// The content edit and the deletion of an intake work item through the
// app (M7 P2b spec 2.4, 2.5, 4). In Web, its intake turned on, wanda,
// acme's admin, is Web's admin, walt its guest whom it shows only his own
// work items. bob changes his submission as his own work item, which he
// created out of the intake, is changed: the same activity, mentions,
// subscriptions, description versions and issue.updated; it stays in the
// intake, his subscription answered. alice, Web's admin, changes his
// snoozed one, snoozed still, her not subscribing answered. walt changes
// his own's name and description, and is refused its priority, given even
// as it is. A work item of no known creator is refused bob and changed by
// alice. Accepted, declined and duplicate ones are closed to bob and to
// alice. bob, who did not submit alice's, is refused its change and its
// deletion; of her Ordinary, never submitted, its change is the one 404 a
// random id answers, and its deletion refused. bob deletes his pending one
// and his accepted one, alice his snoozed, declined, duplicate one and
// one accepted then archived; walt is refused his own's. The four not
// accepted are deleted as deleteIssue deletes a work item, their
// notifications with them; the two accepted stay as they were, their
// intake row alone deleted, out of the list. With Web's intake turned
// off, then Web archived, a change and a deletion are refused 409.
// updateIssue's refusal of a work item in the intake is
// TestAWorkItemInTheIntakeThroughTheApp's.
func TestChangingAndDeletingIntakeItemsThroughTheApp(t *testing.T) {
	w := newIssueWorld(t)
	alice, bob, wanda := w.ids["alice"], w.ids["bob"], w.ids["wanda"]
	w.makeMember(t, "walt", w.web, shared.RoleGuest)
	w.makeMember(t, "wanda", w.web, shared.RoleAdmin)
	triage := stateID(t, w.pool, w.web, "Triage")
	ordinary, twin := w.newWorkItemIn(t, w.web, `{"name":"Ordinary"}`), w.newWorkItemBy(t, "bob", w.web, `{"name":"Login breaks"}`)
	w.intakeOn(t)
	mended, alices, walts := w.submit(t, "bob", `{"name":"Login breaks"}`).ID, w.submit(t, "alice", `{"name":"Alice's"}`).ID,
		w.submit(t, "walt", `{"name":"Guessed"}`).ID
	orphan, late := w.submit(t, "bob", `{"name":"Orphan"}`).ID, w.submit(t, "bob", `{"name":"Late"}`).ID
	items := map[string]uuid.UUID{}
	for _, status := range []string{"pending", "snoozed", "declined", "duplicate", "accepted", "archived"} {
		items[status] = w.submit(t, "bob", fmt.Sprintf(`{"name":%q,"assignee_ids":[%q]}`, status, wanda)).ID
	}
	till := time.Now().Add(time.Hour).UTC().Truncate(time.Microsecond)
	w.triage(t, "alice", items["snoozed"], snoozedTill(till), http.StatusOK)
	w.triage(t, "alice", items["declined"], `{"status":"declined"}`, http.StatusOK)
	w.triage(t, "alice", items["duplicate"], duplicateOf(ordinary), http.StatusOK)
	w.triage(t, "alice", items["accepted"], `{"status":"accepted"}`, http.StatusOK)
	w.triage(t, "alice", items["archived"], `{"status":"accepted"}`, http.StatusOK)
	w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+items["archived"].String(), `{"state_id":"`+w.done.String()+`"}`, http.StatusOK)
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+items["archived"].String()+"/archive", "", http.StatusOK)
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE issues SET created_by_id = NULL WHERE id = $1", orphan); err != nil {
		t.Fatal(err)
	}

	t.Run("its submitter, as an ordinary update", func(t *testing.T) {
		html := "<p>See</p>" + mentionHTML(wanda)
		body := fmt.Sprintf(`{"name":"Login fails","description_html":%q,"priority":"high","label_ids":[%q],"assignee_ids":[%q],`+
			`"target_date":"2026-12-24"}`, html, w.bug, alice)
		raw := w.send(t, "bob", http.MethodPatch, intakeItemPath(mended), body, http.StatusOK)
		w.send(t, "bob", http.MethodPatch, "/api/v0/issues/"+twin.String(), body, http.StatusOK)
		var got struct {
			Name            string      `json:"name"`
			Priority        string      `json:"priority"`
			DescriptionHTML string      `json:"description_html"`
			StateID         uuid.UUID   `json:"state_id"`
			LabelIDs        []uuid.UUID `json:"label_ids"`
			AssigneeIDs     []uuid.UUID `json:"assignee_ids"`
			TargetDate      *string     `json:"target_date"`
		}
		decodeAnswer(t, raw, &got)
		if got.Name != "Login fails" || got.Priority != "high" || got.StateID != triage || !slices.Equal(got.LabelIDs, []uuid.UUID{w.bug}) ||
			!slices.Equal(got.AssigneeIDs, []uuid.UUID{alice}) || got.TargetDate == nil || *got.TargetDate != "2026-12-24" {
			t.Errorf("bob's change answers %s; want it changed, in Web's triage state", raw)
		}
		intakeIs(t, "bob's change", raw, &answeredIntake{Status: "pending"})
		if !subscribedIn(t, raw) {
			t.Errorf("bob's change answers him not subscribed; want subscribed, its creator")
		}
		e := w.effectsOf(t, mended)
		if ordinary := w.effectsOf(t, twin); !reflect.DeepEqual(e, ordinary) {
			t.Errorf("bob's change wrote %+v; want what his change of his own Login breaks out of the intake wrote, %+v", e, ordinary)
		}
		if v := savedVersion(t, w, mended); len(e.activity) == 0 || !slices.Equal(e.mentioned, []uuid.UUID{wanda}) || v.HTML != got.DescriptionHTML ||
			v.OwnedByID != bob {
			t.Errorf("bob's change wrote activity %q, mentions %v, the description version %+v; want each, his", e.activity, e.mentioned, v)
		}
		mentionedAt(t, fanoutOf(t, w.pool, "issue.updated", mended), wanda)
	})
	t.Run("the admin, a snooze on", func(t *testing.T) {
		raw := w.send(t, "alice", http.MethodPatch, intakeItemPath(items["snoozed"]), `{"priority":"low"}`, http.StatusOK)
		intakeIs(t, "alice's change", raw, &answeredIntake{Status: "snoozed", SnoozedTill: &till})
		rows := queryIDs(t, w.pool, "SELECT id FROM issue_subscribers WHERE issue_id = $1 AND subscriber_id = $2 AND deleted_at IS NULL",
			items["snoozed"], alice)
		if subscribedIn(t, raw) || len(rows) != 0 {
			t.Errorf("alice's change answers her subscribed %t, with %d subscriptions; want neither", subscribedIn(t, raw), len(rows))
		}
	})
	t.Run("a guest who submitted it", func(t *testing.T) {
		raw := w.send(t, "walt", http.MethodPatch, intakeItemPath(walts), `{"name":"Guessed again","description_html":"<p>Why</p>"}`, http.StatusOK)
		intakeIs(t, "walt's change", raw, &answeredIntake{Status: "pending"})
		for _, body := range []string{`{"priority":"high"}`, `{"name":"Guessed","priority":"none"}`} {
			w.refusedUnwritten(t, walts, "walt", http.MethodPatch, intakeItemPath(walts), body, http.StatusForbidden, "forbidden")
		}
	})
	t.Run("of no known creator", func(t *testing.T) {
		w.refusedUnwritten(t, orphan, "bob", http.MethodPatch, intakeItemPath(orphan), `{"name":"Mine"}`, http.StatusForbidden, "forbidden")
		w.refusedUnwritten(t, orphan, "bob", http.MethodDelete, intakeItemPath(orphan), "", http.StatusForbidden, "forbidden")
		w.send(t, "alice", http.MethodPatch, intakeItemPath(orphan), `{"name":"Adopted"}`, http.StatusOK)
	})
	t.Run("closed", func(t *testing.T) {
		for _, status := range []string{"accepted", "declined", "duplicate"} {
			for _, name := range []string{"bob", "alice"} {
				w.refusedUnwritten(t, items[status], name, http.MethodPatch, intakeItemPath(items[status]), `{"name":"Late"}`, http.StatusConflict,
					"issue.intake_closed")
			}
		}
	})
	t.Run("a member who did not submit it", func(t *testing.T) {
		w.refusedUnwritten(t, alices, "bob", http.MethodPatch, intakeItemPath(alices), `{"name":"Mine"}`, http.StatusForbidden, "forbidden")
		w.refusedUnwritten(t, alices, "bob", http.MethodDelete, intakeItemPath(alices), "", http.StatusForbidden, "forbidden")
		stamps := w.workItemStamps(t, ordinary)
		none := w.send(t, "bob", http.MethodPatch, intakeItemPath(uuid.NewV7()), `{"name":"Mine"}`, http.StatusNotFound)
		if got := w.send(t, "bob", http.MethodPatch, intakeItemPath(ordinary), `{"name":"Mine"}`, http.StatusNotFound); got != none ||
			refusalOf(t, []byte(got), false) != "issue.intake_not_found" {
			t.Errorf("bob's change of Ordinary = %s; want %s, a random id's, byte for byte", got, none)
		}
		w.refusesAs(t, "bob", http.MethodDelete, intakeItemPath(ordinary), "", http.StatusForbidden, "forbidden")
		if after := w.workItemStamps(t, ordinary); !slices.Equal(after, stamps) {
			t.Errorf("the refused writes changed Ordinary's rows:\n%q\nwant\n%q", after, stamps)
		}
	})
	t.Run("deletions", func(t *testing.T) {
		// Its rows are asserted after the deletions: no start-time run, nor
		// a fan-out, meanwhile (03 §5).
		waitForStartupRuns(t, w.pool)
		waitForFanouts(t, w.pool)
		stamps, notes := map[string][]string{}, map[string]string{}
		for _, status := range []string{"accepted", "archived"} {
			stamps[status], notes[status] = w.workItemStamps(t, items[status]), notificationsOf(t, w.pool, items[status])
		}
		w.refusedUnwritten(t, walts, "walt", http.MethodDelete, intakeItemPath(walts), "", http.StatusForbidden, "forbidden")
		deleter := map[string]string{"pending": "bob", "snoozed": "alice", "declined": "alice", "duplicate": "alice", "accepted": "bob",
			"archived": "alice"}
		for status, name := range deleter {
			w.send(t, name, http.MethodDelete, intakeItemPath(items[status]), "", http.StatusNoContent)
		}
		for _, status := range []string{"pending", "snoozed", "declined", "duplicate"} {
			item, by := items[status], deleter[status]
			at, rowBy := w.intakeRowDeletion(t, item)
			var deletedAt *time.Time
			if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at FROM issues WHERE id = $1", item).Scan(&deletedAt); err != nil ||
				deletedAt == nil || at == nil || !at.Equal(*deletedAt) || rowBy != w.ids[by] {
				t.Errorf("%s: deleted at %v, its intake row at %v by %s, %v; want both at one moment, by %s", status, deletedAt, at, rowBy, err, by)
			}
			for _, line := range w.workItemStamps(t, item) {
				if !strings.Contains(line, "changed by "+by+" changed, deleted changed") {
					t.Errorf("%s: %s; want it changed by %s and deleted at the deletion", status, line, by)
				}
			}
			if bad, all := unstampedNotifications(t, w.pool, item); bad != 0 || all == 0 {
				t.Errorf("%s: %d of its %d notifications not deleted with it; want each (issue.deleted)", status, bad, all)
			}
			w.send(t, "alice", http.MethodGet, "/api/v0/issues/"+item.String(), "", http.StatusNotFound)
		}
		listed := w.intakeOf(t, w.reading("alice"), "").ids()
		for _, status := range []string{"accepted", "archived"} {
			item, by := items[status], deleter[status]
			if at, rowBy := w.intakeRowDeletion(t, item); at == nil || rowBy != w.ids[by] {
				t.Errorf("%s: its intake row deleted at %v by %s; want deleted, by %s", status, at, rowBy, by)
			}
			if got := w.workItemStamps(t, item); !slices.Equal(got, stamps[status]) {
				t.Errorf("%s: its rows\n%q\nwant them as they were\n%q", status, got, stamps[status])
			}
			if got := notificationsOf(t, w.pool, item); got != notes[status] {
				t.Errorf("%s: its notifications\n%s\nwant them as they were\n%s", status, got, notes[status])
			}
			intakeIs(t, status+", read", w.send(t, "alice", http.MethodGet, "/api/v0/issues/"+item.String(), "", http.StatusOK), nil)
		}
		for status, item := range items {
			if slices.Contains(listed, item) {
				t.Errorf("%s is still listed in Web's intake", status)
			}
		}
	})
	t.Run("its intake turned off, then its project archived", func(t *testing.T) {
		w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"intake_view":false}`, http.StatusOK)
		for _, refusal := range []string{"issue.intake_disabled", "project.archived"} {
			if refusal == "project.archived" {
				w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/archive", `{}`, http.StatusOK)
			}
			for _, name := range []string{"bob", "alice"} {
				w.refusedUnwritten(t, late, name, http.MethodPatch, intakeItemPath(late), `{"name":"Later"}`, http.StatusConflict, refusal)
				w.refusedUnwritten(t, late, name, http.MethodDelete, intakeItemPath(late), "", http.StatusConflict, refusal)
			}
		}
	})
}
