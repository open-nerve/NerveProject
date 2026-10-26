package bootstrap

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// Seed through the unaffected membership/fixture paths. A faulty Upsert must
// reach the actual PATCH and corrupt B, rather than fail while making a row.
func TestIssuePreferencesWriteScope(t *testing.T) {
	w := newIssueWorld(t)
	ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Other preferences", "PREF")
	for n, user := range []string{"alice", "bob"} {
		priority := []string{`{"priority":["low"]}`, `{"priority":["medium"]}`}[n]
		tag, err := w.pool.Exec(pgtest.Soon(t), `UPDATE project_user_properties SET filters=$3::jsonb,sort_order=$4,display_filters=$5::jsonb,display_properties=$6::jsonb WHERE project_id=$1 AND user_id=$2 AND deleted_at IS NULL`,
			w.web, w.ids[user], priority, 17+n*12, customDisplay, customProperties)
		if err != nil || tag.RowsAffected() != 1 {
			t.Fatal("sentinel fixture", user, tag, err)
		}
	}
	beforeA, beforeB := preferenceRow(t, w, w.ids["alice"]), preferenceRow(t, w, w.ids["bob"])
	var beforeOther string
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT row_to_json(p)::text FROM project_user_properties p WHERE project_id=$1 AND user_id=$2 AND deleted_at IS NULL`, ops, w.ids["alice"]).Scan(&beforeOther); err != nil {
		t.Fatal(err)
	}
	path := "/api/v0/me/projects/" + w.web.String() + "/preferences"
	want := preferencesJSON("work_items", "[]", 17, `{"priority":["high"]}`, customDisplay, customProperties)
	samePreferencesJSON(t, w.send(t, "alice", http.MethodPatch, path, `{"filters":{"priority":["high"]}}`, 200), want)
	// This assertion is deliberately after the successful caller response,
	// and reads the sibling's actual row, not a GET or a captured parameter.
	if afterB := preferenceRow(t, w, w.ids["bob"]); afterB != beforeB {
		t.Errorf("caller PATCH returned 200 but changed bob's sentinel row:\n%s\nwant unchanged:\n%s", afterB, beforeB)
	}
	var a, old map[string]any
	if err := json.Unmarshal([]byte(preferenceRow(t, w, w.ids["alice"])), &a); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(beforeA), &old); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a["filters"], map[string]any{"priority": []any{"high"}}) || a["updated_by_id"] != w.ids["alice"].String() {
		t.Fatal("caller filters/audit not stored", a)
	}
	for _, key := range []string{"filters", "updated_by_id", "updated_at"} {
		delete(a, key)
		delete(old, key)
	}
	if !reflect.DeepEqual(a, old) {
		t.Fatal("caller patch changed an unrelated column", a, old)
	}
	var afterOther string
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT row_to_json(p)::text FROM project_user_properties p WHERE project_id=$1 AND user_id=$2 AND deleted_at IS NULL`, ops, w.ids["alice"]).Scan(&afterOther); err != nil {
		t.Fatal(err)
	}
	if afterOther != beforeOther {
		t.Fatal("caller patch changed another project", afterOther, beforeOther)
	}
	var count int
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT count(*) FROM project_user_properties WHERE project_id=$1 AND deleted_at IS NULL`, w.web).Scan(&count); err != nil || count != 2 {
		t.Fatal("write scope added/removed a row", count, err)
	}
}
