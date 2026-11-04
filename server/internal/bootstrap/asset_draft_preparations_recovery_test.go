package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"sync/atomic"
	"testing"
	"uuid"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func TestDraftAssetPreparationsRecoverLostAnswer(t *testing.T) {
	w := newIssueWorld(t)
	d := createDraftHTTP(t, w, "alice", `{"payload":{"name":"Recovered"}}`)
	other := createDraftHTTP(t, w, "alice", `{"payload":{}}`)
	neighbor := prepareAssetHTTP(t, w, "alice", "DRAFT_ISSUE_ATTACHMENT", attachmentScope("draft", d.ID), "same.txt", "text/plain", 5)
	elsewhere := prepareAssetHTTP(t, w, "alice", "DRAFT_ISSUE_ATTACHMENT", attachmentScope("draft", other.ID), "same.txt", "text/plain", 5)
	// Discard the successful response completely. The recovery client has no ID,
	// upload grant or storage key from this request and cannot read SQL to find it.
	assetAPICall(t, w, "alice", "POST", "/api/v0/assets", assetPrepareBody("DRAFT_ISSUE_ATTACHMENT", attachmentScope("draft", d.ID), "same.txt", "text/plain", 5), 201)
	page := preparationsHTTP(t, w, "alice", d.ID, "")
	if len(page.Data) != 2 {
		t.Fatalf("unknown preparation was not discoverable: %+v", page)
	}
	var recovered uuid.UUID
	for _, a := range page.Data {
		if a.ID != neighbor.Asset.ID {
			recovered = a.ID
		}
		if a.Name != "same.txt" || a.State != "pending" || a.IsBound {
			t.Fatal(a)
		}
	}
	if recovered == uuid.Nil() || recovered == elsewhere.Asset.ID {
		t.Fatal("recovery crossed draft or guessed by name")
	}
	if again := preparationsHTTP(t, w, "alice", d.ID, ""); !reflect.DeepEqual(again, page) {
		t.Fatal("reopening changed the recovery identities")
	}
	draftRequest(t, w, "alice", "PATCH", draftPath(d.ID), fmt.Sprintf(`{"project_id":%q}`, w.web), 200)
	before := draftAssetSnapshot(t, w)
	if raw := draftRequest(t, w, "alice", "POST", publishPath(d.ID), "", 409); problemCode(t, []byte(raw)) != "asset.not_uploaded" {
		t.Fatal(raw)
	}
	if !reflect.DeepEqual(before, draftAssetSnapshot(t, w)) {
		t.Fatal("pending publish changed graph")
	}
	assetAPICall(t, w, "alice", "DELETE", "/api/v0/assets/"+recovered.String(), "", 204)
	afterDelete := draftAssetSnapshot(t, w)
	assetAPICall(t, w, "alice", "DELETE", "/api/v0/assets/"+recovered.String(), "", 204)
	if !reflect.DeepEqual(afterDelete, draftAssetSnapshot(t, w)) {
		t.Fatal("repeated explicit deletion changed audit/events/tasks")
	}
	left := preparationsHTTP(t, w, "alice", d.ID, "")
	if len(left.Data) != 1 || left.Data[0].ID != neighbor.Asset.ID {
		t.Fatal("explicit deletion removed a same-name neighbor", left)
	}
	otherPage := preparationsHTTP(t, w, "alice", other.ID, "")
	if len(otherPage.Data) != 1 || otherPage.Data[0].ID != elsewhere.Asset.ID {
		t.Fatal("another draft changed", otherPage)
	}
	putAssetHTTP(t, w, neighbor, []byte("other"), 204)
	assetAPICall(t, w, "alice", "POST", completeAssetPath(neighbor.Asset.ID), `{}`, 200)
	afterComplete := draftAssetSnapshot(t, w)
	assetAPICall(t, w, "alice", "POST", completeAssetPath(neighbor.Asset.ID), `{}`, 200)
	if !reflect.DeepEqual(afterComplete, draftAssetSnapshot(t, w)) {
		t.Fatal("repeated completion duplicated binding or effects")
	}
	if len(preparationsHTTP(t, w, "alice", d.ID, "").Data) != 0 {
		t.Fatal("bound attachment remained a preparation")
	}
	var published struct {
		storedIssue
		AttachmentCount int64 `json:"attachment_count"`
	}
	decodeAnswer(t, draftRequest(t, w, "alice", "POST", publishPath(d.ID), "", 201), &published)
	if published.AttachmentCount != 1 || attachmentActivityCount(t, w, published.ID) != 1 {
		t.Fatal("wrong recovered publication effects", published)
	}
	_, actual := readAssetHTTP(t, w, "alice", neighbor.Asset.ID)
	if string(actual) != "other" {
		t.Fatal("neighbor bytes changed")
	}
}

func TestDraftAssetPreparationsRecoverFailedCopy(t *testing.T) {
	var armed atomic.Bool
	w, _ := newAssetCompleteWorld(t, func(c *assetComponents) {
		base := c.sealing.Sealer
		c.sealing.Sealer = assetSealFunc(func(ctx context.Context, src, dst domain.ObjectKey, etag string) (domain.ObjectInfo, error) {
			info, err := base.Seal(ctx, src, dst, etag)
			if err == nil && armed.Load() {
				return info, errors.New("injected failure after sealing the copy")
			}
			return info, err
		})
	})
	source := uploadedAssetHTTP(t, w, "alice", "ISSUE_ATTACHMENT", attachmentScope("issue", w.newWorkItem(t)), "source.txt", "text/plain", []byte("source bytes"))
	d := createDraftHTTP(t, w, "alice", fmt.Sprintf(`{"project_id":%q,"payload":{"name":"Copy recovered"}}`, w.web))
	armed.Store(true)
	raw := assetAPICall(t, w, "alice", "POST", duplicateAssetPath(source.Asset.ID), fmt.Sprintf(`{"purpose":"DRAFT_ISSUE_ATTACHMENT","scope":%s}`, attachmentScope("draft", d.ID)), 503)
	if problemCode(t, []byte(raw)) != "asset.storage_unavailable" {
		t.Fatal(raw)
	}
	listKeys(t, []byte(raw), "status", "code", "title", "detail")
	page := preparationsHTTP(t, w, "alice", d.ID, "")
	if len(page.Data) != 1 || page.Data[0].ID == source.Asset.ID || page.Data[0].State != "pending" || page.Data[0].Name != "source.txt" {
		t.Fatalf("failed copy not recoverable from its public list: %+v", page)
	}
	// Only after the public query has supplied the identity may SQL assert recovery.
	target, err := assetpg.New(w.pool).FindAsset(t.Context(), page.Data[0].ID)
	if err != nil || target.Metadata.Origin != domain.OriginCopy || len(target.Metadata.Candidates) != 1 || target.Metadata.Final != nil || target.Metadata.Staging != nil {
		t.Fatal("failed copy lost its durable recovery record", target, err)
	}
	if raw := draftRequest(t, w, "alice", "POST", publishPath(d.ID), "", 409); problemCode(t, []byte(raw)) != "asset.not_uploaded" {
		t.Fatal(raw)
	}
	_, bytesBefore := readAssetHTTP(t, w, "alice", source.Asset.ID)
	assetAPICall(t, w, "alice", "DELETE", "/api/v0/assets/"+page.Data[0].ID.String(), "", 204)
	draftRequest(t, w, "alice", "POST", publishPath(d.ID), "", 201)
	_, bytesAfter := readAssetHTTP(t, w, "alice", source.Asset.ID)
	if string(bytesBefore) != "source bytes" || !bytes.Equal(bytesBefore, bytesAfter) {
		t.Fatal("discarding the failed copy changed its source")
	}
}

func TestDraftAssetPreparationsCompletedAndHistoricalImages(t *testing.T) {
	w := newIssueWorld(t)
	d := createDraftHTTP(t, w, "alice", `{"payload":{}}`)
	history, current, unused := draftAsset(t, w, d.ID, true), draftAsset(t, w, d.ID, true), draftAsset(t, w, d.ID, true)
	before := preparationsHTTP(t, w, "alice", d.ID, "")
	if len(before.Data) != 3 {
		t.Fatal("completed unbound images omitted", before)
	}
	for _, a := range before.Data {
		if a.State != "uploaded" || a.IsBound || a.Width.MustGet() != 2 || a.Height.MustGet() != 3 {
			t.Fatal("image DTO lost verified metadata", a)
		}
	}
	for _, id := range []uuid.UUID{history.Asset.ID, current.Asset.ID} {
		draftRequest(t, w, "alice", "PATCH", draftPath(d.ID), draftPayloadBody(t, map[string]any{"description_html": issueImageHTML(id)}), 200)
	}
	after := preparationsHTTP(t, w, "alice", d.ID, "")
	if len(after.Data) != 1 || after.Data[0].ID != unused.Asset.ID {
		t.Fatal("current or historical binding exposed as disposable", after)
	}
	for _, id := range []uuid.UUID{history.Asset.ID, current.Asset.ID} {
		raw := assetAPICall(t, w, "alice", "DELETE", "/api/v0/assets/"+id.String(), "", 409)
		if problemCode(t, []byte(raw)) != "asset.bound" {
			t.Fatal(raw)
		}
		_, actual := readAssetHTTP(t, w, "alice", id)
		if !slices.Equal(actual, assetTestPNG(t)) {
			t.Fatal("historical image bytes changed")
		}
	}
	assetAPICall(t, w, "alice", "DELETE", "/api/v0/assets/"+unused.Asset.ID.String(), "", 204)
	if len(preparationsHTTP(t, w, "alice", d.ID, "").Data) != 0 {
		t.Fatal("explicit image discard remained visible")
	}
}
