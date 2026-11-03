package bootstrap

import (
	"fmt"
	"strings"
	"testing"
	"uuid"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
)

func TestProjectCoverCreateRollback(t *testing.T) {
	for _, stage := range []string{"states", "apply", "signer", "commit"} {
		t.Run(stage, func(t *testing.T) {
			var signing *profileSigningFault
			w, _ := newAssetCompleteWorld(t, func(c *assetComponents) { signing = &profileSigningFault{assetStore: c.store}; c.store = signing })
			data := assetTestPNG(t)
			id := uploadedAssetHTTP(t, w, "alice", "PROJECT_COVER", assetScopeBody("new_project", w.acme), "cover.png", "image/png", data).Asset.ID
			waitForCompletedKinds(t, w.pool, map[string]int{"asset.cleanup": 1, "cleanup.purge_deleted": 1}, &lockedBuffer{})
			before := projectCoverSnapshot(t, w)
			table, trigger := "states", "CREATE TRIGGER reject_cover BEFORE INSERT ON states FOR EACH ROW EXECUTE FUNCTION reject_cover()"
			switch stage {
			case "apply":
				table, trigger = "file_assets", fmt.Sprintf("CREATE TRIGGER reject_cover BEFORE UPDATE ON file_assets FOR EACH ROW WHEN (NEW.id='%s' AND NEW.bound_at IS NOT NULL) EXECUTE FUNCTION reject_cover()", id)
			case "commit":
				table, trigger = "projects", "CREATE CONSTRAINT TRIGGER reject_cover AFTER INSERT ON projects DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION reject_cover()"
			case "signer":
				signing.fail.Store(true)
			}
			if stage != "signer" {
				if _, err := w.pool.Exec(t.Context(), `CREATE FUNCTION reject_cover() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'cover creation rejected'; END $$;`+trigger); err != nil {
					t.Fatal(err)
				}
			}
			body := fmt.Sprintf(`{"name":"Atomic cover","identifier":"ATOMIC","project_lead_id":%q,"cover_image_asset_id":%q}`, w.ids["bob"], id)
			got := w.send(t, "alice", "POST", "/api/v0/workspaces/acme/projects", body, 500)
			if problemCode(t, []byte(got)) != "internal_error" || strings.Contains(got, "url") || strings.Contains(got, "payload") {
				t.Fatal("failed creation exposed a partial capability", got)
			}
			if projectCoverSnapshot(t, w) != before {
				t.Fatal("failed POST retained project/children/asset migration or changed audit")
			}
			asset, err := assetpg.New(w.pool).FindAsset(t.Context(), id)
			if err != nil || asset.BoundAt != nil || asset.Placement.ProjectID != uuid.Nil() {
				t.Fatal("preparation was consumed", asset, err)
			}
			if stage == "signer" {
				signing.fail.Store(false)
			} else if _, err := w.pool.Exec(t.Context(), "DROP TRIGGER reject_cover ON "+table); err != nil {
				t.Fatal(err)
			}
			checkImageReply(t, w, w.send(t, "alice", "POST", "/api/v0/workspaces/acme/projects", body, 201), "cover_image", id, data)
		})
	}
}

func TestProjectCoverUpdateRollback(t *testing.T) {
	var signing *profileSigningFault
	w, _ := newAssetCompleteWorld(t, func(c *assetComponents) { signing = &profileSigningFault{assetStore: c.store}; c.store = signing })
	data := assetTestPNG(t)
	old := uploadedAssetHTTP(t, w, "alice", "PROJECT_COVER", assetScopeBody("project", w.web), "old.png", "image/png", data).Asset.ID
	next := uploadedAssetHTTP(t, w, "alice", "PROJECT_COVER", assetScopeBody("project", w.web), "next.png", "image/png", data).Asset.ID
	path := "/api/v0/projects/" + w.web.String()
	w.send(t, "alice", "PATCH", path, fmt.Sprintf(`{"cover_image_asset_id":%q}`, old), 200)
	waitForCompletedKinds(t, w.pool, map[string]int{"asset.cleanup": 1, "cleanup.purge_deleted": 1}, &lockedBuffer{})
	before := projectCoverSnapshot(t, w)
	signing.fail.Store(true)
	body := fmt.Sprintf(`{"cover_image_asset_id":%q,"name":"Must roll back"}`, next)
	w.send(t, "alice", "PATCH", path, body, 500)
	if projectCoverSnapshot(t, w) != before {
		t.Fatal("failed update kept changed project or image bindings")
	}
	signing.fail.Store(false)
	checkImageReply(t, w, w.send(t, "alice", "PATCH", path, body, 200), "cover_image", next, data)
}

func projectCoverSnapshot(t *testing.T, w issueWorld) string {
	t.Helper()
	var snapshot string
	if err := w.pool.QueryRow(t.Context(), `SELECT jsonb_build_array(
 (SELECT jsonb_agg(to_jsonb(p) ORDER BY id) FROM projects p),
 (SELECT jsonb_agg(to_jsonb(m) ORDER BY id) FROM project_members m),
 (SELECT jsonb_agg(to_jsonb(p) ORDER BY id) FROM project_user_properties p),
 (SELECT jsonb_agg(to_jsonb(s) ORDER BY id) FROM states s),
 (SELECT jsonb_agg(to_jsonb(a) ORDER BY id) FROM file_assets a))::text`).Scan(&snapshot); err != nil {
		t.Fatal(err)
	}
	return snapshot
}
