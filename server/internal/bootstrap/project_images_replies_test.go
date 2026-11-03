package bootstrap

import (
	"encoding/json"
	"fmt"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Every response carrying Project must carry the same usable cover capability;
// signing after commit would also make the rollback cases below retain writes.
func TestProjectImageReplies(t *testing.T) {
	w := newIssueWorld(t)
	data := assetTestPNG(t)
	image := uploadedAssetHTTP(t, w, "alice", "PROJECT_COVER", assetScopeBody("new_project", w.acme), "all.png", "image/png", data).Asset.ID
	body := w.send(t, "alice", "POST", "/api/v0/workspaces/acme/projects", fmt.Sprintf(`{"name":"Alpha","identifier":"ALL","cover_image_asset_id":%q}`, image), 201)
	checkImageReply(t, w, body, "cover_image", image, data)
	var created struct{ ID uuid.UUID }
	decodeAnswer(t, body, &created)
	path := "/api/v0/projects/" + created.ID.String()
	checkImageReply(t, w, w.send(t, "alice", "PATCH", path, `{"description":"With image"}`, 200), "cover_image", image, data)
	w.makeMember(t, "walt", created.ID, shared.RoleGuest)
	waitForCompletedKinds(t, w.pool, map[string]int{"asset.cleanup": 1, "cleanup.purge_deleted": 1}, &lockedBuffer{})
	if _, err := w.pool.Exec(t.Context(), `CREATE TABLE project_image_writes(name text NOT NULL);
CREATE FUNCTION audit_project_image() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN INSERT INTO project_image_writes VALUES(TG_TABLE_NAME); RETURN NULL; END $$;
CREATE TRIGGER project_image_write AFTER INSERT OR UPDATE OR DELETE ON projects FOR EACH STATEMENT EXECUTE FUNCTION audit_project_image();
CREATE TRIGGER project_image_asset AFTER INSERT OR UPDATE OR DELETE ON file_assets FOR EACH STATEMENT EXECUTE FUNCTION audit_project_image();`); err != nil {
		t.Fatal(err)
	}
	before := projectCoverSnapshot(t, w)
	for _, who := range []string{"alice", "bob", "walt"} {
		t.Run("get "+who, func(t *testing.T) {
			checkImageReply(t, w, w.send(t, who, "GET", path, "", 200), "cover_image", image, data)
		})
	}
	t.Run("list", func(t *testing.T) {
		list := projectImageList(t, w, "alice", false)
		if len(list) != 2 {
			t.Fatal("list lost unrelated project", len(list))
		}
		checkImageReply(t, w, string(list[0]), "cover_image", image, data)
	})
	var writes int
	if err := w.pool.QueryRow(t.Context(), "SELECT count(*) FROM project_image_writes").Scan(&writes); err != nil || writes != 0 || before != projectCoverSnapshot(t, w) {
		t.Fatal("image reads changed owner or asset audit", writes, err)
	}
	for _, action := range []string{"archive", "unarchive", "join"} {
		t.Run(action, func(t *testing.T) {
			who := "alice"
			if action == "join" {
				who = "wanda"
			}
			checkImageReply(t, w, w.send(t, who, "POST", path+"/"+action, "", 200), "cover_image", image, data)
			if action == "archive" {
				list := projectImageList(t, w, "alice", true)
				if len(list) != 1 {
					t.Fatal("archived filtering lost cover", len(list))
				}
				checkImageReply(t, w, string(list[0]), "cover_image", image, data)
				checkImageReply(t, w, w.send(t, "alice", "GET", path, "", 200), "cover_image", image, data)
			}
		})
	}
	w.send(t, "alice", "PATCH", path, `{"network":0}`, 200)
	hidden := w.send(t, "bob", "GET", path, "", 404)
	missing := w.send(t, "bob", "GET", "/api/v0/projects/"+uuid.New().String(), "", 404)
	if hidden != missing || len(projectImageList(t, w, "bob", false)) != 1 {
		t.Fatal("private project escaped visibility", hidden, missing)
	}
	checkImageReply(t, w, w.send(t, "walt", "GET", path, "", 200), "cover_image", image, data)
	w.send(t, "alice", "DELETE", path, "", 204)
	w.send(t, "alice", "GET", path, "", 404)
	if len(projectImageList(t, w, "alice", false)) != 1 {
		t.Fatal("deleted project survived in directory")
	}
}

func TestProjectImageRepliesSigningRollback(t *testing.T) {
	var signing *profileSigningFault
	w, _ := newAssetCompleteWorld(t, func(c *assetComponents) { signing = &profileSigningFault{assetStore: c.store}; c.store = signing })
	data := assetTestPNG(t)
	image := uploadedAssetHTTP(t, w, "alice", "PROJECT_COVER", assetScopeBody("project", w.web), "cover.png", "image/png", data).Asset.ID
	path := "/api/v0/projects/" + w.web.String()
	w.send(t, "alice", "PATCH", path, fmt.Sprintf(`{"cover_image_asset_id":%q}`, image), 200)
	waitForCompletedKinds(t, w.pool, map[string]int{"asset.cleanup": 1, "cleanup.purge_deleted": 1}, &lockedBuffer{})
	for _, action := range []string{"get", "list", "archive", "unarchive", "join"} {
		t.Run(action, func(t *testing.T) {
			if action == "unarchive" {
				w.send(t, "alice", "POST", path+"/archive", "", 200)
			}
			before := projectCoverSnapshot(t, w)
			method, target, who := "POST", path+"/"+action, "alice"
			switch action {
			case "get":
				method, target = "GET", path
			case "list":
				method, target = "GET", "/api/v0/workspaces/acme/projects"
			case "join":
				who = "wanda"
			}
			signing.fail.Store(true)
			defer signing.fail.Store(false)
			w.send(t, who, method, target, "", 500)
			if before != projectCoverSnapshot(t, w) {
				t.Fatal("failed signing committed project or membership state")
			}
		})
	}
}

func projectImageList(t *testing.T, w issueWorld, who string, archived bool) []json.RawMessage {
	t.Helper()
	var list struct{ Data []json.RawMessage }
	decodeAnswer(t, w.send(t, who, "GET", fmt.Sprintf("/api/v0/workspaces/acme/projects?archived=%t", archived), "", 200), &list)
	return list.Data
}
