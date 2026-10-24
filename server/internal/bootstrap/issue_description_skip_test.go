package bootstrap

import (
	"fmt"
	"net/http"
	"slices"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/content"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

func TestSkipDescriptionPreservesAudit(t *testing.T) {
	w := newIssueWorld(t)
	initial := `<ul data-tight="true"><li><p>Task <a href="https://example.test/a">link</a><input type="checkbox"></p></li></ul>` + mentionHTML(w.ids["bob"])
	id := w.newWorkItemIn(t, w.web, fmt.Sprintf(`{"name":"Skip","description_html":%q}`, initial))
	audit := descriptionAudit(t, w, id)
	activities, versions := descriptionRows(t, w, "issue_activities", id), descriptionRows(t, w, "issue_description_versions", id)
	mentions, subscribers := descriptionRows(t, w, "issue_mentions", id), descriptionRows(t, w, "issue_subscribers", id)
	for _, html := range []string{
		`<ul data-id="list" data-tight="false"><li data-id="item"><p data-id="block">Task <a href="https://example.test/a">link</a><input type="checkbox"></p></li></ul>` + mentionHTML(w.ids["bob"]),
		`<ul><li><p><strong>Task</strong> <a href="https://example.test/a">link</a><input type="checkbox"></p></li></ul>` + mentionHTML(w.ids["bob"]),
	} {
		got := patchDescription(t, w, "bob", id, html, true)
		if got.DescriptionHTML != content.NewSanitizer().Clean(html).HTML {
			t.Fatal("migration HTML was not saved")
		}
		if audit != descriptionAudit(t, w, id) || activities != descriptionRows(t, w, "issue_activities", id) || versions != descriptionRows(t, w, "issue_description_versions", id) {
			t.Fatal("migration update left audit")
		}
		if mentions != descriptionRows(t, w, "issue_mentions", id) || subscribers != descriptionRows(t, w, "issue_subscribers", id) {
			t.Fatal("stable membership rewrote relation rows")
		}
	}
	// A project member becoming inactive changes only relation synchronization.
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE project_members SET is_active=false WHERE project_id=$1 AND member_id=$2", w.web, w.ids["bob"]); err != nil {
		t.Fatal(err)
	}
	patchDescription(t, w, "alice", id, initial, true)
	if got := queryIDs(t, w.pool, "SELECT id FROM issue_mentions WHERE issue_id=$1 AND deleted_at IS NULL", id); len(got) != 0 {
		t.Fatal("inactive mention retained")
	}
	if audit != descriptionAudit(t, w, id) || activities != descriptionRows(t, w, "issue_activities", id) || versions != descriptionRows(t, w, "issue_description_versions", id) {
		t.Fatal("membership sync changed issue audit")
	}
	if subscribers != descriptionRows(t, w, "issue_subscribers", id) {
		t.Fatal("removing mention unsubscribed account")
	}
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE project_members SET is_active=true WHERE project_id=$1 AND member_id=$2", w.web, w.ids["bob"]); err != nil {
		t.Fatal(err)
	}
	// A normal unchanged save must not repair relations or write anything.
	before := descriptionRows(t, w, "issue_mentions", id)
	patchDescription(t, w, "alice", id, initial, false)
	if before != descriptionRows(t, w, "issue_mentions", id) || audit != descriptionAudit(t, w, id) {
		t.Fatal("ordinary no-op repaired relations")
	}
	patchDescription(t, w, "alice", id, initial, true)
	if got := queryIDs(t, w.pool, "SELECT id FROM issue_mentions WHERE issue_id=$1 AND deleted_at IS NULL", id); len(got) != 1 {
		t.Fatal("skip did not re-add effective member")
	}
}

func TestSkipDescriptionCannotHideHrefOrChecked(t *testing.T) {
	for _, html := range []string{`<p>Task <a href="https://example.test/b">link</a><input type="checkbox"></p>`, `<p>Task <a href="https://example.test/a">link</a><input type="checkbox" checked></p>`} {
		t.Run(html, func(t *testing.T) {
			w := newIssueWorld(t)
			initial := `<p>Task <a href="https://example.test/a">link</a><input type="checkbox"></p>`
			id := w.newWorkItemIn(t, w.web, fmt.Sprintf(`{"name":"Identity","description_html":%q}`, initial))
			before := descriptionAudit(t, w, id)
			got := patchDescription(t, w, "bob", id, html, true)
			v := savedVersion(t, w, id)
			var at string
			var issueAt, versionAt bool
			if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT updated_at::text,updated_by_id=$2 FROM issues WHERE id=$1", id, w.ids["bob"]).Scan(&at, &issueAt); err != nil {
				t.Fatal(err)
			}
			if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT last_saved_at=(SELECT updated_at FROM issues WHERE id=$1) FROM issue_description_versions WHERE id=$2", id, v.ID).Scan(&versionAt); err != nil {
				t.Fatal(err)
			}
			var dynamic string
			if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT new_value FROM issue_activities WHERE issue_id=$1 AND field='description' ORDER BY created_at DESC,id DESC LIMIT 1", id).Scan(&dynamic); err != nil {
				t.Fatal(err)
			}
			if before == descriptionAudit(t, w, id) || !issueAt || !versionAt || dynamic != got.DescriptionHTML || v.HTML != got.DescriptionHTML {
				t.Fatal("identity change was suppressed", at, v, dynamic)
			}
		})
	}
}

func TestIssueDescriptionMembershipUsesVisibility(t *testing.T) {
	w := newIssueWorld(t)
	w.adminGuest(t)
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"guest_view_all_features":true}`, 200)
	id := w.newWorkItemIn(t, w.web, fmt.Sprintf(`{"name":"Visible","description_html":%q}`, mentionHTML(w.ids["walt"])))
	if got := queryIDs(t, w.pool, "SELECT subscriber_id FROM issue_subscribers WHERE issue_id=$1 ORDER BY subscriber_id", id); !slices.Contains(got, w.ids["walt"]) {
		t.Fatal("visible guest was not subscribed")
	}
}
