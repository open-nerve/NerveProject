package bootstrap

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"uuid"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	assetdomain "github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func draftAsset(t *testing.T, w issueWorld, draft uuid.UUID, image bool) preparedAssetAnswer {
	t.Helper()
	purpose, name, mime, data := "DRAFT_ISSUE_ATTACHMENT", "file.txt", "text/plain", []byte("draft bytes")
	if image {
		purpose, name, mime, data = "DRAFT_ISSUE_DESCRIPTION", "image.png", "image/png", assetTestPNG(t)
	}
	return uploadedAssetHTTP(t, w, "alice", purpose, attachmentScope("draft", draft), name, mime, data)
}

func draftPayloadBody(t *testing.T, payload map[string]any) string {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"payload": payload})
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func draftAssetSnapshot(t *testing.T, w issueWorld) map[string]string {
	t.Helper()
	out := publishSnapshot(t, w)
	var rows string
	if err := w.pool.QueryRow(t.Context(), "SELECT coalesce(jsonb_agg(to_jsonb(r) ORDER BY id),'[]'::jsonb)::text FROM file_assets r").Scan(&rows); err != nil {
		t.Fatal(err)
	}
	out["file_assets"] = rows
	return out
}

// Saving partial content binds its references without normalizing the stored
// JSON or enforcing publication's business values. Project moves include even
// unreferenced and pending files, in both UUID directions and through null.
func TestDraftAssetsSaveAndMove(t *testing.T) {
	w := newIssueWorld(t)
	d := createDraftHTTP(t, w, "alice", `{"payload":{}}`)
	image := draftAsset(t, w, d.ID, true)
	file := draftAsset(t, w, d.ID, false)
	pending := prepareAssetHTTP(t, w, "alice", "DRAFT_ISSUE_ATTACHMENT", attachmentScope("draft", d.ID), "later.txt", "text/plain", 5)
	html := issueImageHTML(image.Asset.ID) + "<script>discard only when publishing</script>"
	payload := map[string]any{"description_html": html, "priority": "invalid"}
	var saved draftHTTP
	decodeAnswer(t, draftRequest(t, w, "alice", "PATCH", draftPath(d.ID), draftPayloadBody(t, payload), 200), &saved)
	if !reflect.DeepEqual(saved.Payload, payload) {
		t.Fatal("save normalized partial payload", saved.Payload)
	}
	row, err := assetpg.New(w.pool).FindAsset(t.Context(), image.Asset.ID)
	if err != nil || row.BoundAt == nil || !row.BoundAt.Equal(saved.UpdatedAt) {
		t.Fatal("partial save did not bind its image", row, err)
	}
	before := draftAssetSnapshot(t, w)
	draftRequest(t, w, "alice", "PATCH", draftPath(d.ID), `{"payload":{"description_html":3}}`, 400)
	if !reflect.DeepEqual(before, draftAssetSnapshot(t, w)) {
		t.Fatal("structural rejection wrote rows")
	}
	other := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	for _, project := range []*uuid.UUID{&w.web, &other, &w.web, nil} {
		projectJSON := "null"
		want := uuid.Nil()
		if project != nil {
			projectJSON, want = fmt.Sprintf("%q", *project), *project
		}
		decodeAnswer(t, draftRequest(t, w, "alice", "PATCH", draftPath(d.ID), `{"project_id":`+projectJSON+`}`, 200), &saved)
		for _, id := range []uuid.UUID{image.Asset.ID, file.Asset.ID, pending.Asset.ID} {
			got, err := assetpg.New(w.pool).FindAsset(t.Context(), id)
			if err != nil || got.Placement.ProjectID != want || got.Placement.DraftID != d.ID || got.Placement.Scope.Kind != assetdomain.ScopeDraft || !got.UpdatedAt.Equal(saved.UpdatedAt) || got.UpdatedByID == nil || *got.UpdatedByID != w.ids["alice"] {
				t.Fatal("move omitted a file or used another audit", got, err)
			}
		}
	}
	putAssetHTTP(t, w, pending, []byte("later"), 204)
	assetAPICall(t, w, "alice", "POST", completeAssetPath(pending.Asset.ID), `{}`, 200)
	_, actual := readAssetHTTP(t, w, "alice", pending.Asset.ID)
	if string(actual) != "later" {
		t.Fatal("moved pending upload could not complete")
	}
}

func TestDraftAssetsRejectForeignAndUnfinishedReferences(t *testing.T) {
	for _, mode := range []string{"other draft", "pending", "wrong purpose", "missing"} {
		t.Run(mode, func(t *testing.T) {
			w := newIssueWorld(t)
			d := createDraftHTTP(t, w, "alice", `{"payload":{}}`)
			good := draftAsset(t, w, d.ID, true)
			host, purpose := d.ID, "DRAFT_ISSUE_DESCRIPTION"
			if mode == "other draft" {
				host = createDraftHTTP(t, w, "alice", `{"payload":{}}`).ID
			}
			if mode == "wrong purpose" {
				purpose = "DRAFT_ISSUE_ATTACHMENT"
			}
			data := assetTestPNG(t)
			bad := prepareAssetHTTP(t, w, "alice", purpose, attachmentScope("draft", host), "bad.png", "image/png", len(data))
			if mode != "pending" {
				putAssetHTTP(t, w, bad, data, 204)
				assetAPICall(t, w, "alice", "POST", completeAssetPath(bad.Asset.ID), `{}`, 200)
			}
			badID, status := bad.Asset.ID, 404
			if mode == "missing" {
				badID = uuid.NewV7()
			}
			if mode == "pending" {
				status = 409
			}
			body := draftPayloadBody(t, map[string]any{"description_html": issueImageHTML(good.Asset.ID, badID)})
			before := draftAssetSnapshot(t, w)
			draftRequest(t, w, "alice", "PATCH", draftPath(d.ID), body, status)
			draftRequest(t, w, "alice", "POST", draftsPath, body, 404)
			if !reflect.DeepEqual(before, draftAssetSnapshot(t, w)) {
				t.Fatal("invalid reference partially wrote draft or bound another file")
			}
			draftRequest(t, w, "alice", "PATCH", draftPath(d.ID), draftPayloadBody(t, map[string]any{"description_html": issueImageHTML(good.Asset.ID)}), 200)
		})
	}
}

func TestDraftAssetPublish(t *testing.T) {
	w := newIssueWorld(t)
	d := createDraftHTTP(t, w, "alice", fmt.Sprintf(`{"project_id":%q,"payload":{"name":"Published"}}`, w.web))
	history := draftAsset(t, w, d.ID, true)
	current := draftAsset(t, w, d.ID, true)
	unused := draftAsset(t, w, d.ID, true)
	file := draftAsset(t, w, d.ID, false)
	removed := draftAsset(t, w, d.ID, false)
	assetAPICall(t, w, "alice", "DELETE", "/api/v0/assets/"+removed.Asset.ID.String(), "", 204)
	draftRequest(t, w, "alice", "PATCH", draftPath(d.ID), draftPayloadBody(t, map[string]any{"name": "Published", "description_html": issueImageHTML(history.Asset.ID)}), 200)
	draftRequest(t, w, "alice", "PATCH", draftPath(d.ID), draftPayloadBody(t, map[string]any{"name": "Published", "description_html": issueImageHTML(current.Asset.ID)}), 200)
	origins := map[uuid.UUID]assetdomain.Asset{}
	for _, id := range []uuid.UUID{history.Asset.ID, current.Asset.ID, unused.Asset.ID, file.Asset.ID, removed.Asset.ID} {
		row, err := assetpg.New(w.pool).FindAsset(t.Context(), id)
		if err != nil {
			t.Fatal(err)
		}
		origins[id] = row
	}
	var published struct {
		storedIssue
		AttachmentCount int64 `json:"attachment_count"`
	}
	decodeAnswer(t, draftRequest(t, w, "alice", "POST", publishPath(d.ID), "", 201), &published)
	if published.AttachmentCount != 1 || attachmentActivityCount(t, w, published.ID) != 1 {
		t.Fatal("publish omitted actual count or attachment activity", published)
	}
	for id, old := range origins {
		row, err := assetpg.New(w.pool).FindAsset(t.Context(), id)
		if err != nil {
			t.Fatal(err)
		}
		if id == removed.Asset.ID {
			if !reflect.DeepEqual(row, old) {
				t.Fatal("publish revived deleted file")
			}
			continue
		}
		purpose := assetdomain.IssueDescription
		if id == file.Asset.ID {
			purpose = assetdomain.IssueAttachment
		}
		if row.Purpose != purpose || row.Placement.Scope != (assetdomain.Scope{Kind: assetdomain.ScopeIssue, ID: published.ID}) || row.Placement.DraftID != uuid.Nil() || row.Placement.IssueID != published.ID || row.Placement.ProjectID != w.web || row.Placement.UserID != w.ids["alice"] || !row.UpdatedAt.Equal(published.CreatedAt) || row.UpdatedByID == nil || *row.UpdatedByID != w.ids["alice"] || !row.CreatedAt.Equal(old.CreatedAt) || !reflect.DeepEqual(row.Metadata, old.Metadata) || !reflect.DeepEqual(row.BoundAt, old.BoundAt) {
			t.Fatal("migration changed origin, binding, bytes or audit", row, old)
		}
		who := "bob"
		if id == unused.Asset.ID {
			if row.BoundAt != nil {
				t.Fatal("unused image became public")
			}
			assetAPICall(t, w, "bob", "GET", "/api/v0/assets/"+id.String()+"/content", "", 404)
			who = "alice"
		}
		_, data := readAssetHTTP(t, w, who, id)
		want := assetTestPNG(t)
		if id == file.Asset.ID {
			want = []byte("draft bytes")
		}
		if !bytes.Equal(data, want) {
			t.Fatal("migration changed physical bytes")
		}
	}
	before := draftAssetSnapshot(t, w)
	draftRequest(t, w, "alice", "POST", publishPath(d.ID), "", 404)
	if !reflect.DeepEqual(before, draftAssetSnapshot(t, w)) {
		t.Fatal("repeated publication wrote another graph")
	}
	assetAPICall(t, w, "alice", "DELETE", "/api/v0/assets/"+unused.Asset.ID.String(), "", 204)
}

func TestDraftAssetPublishPendingBlocksWholeGraph(t *testing.T) {
	w := newIssueWorld(t)
	d := createDraftHTTP(t, w, "alice", fmt.Sprintf(`{"project_id":%q,"payload":{"name":"Pending"}}`, w.web))
	draftAsset(t, w, d.ID, false)
	pending := prepareAssetHTTP(t, w, "alice", "DRAFT_ISSUE_ATTACHMENT", attachmentScope("draft", d.ID), "pending.txt", "text/plain", 1)
	before := draftAssetSnapshot(t, w)
	draftRequest(t, w, "alice", "POST", publishPath(d.ID), "", 409)
	if !reflect.DeepEqual(before, draftAssetSnapshot(t, w)) {
		t.Fatal("pending file left a partially published graph")
	}
	assetAPICall(t, w, "alice", "DELETE", "/api/v0/assets/"+pending.Asset.ID.String(), "", 204)
	draftRequest(t, w, "alice", "POST", publishPath(d.ID), "", 201)
}

// Existing M4 drafts may already store a reference without an M5 binding. Both
// an unchanged save and a publication must validate it; JSONB no-op audit is
// independent of the newly introduced reference binding.
func TestDraftAssetsLegacyReferences(t *testing.T) {
	for _, publish := range []bool{false, true} {
		for _, foreign := range []bool{false, true} {
			t.Run(fmt.Sprintf("publish=%t/foreign=%t", publish, foreign), func(t *testing.T) {
				w := newIssueWorld(t)
				d := createDraftHTTP(t, w, "alice", fmt.Sprintf(`{"project_id":%q,"payload":{"name":"Legacy"}}`, w.web))
				host := d.ID
				if foreign {
					host = createDraftHTTP(t, w, "alice", fmt.Sprintf(`{"project_id":%q,"payload":{}}`, w.web)).ID
				}
				image := draftAsset(t, w, host, true)
				payload := fmt.Sprintf(`{"name":"Legacy","description_html":%q}`, issueImageHTML(image.Asset.ID))
				if _, err := w.pool.Exec(t.Context(), "UPDATE draft_issues SET payload=$2 WHERE id=$1", d.ID, payload); err != nil {
					t.Fatal(err)
				}
				before := draftAssetSnapshot(t, w)
				method, path, body, want := "PATCH", draftPath(d.ID), `{}`, 200
				if publish {
					method, path, body, want = "POST", publishPath(d.ID), "", 201
				}
				if foreign {
					want = 404
				}
				draftRequest(t, w, "alice", method, path, body, want)
				after := draftAssetSnapshot(t, w)
				if foreign {
					if !reflect.DeepEqual(before, after) {
						t.Fatal("legacy foreign reference partially wrote graph")
					}
					return
				}
				row, err := assetpg.New(w.pool).FindAsset(t.Context(), image.Asset.ID)
				if err != nil || row.BoundAt == nil {
					t.Fatal("legacy reference bypassed binding", err)
				}
				if !publish {
					delete(before, "file_assets")
					delete(after, "file_assets")
					if !reflect.DeepEqual(before, after) {
						t.Fatal("no-op legacy binding changed draft audit or payload")
					}
				}
			})
		}
	}
}
