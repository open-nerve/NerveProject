package bootstrap

import (
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
)

// A revoked/hidden owner cannot issue a new capability after waiting. Archive
// leaves reads available, but must reject a replacement that reaches it later.
func TestImageParentRace(t *testing.T) {
	for _, change := range []string{"workspace revoke", "workspace delete", "project private", "project revoke", "project archive"} {
		for _, changeFirst := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s first=%t", change, changeFirst), func(t *testing.T) {
				w := newObservedIssueWorld(t)
				projectPath := "/api/v0/projects/" + w.web.String()
				readPath, parentTable, barrierTable := projectPath, "projects", "projects"
				method, who, body := "GET", "bob", ""
				changeMethod, changePath, changeBody, changeStatus := "PATCH", projectPath, `{"network":0}`, 200
				purpose, placement, field := "PROJECT_COVER", assetScopeBody("project", w.web), "cover_image"
				if change == "workspace revoke" || change == "workspace delete" {
					readPath, parentTable, barrierTable = "/api/v0/workspaces/acme", "workspaces", "workspaces"
					purpose, placement, field = "WORKSPACE_LOGO", assetScopeBody("workspace", w.acme), "logo"
					changeMethod, changePath, changeBody, changeStatus = "DELETE", readPath, "", 204
				}
				image := bindRaceImage(t, w, "alice", purpose, placement, readPath, field)
				switch change {
				case "workspace revoke", "project revoke":
					scope, parentID := "project", w.web
					if change == "workspace revoke" {
						scope, parentID = "workspace", w.acme
					} else {
						w.send(t, "alice", "PATCH", projectPath, `{"network":0}`, 200)
					}
					barrierTable = scope + "_members"
					member := queryIDs(t, w.pool, "SELECT id FROM "+barrierTable+" WHERE "+scope+"_id=$1 AND member_id=$2", parentID, w.ids["bob"])[0]
					changeMethod, changePath, changeBody, changeStatus = "DELETE", "/api/v0/"+scope+"-members/"+member.String(), "", 204
				case "project private":
					who = "walt" // Visible as a workspace member, not a project member.
				case "project archive":
					w.send(t, "wanda", "POST", projectPath+"/join", "", 200)
					image = uploadedAssetHTTP(t, w, "wanda", purpose, placement, "next.png", "image/png", assetTestPNG(t)).Asset.ID
					method, who, body = "PATCH", "wanda", fmt.Sprintf(`{"cover_image_asset_id":%q}`, image)
					changeMethod, changePath, changeBody = "POST", projectPath+"/archive", ""
				}
				changeReq := newRequest(t, changeMethod, w.base+changePath, w.tokens["alice"], []byte(changeBody))
				signReq := newRequest(t, method, w.base+readPath, w.tokens[who], []byte(body))
				changeProbe, signProbe := w.observe(changeReq), w.observe(signReq)
				var barrier pgx.Tx
				var changed, signed <-chan answer
				if changeFirst {
					barrier = lockedTables(t, w.pool, barrierTable, "SHARE")
					defer requestRelease(t, barrier)
					changed = sendInBackground(changeReq)
					pid := requestBlocked(t, w.pool, changeProbe, barrier.Conn().PgConn().PID(), barrierTable)
					signed = sendInBackground(signReq)
					requestBlocked(t, w.pool, signProbe, pid, parentTable)
				} else {
					barrier = holding(t, w.pool, "SELECT id FROM file_assets WHERE id=$1 FOR SHARE", image)
					defer requestRelease(t, barrier)
					signed = sendInBackground(signReq)
					pid := requestBlocked(t, w.pool, signProbe, barrier.Conn().PgConn().PID(), "file_assets")
					changed = sendInBackground(changeReq)
					requestBlocked(t, w.pool, changeProbe, pid, parentTable)
				}
				requestRelease(t, barrier)
				imageRaceAnswer(t, w, changeReq, changed, changeStatus)
				want := 200
				if changeFirst {
					want = 404
					if change == "project archive" {
						want = 409
					}
				}
				got := imageRaceAnswer(t, w, signReq, signed, want)
				if want == 200 {
					checkImageReply(t, w, string(got.body), field, image, assetTestPNG(t))
				}
				if change == "project archive" {
					if want == 409 && problemCode(t, got.body) != "project.archived" {
						t.Fatal("archive did not precede image lookup", string(got.body))
					}
					a, err := assetpg.New(w.pool).FindAsset(t.Context(), image)
					if err != nil || (a.BoundAt == nil) != changeFirst || a.Placement.ProjectID != w.web {
						t.Fatal("archive race consumed refused preparation", a, err)
					}
				} else {
					w.send(t, who, "GET", readPath, "", 404)
				}
			})
		}
	}
}
