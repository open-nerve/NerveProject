package shared_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The defaults, as JSON, are project_user_properties' column defaults
// (00026, 00038) character for character: a page answers them before its
// caller has a row, and stores them under his first change.
func TestDefaultListDisplaySettings(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value any
		want  string
	}{
		{"display filters", shared.DefaultListDisplayFilters(),
			`{"layout":"list","group_by":null,"sub_group_by":null,"order_by":"-created_at","sub_issue":true,"show_empty_groups":true,"calendar":{"layout":"month","show_weekends":false}}`},
		{"display properties", shared.DefaultListDisplayProperties(),
			`{"assignee":true,"start_date":true,"due_date":true,"labels":true,"key":true,"priority":true,"state":true,"sub_issue_count":true,"link":true,"attachment_count":true,"created_on":true,"updated_on":true}`},
	} {
		got, err := json.Marshal(tc.value)
		if err != nil || string(got) != tc.want {
			t.Errorf("%s = %s, %v; want %s", tc.name, got, err, tc.want)
		}
	}
}

// A nil slice is an omitted key and an empty one a key that filters
// nothing, both in JSON and in the map the settings port takes.
func TestListFiltersKeepOmittedApartFromEmpty(t *testing.T) {
	if m := (shared.ListFilters{}).Map(); m == nil || len(m) != 0 {
		t.Fatalf("no keys: %#v, want an empty map", m)
	}
	f := shared.ListFilters{Priority: []string{"low"}, LabelID: []string{}}
	m := f.Map()
	if want := map[string][]string{"priority": {"low"}, "label_id": {}}; !reflect.DeepEqual(m, want) || m["label_id"] == nil {
		t.Fatalf("Map() = %#v, want %#v", m, want)
	}
	if b, err := json.Marshal(f); err != nil || string(b) != `{"priority":["low"],"label_id":[]}` {
		t.Fatalf("JSON = %s, %v", b, err)
	}
	var read shared.ListFilters
	if err := json.Unmarshal([]byte(`{"label_id":[]}`), &read); err != nil || read.LabelID == nil || len(read.LabelID) != 0 || read.Priority != nil {
		t.Fatalf("read %#v, %v", read, err)
	}
	// Every key, by the name JSON gives it.
	all := shared.ListFilters{StateID: []string{"1"}, StateGroup: []string{"2"}, Priority: []string{"3"}, LabelID: []string{"4"}, AssigneeID: []string{"5"},
		CreatedByID: []string{"6"}, MentionID: []string{"7"}, SubscriberID: []string{"8"}, StartDate: []string{"9"}, TargetDate: []string{"10"}}
	b, err := json.Marshal(all)
	if err != nil {
		t.Fatal(err)
	}
	var byJSON map[string][]string
	if err := json.Unmarshal(b, &byJSON); err != nil || len(byJSON) != 10 || !reflect.DeepEqual(all.Map(), byJSON) {
		t.Fatalf("Map() = %v; JSON %s", all.Map(), b)
	}
}
