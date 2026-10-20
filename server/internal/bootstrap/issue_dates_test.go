package bootstrap

import (
	"net/http"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// The dates as bootstrap wires them (M4 design 3.7, 3.13): alice creates a
// work item in Web from the 12th to the 19th of October. bob gives it a
// start date alone after the target date it keeps, the 1st of November (a
// later month, an earlier day), then a target date alone before the start
// date it keeps: each refused (422), at start_date and at target_date,
// nothing written. He moves both past the target date it had, then clears
// its target date: each written, each date changed a row of his activity
// (rows 8, 9). Then he gives it the dates it has, which changes nothing
// and writes no row. Each answer shows the dates as the database has them.
// A creation with its start date after its target date, the 1st of
// November and the 19th of October, is refused (422 at target_date) and
// takes no number; one with both on one day is made, with the next number.
func TestDatesAreKeptInOrder(t *testing.T) {
	w := newIssueWorld(t)
	item := w.newWorkItemIn(t, w.web, `{"name":"Fix","start_date":"2026-10-12","target_date":"2026-10-19"}`)
	path := "/api/v0/issues/" + item.String()
	for _, step := range []struct {
		body, refusal, stored string // refusal "" for a change answered 200
		writes                bool   // whether it writes the work item
	}{
		{`{"start_date":"2026-11-01"}`, "validation_failed start_date not_allowed: must not be after the target date", "2026-10-12 2026-10-19",
			false},
		{`{"target_date":"2026-10-11"}`, "validation_failed target_date not_allowed: must not be before the start date", "2026-10-12 2026-10-19",
			false},
		{`{"start_date":"2026-10-20","target_date":"2026-10-25"}`, "", "2026-10-20 2026-10-25", true},
		{`{"target_date":null}`, "", "2026-10-20 none", true},
		{`{"start_date":"2026-10-20","target_date":null}`, "", "2026-10-20 none", false},
	} {
		_, changed, by := archivalOf(t, w, item)
		if step.refusal != "" {
			answer := w.send(t, "bob", http.MethodPatch, path, step.body, http.StatusUnprocessableEntity)
			if got := refusalOf(t, []byte(answer), true); got != step.refusal {
				t.Errorf("PATCH %s answers %s, want %s", step.body, got, step.refusal)
			}
		} else if got := answeredDates(t, w.send(t, "bob", http.MethodPatch, path, step.body, http.StatusOK)); got != step.stored {
			t.Errorf("PATCH %s answers the dates %s, want %s", step.body, got, step.stored)
		}
		if got := w.datesOf(t, item); got != step.stored {
			t.Errorf("after PATCH %s the dates are %s, want %s", step.body, got, step.stored)
		}
		if _, at, who := archivalOf(t, w, item); (at.Equal(changed) && who == by) == step.writes {
			t.Errorf("after PATCH %s the work item was changed last by %s at %v, before it by %s at %v; want it written: %v", step.body, who,
				at, by, changed, step.writes)
		}
	}
	if said, want := w.activityBy(t, item, w.ids["bob"]), []string{"start_date: 2026-10-12 → 2026-10-20",
		"target_date: 2026-10-19 → 2026-10-25", "target_date: 2026-10-25 → "}; !slices.Equal(said, want) {
		t.Errorf("bob's activity says %q, want %q", said, want)
	}

	create := "/api/v0/projects/" + w.web.String() + "/issues"
	answer := w.send(t, "alice", http.MethodPost, create, `{"name":"Fix","start_date":"2026-11-01","target_date":"2026-10-19"}`,
		http.StatusUnprocessableEntity)
	if got, want := refusalOf(t, []byte(answer), true), "validation_failed target_date not_allowed: must not be before the start date"; got != want {
		t.Errorf("POST answers %s, want %s", got, want)
	}
	sameDay := w.newWorkItemIn(t, w.web, `{"name":"Fix","start_date":"2026-10-19","target_date":"2026-10-19"}`)
	var number int
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT sequence_id FROM issues WHERE id = $1", sameDay).Scan(&number); err != nil {
		t.Fatal(err)
	}
	if got := w.datesOf(t, sameDay); got != "2026-10-19 2026-10-19" || number != 2 {
		t.Errorf("the work item made on one day: from %s, number %d; want 2026-10-19 2026-10-19, number 2", got, number)
	}
}

// datesOf are the dates of the work item id as the database has them:
// "<start> <target>", each YYYY-MM-DD, or none.
func (w issueWorld) datesOf(t *testing.T, id uuid.UUID) string {
	t.Helper()
	var dates string
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT coalesce(to_char(start_date, 'YYYY-MM-DD'), 'none') || ' ' ||
		coalesce(to_char(target_date, 'YYYY-MM-DD'), 'none') FROM issues WHERE id = $1`, id).Scan(&dates); err != nil {
		t.Fatal(err)
	}
	return dates
}

// answeredDates are the dates of the work item an answer shows, as datesOf
// writes them.
func answeredDates(t *testing.T, answer string) string {
	t.Helper()
	var i storedIssue
	decodeAnswer(t, answer, &i)
	return i.dates()
}
