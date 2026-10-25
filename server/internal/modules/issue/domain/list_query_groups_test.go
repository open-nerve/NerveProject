package domain

import (
	"encoding/json"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestParseListQueryGroups(t *testing.T) {
	id := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	cursor, err := EncodeListCursor(ListCursor{Order: "-created_at", Key: json.RawMessage(`"2026-01-01T00:00:00.000000Z"`), ID: id})
	if err != nil {
		t.Fatal(err)
	}
	// Exercise all presence combinations. The expected errors are literal business
	// precedence, independent of the parser's validation helpers.
	cases := []struct {
		g, s, k, sk, c bool
		field          string
	}{
		{false, false, false, false, false, ""}, {false, false, false, false, true, ""},
		{false, false, false, true, false, "sub_group_key"}, {false, false, false, true, true, "sub_group_key"},
		{false, false, true, false, false, "group_key"}, {false, false, true, false, true, "group_key"},
		{false, false, true, true, false, "group_key"}, {false, false, true, true, true, "group_key"},
		{false, true, false, false, false, "sub_group_by"}, {false, true, false, false, true, "sub_group_by"},
		{false, true, false, true, false, "sub_group_by"}, {false, true, false, true, true, "sub_group_by"},
		{false, true, true, false, false, "sub_group_by"}, {false, true, true, false, true, "sub_group_by"},
		{false, true, true, true, false, "sub_group_by"}, {false, true, true, true, true, "sub_group_by"},
		{true, false, false, false, false, ""}, {true, false, false, false, true, "cursor"},
		{true, false, false, true, false, "sub_group_key"}, {true, false, false, true, true, "sub_group_key"},
		{true, false, true, false, false, ""}, {true, false, true, false, true, ""},
		{true, false, true, true, false, "sub_group_key"}, {true, false, true, true, true, "sub_group_key"},
		{true, true, false, false, false, ""}, {true, true, false, false, true, "cursor"},
		{true, true, false, true, false, "sub_group_key"}, {true, true, false, true, true, "sub_group_key"},
		{true, true, true, false, false, "sub_group_key"}, {true, true, true, false, true, "cursor"},
		{true, true, true, true, false, ""}, {true, true, true, true, true, ""},
	}
	for _, tc := range cases {
		raw := map[string][]string{}
		if tc.g {
			raw["group_by"] = []string{"label_id"}
		}
		if tc.s {
			raw["sub_group_by"] = []string{"target_date"}
		}
		if tc.k {
			raw["group_key"] = []string{"none"}
		}
		if tc.sk {
			raw["sub_group_key"] = []string{"2026-01-01"}
		}
		if tc.c {
			raw["cursor"] = []string{cursor}
		}
		q, err := ParseListQuery(raw, ProjectList, id)
		if tc.field != "" {
			assertListProblem(t, err, shared.KindInvalid, tc.field, shared.FieldNotAllowed)
			continue
		}
		if err != nil {
			t.Fatalf("raw=%v err=%v", raw, err)
		}
		if (q.Cursor != nil) != tc.c || (q.GroupKey != nil) != tc.k || (q.SubGroupKey != nil) != tc.sk {
			t.Fatalf("raw=%v query=%+v", raw, q)
		}
	}
	for _, raw := range []map[string][]string{
		{"group_by": {"label_id"}, "sub_group_by": {"label_id"}},
		{"group_by": {"state_group"}, "group_key": {"none"}},
		{"group_by": {"target_date"}, "group_key": {"2026-01-01.."}},
		{"group_by": {"label_id"}, "group_key": {""}},
		{"group_by": {"priority"}, "group_key": {"none"}, "sub_group_by": {"assignee_id"}, "sub_group_key": {"me"}},
	} {
		_, err := ParseListQuery(raw, ProjectList, id)
		if err == nil {
			t.Fatalf("accepted %v", raw)
		}
	}
	q, err := ParseListQuery(map[string][]string{"label_id": {id.String()}, "group_by": {"label_id"}, "group_key": {"none"}, "target_date": {"2026-01-01"}}, ProjectList, id)
	if err != nil || len(q.Filters.Labels.IDs) != 1 || q.Filters.Labels.IDs[0] != id || q.Filters.Labels.None || len(q.Filters.Target) != 1 {
		t.Fatalf("group key replaced filters: %+v %v", q, err)
	}
	for _, raw := range []map[string][]string{{"cursor": {""}}, {"cursor": {"broken"}}, {"cursor": {cursor}, "order_by": {"priority"}}} {
		_, err := ParseListQuery(raw, ProjectList, id)
		assertListProblem(t, err, shared.KindBadRequest, "cursor", shared.FieldInvalidFormat)
	}
}
