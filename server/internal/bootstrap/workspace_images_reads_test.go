package bootstrap

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
	"uuid"
)

func TestWorkspaceImagesReadsAndRollback(t *testing.T) {
	var signing *profileSigningFault
	w, _ := newAssetCompleteWorld(t, func(c *assetComponents) { signing = &profileSigningFault{assetStore: c.store}; c.store = signing })
	data := assetTestPNG(t)
	upload := func() uuid.UUID {
		return uploadedAssetHTTP(t, w, "alice", "WORKSPACE_LOGO", fmt.Sprintf(`{"kind":"workspace","id":%q}`, w.acme), "logo.png", "image/png", data).Asset.ID
	}
	old, next := upload(), upload()
	body := w.send(t, "alice", "PATCH", "/api/v0/workspaces/acme", fmt.Sprintf(`{"logo_asset_id":%q}`, old), 200)
	checkImageReply(t, w, body, "logo", old, data)
	var ownerAt, boundAt, assetAt time.Time
	if err := w.pool.QueryRow(t.Context(), `SELECT w.updated_at,a.bound_at,a.updated_at FROM workspaces w JOIN file_assets a ON a.id=w.logo_asset_id WHERE w.id=$1`, w.acme).Scan(&ownerAt, &boundAt, &assetAt); err != nil || !ownerAt.Equal(boundAt) || !ownerAt.Equal(assetAt) {
		t.Fatal("logo audit time differs", ownerAt, boundAt, assetAt, err)
	}
	waitForCompletedKinds(t, w.pool, map[string]int{"asset.cleanup": 1, "cleanup.purge_deleted": 1}, &lockedBuffer{})
	if _, err := w.pool.Exec(t.Context(), `CREATE TABLE workspace_image_writes(name text NOT NULL);
CREATE FUNCTION audit_workspace_image() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN INSERT INTO workspace_image_writes VALUES(TG_TABLE_NAME); RETURN NULL; END $$;
CREATE TRIGGER workspace_image_write AFTER INSERT OR UPDATE OR DELETE ON workspaces FOR EACH STATEMENT EXECUTE FUNCTION audit_workspace_image();
CREATE TRIGGER workspace_image_asset AFTER INSERT OR UPDATE OR DELETE ON file_assets FOR EACH STATEMENT EXECUTE FUNCTION audit_workspace_image();`); err != nil {
		t.Fatal(err)
	}
	before := workspaceImageSnapshot(t, w)
	for range 3 {
		checkImageReply(t, w, w.send(t, "bob", "GET", "/api/v0/workspaces/acme", "", 200), "logo", old, data)
		var list struct {
			Data []json.RawMessage `json:"data"`
		}
		decodeAnswer(t, w.send(t, "bob", "GET", "/api/v0/workspaces", "", 200), &list)
		if len(list.Data) != 1 {
			t.Fatal("list lost workspace")
		}
		checkImageReply(t, w, string(list.Data[0]), "logo", old, data)
	}
	var writes int
	if err := w.pool.QueryRow(t.Context(), "SELECT count(*) FROM workspace_image_writes").Scan(&writes); err != nil || writes != 0 || before != workspaceImageSnapshot(t, w) {
		t.Fatal("read changed business rows", writes, err)
	}
	signing.fail.Store(true)
	patch := fmt.Sprintf(`{"name":"must roll back","logo_asset_id":%q}`, next)
	w.send(t, "alice", "PATCH", "/api/v0/workspaces/acme", patch, 500)
	w.send(t, "bob", "GET", "/api/v0/workspaces/acme", "", 500)
	w.send(t, "bob", "GET", "/api/v0/workspaces", "", 500)
	if before != workspaceImageSnapshot(t, w) {
		t.Fatal("failed response retained a partial logo/name/binding")
	}
	signing.fail.Store(false)
	checkImageReply(t, w, w.send(t, "alice", "PATCH", "/api/v0/workspaces/acme", patch, 200), "logo", next, data)
}
