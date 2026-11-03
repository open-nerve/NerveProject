package bootstrap

import (
	"encoding/json"
	"fmt"
	"testing"
	"uuid"
)

func TestMemberAvatarReplies(t *testing.T) {
	w := newIssueWorld(t)
	data := assetTestPNG(t)
	id := uploadedAssetHTTP(t, w, "bob", "USER_AVATAR", `{"kind":"user"}`, "avatar.png", "image/png", data).Asset.ID
	w.send(t, "bob", "PATCH", "/api/v0/me", fmt.Sprintf(`{"avatar_asset_id":%q}`, id), 200)
	read := func(who string, emails bool) {
		t.Helper()
		var list struct {
			Data []struct {
				Member json.RawMessage `json:"member"`
			} `json:"data"`
		}
		decodeAnswer(t, w.send(t, who, "GET", "/api/v0/workspaces/acme/members", "", 200), &list)
		found := false
		for _, entry := range list.Data {
			var m struct {
				ID    uuid.UUID `json:"id"`
				Email *string   `json:"email"`
			}
			decodeAnswer(t, string(entry.Member), &m)
			if (m.Email != nil) != emails {
				t.Fatal("guest email boundary changed", m)
			}
			if m.ID == w.ids["bob"] {
				checkSignedImageReply(t, w, string(entry.Member), "avatar", data)
				found = true
			}
		}
		if !found {
			t.Fatal("lost member profile")
		}
	}
	read("alice", true)
	membership := queryIDs(t, w.pool, "SELECT id FROM workspace_members WHERE workspace_id=$1 AND member_id=$2", w.acme, w.ids["bob"])[0]
	var updated struct {
		Member json.RawMessage `json:"member"`
	}
	decodeAnswer(t, w.send(t, "alice", "PATCH", "/api/v0/workspace-members/"+membership.String(), `{"role":20}`, 200), &updated)
	checkSignedImageReply(t, w, string(updated.Member), "avatar", data)
	guest := queryIDs(t, w.pool, "SELECT id FROM workspace_members WHERE workspace_id=$1 AND member_id=$2", w.acme, w.ids["walt"])[0]
	w.send(t, "alice", "PATCH", "/api/v0/workspace-members/"+guest.String(), `{"role":5}`, 200)
	read("walt", false)
	w.send(t, "bob", "POST", "/api/v0/me/deactivate", "", 204)
	read("alice", true)
}
