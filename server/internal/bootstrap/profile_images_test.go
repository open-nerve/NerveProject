package bootstrap

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"testing"
	"time"
	"uuid"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
)

// Images enter through authenticated preparation, real PUT and completion.
// This tests the owner API, not just the asset binding port in isolation.
func TestProfileImages(t *testing.T) {
	w := newIssueWorld(t)
	data := assetTestPNG(t)
	avatar := uploadedAssetHTTP(t, w, "alice", "USER_AVATAR", `{"kind":"user"}`, "avatar.png", "image/png", data).Asset.ID
	cover := uploadedAssetHTTP(t, w, "alice", "USER_COVER", `{"kind":"user"}`, "cover.png", "image/png", data).Asset.ID
	patch := fmt.Sprintf(`{"avatar_asset_id":%q,"cover_image_asset_id":%q}`, avatar, cover)
	check := func(body string) {
		t.Helper()
		checkImageReply(t, w, body, "avatar", avatar, data)
		checkImageReply(t, w, body, "cover_image", cover, data)
	}
	check(w.send(t, "alice", "PATCH", "/api/v0/me", patch, 200))
	rows := assetpg.New(w.pool)
	oldAvatar, err := rows.FindAsset(t.Context(), avatar)
	if err != nil {
		t.Fatal(err)
	}
	oldCover, err := rows.FindAsset(t.Context(), cover)
	if err != nil {
		t.Fatal(err)
	}
	var updated time.Time
	if err := w.pool.QueryRow(t.Context(), "SELECT updated_at FROM users WHERE id=$1", w.ids["alice"]).Scan(&updated); err != nil {
		t.Fatal(err)
	}
	if oldAvatar.BoundAt == nil || oldCover.BoundAt == nil || !oldAvatar.UpdatedAt.Equal(updated) || !oldCover.UpdatedAt.Equal(updated) || !oldAvatar.BoundAt.Equal(updated) || !oldCover.BoundAt.Equal(updated) {
		t.Fatalf("owner/assets did not share the last-lock time: %v %+v %+v", updated, oldAvatar, oldCover)
	}
	for _, body := range []string{patch, `{"first_name":"Changed"}`, `{}`} {
		check(w.send(t, "alice", "PATCH", "/api/v0/me", body, 200))
		check(w.send(t, "alice", "GET", "/api/v0/me", "", 200))
	}
	afterAvatar, err := rows.FindAsset(t.Context(), avatar)
	if err != nil {
		t.Fatal(err)
	}
	afterCover, err := rows.FindAsset(t.Context(), cover)
	if err != nil || !reflect.DeepEqual(oldAvatar, afterAvatar) || !reflect.DeepEqual(oldCover, afterCover) {
		t.Fatal("idempotent/omitted images changed asset audit", err)
	}
	// The generic endpoint still cannot delete a bound image or read another user's.
	assetAPICall(t, w, "alice", "DELETE", "/api/v0/assets/"+avatar.String(), "", 409)
	assetAPICall(t, w, "bob", "GET", "/api/v0/assets/"+avatar.String()+"/content", "", 404)
	next := uploadedAssetHTTP(t, w, "alice", "USER_AVATAR", `{"kind":"user"}`, "next.png", "image/png", data).Asset.ID
	body := w.send(t, "alice", "PATCH", "/api/v0/me", fmt.Sprintf(`{"avatar_asset_id":%q}`, next), 200)
	checkImageReply(t, w, body, "avatar", next, data)
	checkImageReply(t, w, body, "cover_image", cover, data)
	removed, err := rows.FindAsset(t.Context(), avatar)
	if err != nil || removed.DeletedAt == nil || removed.DeletedByID == nil || *removed.DeletedByID != w.ids["alice"] {
		t.Fatal("replacement did not soft-delete old image", removed, err)
	}
	body = w.send(t, "alice", "PATCH", "/api/v0/me", `{"avatar_asset_id":null}`, 200)
	checkNullImageReply(t, body, "avatar")
	checkImageReply(t, w, body, "cover_image", cover, data)
	body = w.send(t, "alice", "PATCH", "/api/v0/me", `{"cover_image_asset_id":null}`, 200)
	checkNullImageReply(t, body, "avatar")
	checkNullImageReply(t, body, "cover_image")
}

func checkImageReply(t *testing.T, w issueWorld, body, field string, id uuid.UUID, data []byte) {
	t.Helper()
	var reply map[string]json.RawMessage
	decodeAnswer(t, body, &reply)
	var gotID uuid.UUID
	if err := json.Unmarshal(reply[field+"_asset_id"], &gotID); err != nil || gotID != id {
		t.Fatalf("wrong %s reference in %s: %v", field, body, err)
	}
	checkSignedImageReply(t, w, body, field, data)
}

func checkSignedImageReply(t *testing.T, w issueWorld, body, field string, data []byte) {
	t.Helper()
	var reply map[string]json.RawMessage
	decodeAnswer(t, body, &reply)
	var address string
	var expires time.Time
	for key, target := range map[string]any{field + "_url": &address, field + "_url_expires_at": &expires} {
		if err := json.Unmarshal(reply[key], target); err != nil {
			t.Fatalf("%s missing/invalid in %s: %v", key, body, err)
		}
	}
	if address == "" || !expires.After(time.Now()) || expires.After(time.Now().Add(65*time.Minute)) {
		t.Fatalf("invalid image response %s", body)
	}
	u, err := url.Parse(address)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := base64.RawURLEncoding.DecodeString(u.Query().Get("payload"))
	if err != nil {
		t.Fatal(err)
	}
	var claim struct {
		ExpiresAt int64 `json:"expires_at"`
		Inline    bool  `json:"inline"`
	}
	if err := json.Unmarshal(payload, &claim); err != nil {
		t.Fatal(err)
	}
	if !claim.Inline || !expires.Equal(time.Unix(claim.ExpiresAt, 0)) {
		t.Fatalf("expiry differs from signature: %s %v", address, expires)
	}
	res, got := send(t, newRequest(t, "GET", w.base+address, "", nil))
	if res.StatusCode != 200 || !bytes.Equal(got, data) {
		t.Fatalf("owner image content: %d %q", res.StatusCode, got)
	}
}

func checkNullImageReply(t *testing.T, body, field string) {
	t.Helper()
	var reply map[string]json.RawMessage
	decodeAnswer(t, body, &reply)
	for _, key := range []string{field + "_asset_id", field + "_url", field + "_url_expires_at"} {
		if string(reply[key]) != "null" {
			t.Fatalf("%s must be present null in %s", key, body)
		}
	}
}
