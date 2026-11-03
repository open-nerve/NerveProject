package bootstrap

import (
	"fmt"
	"reflect"
	"testing"
	"uuid"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
)

func TestBoundAssetAccess(t *testing.T) {
	w := newIssueWorld(t)
	item := w.newWorkItemBy(t, "bob", w.web, `{"name":"Shared attachment"}`)
	prepared := prepareAssetHTTP(t, w, "bob", "ISSUE_ATTACHMENT", attachmentScope("issue", item), "bob-private.txt", "text/plain", 3)
	path := "/api/v0/assets/" + prepared.Asset.ID.String()
	assetAPICall(t, w, "alice", "GET", path, "", 404)
	putAssetHTTP(t, w, prepared, []byte("abc"), 204)
	assetAPICall(t, w, "bob", "POST", completeAssetPath(prepared.Asset.ID), `{}`, 200)
	assetAPICall(t, w, "alice", "GET", path, "", 200)
	_, data := readAssetHTTP(t, w, "alice", prepared.Asset.ID)
	if string(data) != "abc" {
		t.Fatal("authorized member did not read the published bytes")
	}
	// Reading a bound file never permits completing another applicant's upload.
	assetAPICall(t, w, "alice", "POST", completeAssetPath(prepared.Asset.ID), `{}`, 404)
	copyBody := fmt.Sprintf(`{"scope":%s,"purpose":"ISSUE_ATTACHMENT"}`, attachmentScope("issue", item))
	assetAPICall(t, w, "alice", "POST", path+"/duplicate", copyBody, 201)
	assetAPICall(t, w, "alice", "DELETE", path, "", 204)
	row, err := assetpg.New(w.pool).FindAsset(t.Context(), prepared.Asset.ID)
	if err != nil || row.DeletedAt == nil || row.DeletedByID == nil || *row.DeletedByID != w.ids["alice"] {
		t.Fatalf("host-authorized deletion missing: %+v %v", row, err)
	}
	assetAPICall(t, w, "alice", "DELETE", path, "", 204)
	again, err := assetpg.New(w.pool).FindAsset(t.Context(), prepared.Asset.ID)
	if err != nil || !reflect.DeepEqual(row, again) {
		t.Fatalf("idempotent delete changed first audit: %+v %v", again, err)
	}
	assetAPICall(t, w, "alice", "GET", path, "", 404)
	assetAPICall(t, w, "alice", "GET", path+"/content", "", 404)
	assetAPICall(t, w, "alice", "DELETE", "/api/v0/assets/"+uuid.NewV7().String(), "", 404)
	// Owner deletion removes visibility even for the previous successful caller.
	w.send(t, "alice", "DELETE", "/api/v0/issues/"+item.String(), "", 204)
	assetAPICall(t, w, "alice", "DELETE", path, "", 404)
}

func TestBoundAssetAccessKeepsPreparationsPrivate(t *testing.T) {
	w := newIssueWorld(t)
	item := w.newWorkItem(t)
	for _, tc := range []struct{ purpose, scope string }{
		{"USER_AVATAR", `{"kind":"user"}`},
		{"ISSUE_DESCRIPTION", attachmentScope("issue", item)},
		{"COMMENT_ATTACHMENT", attachmentScope("new_comment", item)},
	} {
		t.Run(tc.purpose, func(t *testing.T) {
			data := assetTestPNG(t)
			p := prepareAssetHTTP(t, w, "bob", tc.purpose, tc.scope, "not-shared.png", "image/png", len(data))
			putAssetHTTP(t, w, p, data, 204)
			assetAPICall(t, w, "bob", "POST", completeAssetPath(p.Asset.ID), `{}`, 200)
			path := "/api/v0/assets/" + p.Asset.ID.String()
			for _, suffix := range []string{"", "/content"} {
				assetAPICall(t, w, "alice", "GET", path+suffix, "", 404)
			}
			assetAPICall(t, w, "alice", "DELETE", path, "", 404)
		})
	}
}
