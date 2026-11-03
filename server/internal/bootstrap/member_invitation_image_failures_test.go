package bootstrap

import (
	"fmt"
	"net/url"
	"strings"
	"testing"
)

// Image signing is part of the owner transaction: a failed response may not
// retain a membership cascade, accepted invitation or a partial capability.
func TestMemberAvatarAndInvitationImageRollback(t *testing.T) {
	var signing *profileSigningFault
	w, _ := newAssetCompleteWorld(t, func(c *assetComponents) { signing = &profileSigningFault{assetStore: c.store}; c.store = signing })
	data := assetTestPNG(t)
	avatar := uploadedAssetHTTP(t, w, "bob", "USER_AVATAR", `{"kind":"user"}`, "avatar.png", "image/png", data).Asset.ID
	w.send(t, "bob", "PATCH", "/api/v0/me", fmt.Sprintf(`{"avatar_asset_id":%q}`, avatar), 200)
	logo := uploadedAssetHTTP(t, w, "alice", "WORKSPACE_LOGO", assetScopeBody("workspace", w.acme), "logo.png", "image/png", data).Asset.ID
	w.send(t, "alice", "PATCH", "/api/v0/workspaces/acme", fmt.Sprintf(`{"logo_asset_id":%q}`, logo), 200)
	link := invite(t, w.contract, w.base, w.tokens["alice"], "acme", "olga@example.com")
	member := queryIDs(t, w.pool, "SELECT id FROM workspace_members WHERE workspace_id=$1 AND member_id=$2", w.acme, w.ids["bob"])[0]
	memberPath := "/api/v0/workspace-members/" + member.String()
	invitePath := "/api/v0/workspace-invitations/" + link.id.String()
	before := memberInvitationImageSnapshot(t, w)
	signing.fail.Store(true)
	for _, request := range []struct{ who, method, path, body string }{
		{"alice", "PATCH", memberPath, `{"role":5}`},
		{"alice", "GET", "/api/v0/workspaces/acme/members", ""},
		{"olga", "POST", invitePath + "/accept", fmt.Sprintf(`{"token":%q}`, link.token)},
		{"", "GET", invitePath + "?token=" + url.QueryEscape(link.token), ""},
	} {
		body := w.send(t, request.who, request.method, request.path, request.body, 500)
		if problemCode(t, []byte(body)) != "internal_error" || strings.Contains(body, "url") || strings.Contains(body, "payload") {
			t.Fatal("failure leaked image capability", body)
		}
		if memberInvitationImageSnapshot(t, w) != before {
			t.Fatal("signing failure left membership/invitation/image writes", request)
		}
	}
	signing.fail.Store(false)
	w.send(t, "alice", "PATCH", memberPath, `{"role":5}`, 200)
	joined := w.send(t, "olga", "POST", invitePath+"/accept", fmt.Sprintf(`{"token":%q}`, link.token), 200)
	checkImageReply(t, w, joined, "logo", logo, data)
}

func memberInvitationImageSnapshot(t *testing.T, w issueWorld) string {
	t.Helper()
	var out string
	if err := w.pool.QueryRow(t.Context(), `SELECT jsonb_build_array(
 (SELECT jsonb_agg(to_jsonb(m) ORDER BY id) FROM workspace_members m),
 (SELECT jsonb_agg(to_jsonb(m) ORDER BY id) FROM project_members m),
 (SELECT jsonb_agg(to_jsonb(i) ORDER BY id) FROM workspace_member_invites i),
 (SELECT jsonb_agg(to_jsonb(a) ORDER BY id) FROM file_assets a))::text`).Scan(&out); err != nil {
		t.Fatal(err)
	}
	return out
}
