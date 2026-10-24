package bootstrap

import (
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/content"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

func descriptionRows(t *testing.T, w issueWorld, table string, id uuid.UUID) string {
	t.Helper()
	var text string
	condition := "issue_id=$1"
	if table == "issues" {
		condition = "id=$1"
	}
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT coalesce(jsonb_agg(to_jsonb(r) ORDER BY id),'[]')::text FROM "+table+" r WHERE "+condition, id).Scan(&text); err != nil {
		t.Fatal(err)
	}
	return text
}
func descriptionAudit(t *testing.T, w issueWorld, id uuid.UUID) string {
	t.Helper()
	var text string
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT (to_jsonb(i)-'description_html'-'description_stripped')::text FROM issues i WHERE id=$1", id).Scan(&text); err != nil {
		t.Fatal(err)
	}
	return text
}
func savedVersion(t *testing.T, w issueWorld, id uuid.UUID) domain.DescriptionVersion {
	t.Helper()
	var v domain.DescriptionVersion
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT id,owned_by_id,created_by_id,updated_by_id,description_html,coalesce(description_stripped,''),last_saved_at,created_at,updated_at FROM issue_description_versions WHERE issue_id=$1 AND deleted_at IS NULL ORDER BY last_saved_at DESC,id DESC LIMIT 1`, id).Scan(&v.ID, &v.OwnedByID, &v.CreatedByID, &v.UpdatedByID, &v.HTML, &v.Text, &v.LastSavedAt, &v.CreatedAt, &v.UpdatedAt); err != nil {
		t.Fatal(err)
	}
	return v
}
func patchDescription(t *testing.T, w issueWorld, actor string, id uuid.UUID, html string, skip bool) domain.Issue {
	t.Helper()
	var got struct {
		HTML string `json:"description_html"`
	}
	body := fmt.Sprintf(`{"description_html":%q}`, html)
	if skip {
		body = fmt.Sprintf(`{"description_html":%q,"skip_activity":true}`, html)
	}
	decodeAnswer(t, w.send(t, actor, http.MethodPatch, "/api/v0/issues/"+id.String(), body, 200), &got)
	return domain.Issue{DescriptionHTML: got.HTML}
}

func TestDescriptionWriteIsAtomic(t *testing.T) {
	w := newIssueWorld(t)
	w.adminGuest(t)
	html := mentionHTML(w.ids["walt"], w.ids["bob"], w.ids["bob"], w.ids["olga"]) + `<mention-component entity_name="user_mention" entity_identifier="bad"></mention-component><script>bad()</script>`
	clean := content.NewSanitizer().Clean(html)
	id := w.newWorkItemIn(t, w.web, fmt.Sprintf(`{"name":"Atomic","description_html":%q}`, html))
	version := savedVersion(t, w, id)
	var stored, text string
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT description_html,description_stripped FROM issues WHERE id=$1", id).Scan(&stored, &text); err != nil {
		t.Fatal(err)
	}
	if stored != clean.HTML || version.HTML != clean.HTML || text != clean.Text || version.Text != clean.Text {
		t.Fatal("saved bodies differ", stored, version)
	}
	if version.OwnedByID != w.ids["alice"] || version.CreatedByID != w.ids["alice"] || version.UpdatedByID != w.ids["alice"] {
		t.Fatal("version writer", version)
	}
	mentions := queryIDs(t, w.pool, "SELECT mention_id FROM issue_mentions WHERE issue_id=$1 AND deleted_at IS NULL ORDER BY mention_id", id)
	want := []uuid.UUID{w.ids["walt"], w.ids["bob"]}
	slices.SortFunc(want, uuid.UUID.Compare)
	if !slices.Equal(mentions, want) {
		t.Fatal("mentions did not filter members and retain hidden guests", mentions, want)
	}
	subscribers := queryIDs(t, w.pool, "SELECT subscriber_id FROM issue_subscribers WHERE issue_id=$1 AND deleted_at IS NULL ORDER BY subscriber_id", id)
	want = []uuid.UUID{w.ids["alice"], w.ids["bob"]}
	slices.SortFunc(want, uuid.UUID.Compare)
	if !slices.Equal(subscribers, want) {
		t.Fatal("hidden mention subscribed", subscribers, want)
	}
	var kept string
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT to_jsonb(m)::text FROM issue_mentions m WHERE issue_id=$1 AND mention_id=$2", id, w.ids["bob"]).Scan(&kept); err != nil {
		t.Fatal(err)
	}
	old := queryIDs(t, w.pool, "SELECT id FROM issue_mentions WHERE issue_id=$1 AND mention_id=$2 AND deleted_at IS NULL", id, w.ids["walt"])[0]
	patchDescription(t, w, "alice", id, mentionHTML(w.ids["bob"]), false)
	var after string
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT to_jsonb(m)::text FROM issue_mentions m WHERE issue_id=$1 AND mention_id=$2", id, w.ids["bob"]).Scan(&after); err != nil || kept != after {
		t.Fatal("retained mention rewritten", err)
	}
	if got := queryIDs(t, w.pool, "SELECT id FROM issue_mentions WHERE id=$1 AND deleted_at IS NOT NULL", old); len(got) != 1 {
		t.Fatal("removed mention was not soft deleted")
	}
	patchDescription(t, w, "alice", id, mentionHTML(w.ids["bob"], w.ids["walt"]), false)
	if got := queryIDs(t, w.pool, "SELECT id FROM issue_mentions WHERE issue_id=$1 AND mention_id=$2 AND deleted_at IS NULL", id, w.ids["walt"]); len(got) != 1 || got[0] == old {
		t.Fatal("mention readd reused row", got)
	}
	before := descriptionRows(t, w, "issue_mentions", id)
	versions := descriptionRows(t, w, "issue_description_versions", id)
	w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+id.String(), fmt.Sprintf(`{"state_id":%q}`, w.done), 200)
	if before != descriptionRows(t, w, "issue_mentions", id) || versions != descriptionRows(t, w, "issue_description_versions", id) {
		t.Fatal("state-only patch touched description rows")
	}
	// A descriptor is not an automatic subscriber just because they edit.
	other := w.newWorkItemIn(t, w.web, `{"name":"Non-subscriber"}`)
	patchDescription(t, w, "bob", other, "<p>Edited by Bob</p>", false)
	if got := queryIDs(t, w.pool, "SELECT id FROM issue_subscribers WHERE issue_id=$1 AND subscriber_id=$2", other, w.ids["bob"]); len(got) != 0 {
		t.Fatal("description editor automatically subscribed")
	}
	// Failure at the final version statement rolls back every prior table.
	tables := []string{"issues", "issue_assignees", "issue_labels", "issue_subscribers", "issue_mentions", "issue_activities", "issue_description_versions"}
	snapshot := map[string]string{}
	for _, table := range tables {
		snapshot[table] = descriptionRows(t, w, table, other)
	}
	if _, err := w.pool.Exec(pgtest.Soon(t), `ALTER TABLE issue_description_versions ADD CONSTRAINT reject_description CHECK (description_html NOT LIKE '%Rejected%')`); err != nil {
		t.Fatal(err)
	}
	failure := fmt.Sprintf(`{"name":"Changed","description_html":%q,"assignee_ids":[%q],"label_ids":[%q]}`, "<p>Rejected</p>"+mentionHTML(w.ids["walt"]), w.ids["bob"], w.bug)
	w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+other.String(), failure, 500)
	for _, table := range tables {
		if snapshot[table] != descriptionRows(t, w, table, other) {
			t.Fatal("version failure did not roll back", table)
		}
	}
	count := len(queryIDs(t, w.pool, "SELECT id FROM issues WHERE project_id=$1", w.web))
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/issues", `{"name":"Fail create","description_html":"<p>Rejected</p>"}`, 500)
	if got := len(queryIDs(t, w.pool, "SELECT id FROM issues WHERE project_id=$1", w.web)); got != count {
		t.Fatal("failed create committed issue")
	}
}

func TestIssueDescriptionVersionsSlideFromLastSave(t *testing.T) {
	w := newIssueWorld(t)
	id := w.newWorkItemIn(t, w.web, `{"name":"Sliding"}`)
	v := savedVersion(t, w, id)
	if v.HTML != "<p></p>" {
		t.Fatal("default creation lacks initial version", v)
	}
	// Creation is long ago, while the last save is recent: only a sliding clock merges.
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE issue_description_versions SET created_at=$2,last_saved_at=$3 WHERE id=$1", v.ID, time.Now().Add(-time.Hour), time.Now().Add(-500*time.Second)); err != nil {
		t.Fatal(err)
	}
	before := savedVersion(t, w, id)
	patchDescription(t, w, "alice", id, "<p>Same owner recent save</p>", false)
	after := savedVersion(t, w, id)
	if before.ID != after.ID || !after.CreatedAt.Equal(before.CreatedAt) || after.CreatedByID != before.CreatedByID || !after.LastSavedAt.After(before.LastSavedAt) || !after.UpdatedAt.Equal(after.LastSavedAt) {
		t.Fatal("sliding save replaced version or origin", before, after)
	}
	patchDescription(t, w, "bob", id, "<p>Another owner</p>", false)
	other := savedVersion(t, w, id)
	if other.ID == after.ID || other.OwnedByID != w.ids["bob"] || other.CreatedByID != w.ids["bob"] || other.UpdatedByID != w.ids["bob"] {
		t.Fatal("new writer did not own new version", other)
	}
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE issue_description_versions SET last_saved_at=$2 WHERE id=$1", other.ID, time.Now().Add(-601*time.Second)); err != nil {
		t.Fatal(err)
	}
	// Keep Bob's expired row latest: an owner-only policy must fail this case.
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE issue_description_versions SET last_saved_at=$2 WHERE issue_id=$1 AND id<>$3", id, time.Now().Add(-time.Hour), other.ID); err != nil {
		t.Fatal(err)
	}
	latest := savedVersion(t, w, id)
	if latest.ID != other.ID || latest.OwnedByID != w.ids["bob"] || time.Since(latest.LastSavedAt) <= 600*time.Second {
		t.Fatal("window test requires Bob's own expired version to be latest", latest)
	}
	patchDescription(t, w, "bob", id, "<p>Outside window</p>", false)
	if got := savedVersion(t, w, id); got.ID == other.ID {
		t.Fatal("expired version merged")
	}
	// A legacy issue does not get a fabricated initial snapshot on migration saves.
	if _, err := w.pool.Exec(pgtest.Soon(t), "DELETE FROM issue_description_versions WHERE issue_id=$1", id); err != nil {
		t.Fatal(err)
	}
	patchDescription(t, w, "bob", id, `<p data-id="migration">Outside window</p>`, true)
	if got := queryIDs(t, w.pool, "SELECT id FROM issue_description_versions WHERE issue_id=$1", id); len(got) != 0 {
		t.Fatal("migration fabricated legacy version")
	}
	patchDescription(t, w, "bob", id, "<p>First real legacy save</p>", false)
	if got := queryIDs(t, w.pool, "SELECT id FROM issue_description_versions WHERE issue_id=$1", id); len(got) != 1 {
		t.Fatal("first real save should insert one version")
	}
}
