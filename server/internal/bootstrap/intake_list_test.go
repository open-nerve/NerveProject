package bootstrap

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// intakeRow is a row of a project's intake list as the tests read it: its
// id, its parent and its number of sub-items as its reader is shown them,
// when it was archived, and its intake row as answered, raw.
type intakeRow struct {
	ID             uuid.UUID       `json:"id"`
	ParentID       *uuid.UUID      `json:"parent_id"`
	SubIssuesCount int             `json:"sub_issues_count"`
	ArchivedAt     *time.Time      `json:"archived_at"`
	Intake         json.RawMessage `json:"intake"`
}

// intakePage is a page of a project's intake list (IntakeIssuePage).
type intakePage struct {
	Total      int64       `json:"total"`
	Data       []intakeRow `json:"data"`
	NextCursor *string     `json:"next_cursor"`
}

// ids are the page's work items, in its order.
func (p intakePage) ids() []uuid.UUID {
	ids := make([]uuid.UUID, len(p.Data))
	for n, row := range p.Data {
		ids[n] = row.ID
	}
	return ids
}

// row is the page's row of the work item id, which it must list.
func (p intakePage) row(t *testing.T, id uuid.UUID) intakeRow {
	t.Helper()
	for _, row := range p.Data {
		if row.ID == id {
			return row
		}
	}
	t.Fatalf("the page %v does not list %s", p.ids(), id)
	return intakeRow{}
}

// reader is an account's GET of a path, answered 200 and checked against
// the contract: through the app (reading), or through the issue module
// wired as bootstrap wires it on a test clock (readingOn).
type reader func(t *testing.T, path string) string

func (w issueWorld) reading(name string) reader {
	return func(t *testing.T, path string) string {
		t.Helper()
		return w.send(t, name, http.MethodGet, path, "", http.StatusOK)
	}
}

func (w issueWorld) readingOn(route moduleRoute, name string) reader {
	return func(t *testing.T, path string) string {
		t.Helper()
		return route.answerWithin(t, w.contract, http.MethodGet, path, w.ids[name], "", http.StatusOK)
	}
}

// intakeOf is the page read answers of Web's intake list with query: its
// three keys, and a list of rows, never null.
func (w issueWorld) intakeOf(t *testing.T, read reader, query string) intakePage {
	t.Helper()
	raw := read(t, intakePath(w.web)+"?"+query)
	listKeys(t, []byte(raw), "total", "data", "next_cursor")
	var page intakePage
	decodeAnswer(t, raw, &page)
	if page.Data == nil {
		t.Fatalf("the intake list %s: data must be a list", raw)
	}
	return page
}

// listsIntake checks read lists Web's intake with query on one page: want,
// in its order, its total theirs, without a cursor; and returns the page.
func (w issueWorld) listsIntake(t *testing.T, read reader, query string, want ...uuid.UUID) intakePage {
	t.Helper()
	page := w.intakeOf(t, read, query)
	if got := page.ids(); !slices.Equal(got, want) || page.Total != int64(len(want)) || page.NextCursor != nil {
		t.Errorf("?%s lists %v (total %d, cursor %v); want %v", query, got, page.Total, page.NextCursor, want)
	}
	return page
}

// walksIntake checks read pages through Web's intake list with query, limit
// rows a page, each page's cursor to the next: want, in its order, each
// page totalling every one of them, and the last, alone, without a cursor.
func (w issueWorld) walksIntake(t *testing.T, read reader, query string, limit int, want []uuid.UUID) {
	t.Helper()
	q, err := url.ParseQuery(query)
	if err != nil {
		t.Fatal(err)
	}
	q.Set("limit", strconv.Itoa(limit))
	var got []uuid.UUID
	pages := max(1, (len(want)+limit-1)/limit)
	for n := 1; ; n++ {
		page := w.intakeOf(t, read, q.Encode())
		if page.Total != int64(len(want)) || len(page.Data) > limit {
			t.Fatalf("?%s, page %d: %d rows, total %d; want at most %d, total %d", query, n, len(page.Data), page.Total, limit, len(want))
		}
		got = append(got, page.ids()...)
		if page.NextCursor == nil || n == pages {
			if page.NextCursor != nil || n != pages {
				t.Fatalf("?%s ends on page %d, its cursor given %t; want it to end, without one, on page %d", query, n, page.NextCursor != nil, pages)
			}
			break
		}
		q.Set("cursor", *page.NextCursor)
	}
	if !slices.Equal(got, want) {
		t.Errorf("?%s pages through %v; want %v", query, got, want)
	}
}

// intakesAsDetailed checks each row of page carries the intake row read's
// detail of its work item carries (getIssue), value for value: the same
// JSON.
func intakesAsDetailed(t *testing.T, read reader, page intakePage) {
	t.Helper()
	for _, row := range page.Data {
		var detail map[string]json.RawMessage
		decodeAnswer(t, read(t, "/api/v0/issues/"+row.ID.String()), &detail)
		var listed, detailed bytes.Buffer
		if json.Compact(&listed, row.Intake) != nil || json.Compact(&detailed, detail["intake"]) != nil || listed.String() != detailed.String() {
			t.Errorf("%s is listed with the intake %s, its detail %s; want the same", row.ID, row.Intake, detail["intake"])
		}
	}
}

// rowIntakeIs checks the row carries the intake row want (intakeIs).
func rowIntakeIs(t *testing.T, what string, row intakeRow, want *answeredIntake) {
	t.Helper()
	intakeIs(t, what, fmt.Sprintf(`{"intake":%s}`, row.Intake), want)
}

// H9's list through the app (M7 P2b spec 2.3, 4). Web, its intake turned
// on, has alice's Target, Parent and Child, out of its intake, and in it,
// in the order of their numbers: alice's Snoozed, urgent, labelled Bug,
// assigned to bob, snoozed till an hour on; bob's Pending, high; walt's
// Guessed, walt being Web's guest whom it shows only his own work items,
// a duplicate of Target, which he does not see; bob's Declined; alice's
// Accepted, accepted, then in Done and archived; walt's Adopted,
// accepted, which alice puts under Parent and over Child; then alice's
// Last of the day and The next day, submitted through the issue module on
// a clock at 2001-02-03 23:59:59.999999 UTC, then a microsecond on; and
// last, bob changes Pending. alice, Web's admin, and bob, its member, list
// all of them, newest first; walt his own. Each order pages through them,
// three a page, each page's total all of them; each filter lists those it
// selects, alternatives of a filter as one, filters together; each day
// condition takes the last microsecond of its last day, and not the next
// midnight, and none is refused. Through the issue module on a clock, a
// microsecond before Snoozed's time, it is snoozed; at its time, pending,
// listed and filtered so. Each row's intake is its detail's, Guessed
// naming Target to alice and nothing to walt; Accepted and Adopted are
// listed, archived or not; to walt, Adopted has no parent and no sub-item.
func TestListingTheIntakeThroughTheApp(t *testing.T) {
	w := newIssueWorld(t)
	alice, bob := w.ids["alice"], w.ids["bob"]
	w.makeMember(t, "walt", w.web, shared.RoleGuest)
	target, parent, child := w.newWorkItemIn(t, w.web, `{"name":"Target"}`), w.newWorkItemIn(t, w.web, `{"name":"Parent"}`),
		w.newWorkItemIn(t, w.web, `{"name":"Child"}`)
	w.intakeOn(t)
	snoozed := w.submit(t, "alice", fmt.Sprintf(`{"name":"Snoozed","priority":"urgent","label_ids":[%q],"assignee_ids":[%q]}`, w.bug, bob)).ID
	pending := w.submit(t, "bob", `{"name":"Pending","priority":"high"}`).ID
	guessed := w.submit(t, "walt", `{"name":"Guessed"}`).ID
	declined := w.submit(t, "bob", `{"name":"Declined"}`).ID
	accepted, adopted := w.submit(t, "alice", `{"name":"Accepted"}`).ID, w.submit(t, "walt", `{"name":"Adopted"}`).ID
	c := clockAt(time.Date(2001, 2, 3, 23, 59, 59, 999999000, time.UTC))
	route := newIssueRouteWithClock(t, w.pool, postgres.NewTxManager(w.pool, 2*time.Second), c)
	onTheClock := func(name string) uuid.UUID {
		var s submission
		decodeAnswer(t, route.answerWithin(t, w.contract, http.MethodPost, intakePath(w.web), alice, `{"name":"`+name+`"}`, http.StatusCreated), &s)
		return s.ID
	}
	lastOfDay := onTheClock("Last of the day")
	c.advance(time.Microsecond)
	nextDay := onTheClock("The next day")
	till := time.Now().Add(time.Hour).UTC().Truncate(time.Microsecond)
	w.triage(t, "alice", snoozed, snoozedTill(till), http.StatusOK)
	w.triage(t, "alice", guessed, duplicateOf(target), http.StatusOK)
	w.triage(t, "alice", declined, `{"status":"declined"}`, http.StatusOK)
	w.triage(t, "alice", accepted, `{"status":"accepted"}`, http.StatusOK)
	w.triage(t, "alice", adopted, `{"status":"accepted"}`, http.StatusOK)
	w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+accepted.String(), `{"state_id":"`+w.done.String()+`"}`, http.StatusOK)
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+accepted.String()+"/archive", "", http.StatusOK)
	w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+adopted.String(), `{"parent_id":"`+parent.String()+`"}`, http.StatusOK)
	w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+child.String(), `{"parent_id":"`+adopted.String()+`"}`, http.StatusOK)
	w.send(t, "bob", http.MethodPatch, "/api/v0/intake-issues/"+pending.String(), `{"name":"Pending again"}`, http.StatusOK)

	bySequence := []uuid.UUID{snoozed, pending, guessed, declined, accepted, adopted, lastOfDay, nextDay}
	byCreation := []uuid.UUID{lastOfDay, nextDay, snoozed, pending, guessed, declined, accepted, adopted}
	byChange := []uuid.UUID{lastOfDay, nextDay, snoozed, guessed, declined, accepted, adopted, pending}
	backwards := func(ids []uuid.UUID) []uuid.UUID {
		r := slices.Clone(ids)
		slices.Reverse(r)
		return r
	}
	newestFirst := backwards(byCreation)
	// among are the work items of set, newest first, as a list's default
	// order lists them.
	among := func(set ...uuid.UUID) []uuid.UUID {
		var in []uuid.UUID
		for _, id := range newestFirst {
			if slices.Contains(set, id) {
				in = append(in, id)
			}
		}
		return in
	}
	everyone := among(byCreation...)
	asAlice := w.reading("alice")

	t.Run("each reader", func(t *testing.T) {
		for _, name := range []string{"alice", "bob"} {
			intakesAsDetailed(t, w.reading(name), w.listsIntake(t, w.reading(name), "", everyone...))
		}
		intakesAsDetailed(t, w.reading("walt"), w.listsIntake(t, w.reading("walt"), "", adopted, guessed))
		w.walksIntake(t, w.reading("walt"), "order_by=sequence_id", 1, []uuid.UUID{guessed, adopted})
	})
	t.Run("each order", func(t *testing.T) {
		for order, want := range map[string][]uuid.UUID{"created_at": byCreation, "updated_at": byChange, "sequence_id": bySequence} {
			w.walksIntake(t, asAlice, "order_by="+order, 3, want)
			w.walksIntake(t, asAlice, "order_by=-"+order, 3, backwards(want))
		}
		w.walksIntake(t, asAlice, "", 3, everyone)
	})
	t.Run("each filter", func(t *testing.T) {
		for _, c := range []struct {
			name, query string
			want        []uuid.UUID
		}{
			{"alice", "status=pending", among(pending, lastOfDay, nextDay)}, {"alice", "status=snoozed", among(snoozed)},
			{"alice", "status=accepted", among(accepted, adopted)}, {"alice", "status=declined", among(declined)},
			{"alice", "status=duplicate", among(guessed)}, {"alice", "status=pending,snoozed", among(pending, snoozed, lastOfDay, nextDay)},
			{"walt", "status=duplicate,declined", among(guessed)},
			{"alice", "priority=urgent", among(snoozed)}, {"alice", "priority=urgent,high", among(snoozed, pending)},
			{"alice", "priority=none", among(guessed, declined, accepted, adopted, lastOfDay, nextDay)},
			{"alice", "label_id=" + w.bug.String(), among(snoozed)},
			{"alice", "label_id=none", among(pending, guessed, declined, accepted, adopted, lastOfDay, nextDay)},
			{"alice", "assignee_id=" + bob.String(), among(snoozed)}, {"bob", "assignee_id=me", among(snoozed)},
			{"alice", "assignee_id=none", among(pending, guessed, declined, accepted, adopted, lastOfDay, nextDay)},
			{"alice", "created_by_id=" + bob.String(), among(pending, declined)}, {"walt", "created_by_id=me", among(guessed, adopted)},
			{"alice", "created_by_id=" + w.ids["walt"].String() + "," + bob.String(), among(pending, guessed, declined, adopted)},
			{"bob", "status=pending&created_by_id=me", among(pending)},
			{"alice", "status=accepted&priority=none&label_id=none&assignee_id=none", among(accepted, adopted)},
		} {
			w.listsIntake(t, w.reading(c.name), c.query, c.want...)
		}
		w.walksIntake(t, asAlice, "status=pending,snoozed", 3, among(pending, snoozed, lastOfDay, nextDay))
	})
	t.Run("the days", func(t *testing.T) {
		later := among(snoozed, pending, guessed, declined, accepted, adopted, nextDay)
		for query, want := range map[string][]uuid.UUID{
			"created_at=..2001-02-03": {lastOfDay}, "created_at=2001-02-03": {lastOfDay}, "created_at=2001-02-04": {nextDay},
			"created_at=2001-02-03..2001-02-04": {nextDay, lastOfDay}, "created_at=2001-02-04..": later,
			"updated_at=2001-02-03": {lastOfDay}, "updated_at=..2001-02-04": {nextDay, lastOfDay}, "updated_at=2001-02-04..": later,
		} {
			w.listsIntake(t, asAlice, query, want...)
		}
		for _, field := range []string{"created_at", "updated_at"} {
			w.refusesAs(t, "alice", http.MethodGet, intakePath(w.web)+"?"+field+"=none", "", http.StatusUnprocessableEntity,
				"validation_failed "+field+" not_allowed")
		}
	})
	t.Run("a snooze whose time has come", func(t *testing.T) {
		c := clockAt(till.Add(-time.Microsecond))
		read := w.readingOn(newIssueRouteWithClock(t, w.pool, postgres.NewTxManager(w.pool, 2*time.Second), c), "alice")
		w.listsIntake(t, read, "status=pending", among(pending, lastOfDay, nextDay)...)
		page := w.listsIntake(t, read, "status=snoozed", snoozed)
		rowIntakeIs(t, "Snoozed a microsecond before its time", page.row(t, snoozed), &answeredIntake{Status: "snoozed", SnoozedTill: &till})
		intakesAsDetailed(t, read, page)
		c.advance(time.Microsecond)
		w.listsIntake(t, read, "status=snoozed")
		page = w.listsIntake(t, read, "status=pending", among(pending, snoozed, lastOfDay, nextDay)...)
		rowIntakeIs(t, "Snoozed at its time", page.row(t, snoozed), &answeredIntake{Status: "pending"})
		intakesAsDetailed(t, read, page)
	})
	t.Run("a duplicate's target as its reader sees it", func(t *testing.T) {
		rowIntakeIs(t, "Guessed to alice", w.intakeOf(t, asAlice, "").row(t, guessed), &answeredIntake{Status: "duplicate", DuplicateToID: &target})
		rowIntakeIs(t, "Guessed to walt", w.intakeOf(t, w.reading("walt"), "").row(t, guessed), &answeredIntake{Status: "duplicate"})
	})
	t.Run("accepted, archived or not, trimmed as the project list", func(t *testing.T) {
		page := w.intakeOf(t, asAlice, "status=accepted")
		if row := page.row(t, accepted); row.ArchivedAt == nil {
			t.Errorf("Accepted is listed unarchived; want it archived")
		}
		for _, item := range []uuid.UUID{accepted, adopted} {
			rowIntakeIs(t, "an acceptance", page.row(t, item), &answeredIntake{Status: "accepted"})
		}
		if row := page.row(t, adopted); row.ParentID == nil || *row.ParentID != parent || row.SubIssuesCount != 1 {
			t.Errorf("alice lists Adopted under %v with %d sub-items; want under Parent, %s, with Child", row.ParentID, row.SubIssuesCount, parent)
		}
		if row := w.intakeOf(t, w.reading("walt"), "status=accepted").row(t, adopted); row.ParentID != nil || row.SubIssuesCount != 0 {
			t.Errorf("walt lists Adopted under %v with %d sub-items; want neither, both alice's", row.ParentID, row.SubIssuesCount)
		}
	})
}
