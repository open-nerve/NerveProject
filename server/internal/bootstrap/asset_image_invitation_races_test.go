package bootstrap

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestInvitationImageRace(t *testing.T) {
	for _, ending := range []string{"revoke", "accept", "deactivate nonmember"} {
		for _, endingFirst := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s first=%t", ending, endingFirst), func(t *testing.T) {
				w := newObservedIssueWorld(t)
				logo := bindRaceImage(t, w, "alice", "WORKSPACE_LOGO", assetScopeBody("workspace", w.acme), "/api/v0/workspaces/acme", "logo")
				link := invite(t, w.contract, w.base, w.tokens["alice"], "acme", "olga@example.com")
				path := "/api/v0/workspace-invitations/" + link.id.String()
				preview := newRequest(t, "GET", w.base+path+"?token="+url.QueryEscape(link.token), "", nil)
				method, endPath, who, body, endStatus := "DELETE", path, "alice", "", 204
				if ending == "accept" {
					method, endPath, who, body, endStatus = "POST", path+"/accept", "olga", fmt.Sprintf(`{"token":%q}`, link.token), 200
				}
				if ending == "deactivate nonmember" {
					method, endPath, who = "POST", "/api/v0/me/deactivate", "olga"
				}
				end := newRequest(t, method, w.base+endPath, w.tokens[who], []byte(body))
				previewProbe, endProbe := w.observe(preview), w.observe(end)
				var barrier pgx.Tx
				var seen, ended <-chan answer
				waitOn := "workspace_member_invites"
				if ending == "accept" {
					waitOn = "workspaces"
				}
				if endingFirst {
					barrier = lockedTables(t, w.pool, "workspace_member_invites", "SHARE")
					defer requestRelease(t, barrier)
					ended = sendInBackground(end)
					pid := requestBlocked(t, w.pool, endProbe, barrier.Conn().PgConn().PID(), "workspace_member_invites")
					seen = sendInBackground(preview)
					requestBlocked(t, w.pool, previewProbe, pid, waitOn)
				} else {
					barrier = holding(t, w.pool, "SELECT id FROM file_assets WHERE id=$1 FOR SHARE", logo)
					defer requestRelease(t, barrier)
					seen = sendInBackground(preview)
					pid := requestBlocked(t, w.pool, previewProbe, barrier.Conn().PgConn().PID(), "file_assets")
					ended = sendInBackground(end)
					requestBlocked(t, w.pool, endProbe, pid, waitOn)
				}
				requestRelease(t, barrier)
				imageRaceAnswer(t, w, end, ended, endStatus)
				want := 200
				if endingFirst {
					want = 404
				}
				got := imageRaceAnswer(t, w, preview, seen, want)
				if want == 200 {
					checkSignedImageReply(t, w, string(got.body), "workspace_logo", assetTestPNG(t))
				}
				w.send(t, "", "GET", path+"?token="+url.QueryEscape(link.token), "", 404)
			})
		}
	}
}
