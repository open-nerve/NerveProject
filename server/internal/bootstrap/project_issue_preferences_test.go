package bootstrap

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

const defaultDisplay = `{"calendar":{"layout":"month","show_weekends":false},"group_by":null,"layout":"list","order_by":"-created_at","show_empty_groups":true,"sub_group_by":null,"sub_issue":true}`
const defaultProperties = `{"assignee":true,"created_on":true,"due_date":true,"key":true,"labels":true,"link":true,"priority":true,"start_date":true,"state":true,"sub_issue_count":true,"updated_on":true}`
const defaultIssueSettings = `"display_filters":` + defaultDisplay + `,"display_properties":` + defaultProperties + `,"filters":{},`
const customDisplay = `{"layout":"calendar","group_by":"state_id","sub_group_by":"priority","order_by":"priority","sub_issue":false,"show_empty_groups":false,"calendar":{"layout":"week","show_weekends":true}}`
const customProperties = `{"assignee":false,"start_date":true,"due_date":false,"labels":true,"key":false,"priority":true,"state":false,"sub_issue_count":true,"link":false,"created_on":true,"updated_on":false}`
const customFilters = `{"priority":["high"],"label_id":[],"assignee_id":["me"],"start_date":["2026-10-01..2026-10-02","2026-10-04..2026-10-05"]}`

func preferencesJSON(tab, hidden string, sort int, filters, display, properties string) string {
	return fmt.Sprintf(`{"navigation":{"default_tab":%q,"hide_in_more_menu":%s},"sort_order":%d,"filters":%s,"display_filters":%s,"display_properties":%s}`, tab, hidden, sort, filters, display, properties)
}
func samePreferencesJSON(t *testing.T, got, want string) {
	t.Helper()
	var a, b any
	if err := json.Unmarshal([]byte(got), &a); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(want), &b); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("preferences = %s; want %s", got, want)
	}
}
func preferenceRow(t *testing.T, w issueWorld, user uuid.UUID) string {
	t.Helper()
	var row string
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT row_to_json(p)::text FROM project_user_properties p WHERE project_id=$1 AND user_id=$2 AND deleted_at IS NULL`, w.web, user).Scan(&row); err != nil {
		t.Fatal(err)
	}
	return row
}

func TestIssuePreferencesAPI(t *testing.T) { exerciseIssuePreferences(t, newIssueWorld(t)) }
func TestIssuePreferencesAsRuntimeRole(t *testing.T) {
	roles := newSplitRoles(t)
	exerciseIssuePreferences(t, newIssueWorldOn(t, roles.serverURL, roles.server, func(*config.Config) {}))
}
func exerciseIssuePreferences(t *testing.T, w issueWorld) {
	t.Helper()
	path := "/api/v0/me/projects/" + w.web.String() + "/preferences"
	defaults := preferencesJSON("work_items", "[]", 65535, `{}`, defaultDisplay, defaultProperties)
	// No API removes only a preference row; expose the missing-row read without removing membership.
	if _, err := w.pool.Exec(pgtest.Soon(t), `UPDATE project_user_properties SET deleted_at=now() WHERE project_id=$1 AND user_id=$2`, w.web, w.ids["bob"]); err != nil {
		t.Fatal(err)
	}
	samePreferencesJSON(t, w.send(t, "bob", http.MethodGet, path, "", 200), defaults)
	var rows int
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT count(*) FROM project_user_properties WHERE project_id=$1 AND user_id=$2 AND deleted_at IS NULL`, w.web, w.ids["bob"]).Scan(&rows); err != nil || rows != 0 {
		t.Fatalf("GET inserted row: %d/%v", rows, err)
	}
	ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	opsPath := "/api/v0/me/projects/" + ops.String() + "/preferences"
	alice := w.send(t, "alice", http.MethodPatch, path, preferencesJSON("views", "[]", 27, `{"priority":["low"]}`, defaultDisplay, customProperties), 200)
	aliceOps := w.send(t, "alice", http.MethodGet, opsPath, "", 200)
	custom := preferencesJSON("cycles", `["intake"]`, 17, customFilters, customDisplay, customProperties)
	samePreferencesJSON(t, w.send(t, "bob", http.MethodPatch, path, custom, 200), custom)
	samePreferencesJSON(t, w.send(t, "bob", http.MethodGet, path, "", 200), custom)
	// Each named top-level field replaces the old one; no deep merge.
	one := preferencesJSON("cycles", `["intake"]`, 17, `{"priority":["medium"]}`, customDisplay, customProperties)
	samePreferencesJSON(t, w.send(t, "bob", http.MethodPatch, path, `{"filters":{"priority":["medium"]}}`, 200), one)
	clear := preferencesJSON("cycles", `["intake"]`, 17, `{}`, customDisplay, customProperties)
	samePreferencesJSON(t, w.send(t, "bob", http.MethodPatch, path, `{"filters":{}}`, 200), clear)
	samePreferencesJSON(t, w.send(t, "bob", http.MethodPatch, path, `{}`, 200), clear)
	samePreferencesJSON(t, w.send(t, "bob", http.MethodPatch, path, custom, 200), custom)
	samePreferencesJSON(t, w.send(t, "alice", http.MethodGet, path, "", 200), alice)
	samePreferencesJSON(t, w.send(t, "alice", http.MethodGet, opsPath, "", 200), aliceOps)
	before := preferenceRow(t, w, w.ids["bob"])
	w.send(t, "bob", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/leave", "", 204)
	w.makeMember(t, "bob", w.web, 15)
	samePreferencesJSON(t, w.send(t, "bob", http.MethodGet, path, "", 200), custom)
	if got := preferenceRow(t, w, w.ids["bob"]); got != before {
		t.Fatalf("restore changed preferences row:\n%s\nwant %s", got, before)
	}
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/archive", "", 200)
	samePreferencesJSON(t, w.send(t, "bob", http.MethodGet, path, "", 200), custom)
	samePreferencesJSON(t, w.send(t, "bob", http.MethodPatch, path, `{"filters":{}}`, 200), clear)
	checkPreferenceWords(t, w, path)
	for _, order := range []string{"link_count", "-link_count"} {
		body := strings.Replace(custom, `"order_by":"priority"`, `"order_by":"`+order+`"`, 1)
		if body == custom {
			t.Fatal("custom order fixture was not replaced")
		}
		samePreferencesJSON(t, w.send(t, "bob", http.MethodPatch, path, body, 200), body)
		samePreferencesJSON(t, w.send(t, "bob", http.MethodGet, path, "", 200), body)
	}
	bad := `{"display_properties":` + strings.TrimSuffix(customProperties, "}") + `,"link_count":true}}`
	answer := draftRequest(t, w, "bob", http.MethodPatch, path, bad, 400)
	if oneError(t, []byte(answer)) != "display_properties.link_count not_allowed" {
		t.Fatal(answer)
	}
}

func checkPreferenceWords(t *testing.T, w issueWorld, path string) {
	t.Helper()
	before := preferenceRow(t, w, w.ids["bob"])
	// Valid structure alone must not permit an invalid filter atom.
	w.send(t, "bob", http.MethodPatch, path, `{"filters":{"label_id":["bad-uuid"]}}`, 422)
	// One request exercises deterministic aggregation across navigation and both new word-bearing fields.
	body := `{"navigation":{"default_tab":"views","hide_in_more_menu":["views","views"]},"filters":{"target_date":["none","none"],"label_id":["bad-uuid","none,me"],"start_date":["2026-02-30"]},"display_filters":{"layout":"future","group_by":"cycle_id","sub_group_by":"module_id","order_by":"-sort_order","sub_issue":false,"show_empty_groups":false,"calendar":{"layout":"year","show_weekends":false}}}`
	answer := w.send(t, "bob", http.MethodPatch, path, body, 422)
	var problem struct {
		Code   string
		Errors []struct{ Field, Code string }
	}
	decodeAnswer(t, answer, &problem)
	got := []string{}
	for _, e := range problem.Errors {
		got = append(got, e.Field+" "+e.Code)
	}
	want := []string{"navigation.hide_in_more_menu[1] duplicate", "filters.label_id[0] not_allowed", "filters.label_id[1] not_allowed", "filters.start_date[0] not_allowed", "filters.target_date[0] not_allowed", "filters.target_date[1] not_allowed", "display_filters.layout not_allowed", "display_filters.group_by not_allowed", "display_filters.sub_group_by not_allowed", "display_filters.order_by not_allowed", "display_filters.calendar.layout not_allowed"}
	if problem.Code != "validation_failed" || !reflect.DeepEqual(got, want) {
		t.Fatalf("errors %s; want %v", answer, want)
	}
	if after := preferenceRow(t, w, w.ids["bob"]); after != before {
		t.Fatal("invalid words changed stored preferences")
	}
}
