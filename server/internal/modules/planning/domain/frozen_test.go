package domain_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
)

// frozenDoc is a frozen progress as a cycle's progress_snapshot stores it: the API's FrozenProgress, by its field
// names (M6 P4 spec 2.3, 2.4).
const frozenDoc = `{"total_issues":3,"backlog_issues":0,"unstarted_issues":1,"started_issues":0,"completed_issues":1,` +
	`"cancelled_issues":1,` +
	`"assignees":[{"assignee_id":"0a000000-0000-7000-8000-000000000001","total_issues":1,"completed_issues":1,"pending_issues":0},` +
	`{"assignee_id":null,"total_issues":1,"completed_issues":0,"pending_issues":1}],` +
	`"labels":[{"label_id":"1a000000-0000-7000-8000-000000000001","total_issues":2,"completed_issues":1,"pending_issues":1}],` +
	`"burndown":{"start_date":"2026-11-01","end_date":"2026-11-02","scope":2,` +
	`"points":[{"date":"2026-11-01","remaining":2},{"date":"2026-11-02","remaining":null}]},` +
	`"frozen_at":"2026-11-10T08:30:00.123456Z"}`

// frozenValue is frozenDoc's value.
func frozenValue(t *testing.T) domain.Frozen {
	two := int64(2)
	return domain.Frozen{
		Progress: domain.Progress{
			Counts:    domain.Counts{Total: 3, Unstarted: 1, Completed: 1, Cancelled: 1},
			Assignees: []domain.AssigneeDistribution{{AssigneeID: &alice, Tally: tally(1, 1, 0)}, {Tally: tally(1, 0, 1)}},
			Labels:    []domain.LabelDistribution{{LabelID: &ui, Tally: tally(2, 1, 1)}},
			Burndown: &domain.Burndown{StartDate: *day(t, "2026-11-01"), EndDate: *day(t, "2026-11-02"), Scope: 2,
				Points: []domain.BurndownPoint{{Date: *day(t, "2026-11-01"), Remaining: &two}, {Date: *day(t, "2026-11-02")}}},
		},
		FrozenAt: time.Date(2026, 11, 10, 8, 30, 0, 123456000, time.UTC),
	}
}

// tree is a JSON document as its values, numbers as written.
func tree(t *testing.T, doc []byte) any {
	t.Helper()
	d := json.NewDecoder(bytes.NewReader(doc))
	d.UseNumber()
	var v any
	if err := d.Decode(&v); err != nil {
		t.Fatalf("%s: %v", doc, err)
	}
	return v
}

// A frozen progress is stored by the API's field names, dates as YYYY-MM-DD, an id or a remaining of none as null, and
// reads back as it was.
func TestFrozenEncoding(t *testing.T) {
	want := frozenValue(t)
	doc, err := domain.EncodeFrozen(want)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(tree(t, doc), tree(t, []byte(frozenDoc))) {
		t.Fatalf("encoded\n%s\nwant\n%s", doc, frozenDoc)
	}
	got, err := domain.DecodeFrozen([]byte(frozenDoc))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("decoded %+v\nwant %+v", got, want)
	}
}

// What ProgressOf computes reads back as it was, an empty progress too: its lists empty, not null, and no burndown.
func TestFrozenRoundTrip(t *testing.T) {
	full := spanned(t, "Asia/Shanghai", "2026-11-01", "2026-11-05", "2026-11-03",
		done(t, completed, "2026-11-02T04:00:00Z"), in(started, of(alice, bob), of(ui)), in(cancelled, nil, of(bug)))
	for _, tt := range []struct {
		name  string
		facts domain.ProgressFacts
	}{
		{"with work items and a burndown", full},
		{"empty", undated(t)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			progress, err := domain.ProgressOf(tt.facts)
			if err != nil {
				t.Fatal(err)
			}
			frozen := domain.Frozen{Progress: progress, FrozenAt: time.Date(2026, 11, 10, 8, 30, 0, 0, time.UTC)}
			doc, err := domain.EncodeFrozen(frozen)
			if err != nil {
				t.Fatal(err)
			}
			got, err := domain.DecodeFrozen(doc)
			if err != nil {
				t.Fatalf("%s: %v", doc, err)
			}
			if !reflect.DeepEqual(got, frozen) {
				t.Fatalf("decoded %+v\nwant %+v", got, frozen)
			}
		})
	}
	empty, err := domain.ProgressOf(undated(t))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := domain.EncodeFrozen(domain.Frozen{Progress: empty})
	if err != nil {
		t.Fatal(err)
	}
	fields := tree(t, doc).(map[string]any)
	if !reflect.DeepEqual(fields["assignees"], []any{}) || !reflect.DeepEqual(fields["labels"], []any{}) ||
		fields["burndown"] != nil {
		t.Fatalf("empty progress encoded as %s", doc)
	}
}

// countsOnly is frozenDoc without the distributions and the burndown, as a cycle's list row reads the column (M6 P4
// spec R5).
func countsOnly(t *testing.T) []byte {
	fields := tree(t, []byte(frozenDoc)).(map[string]any)
	for _, key := range []string{"assignees", "labels", "burndown"} {
		delete(fields, key)
	}
	doc, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

// A list row reads the six counts and the moment of a frozen progress, and only those: the whole document is refused,
// so that a list never reads the distributions and the burndown.
func TestDecodeFrozenCounts(t *testing.T) {
	got, err := domain.DecodeFrozenCounts(countsOnly(t))
	if err != nil {
		t.Fatal(err)
	}
	want := frozenValue(t)
	if !reflect.DeepEqual(got, domain.FrozenCounts{Counts: want.Counts, FrozenAt: want.FrozenAt}) {
		t.Fatalf("decoded %+v", got)
	}
	for _, tt := range []struct {
		name string
		edit func(map[string]any)
	}{
		{"with the assignees", func(d map[string]any) { d["assignees"] = []any{} }},
		{"an unknown key", func(d map[string]any) { d["note"] = "frozen by hand" }},
		{"a count missing", func(d map[string]any) { delete(d, "cancelled_issues") }},
		{"frozen_at missing", func(d map[string]any) { delete(d, "frozen_at") }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fields := tree(t, countsOnly(t)).(map[string]any)
			tt.edit(fields)
			doc, err := json.Marshal(fields)
			if err != nil {
				t.Fatal(err)
			}
			if got, err := domain.DecodeFrozenCounts(doc); err == nil {
				t.Fatalf("read %s as %+v", doc, got)
			}
		})
	}
	if got, err := domain.DecodeFrozenCounts([]byte(frozenDoc)); err == nil {
		t.Fatalf("read the whole document as %+v", got)
	}
}

// A stored frozen progress is the module's own record: a document that is not exactly as EncodeFrozen writes one is
// a defect, refused rather than read as zeros or as nothing (M6 P4 spec R1, R5).
func TestDecodeFrozenRefuses(t *testing.T) {
	burndown := func(d map[string]any) map[string]any { return d["burndown"].(map[string]any) }
	point := func(d map[string]any, i int) map[string]any { return burndown(d)["points"].([]any)[i].(map[string]any) }
	assignee := func(d map[string]any, i int) map[string]any { return d["assignees"].([]any)[i].(map[string]any) }
	for _, tt := range []struct {
		name string
		edit func(map[string]any)
	}{
		{"an unknown key", func(d map[string]any) { d["note"] = "frozen by hand" }},
		{"an unknown key in a row", func(d map[string]any) { assignee(d, 0)["name"] = "Alice" }},
		{"an unknown key in a point", func(d map[string]any) { point(d, 0)["ideal"] = 2 }},
		{"a count missing", func(d map[string]any) { delete(d, "backlog_issues") }},
		{"frozen_at missing", func(d map[string]any) { delete(d, "frozen_at") }},
		{"the burndown missing", func(d map[string]any) { delete(d, "burndown") }},
		{"the scope missing", func(d map[string]any) { delete(burndown(d), "scope") }},
		{"the id of none missing", func(d map[string]any) { delete(assignee(d, 1), "assignee_id") }},
		{"a remaining of none missing", func(d map[string]any) { delete(point(d, 1), "remaining") }},
		{"a key written in another case", func(d map[string]any) { d["Total_Issues"] = d["total_issues"]; delete(d, "total_issues") }},
		{"the assignees null", func(d map[string]any) { d["assignees"] = nil }},
		{"the labels null", func(d map[string]any) { d["labels"] = nil }},
		{"the points null", func(d map[string]any) { burndown(d)["points"] = nil }},
		{"a count as text", func(d map[string]any) { d["total_issues"] = "3" }},
		{"a count not whole", func(d map[string]any) { d["total_issues"] = 3.5 }},
		{"a day not as YYYY-MM-DD", func(d map[string]any) { burndown(d)["start_date"] = "2026-11-1" }},
		{"an id not a uuid", func(d map[string]any) { assignee(d, 0)["assignee_id"] = "alice" }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fields := tree(t, []byte(frozenDoc)).(map[string]any)
			tt.edit(fields)
			doc, err := json.Marshal(fields)
			if err != nil {
				t.Fatal(err)
			}
			if got, err := domain.DecodeFrozen(doc); err == nil {
				t.Fatalf("read %s as %+v", doc, got)
			}
		})
	}
	for _, doc := range []string{"null", "[]", "{}", frozenDoc + "{}", ""} {
		if got, err := domain.DecodeFrozen([]byte(doc)); err == nil {
			t.Fatalf("read %q as %+v", doc, got)
		}
	}
}
