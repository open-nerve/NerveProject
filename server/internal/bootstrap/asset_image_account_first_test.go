package bootstrap

import (
	"fmt"
	"testing"
)

// All contested parent/asset tables are blocked at once. A real request's
// first locking SQL must wait on users; observing any later table fails.
func TestImageAuthAccountFirst(t *testing.T) {
	w := newObservedIssueWorld(t)
	p := "/api/v0/projects/" + w.web.String()
	bindRaceImage(t, w, "alice", "USER_AVATAR", `{"kind":"user"}`, "/api/v0/me", "avatar")
	bindRaceImage(t, w, "alice", "WORKSPACE_LOGO", assetScopeBody("workspace", w.acme), "/api/v0/workspaces/acme", "logo")
	bindRaceImage(t, w, "alice", "PROJECT_COVER", assetScopeBody("project", w.web), p, "cover_image")
	member := queryIDs(t, w.pool, "SELECT id FROM workspace_members WHERE workspace_id=$1 AND member_id=$2", w.acme, w.ids["bob"])[0]
	link := invite(t, w.contract, w.base, w.tokens["alice"], "acme", "olga@example.com")
	prepared := uploadedAssetHTTP(t, w, "alice", "PROJECT_COVER", assetScopeBody("new_project", w.acme), "new.png", "image/png", assetTestPNG(t)).Asset.ID
	for _, tc := range []struct {
		name, who, method, path, body string
		status                        int
	}{
		{"me get", "alice", "GET", "/api/v0/me", "", 200},
		{"me update", "alice", "PATCH", "/api/v0/me", `{}`, 200},
		{"workspace get", "alice", "GET", "/api/v0/workspaces/acme", "", 200},
		{"workspace list", "alice", "GET", "/api/v0/workspaces", "", 200},
		{"workspace update", "alice", "PATCH", "/api/v0/workspaces/acme", `{}`, 200},
		{"member list", "alice", "GET", "/api/v0/workspaces/acme/members", "", 200},
		{"member update", "alice", "PATCH", "/api/v0/workspace-members/" + member.String(), `{"role":15}`, 200},
		{"invitation accept", "olga", "POST", "/api/v0/workspace-invitations/" + link.id.String() + "/accept", fmt.Sprintf(`{"token":%q}`, link.token), 200},
		{"project get", "alice", "GET", p, "", 200},
		{"project list", "alice", "GET", "/api/v0/workspaces/acme/projects", "", 200},
		{"project create", "alice", "POST", "/api/v0/workspaces/acme/projects", fmt.Sprintf(`{"name":"First lock","identifier":"FIRST","cover_image_asset_id":%q}`, prepared), 201},
		{"project update", "alice", "PATCH", p, `{}`, 200},
		{"project archive", "alice", "POST", p + "/archive", "", 200},
		{"project unarchive", "alice", "POST", p + "/unarchive", "", 200},
		{"project join", "wanda", "POST", p + "/join", "", 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			barrier := lockedTables(t, w.pool, "users, workspaces, projects, file_assets", "EXCLUSIVE")
			defer requestRelease(t, barrier)
			req := newRequest(t, tc.method, w.base+tc.path, w.tokens[tc.who], []byte(tc.body))
			probe := w.observe(req)
			response := sendInBackground(req)
			requestTableBlocked(t, w.pool, probe, barrier.Conn().PgConn().PID(), "users")
			requestRelease(t, barrier)
			imageRaceAnswer(t, w, req, response, tc.status)
		})
	}
}
