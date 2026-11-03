package bootstrap

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"uuid"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
)

// All six issue/comment/draft purposes cross real storage and owner APIs. SQL
// only observes facts; no normal fixture row or binding is written directly.
func TestContentStorageContract(t *testing.T) {
	for _, driver := range []string{"local", "s3"} {
		t.Run(driver, func(t *testing.T) {
			w, _ := newImageStorageWorld(t, driver)
			item := w.newWorkItemBy(t, "alice", w.web, `{"name":"Stored content"}`)
			var comment answeredComment
			decodeAnswer(t, w.send(t, "alice", "POST", commentsOf(item), commentAssetsBody(t, "<p>Stored comment</p>"), 201), &comment)
			draft := createDraftHTTP(t, w, "alice", fmt.Sprintf(`{"project_id":%q,"payload":{"name":"Stored draft"}}`, w.web))
			base, status := w.base, 204
			if driver == "s3" {
				base, status = "", 200
			}
			var draftImage uuid.UUID
			for _, tc := range []struct {
				kind, purpose string
				host          uuid.UUID
				image         bool
			}{
				{"issue", "ISSUE_ATTACHMENT", item, false},
				{"issue", "ISSUE_DESCRIPTION", item, true},
				{"comment", "COMMENT_ATTACHMENT", comment.ID, false},
				{"comment", "COMMENT_DESCRIPTION", comment.ID, true},
				{"draft", "DRAFT_ISSUE_ATTACHMENT", draft.ID, false},
				{"draft", "DRAFT_ISSUE_DESCRIPTION", draft.ID, true},
			} {
				t.Run(tc.purpose, func(t *testing.T) {
					data, name, mime, disposition := []byte(tc.purpose), "file.txt", "text/plain", "attachment;"
					if tc.image {
						data, name, mime, disposition = assetTestPNG(t), "image.png", "image/png", "inline;"
					}
					prepared := uploadedAssetHTTPAt(t, w, base, status, "alice", tc.purpose, assetScopeBody(tc.kind, tc.host), name, mime, data)
					if prepared.Asset.IsBound == tc.image {
						t.Fatal("ordinary completion and unreferenced image binding conflated", prepared.Asset)
					}
					before, err := assetpg.New(w.pool).FindAsset(t.Context(), prepared.Asset.ID)
					if err != nil {
						t.Fatal(err)
					}
					if tc.image {
						html := issueImageHTML(prepared.Asset.ID)
						switch tc.kind {
						case "issue":
							saveIssueImage(t, w, "alice", item, html, false, 200)
						case "comment":
							w.send(t, "alice", "PATCH", commentPath(comment.ID), commentAssetsBody(t, html), 200)
						case "draft":
							draftImage = prepared.Asset.ID
							draftRequest(t, w, "alice", "PATCH", draftPath(draft.ID), draftPayloadBody(t, map[string]any{"name": "Stored draft", "description_html": html}), 200)
						}
					}
					bound, err := assetpg.New(w.pool).FindAsset(t.Context(), prepared.Asset.ID)
					if err != nil || bound.BoundAt == nil || !bound.CreatedAt.Equal(before.CreatedAt) || !reflect.DeepEqual(bound.Metadata, before.Metadata) {
						t.Fatal("binding lost origin or physical identity", bound, err)
					}
					response, actual := readAssetHTTP(t, w, "alice", prepared.Asset.ID)
					if !bytes.Equal(actual, data) || !strings.HasPrefix(response.Header.Get("Content-Disposition"), disposition) {
						t.Fatal("bytes or disposition changed across binding", response.Header, actual)
					}
					assetAPICall(t, w, "alice", "POST", completeAssetPath(prepared.Asset.ID), `{}`, 200)
					after, err := assetpg.New(w.pool).FindAsset(t.Context(), prepared.Asset.ID)
					if err != nil || !reflect.DeepEqual(after, bound) {
						t.Fatal("repeat completion mutated binding", after, err)
					}
				})
			}
			var published struct {
				ID              uuid.UUID `json:"id"`
				AttachmentCount int64     `json:"attachment_count"`
				HTML            string    `json:"description_html"`
			}
			decodeAnswer(t, draftRequest(t, w, "alice", "POST", publishPath(draft.ID), "", 201), &published)
			if published.AttachmentCount != 1 || published.HTML != issueImageHTML(draftImage) || attachmentActivityCount(t, w, published.ID) != 1 {
				t.Fatal("publication omitted real file count, content or event", published)
			}
			if _, data := readAssetHTTP(t, w, "alice", draftImage); !bytes.Equal(data, assetTestPNG(t)) {
				t.Fatal("publication changed image bytes")
			}
			t.Run("remote source remains a reference without an outbound fetch", func(t *testing.T) {
				var requests atomic.Int32
				data := assetTestPNG(t)
				remote := httptest.NewServer(http.HandlerFunc(func(out http.ResponseWriter, _ *http.Request) {
					requests.Add(1)
					out.Header().Set("Content-Type", "image/png")
					_, _ = out.Write(data)
				}))
				defer remote.Close()
				before := queryIDs(t, w.pool, "SELECT id FROM file_assets ORDER BY id")
				html := fmt.Sprintf(`<p>External</p><image-component src="%s/image.png"></image-component>`, remote.URL)
				var saved struct {
					HTML string `json:"description_html"`
				}
				decodeAnswer(t, saveIssueImage(t, w, "alice", item, html, true, 200), &saved)
				if saved.HTML != html || requests.Load() != 0 || !reflect.DeepEqual(before, queryIDs(t, w.pool, "SELECT id FROM file_assets ORDER BY id")) {
					t.Fatal("remote reference was fetched, materialized or changed", saved, requests.Load())
				}
			})
		})
	}
}
