package bootstrap

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"uuid"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
)

func TestProjectCoverCreate(t *testing.T) {
	w := newIssueWorld(t)
	for _, builtin := range []string{"cover-01", "cover-29"} {
		name := "Builtin " + builtin[6:]
		body := w.send(t, "alice", "POST", "/api/v0/workspaces/acme/projects", fmt.Sprintf(`{"name":%q,"identifier":%q,"cover_builtin_id":%q}`, name, "B"+builtin[6:], builtin), 201)
		checkProjectBuiltin(t, body, builtin)
	}
	data := assetTestPNG(t)
	image := uploadedAssetHTTP(t, w, "alice", "PROJECT_COVER", assetScopeBody("new_project", w.acme), "cover.png", "image/png", data).Asset.ID
	body := w.send(t, "alice", "POST", "/api/v0/workspaces/acme/projects", fmt.Sprintf(`{"name":"With cover","identifier":"COVER","project_lead_id":%q,"cover_image_asset_id":%q}`, w.ids["bob"], image), 201)
	checkImageReply(t, w, body, "cover_image", image, data)
	var project struct{ ID uuid.UUID }
	decodeAnswer(t, body, &project)
	var complete bool
	err := w.pool.QueryRow(t.Context(), `SELECT p.cover_image_asset_id=$2 AND p.cover_builtin_id IS NULL
 AND a.project_id=p.id AND a.workspace_id=p.workspace_id AND a.scope_kind='project'
 AND a.bound_at=p.created_at AND a.updated_at=p.created_at
 AND (SELECT count(*) FROM project_members m WHERE m.project_id=p.id AND m.role=20 AND m.created_at=p.created_at)=2
 AND (SELECT count(*) FROM project_user_properties m WHERE m.project_id=p.id AND m.created_at=p.created_at)=2
 AND (SELECT count(*) FROM states s WHERE s.project_id=p.id AND s.created_at=p.created_at)=6
 FROM projects p JOIN file_assets a ON a.id=p.cover_image_asset_id WHERE p.id=$1`, project.ID, image).Scan(&complete)
	if err != nil || !complete {
		t.Fatal("POST did not atomically create the whole project and bind at one time", complete, err)
	}
	for _, builtin := range []string{"", "cover-00", "cover-30", "COVER-01", "https://example.com/image.png"} {
		assetAPICall(t, w, "alice", "POST", "/api/v0/workspaces/acme/projects", fmt.Sprintf(`{"name":"Invalid","identifier":"INVALID","cover_builtin_id":%q}`, builtin), 422)
	}
	assetAPICall(t, w, "alice", "POST", "/api/v0/workspaces/acme/projects", fmt.Sprintf(`{"name":"Invalid","identifier":"INVALID","cover_builtin_id":"cover-01","cover_image_asset_id":%q}`, uuid.New()), 422)
	preparation := uploadedAssetHTTP(t, w, "alice", "PROJECT_COVER", assetScopeBody("new_project", w.acme), "owned.png", "image/png", data).Asset.ID
	assetAPICall(t, w, "bob", "POST", "/api/v0/workspaces/acme/projects", fmt.Sprintf(`{"name":"Not mine","identifier":"NOTMINE","cover_image_asset_id":%q}`, preparation), 404)
	wrongScope := uploadedAssetHTTP(t, w, "alice", "PROJECT_COVER", assetScopeBody("project", w.web), "old.png", "image/png", data).Asset.ID
	assetAPICall(t, w, "alice", "POST", "/api/v0/workspaces/acme/projects", fmt.Sprintf(`{"name":"Wrong scope","identifier":"WRONG","cover_image_asset_id":%q}`, wrongScope), 404)
}

func TestProjectCoverUpdate(t *testing.T) {
	w := newIssueWorld(t)
	data := assetTestPNG(t)
	path := "/api/v0/projects/" + w.web.String()
	upload := func(who string) uuid.UUID {
		return uploadedAssetHTTP(t, w, who, "PROJECT_COVER", assetScopeBody("project", w.web), "cover.png", "image/png", data).Asset.ID
	}
	old := upload("alice")
	checkImageReply(t, w, w.send(t, "alice", "PATCH", path, fmt.Sprintf(`{"cover_image_asset_id":%q}`, old), 200), "cover_image", old, data)
	for _, body := range []string{`{}`, `{"cover_builtin_id":null}`, fmt.Sprintf(`{"cover_image_asset_id":%q}`, old)} {
		checkImageReply(t, w, w.send(t, "alice", "PATCH", path, body, 200), "cover_image", old, data)
	}
	assetAPICall(t, w, "bob", "PATCH", path, fmt.Sprintf(`{"cover_image_asset_id":%q}`, uuid.New()), 403)
	hidden := assetAPICall(t, w, "olga", "PATCH", path, fmt.Sprintf(`{"cover_image_asset_id":%q}`, uuid.New()), 404)
	missing := assetAPICall(t, w, "olga", "PATCH", "/api/v0/projects/"+uuid.New().String(), fmt.Sprintf(`{"cover_image_asset_id":%q}`, uuid.New()), 404)
	if hidden != missing {
		t.Fatal("hidden project disclosed image state", hidden, missing)
	}
	w.send(t, "wanda", "POST", path+"/join", "", 200)
	pending := upload("alice")
	assetAPICall(t, w, "wanda", "PATCH", path, fmt.Sprintf(`{"cover_image_asset_id":%q}`, pending), 404)
	next := upload("wanda")
	checkImageReply(t, w, w.send(t, "wanda", "PATCH", path, fmt.Sprintf(`{"cover_image_asset_id":%q}`, next), 200), "cover_image", next, data)
	rows := assetpg.New(w.pool)
	removed, err := rows.FindAsset(t.Context(), old)
	if err != nil || removed.DeletedAt == nil || removed.DeletedByID == nil || *removed.DeletedByID != w.ids["wanda"] {
		t.Fatal("old applicant prevented successor replacement", removed, err)
	}
	before, err := rows.FindAsset(t.Context(), next)
	if err != nil {
		t.Fatal(err)
	}
	checkImageReply(t, w, w.send(t, "alice", "PATCH", path, fmt.Sprintf(`{"cover_image_asset_id":%q}`, next), 200), "cover_image", next, data)
	after, err := rows.FindAsset(t.Context(), next)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("current image idempotence changed audit", err)
	}
	checkProjectBuiltin(t, w.send(t, "alice", "PATCH", path, `{"cover_builtin_id":"cover-29"}`, 200), "cover-29")
	checkProjectBuiltin(t, w.send(t, "alice", "PATCH", path, `{"cover_image_asset_id":null}`, 200), "cover-29")
	removed, err = rows.FindAsset(t.Context(), next)
	if err != nil || removed.DeletedAt == nil {
		t.Fatal("builtin switch left prior image alive", removed, err)
	}
	last := upload("alice")
	body := w.send(t, "alice", "PATCH", path, fmt.Sprintf(`{"cover_image_asset_id":%q}`, last), 200)
	checkImageReply(t, w, body, "cover_image", last, data)
	var reply map[string]json.RawMessage
	decodeAnswer(t, body, &reply)
	if string(reply["cover_builtin_id"]) != "null" {
		t.Fatal("upload did not clear builtin", body)
	}
	checkNullImageReply(t, w.send(t, "alice", "PATCH", path, `{"cover_image_asset_id":null,"cover_builtin_id":null}`, 200), "cover_image")
	w.send(t, "alice", "POST", path+"/archive", "", 200)
	refused := assetAPICall(t, w, "alice", "PATCH", path, fmt.Sprintf(`{"cover_image_asset_id":%q}`, uuid.New()), 409)
	if problemCode(t, []byte(refused)) != "project.archived" {
		t.Fatal("asset lookup preceded archive decision", refused)
	}
}

func checkProjectBuiltin(t *testing.T, body, builtin string) {
	t.Helper()
	checkNullImageReply(t, body, "cover_image")
	var got struct {
		Builtin *string `json:"cover_builtin_id"`
	}
	decodeAnswer(t, body, &got)
	if got.Builtin == nil || *got.Builtin != builtin {
		t.Fatal("builtin ID not preserved", body)
	}
}
