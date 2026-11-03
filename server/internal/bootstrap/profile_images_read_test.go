package bootstrap

import (
	"encoding/json"
	"fmt"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
)

func TestProfileImagesReadsDoNotWrite(t *testing.T) {
	w := newIssueWorld(t)
	data := assetTestPNG(t)
	avatar := uploadedAssetHTTP(t, w, "alice", "USER_AVATAR", `{"kind":"user"}`, "avatar.png", "image/png", data).Asset.ID
	patch := fmt.Sprintf(`{"avatar_asset_id":%q}`, avatar)
	w.send(t, "alice", "PATCH", "/api/v0/me", patch, 200)
	waitForCompletedKinds(t, w.pool, map[string]int{"asset.cleanup": 1, "cleanup.purge_deleted": 1}, &lockedBuffer{})
	if _, err := w.pool.Exec(t.Context(), `CREATE TABLE profile_read_writes (name text NOT NULL);
CREATE FUNCTION audit_profile_read() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN INSERT INTO profile_read_writes VALUES(TG_TABLE_NAME); RETURN NULL; END $$;
CREATE TRIGGER profile_read_user AFTER INSERT OR UPDATE OR DELETE ON users FOR EACH STATEMENT EXECUTE FUNCTION audit_profile_read();
CREATE TRIGGER profile_read_asset AFTER INSERT OR UPDATE OR DELETE ON file_assets FOR EACH STATEMENT EXECUTE FUNCTION audit_profile_read();`); err != nil {
		t.Fatal(err)
	}
	before := profileImageSnapshot(t, w)
	for range 3 {
		checkImageReply(t, w, w.send(t, "alice", "GET", "/api/v0/me", "", 200), "avatar", avatar, data)
	}
	var writes int
	if err := w.pool.QueryRow(t.Context(), "SELECT count(*) FROM profile_read_writes").Scan(&writes); err != nil || writes != 0 || profileImageSnapshot(t, w) != before {
		t.Fatalf("GET changed business rows: %d %v", writes, err)
	}
	profiles, err := identity.Provide(w.pool).PublicProfiles.PublicProfiles(t.Context(), []uuid.UUID{w.ids["alice"]})
	if err != nil || len(profiles) != 1 || profiles[0].AvatarAssetID == nil || *profiles[0].AvatarAssetID != avatar {
		t.Fatalf("public profile lost factual reference: %+v %v", profiles, err)
	}
	for _, body := range []string{patch, `{}`, `{"display_name":"alice"}`} {
		w.send(t, "alice", "PATCH", "/api/v0/me", body, 200)
	}
	if err := w.pool.QueryRow(t.Context(), "SELECT count(*) FROM profile_read_writes WHERE name='file_assets'").Scan(&writes); err != nil || writes != 0 {
		t.Fatalf("unchanged image wrote asset: %d %v", writes, err)
	}
	// An explicit stale-row fault must yield null URLs, not a new capability.
	if _, err := w.pool.Exec(t.Context(), "UPDATE file_assets SET deleted_at=CURRENT_TIMESTAMP,is_deleted=true WHERE id=$1", avatar); err != nil {
		t.Fatal(err)
	}
	var reply map[string]json.RawMessage
	decodeAnswer(t, w.send(t, "alice", "GET", "/api/v0/me", "", 200), &reply)
	if string(reply["avatar_url"]) != "null" || string(reply["avatar_url_expires_at"]) != "null" {
		t.Fatal("signed stale image", reply)
	}
}
