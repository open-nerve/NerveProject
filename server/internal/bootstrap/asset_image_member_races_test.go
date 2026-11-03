package bootstrap

import (
	"encoding/json"
	"fmt"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5"
)

// Taking a target account lock after W would wait on Bob while his deactivation
// waits on W. Current public-profile reads must not take that reverse edge.
func TestMemberAvatarLockOrder(t *testing.T) {
	for _, caller := range []string{"alice", "wanda"} {
		for _, change := range []string{"replace", "deactivate"} {
			for _, changeFirst := range []bool{true, false} {
				t.Run(fmt.Sprintf("%s %s first=%t", caller, change, changeFirst), func(t *testing.T) {
					w := newObservedIssueWorld(t)
					old := bindRaceImage(t, w, "bob", "USER_AVATAR", `{"kind":"user"}`, "/api/v0/me", "avatar")
					method, path, body, status := "POST", "/api/v0/me/deactivate", "", 204
					if change == "replace" {
						next := uploadedAssetHTTP(t, w, "bob", "USER_AVATAR", `{"kind":"user"}`, "new.png", "image/png", assetTestPNG(t)).Asset.ID
						method, path, body, status = "PATCH", "/api/v0/me", fmt.Sprintf(`{"avatar_asset_id":%q}`, next), 200
					}
					readReq := newRequest(t, "GET", w.base+"/api/v0/workspaces/acme/members", w.tokens[caller], nil)
					changeReq := newRequest(t, method, w.base+path, w.tokens["bob"], []byte(body))
					readProbe, changeProbe := w.observe(readReq), w.observe(changeReq)
					var changed, read <-chan answer
					var barrier pgx.Tx
					if changeFirst && change == "replace" {
						// The replacement already owns old/new assets, but the old user
						// reference remains visible until its transaction commits.
						barrier = lockedTables(t, w.pool, "users", "SHARE")
						defer requestRelease(t, barrier)
						changed = sendInBackground(changeReq)
						pid := requestBlocked(t, w.pool, changeProbe, barrier.Conn().PgConn().PID(), "users")
						read = sendInBackground(readReq)
						requestBlocked(t, w.pool, readProbe, pid, "file_assets")
					} else {
						barrier = holding(t, w.pool, "SELECT id FROM file_assets WHERE id=$1 FOR SHARE", old)
						defer requestRelease(t, barrier)
						var beforeW pgx.Tx
						if changeFirst {
							beforeW = lockedTables(t, w.pool, "profiles", "SHARE")
							defer requestRelease(t, beforeW)
							changed = sendInBackground(changeReq)
							requestBlocked(t, w.pool, changeProbe, beforeW.Conn().PgConn().PID(), "profiles")
						}
						read = sendInBackground(readReq)
						pid := requestBlocked(t, w.pool, readProbe, barrier.Conn().PgConn().PID(), "file_assets")
						if changeFirst {
							requestRelease(t, beforeW)
						} else {
							changed = sendInBackground(changeReq)
						}
						waitTable := "workspaces"
						if change == "replace" {
							waitTable = "file_assets"
						}
						requestBlocked(t, w.pool, changeProbe, pid, waitTable)
					}
					requestRelease(t, barrier)
					imageRaceAnswer(t, w, changeReq, changed, status)
					got := imageRaceAnswer(t, w, readReq, read, 200)
					member := memberImageBody(t, string(got.body), w.ids["bob"])
					if changeFirst && change == "replace" {
						var image struct {
							URL    *string `json:"avatar_url"`
							Expiry *string `json:"avatar_url_expires_at"`
						}
						decodeAnswer(t, member, &image)
						if image.URL != nil || image.Expiry != nil {
							t.Fatal("signed a replaced reference or chased a new one", member)
						}
					} else {
						checkSignedImageReply(t, w, member, "avatar", assetTestPNG(t))
					}
					// Ended/deactivated public profiles remain visible by the M3 contract.
					fresh := w.send(t, caller, "GET", "/api/v0/workspaces/acme/members", "", 200)
					checkSignedImageReply(t, w, memberImageBody(t, fresh, w.ids["bob"]), "avatar", assetTestPNG(t))
				})
			}
		}
	}
}

func memberImageBody(t *testing.T, body string, user uuid.UUID) string {
	t.Helper()
	var list struct {
		Data []struct{ Member json.RawMessage }
	}
	decodeAnswer(t, body, &list)
	for _, m := range list.Data {
		var member struct{ ID uuid.UUID }
		decodeAnswer(t, string(m.Member), &member)
		if member.ID == user {
			return string(m.Member)
		}
	}
	t.Fatal("lost target public profile", body)
	return ""
}
