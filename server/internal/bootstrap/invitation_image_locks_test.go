package bootstrap

import (
	"context"
	"fmt"
	"net/url"
	"testing"
	"time"
	"uuid"

	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Revocation shares W with preview; a nonmember recipient's deactivation need
// not lock W at all. Only the fixed invitation lock keeps both behind signing.
func TestInvitationImageLockBoundary(t *testing.T) {
	for _, ending := range []string{"revoke", "deactivate nonmember", "revoked first"} {
		t.Run(ending, func(t *testing.T) {
			w, _ := newAssetCompleteWorld(t, func(*assetComponents) {})
			data := assetTestPNG(t)
			logo := uploadedAssetHTTP(t, w, "alice", "WORKSPACE_LOGO", assetScopeBody("workspace", w.acme), "logo.png", "image/png", data).Asset.ID
			w.send(t, "alice", "PATCH", "/api/v0/workspaces/acme", fmt.Sprintf(`{"logo_asset_id":%q}`, logo), 200)
			link := invite(t, w.contract, w.base, w.tokens["alice"], "acme", "olga@example.com")
			path := "/api/v0/workspace-invitations/" + link.id.String() + "?token=" + url.QueryEscape(link.token)
			req := newRequest(t, "GET", w.base+path, "", nil)
			probe := w.observe(req)
			var response <-chan answer
			want := 200
			if ending == "revoked first" {
				stopped := newGate()
				defer requestOpen(stopped)
				tx := requestTx(w.pool)
				store := previewRevoker{Store: workspacepg.New(w.pool), gate: stopped}
				collect := requestBackground(t, func() error {
					return workspaceapp.NewDeleteWorkspaceInvitation(store, authorizerOn(w.pool), tx, clock.System{}).
						Execute(shared.WithActor(t.Context(), shared.Actor{UserID: w.ids["alice"]}), link.id)
				}, func() { requestOpen(stopped) })
				receiveWithin(t, stopped.held, 5*time.Second, "revocation written but not committed")
				response = sendInBackground(req)
				requestBlocked(t, w.pool, probe, requestPID(t, tx), "workspace_member_invites")
				requestOpen(stopped)
				collect()
				want = 404
			} else {
				held := holding(t, w.pool, "SELECT id FROM file_assets WHERE id=$1 FOR SHARE", logo)
				defer requestRelease(t, held)
				response = sendInBackground(req)
				pid := requestBlocked(t, w.pool, probe, held.Conn().PgConn().PID(), "file_assets")
				end := newRequest(t, "DELETE", w.base+"/api/v0/workspace-invitations/"+link.id.String(), w.tokens["alice"], nil)
				if ending == "deactivate nonmember" {
					if ids := queryIDs(t, w.pool, "SELECT id FROM workspace_members WHERE workspace_id=$1 AND member_id=$2", w.acme, w.ids["olga"]); len(ids) != 0 {
						t.Fatal("fixture recipient is a member", ids)
					}
					end = newRequest(t, "POST", w.base+"/api/v0/me/deactivate", w.tokens["olga"], nil)
				}
				endProbe := w.observe(end)
				ended := sendInBackground(end)
				requestBlocked(t, w.pool, endProbe, pid, "workspace_member_invites")
				requestRelease(t, held)
				got := receiveWithin(t, ended, 5*time.Second, "invitation ends after preview")
				if got.err != nil || got.res.StatusCode != 204 {
					t.Fatalf("ending: %+v", got)
				}
				w.contract.CheckResponse(t, end, got.res)
			}
			got := receiveWithin(t, response, 5*time.Second, "preview concurrent with ending")
			if got.err != nil || got.res.StatusCode != want {
				t.Fatalf("preview=%+v want=%d", got, want)
			}
			w.contract.CheckResponse(t, req, got.res)
			if want == 200 {
				checkSignedImageReply(t, w, string(got.body), "workspace_logo", data)
			}
			w.send(t, "", "GET", path, "", 404)
			requestEventually(t, "preview and ending release connections", func(context.Context) bool {
				return w.requests.appPool.Stat().AcquiredConns() == 0
			})
		})
	}
}

type previewRevoker struct {
	*workspacepg.Store
	gate *gate
}

func (s previewRevoker) DeleteInvitation(ctx context.Context, id, by uuid.UUID, at time.Time) error {
	if err := s.Store.DeleteInvitation(ctx, id, by, at); err != nil {
		return err
	}
	return s.gate.wait(ctx)
}
