package domain

import (
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// utcAt is the instant of the day at 00:00 UTC, plus the days after it.
func utcAt(t *testing.T, day string, after int) *time.Time {
	t.Helper()
	d, err := shared.ParseDate(day)
	if err != nil {
		t.Fatal(err)
	}
	at := d.Time().AddDate(0, 0, after)
	return &at
}

// An intake list's query (M7 P2b spec 2.3): absent, every status, no
// filter, newest first, 50 rows; each filter as the project list reads its
// atoms (CSV, me, none), the statuses by their five names, and the two
// days as half-open UTC ranges, [the first day 00:00, the day after the
// last 00:00), an open end nil.
func TestParseIntakeQuery(t *testing.T) {
	actor := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	other := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	t.Run("defaults", func(t *testing.T) {
		q, err := ParseIntakeQuery(nil, actor)
		if err != nil || !reflect.DeepEqual(q, IntakeQuery{Order: "-created_at", Limit: 50}) {
			t.Fatalf("defaults = %+v, %v", q, err)
		}
	})
	t.Run("every filter", func(t *testing.T) {
		q, err := ParseIntakeQuery(map[string][]string{
			"status": {"pending,declined,duplicate"}, "priority": {"urgent,none"}, "label_id": {other.String() + ",none"},
			"assignee_id": {"me,none," + other.String()}, "created_by_id": {"me," + other.String()},
			"created_at": {"2026-11-01..2026-11-03"}, "updated_at": {"2026-11-05"}, "order_by": {"sequence_id"}, "limit": {"100"},
		}, actor)
		if err != nil {
			t.Fatal(err)
		}
		want := IntakeQuery{Order: "sequence_id", Limit: 100, Filters: IntakeFilters{
			Statuses: []IntakeStatus{IntakePending, IntakeDeclined, IntakeDuplicate}, Priorities: []Priority{"urgent", "none"},
			Labels: IDFilter{IDs: []uuid.UUID{other}, None: true}, Assignees: IDFilter{IDs: []uuid.UUID{actor, other}, None: true},
			CreatorIDs: []uuid.UUID{actor, other},
			Created:    InstantRange{From: utcAt(t, "2026-11-01", 0), To: utcAt(t, "2026-11-03", 1)},
			Updated:    InstantRange{From: utcAt(t, "2026-11-05", 0), To: utcAt(t, "2026-11-05", 1)},
		}}
		if !reflect.DeepEqual(q, want) {
			t.Fatalf("query = %+v\nwant %+v", q, want)
		}
	})
	// The range's ends are instants in UTC: the first day's midnight, in
	// it, and the midnight after the last day, out of it; an open end has
	// none. A closed range, or one that ends at its last day's midnight,
	// would leave that day out or let the next day's first instant in.
	t.Run("half-open UTC ranges", func(t *testing.T) {
		for value, want := range map[string]InstantRange{
			"2026-11-19":             {From: utcAt(t, "2026-11-19", 0), To: utcAt(t, "2026-11-20", 0)},
			"2026-12-31..2027-01-01": {From: utcAt(t, "2026-12-31", 0), To: utcAt(t, "2027-01-02", 0)},
			"2026-11-01..":           {From: utcAt(t, "2026-11-01", 0)},
			"..2026-02-28":           {To: utcAt(t, "2026-03-01", 0)},
			"..2028-02-28":           {To: utcAt(t, "2028-02-29", 0)},
		} {
			for _, field := range []string{"created_at", "updated_at"} {
				q, err := ParseIntakeQuery(map[string][]string{field: {value}}, actor)
				if err != nil {
					t.Fatalf("%s=%s: %v", field, value, err)
				}
				got := q.Filters.Created
				if field == "updated_at" {
					got = q.Filters.Updated
				}
				if !reflect.DeepEqual(got, want) {
					t.Errorf("%s=%s = %+v, want %+v", field, value, got, want)
				}
				for _, end := range []*time.Time{got.From, got.To} {
					if end != nil && (end.Location() != time.UTC || end.Hour() != 0 || end.Minute() != 0 || end.Nanosecond() != 0) {
						t.Errorf("%s=%s has an end %v, want a midnight in UTC", field, value, end)
					}
				}
			}
		}
	})
	t.Run("each order word", func(t *testing.T) {
		for _, order := range []IntakeOrder{"created_at", "-created_at", "updated_at", "-updated_at", "sequence_id", "-sequence_id"} {
			q, err := ParseIntakeQuery(map[string][]string{"order_by": {string(order)}}, actor)
			if err != nil || q.Order != order {
				t.Errorf("order_by=%s = %+v, %v", order, q, err)
			}
		}
	})
	// Each refusal is one 422 not_allowed on its parameter, as the other
	// lists answer; a limit that is no integer is the binding's 400, one
	// out of range the page size's 422.
	t.Run("refusals", func(t *testing.T) {
		for _, tc := range []struct {
			raw         map[string][]string
			field, code string
			kind        shared.Kind
		}{
			{map[string][]string{"status": {"open"}}, "status", shared.FieldNotAllowed, shared.KindInvalid},
			{map[string][]string{"status": {"Pending"}}, "status", shared.FieldNotAllowed, shared.KindInvalid},
			{map[string][]string{"status": {""}}, "status", shared.FieldNotAllowed, shared.KindInvalid},
			{map[string][]string{"status": {"pending,"}}, "status", shared.FieldNotAllowed, shared.KindInvalid},
			{map[string][]string{"status": {"pending", "declined"}}, "status", shared.FieldNotAllowed, shared.KindInvalid},
			{map[string][]string{"priority": {"huge"}}, "priority", shared.FieldNotAllowed, shared.KindInvalid},
			{map[string][]string{"label_id": {"me"}}, "label_id", shared.FieldNotAllowed, shared.KindInvalid},
			{map[string][]string{"assignee_id": {"nobody"}}, "assignee_id", shared.FieldNotAllowed, shared.KindInvalid},
			{map[string][]string{"created_by_id": {"none"}}, "created_by_id", shared.FieldNotAllowed, shared.KindInvalid},
			{map[string][]string{"created_at": {"none"}}, "created_at", shared.FieldNotAllowed, shared.KindInvalid},
			{map[string][]string{"updated_at": {"none"}}, "updated_at", shared.FieldNotAllowed, shared.KindInvalid},
			{map[string][]string{"created_at": {".."}}, "created_at", shared.FieldNotAllowed, shared.KindInvalid},
			{map[string][]string{"created_at": {"2026-11-03..2026-11-01"}}, "created_at", shared.FieldNotAllowed, shared.KindInvalid},
			{map[string][]string{"updated_at": {"2026-11-19T00:00:00Z"}}, "updated_at", shared.FieldNotAllowed, shared.KindInvalid},
			{map[string][]string{"created_at": {"2026-11-01", "2026-11-02"}}, "created_at", shared.FieldNotAllowed, shared.KindInvalid},
			{map[string][]string{"order_by": {"priority"}}, "order_by", shared.FieldNotAllowed, shared.KindInvalid},
			{map[string][]string{"order_by": {"+sequence_id"}}, "order_by", shared.FieldNotAllowed, shared.KindInvalid},
			{map[string][]string{"order_by": {"--created_at"}}, "order_by", shared.FieldNotAllowed, shared.KindInvalid},
			{map[string][]string{"state_id": {other.String()}}, "state_id", shared.FieldNotAllowed, shared.KindInvalid},
			{map[string][]string{"start_date": {"2026-11-01"}}, "start_date", shared.FieldNotAllowed, shared.KindInvalid},
			{map[string][]string{"archived": {"true"}}, "archived", shared.FieldNotAllowed, shared.KindInvalid},
			{map[string][]string{"group_by": {"priority"}}, "group_by", shared.FieldNotAllowed, shared.KindInvalid},
			{map[string][]string{"limit": {"0"}}, "limit", shared.FieldOutOfRange, shared.KindInvalid},
			{map[string][]string{"limit": {"101"}}, "limit", shared.FieldOutOfRange, shared.KindInvalid},
			{map[string][]string{"limit": {"ten"}}, "limit", shared.FieldInvalidFormat, shared.KindBadRequest},
			{map[string][]string{"cursor": {"x"}}, "cursor", shared.FieldInvalidFormat, shared.KindBadRequest},
		} {
			_, err := ParseIntakeQuery(tc.raw, actor)
			assertListProblem(t, err, tc.kind, tc.field, tc.code)
		}
	})
	t.Run("a cursor of the order asked for", func(t *testing.T) {
		at := time.Date(2026, 11, 19, 9, 0, 0, 123456000, time.UTC)
		newest, err := EncodeIntakeCursor(IntakeCursor{Order: "-created_at", At: at, ID: other})
		if err != nil {
			t.Fatal(err)
		}
		q, err := ParseIntakeQuery(map[string][]string{"cursor": {newest}}, actor)
		if err != nil || q.Cursor == nil || *q.Cursor != (IntakeCursor{Order: "-created_at", At: at, ID: other}) {
			t.Fatalf("cursor = %+v, %v", q.Cursor, err)
		}
		_, err = ParseIntakeQuery(map[string][]string{"cursor": {newest}, "order_by": {"created_at"}}, actor)
		assertListProblem(t, err, shared.KindBadRequest, "cursor", shared.FieldInvalidFormat)
	})
}

// sequence_id orders the intake list alone (DP-13): the project's and
// the member's lists, their cursors and a view's settings, which ask
// ListOrderAllowed, refuse it; nor do the lists take the intake list's
// status and day filters. The intake list takes none of the lists' other
// order words.
func TestTheIntakeVocabularyStaysItsOwn(t *testing.T) {
	actor := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	for _, order := range []string{"sequence_id", "-sequence_id"} {
		if ListOrderAllowed(ListOrder(order)) {
			t.Errorf("ListOrderAllowed(%q) = true; the project and member lists and view settings must refuse it", order)
		}
		for _, scope := range []ListScope{ProjectList, MemberList} {
			raw := map[string][]string{"order_by": {order}}
			if scope == MemberList {
				raw["relation"] = []string{"created"}
			}
			_, err := ParseListQuery(raw, scope, actor)
			assertListProblem(t, err, shared.KindInvalid, "order_by", shared.FieldNotAllowed)
		}
		if _, err := canonicalListKey(ListOrder(order), []byte("1")); err == nil {
			t.Errorf("canonicalListKey(%q) = nil error; a list cursor must not carry it", order)
		}
	}
	for _, field := range []string{"status", "created_at", "updated_at"} {
		if ListFilterAllowed(field, ProjectList) || ListFilterAllowed(field, MemberList) {
			t.Errorf("ListFilterAllowed(%q) = true; it filters the intake list alone", field)
		}
	}
	for _, order := range []IntakeOrder{"", "sort_order", "priority", "-priority", "start_date", "state_name", "sub_issues_count", "link_count"} {
		if IntakeOrderAllowed(order) {
			t.Errorf("IntakeOrderAllowed(%q) = true; the intake list orders by created_at, updated_at and sequence_id alone", order)
		}
	}
}
