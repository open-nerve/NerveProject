package httpadapter_test

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

const displayDefaultJSON = `{"calendar":{"layout":"month","show_weekends":false},"group_by":null,"layout":"list","order_by":"-created_at","show_empty_groups":true,"sub_group_by":null,"sub_issue":true}`
const propertiesDefaultJSON = `{"assignee":true,"created_on":true,"due_date":true,"key":true,"labels":true,"link":true,"priority":true,"start_date":true,"state":true,"sub_issue_count":true,"updated_on":true}`
const issueDefaultsJSON = `"display_filters":` + displayDefaultJSON + `,"display_properties":` + propertiesDefaultJSON + `,"filters":{},`

func TestIssuePreferencesHTTPValues(t *testing.T) {
	p := domain.DefaultPreferences()
	p.Filters = domain.Filters{LabelID: []string{}, Priority: []string{"low"}}
	p.DisplayFilters.GroupBy = ptr("priority")
	p.DisplayFilters.SubIssue = false
	p.DisplayProperties.Link = false
	prefs := &fakePreferences{answer: p}
	h := newServer(t, fakes{prefs: prefs})
	path := "/api/v0/me/projects/" + webID.String() + "/preferences"
	display := `{"layout":"calendar","group_by":null,"sub_group_by":null,"order_by":"name","sub_issue":false,"show_empty_groups":false,"calendar":{"layout":"week","show_weekends":true}}`
	properties := `{"assignee":false,"start_date":true,"due_date":false,"labels":true,"key":false,"priority":true,"state":false,"sub_issue_count":true,"link":false,"created_on":true,"updated_on":false}`
	body := `{"filters":{"state_id":["bad-uuid"],"state_group":["future"],"priority":["future"],"label_id":[],"assignee_id":["me"],"created_by_id":["me"],"mention_id":["me"],"subscriber_id":["me"],"start_date":["bad-date"],"target_date":["none"]},"display_filters":` + display + `,"display_properties":` + properties + `}`
	for _, b := range []string{body, `{"filters":{}}`, `{}`} {
		res, answer := do(t, h, request(http.MethodPatch, path, "bob", b))
		if res.StatusCode != 200 {
			t.Fatalf("%s: %d %s", b, res.StatusCode, answer)
		}
		var got map[string]json.RawMessage
		if err := json.Unmarshal([]byte(answer), &got); err != nil {
			t.Fatal(err)
		}
		if string(got["filters"]) != `{"label_id":[],"priority":["low"]}` {
			t.Fatalf("filters: %s", answer)
		}
		var d map[string]any
		if err := json.Unmarshal(got["display_filters"], &d); err != nil {
			t.Fatal(err)
		}
		var props map[string]bool
		if err := json.Unmarshal(got["display_properties"], &props); err != nil {
			t.Fatal(err)
		}
		if d["group_by"] != "priority" || d["sub_group_by"] != nil || d["sub_issue"] != false || props["link"] != false || len(props) != 11 {
			t.Fatal(answer)
		}
	}
	wantFilters := domain.Filters{StateID: []string{"bad-uuid"}, StateGroup: []string{"future"}, Priority: []string{"future"}, LabelID: []string{}, AssigneeID: []string{"me"}, CreatedByID: []string{"me"}, MentionID: []string{"me"}, SubscriberID: []string{"me"}, StartDate: []string{"bad-date"}, TargetDate: []string{"none"}}
	wantDisplay := domain.DisplayFilters{Layout: "calendar", OrderBy: "name", Calendar: domain.DisplayCalendar{Layout: "week", ShowWeekends: true}}
	wantProps := domain.DisplayProperties{StartDate: true, Labels: true, Priority: true, SubIssueCount: true, CreatedOn: true}
	want := []domain.PreferencesPatch{{Filters: &wantFilters, DisplayFilters: &wantDisplay, DisplayProperties: &wantProps}, {Filters: &domain.Filters{}}, {}}
	if !reflect.DeepEqual(prefs.got, want) {
		t.Fatalf("patches %+v; want %+v", prefs.got, want)
	}
}

func TestIssuePreferencesHTTPStructure(t *testing.T) {
	prefs := &fakePreferences{answer: domain.DefaultPreferences()}
	h := newServer(t, fakes{prefs: prefs})
	path := "/api/v0/me/projects/" + webID.String() + "/preferences"
	bad := []string{}
	for _, key := range []string{"filters", "display_filters", "display_properties"} {
		for _, v := range []string{`null`, `[]`, `"x"`, `1`, `true`} {
			bad = append(bad, `{"`+key+`":`+v+`}`)
		}
	}
	for _, v := range []string{`null`, `"x"`, `1`, `true`, `{}`, `[1]`, `[null]`, `[true]`, `[{}]`, `[["x"]]`} {
		bad = append(bad, `{"filters":{"label_id":`+v+`}}`)
	}
	bad = append(bad, `{"filters":{"archived":[]}}`, `{"filters":{"project_id":[]}}`, `{"filters":{"sub_issue":[]}}`)
	for key, raw := range map[string]string{"display_filters": displayDefaultJSON, "display_properties": propertiesDefaultJSON} {
		var obj map[string]json.RawMessage
		if err := json.Unmarshal([]byte(raw), &obj); err != nil {
			t.Fatal(err)
		}
		wrap := func() {
			b, err := json.Marshal(map[string]any{key: obj})
			if err != nil {
				t.Fatal(err)
			}
			bad = append(bad, string(b))
		}
		for field, v := range obj {
			delete(obj, field)
			wrap()
			obj[field] = json.RawMessage(`42`)
			wrap()
			obj[field] = v
		}
		obj["extra"] = json.RawMessage(`false`)
		wrap()
	}
	for _, calendar := range []string{`null`, `[]`, `"x"`, `{}`, `{"layout":"month"}`, `{"show_weekends":false}`, `{"layout":false,"show_weekends":false}`, `{"layout":"month","show_weekends":null}`, `{"layout":"month","show_weekends":false,"extra":true}`} {
		var obj map[string]json.RawMessage
		if err := json.Unmarshal([]byte(displayDefaultJSON), &obj); err != nil {
			t.Fatal(err)
		}
		obj["calendar"] = json.RawMessage(calendar)
		b, err := json.Marshal(map[string]any{"display_filters": obj})
		if err != nil {
			t.Fatal(err)
		}
		bad = append(bad, string(b))
	}
	for _, body := range bad {
		res, answer := do(t, h, request(http.MethodPatch, path, "bob", body))
		if res.StatusCode != 400 {
			t.Errorf("%s = %d %s", body, res.StatusCode, answer)
		}
	}
	if len(prefs.calls) != 0 {
		t.Fatalf("invalid structure reached use case: %v", prefs.calls)
	}
}
