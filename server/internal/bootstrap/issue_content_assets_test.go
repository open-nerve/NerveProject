package bootstrap

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
	"uuid"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
)

func issueImageHTML(ids ...uuid.UUID) string {
	html := "<p>Body</p>"
	for _, id := range ids {
		html += fmt.Sprintf(`<image-component src="%s"></image-component>`, id)
	}
	return html
}
func saveIssueImage(t *testing.T, w issueWorld, who string, item uuid.UUID, html string, skip bool, status int) string {
	t.Helper()
	return w.send(t, who, "PATCH", "/api/v0/issues/"+item.String(), fmt.Sprintf(`{"description_html":%q,"skip_activity":%t}`, html, skip), status)
}

func TestIssueContentAssets(t *testing.T) {
	t.Run("bind current and retain history and another author's bound image", func(t *testing.T) {
		w := newIssueWorld(t)
		item := w.newWorkItem(t)
		image := uploadedAssetHTTP(t, w, "bob", "ISSUE_DESCRIPTION", attachmentScope("issue", item), "body.png", "image/png", assetTestPNG(t))
		before, err := assetpg.New(w.pool).FindAsset(t.Context(), image.Asset.ID)
		if err != nil || before.BoundAt != nil {
			t.Fatal(before, err)
		}
		var result struct {
			UpdatedAt time.Time `json:"updated_at"`
		}
		decodeAnswer(t, saveIssueImage(t, w, "bob", item, issueImageHTML(image.Asset.ID), true, 200), &result)
		bound, err := assetpg.New(w.pool).FindAsset(t.Context(), image.Asset.ID)
		if err != nil || bound.BoundAt == nil || !bound.BoundAt.Equal(result.UpdatedAt) || bound.UpdatedByID == nil || *bound.UpdatedByID != w.ids["bob"] {
			t.Fatalf("content write did not bind at its one business moment: %+v %v", bound, err)
		}
		if !bound.CreatedAt.Equal(before.CreatedAt) || !reflect.DeepEqual(bound.Metadata, before.Metadata) {
			t.Fatal("binding changed origin or physical object")
		}
		saveIssueImage(t, w, "alice", item, issueImageHTML(image.Asset.ID, image.Asset.ID), true, 200)
		saveIssueImage(t, w, "alice", item, "<p>Removed image</p>", false, 200)
		saveIssueImage(t, w, "alice", item, issueImageHTML(image.Asset.ID), false, 200)
		after, err := assetpg.New(w.pool).FindAsset(t.Context(), image.Asset.ID)
		if err != nil || !reflect.DeepEqual(after, bound) {
			t.Fatal("history/reuse changed asset binding", err)
		}
		assetAPICall(t, w, "alice", "GET", "/api/v0/assets/"+image.Asset.ID.String(), "", 200)
		assetAPICall(t, w, "bob", "DELETE", "/api/v0/assets/"+image.Asset.ID.String(), "", 409)
	})
	for _, mode := range []string{"foreign preparation", "other issue", "wrong purpose", "pending", "zero ID", "create cannot take existing image", "create cannot take draft image"} {
		t.Run(mode, func(t *testing.T) {
			w := newIssueWorld(t)
			item := w.newWorkItem(t)
			host := item
			purpose := "ISSUE_DESCRIPTION"
			actor := "bob"
			if mode == "foreign preparation" {
				actor = "alice"
			}
			if mode == "other issue" {
				host = w.newWorkItem(t)
			}
			if mode == "wrong purpose" {
				purpose = "ISSUE_ATTACHMENT"
			}
			data := assetTestPNG(t)
			scope := attachmentScope("issue", host)
			if mode == "create cannot take draft image" {
				draft := createDraftHTTP(t, w, "bob", `{"payload":{}}`)
				scope, purpose = attachmentScope("draft", draft.ID), "DRAFT_ISSUE_DESCRIPTION"
			}
			prepared := prepareAssetHTTP(t, w, actor, purpose, scope, "private.png", "image/png", len(data))
			if mode != "pending" {
				putAssetHTTP(t, w, prepared, data, 204)
				assetAPICall(t, w, actor, "POST", "/api/v0/assets/"+prepared.Asset.ID.String()+"/complete", `{}`, 200)
			}
			id := prepared.Asset.ID
			if mode == "zero ID" {
				id = uuid.Nil()
			}
			status := 404
			if mode == "pending" {
				status = 409
			}
			before, err := assetpg.New(w.pool).FindAsset(t.Context(), prepared.Asset.ID)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "create cannot take existing image" || mode == "create cannot take draft image" {
				w.send(t, "bob", "POST", "/api/v0/projects/"+w.web.String()+"/issues", fmt.Sprintf(`{"name":"Must not be created","description_html":%q}`, issueImageHTML(id)), 404)
			} else {
				saveIssueImage(t, w, "bob", item, issueImageHTML(id), false, status)
			}
			after, err := assetpg.New(w.pool).FindAsset(t.Context(), prepared.Asset.ID)
			if err != nil || !reflect.DeepEqual(after, before) {
				t.Fatal("refused content changed asset", err)
			}
		})
	}
}

func TestIssueContentAssetsImageOrderIsNotMigration(t *testing.T) {
	w := newIssueWorld(t)
	id := w.newWorkItem(t)
	a := uploadedAssetHTTP(t, w, "alice", "ISSUE_DESCRIPTION", attachmentScope("issue", id), "a.png", "image/png", assetTestPNG(t))
	b := uploadedAssetHTTP(t, w, "alice", "ISSUE_DESCRIPTION", attachmentScope("issue", id), "b.png", "image/png", assetTestPNG(t))
	original := issueImageHTML(a.Asset.ID, b.Asset.ID)
	saveIssueImage(t, w, "alice", id, original, false, 200)
	before := descriptionAudit(t, w, id)
	versions := descriptionRows(t, w, "issue_description_versions", id)
	activity := descriptionRows(t, w, "issue_activities", id)
	migration := strings.Replace(original, "<p>", `<p data-id="body">`, 1)
	saveIssueImage(t, w, "alice", id, migration, true, 200)
	if descriptionAudit(t, w, id) != before || descriptionRows(t, w, "issue_description_versions", id) != versions || descriptionRows(t, w, "issue_activities", id) != activity {
		t.Fatal("block-ID migration changed issue audit, version or activity")
	}
	reordered := issueImageHTML(b.Asset.ID, a.Asset.ID)
	saveIssueImage(t, w, "alice", id, reordered, true, 200)
	if descriptionAudit(t, w, id) == before || descriptionRows(t, w, "issue_description_versions", id) == versions || descriptionRows(t, w, "issue_activities", id) == activity || savedVersion(t, w, id).HTML != reordered {
		t.Fatal("image order change was swallowed by skip_activity")
	}
	for _, image := range []uuid.UUID{a.Asset.ID, b.Asset.ID} {
		row, err := assetpg.New(w.pool).FindAsset(t.Context(), image)
		if err != nil || row.BoundAt == nil {
			t.Fatal("reorder lost binding", err)
		}
	}
}
