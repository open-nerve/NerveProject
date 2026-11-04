package bootstrap

import (
	"fmt"
	"net/url"
	"reflect"
	"testing"
	"uuid"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	assetdomain "github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func TestDraftAssetPreparationsStaleSelection(t *testing.T) {
	for _, image := range []bool{false, true} {
		for _, deleteFirst := range []bool{false, true} {
			t.Run(fmt.Sprintf("image=%t/delete_first=%t", image, deleteFirst), func(t *testing.T) {
				w := newIssueWorld(t)
				d := createDraftHTTP(t, w, "alice", `{"payload":{}}`)
				purpose, mime, data := "DRAFT_ISSUE_ATTACHMENT", "text/plain", []byte("late")
				if image {
					purpose, mime, data = "DRAFT_ISSUE_DESCRIPTION", "image/png", assetTestPNG(t)
				}
				prepared := prepareAssetHTTP(t, w, "alice", purpose, attachmentScope("draft", d.ID), "selected", mime, len(data))
				page := preparationsHTTP(t, w, "alice", d.ID, "")
				if len(page.Data) != 1 || page.Data[0].ID != prepared.Asset.ID {
					t.Fatal(page)
				}
				selected := page.Data[0].ID
				if deleteFirst {
					assetAPICall(t, w, "alice", "DELETE", "/api/v0/assets/"+selected.String(), "", 204)
					before := draftAssetSnapshot(t, w)
					putAssetHTTP(t, w, prepared, data, 204)
					assetAPICall(t, w, "alice", "POST", completeAssetPath(selected), `{}`, 404)
					assetAPICall(t, w, "alice", "GET", "/api/v0/assets/"+selected.String(), "", 404)
					if !reflect.DeepEqual(before, draftAssetSnapshot(t, w)) {
						t.Fatal("late PUT/complete revived deleted preparation")
					}
				} else {
					putAssetHTTP(t, w, prepared, data, 204)
					assetAPICall(t, w, "alice", "POST", completeAssetPath(selected), `{}`, 200)
					status := 204
					if image {
						draftRequest(t, w, "alice", "PATCH", draftPath(d.ID), draftPayloadBody(t, map[string]any{"description_html": issueImageHTML(selected)}), 200)
						status = 409
					}
					before := draftAssetSnapshot(t, w)
					raw := assetAPICall(t, w, "alice", "DELETE", "/api/v0/assets/"+selected.String(), "", status)
					if image && (problemCode(t, []byte(raw)) != "asset.bound" || !reflect.DeepEqual(before, draftAssetSnapshot(t, w))) {
						t.Fatal("stale preparation list bypassed bound image protection", raw)
					}
					row, err := assetpg.New(w.pool).FindAsset(t.Context(), selected)
					if err != nil || row.BoundAt == nil || (row.DeletedAt == nil) != image {
						t.Fatal("ordinary attachment and description delete rules collapsed", row, err)
					}
				}
				if len(preparationsHTTP(t, w, "alice", d.ID, "").Data) != 0 {
					t.Fatal("consumed preparation remained visible")
				}
			})
		}
	}
}

func TestDraftAssetPreparationsPublishVersusNewPrepare(t *testing.T) {
	for _, publishFirst := range []bool{false, true} {
		t.Run(fmt.Sprintf("publish_first=%t", publishFirst), func(t *testing.T) {
			w := newObservedIssueWorld(t)
			d := createDraftHTTP(t, w, "alice", fmt.Sprintf(`{"project_id":%q,"payload":{"name":"Serialized"}}`, w.web))
			if len(preparationsHTTP(t, w, "alice", d.ID, "").Data) != 0 {
				t.Fatal("new draft not empty")
			}
			held := holding(t, w.pool, "SELECT id FROM draft_issues WHERE id=$1 FOR UPDATE", d.ID)
			defer requestRelease(t, held)
			prepare := newRequest(t, "POST", w.base+"/api/v0/assets", w.tokens["alice"], []byte(assetPrepareBody("DRAFT_ISSUE_ATTACHMENT", attachmentScope("draft", d.ID), "new.txt", "text/plain", 1)))
			publish := newRequest(t, "POST", w.base+publishPath(d.ID), w.tokens["alice"], nil)
			firstReq, secondReq := prepare, publish
			if publishFirst {
				firstReq, secondReq = publish, prepare
			}
			firstProbe, secondProbe := w.observe(firstReq), w.observe(secondReq)
			first := sendInBackground(firstReq)
			pid := requestBlocked(t, w.pool, firstProbe, held.Conn().PgConn().PID(), "draft_issues")
			second := sendInBackground(secondReq)
			secondPID := requestBlocked(t, w.pool, secondProbe, pid, "projects")
			t.Logf("draft write order first=%s PID=%d second=%s PID=%d", firstReq.URL.Path, pid, secondReq.URL.Path, secondPID)
			requestRelease(t, held)
			imageRaceAnswer(t, w, firstReq, first, 201)
			status := 409
			if publishFirst {
				status = 404
			}
			loser := imageRaceAnswer(t, w, secondReq, second, status)
			if publishFirst {
				if problemCode(t, loser.body) != "asset.not_found" || len(queryIDs(t, w.pool, "SELECT id FROM file_assets")) != 0 {
					t.Fatal("late preparation escaped consumed draft", loser)
				}
			} else {
				if problemCode(t, loser.body) != "asset.not_uploaded" || len(queryIDs(t, w.pool, "SELECT id FROM issues")) != 0 {
					t.Fatal("publish ignored newly committed pending", loser)
				}
				page := preparationsHTTP(t, w, "alice", d.ID, "")
				if len(page.Data) != 1 {
					t.Fatal(page)
				}
				assetAPICall(t, w, "alice", "DELETE", "/api/v0/assets/"+page.Data[0].ID.String(), "", 204)
				draftRequest(t, w, "alice", "POST", publishPath(d.ID), "", 201)
			}
		})
	}
}

func TestDraftAssetPreparationsChangingPages(t *testing.T) {
	w := newIssueWorld(t)
	d := createDraftHTTP(t, w, "alice", `{"payload":{}}`)
	other := createDraftHTTP(t, w, "bob", `{"payload":{}}`)
	prepare := func(name string) preparedAssetAnswer {
		return prepareAssetHTTP(t, w, "alice", "DRAFT_ISSUE_ATTACHMENT", attachmentScope("draft", d.ID), name, "text/plain", 1)
	}
	first, removed, last := prepare("first"), prepare("removed"), prepare("last")
	foreign := prepareAssetHTTP(t, w, "bob", "DRAFT_ISSUE_ATTACHMENT", attachmentScope("draft", other.ID), "foreign", "text/plain", 1)
	page := preparationsHTTP(t, w, "alice", d.ID, "?limit=1")
	if len(page.Data) != 1 || page.Data[0].ID != first.Asset.ID || page.NextCursor == nil {
		t.Fatal(page)
	}
	assetAPICall(t, w, "alice", "DELETE", "/api/v0/assets/"+removed.Asset.ID.String(), "", 204)
	added := prepare("added")
	seen := map[uuid.UUID]bool{first.Asset.ID: true}
	cursor := page.NextCursor
	for range 4 {
		if cursor == nil {
			break
		}
		next := preparationsHTTP(t, w, "alice", d.ID, "?limit=1&cursor="+url.QueryEscape(*cursor))
		for _, a := range next.Data {
			if seen[a.ID] || a.ID == foreign.Asset.ID || a.ID == removed.Asset.ID {
				t.Fatal("changing page repeated or crossed owner", a)
			}
			seen[a.ID] = true
		}
		cursor = next.NextCursor
	}
	if cursor != nil || len(seen) != 3 || !seen[last.Asset.ID] || !seen[added.Asset.ID] {
		t.Fatal("changing pagination did not terminate completely", seen)
	}
	end, err := assetdomain.EncodeDraftPreparationCursor(d.ID, uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff"))
	if err != nil {
		t.Fatal(err)
	}
	if p := preparationsHTTP(t, w, "alice", d.ID, "?cursor="+url.QueryEscape(end)); len(p.Data) != 0 || p.NextCursor != nil {
		t.Fatal("beyond-end cursor did not terminate", p)
	}
	// A valid cursor never grants another author this draft's rows.
	assetAPICall(t, w, "bob", "GET", preparationsPath(d.ID)+"?cursor="+url.QueryEscape(*page.NextCursor), "", 404)
	assetAPICall(t, w, "bob", "GET", preparationsPath(other.ID)+"?cursor="+url.QueryEscape(*page.NextCursor), "", 400)
}
