package bootstrap

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"uuid"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
)

// A current bound logo belongs to the workspace; an unbound preparation still
// belongs to its applicant. Confusing these lets admins steal preparations or
// prevents a successor from replacing the former admin's logo.
func TestWorkspaceImages(t *testing.T) {
	w := newIssueWorld(t)
	data := assetTestPNG(t)
	scope := fmt.Sprintf(`{"kind":"workspace","id":%q}`, w.acme)
	upload := func(who, name string) uuid.UUID {
		return uploadedAssetHTTP(t, w, who, "WORKSPACE_LOGO", scope, name, "image/png", data).Asset.ID
	}
	patch := func(who string, id uuid.UUID, status int) string {
		return w.send(t, who, "PATCH", "/api/v0/workspaces/acme", fmt.Sprintf(`{"logo_asset_id":%q}`, id), status)
	}
	old := upload("alice", "old.png")
	checkImageReply(t, w, patch("alice", old, 200), "logo", old, data)
	for _, who := range []string{"alice", "wanda", "bob"} {
		checkImageReply(t, w, w.send(t, who, "GET", "/api/v0/workspaces/acme", "", 200), "logo", old, data)
		var list struct {
			Data []json.RawMessage `json:"data"`
		}
		decodeAnswer(t, w.send(t, who, "GET", "/api/v0/workspaces", "", 200), &list)
		if len(list.Data) != 1 {
			t.Fatalf("%s list: %s", who, list.Data)
		}
		checkImageReply(t, w, string(list.Data[0]), "logo", old, data)
	}
	pending := upload("alice", "not-bound.png")
	before := workspaceImageSnapshot(t, w)
	patch("wanda", pending, 404)
	patch("bob", uuid.New(), 403)
	member := queryIDs(t, w.pool, "SELECT id FROM workspace_members WHERE workspace_id=$1 AND member_id=$2", w.acme, w.ids["walt"])[0]
	w.send(t, "alice", "PATCH", "/api/v0/workspace-members/"+member.String(), `{"role":5}`, 200)
	patch("walt", uuid.New(), 403)
	hidden := patch("olga", uuid.New(), 404)
	missing := w.send(t, "olga", "PATCH", "/api/v0/workspaces/missing", fmt.Sprintf(`{"logo_asset_id":%q}`, uuid.New()), 404)
	if hidden != missing || before != workspaceImageSnapshot(t, w) {
		t.Fatal("rejection revealed a workspace or changed image state", hidden, missing)
	}
	next := upload("wanda", "next.png")
	checkImageReply(t, w, patch("wanda", next, 200), "logo", next, data)
	rows := assetpg.New(w.pool)
	removed, err := rows.FindAsset(t.Context(), old)
	if err != nil || removed.DeletedAt == nil || removed.DeletedByID == nil || *removed.DeletedByID != w.ids["wanda"] {
		t.Fatal("successor did not remove old logo", removed, err)
	}
	current, err := rows.FindAsset(t.Context(), next)
	if err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{fmt.Sprintf(`{"logo_asset_id":%q}`, next), `{}`, `{"name":"Renamed"}`} {
		checkImageReply(t, w, w.send(t, "wanda", "PATCH", "/api/v0/workspaces/acme", body, 200), "logo", next, data)
	}
	after, err := rows.FindAsset(t.Context(), next)
	if err != nil || !reflect.DeepEqual(current, after) {
		t.Fatal("unchanged logo rewrote asset", err)
	}
	checkNullImageReply(t, w.send(t, "wanda", "PATCH", "/api/v0/workspaces/acme", `{"logo_asset_id":null}`, 200), "logo")
	for i, size := range []string{"Just myself", "2-10", "11-50", "51-200", "201-500", "500+"} {
		body := w.send(t, "alice", "POST", "/api/v0/workspaces", fmt.Sprintf(`{"name":"Size %d","slug":"size-%d","organization_size":%q}`, i, i, size), 201)
		checkNullImageReply(t, body, "logo")
		var got struct {
			Size string `json:"organization_size"`
		}
		decodeAnswer(t, body, &got)
		if got.Size != size {
			t.Fatal("size changed", got, size)
		}
	}
	checkNullImageReply(t, w.send(t, "alice", "POST", "/api/v0/workspaces", `{"name":"None","slug":"none"}`, 201), "logo")
	assetAPICall(t, w, "alice", "PATCH", "/api/v0/workspaces/acme", `{"organization_size":"unknown"}`, 422)
	assetAPICall(t, w, "alice", "PATCH", "/api/v0/workspaces/acme", `{"organization_size":null}`, 400)
}

func workspaceImageSnapshot(t *testing.T, w issueWorld) string {
	t.Helper()
	var out string
	err := w.pool.QueryRow(t.Context(), `SELECT jsonb_build_array((SELECT to_jsonb(w) FROM workspaces w WHERE id=$1),
 (SELECT jsonb_agg(to_jsonb(a) ORDER BY id) FROM file_assets a))::text`, w.acme).Scan(&out)
	if err != nil {
		t.Fatal(err)
	}
	return out
}
