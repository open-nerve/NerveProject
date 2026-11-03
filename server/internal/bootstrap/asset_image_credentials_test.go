package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	identitypg "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	identitydomain "github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
)

// Removing account/credential locking can sign with a revoked session. These
// real HTTP requests authenticate before revocation commits, then must recheck.
func TestImageCredentialRace(t *testing.T) {
	for _, scope := range []string{"user", "workspace", "project"} {
		for _, ending := range []string{"reset", "deactivate"} {
			for _, endingFirst := range []bool{true, false} {
				t.Run(fmt.Sprintf("%s %s first=%t", scope, ending, endingFirst), func(t *testing.T) {
					w := newObservedIssueWorld(t)
					// Allow Alice's real deactivation without violating sole-admin rules.
					w.send(t, "wanda", "POST", "/api/v0/projects/"+w.web.String()+"/join", "", 200)
					path, purpose, placement, field := "/api/v0/me", "USER_AVATAR", `{"kind":"user"}`, "avatar"
					switch scope {
					case "workspace":
						path, purpose, placement, field = "/api/v0/workspaces/acme", "WORKSPACE_LOGO", assetScopeBody("workspace", w.acme), "logo"
					case "project":
						path, purpose, placement, field = "/api/v0/projects/"+w.web.String(), "PROJECT_COVER", assetScopeBody("project", w.web), "cover_image"
					}
					image := bindRaceImage(t, w, "alice", purpose, placement, path, field)
					req := newRequest(t, "GET", w.base+path, w.tokens["alice"], nil)
					probe := w.observe(req)
					endTx, stopped := requestTx(w.pool), newGate()
					defer requestOpen(stopped)
					store := identitypg.New(w.pool)
					var passwords identityapp.PasswordHashWriter = store
					var sessions identityapp.SessionRevoker = store
					if endingFirst {
						passwords = gatedPasswords{PasswordHashWriter: store, gate: stopped}
						sessions = gatedSessions{SessionRevoker: store, gate: stopped}
					}
					end := func() error {
						if ending == "reset" {
							_, err := identityapp.NewResetPassword(identityapp.ResetPasswordDeps{Accounts: store, Passwords: passwords, Sessions: store, APITokens: store, Hasher: fixedHasher{}, Rules: identitydomain.NewPasswordRules(), Tx: endTx, Clock: clock.System{}, Logger: slog.New(slog.DiscardHandler)}).Execute(t.Context(), "alice@example.com", "N3w-Passw0rd!")
							return err
						}
						_, err := identityapp.NewDeactivate(identityapp.DeactivateDeps{Accounts: store, Users: store, Profiles: store, Sessions: sessions,
							Memberships: workspaceapp.NewDeactivator(workspacepg.New(w.pool), cascadeOn(w.pool), clock.System{}), Tx: endTx, Clock: clock.System{}, Logger: slog.New(slog.DiscardHandler)}).ExecuteByEmail(t.Context(), "alice@example.com")
						return err
					}
					var response <-chan answer
					var collectEnd func()
					if endingFirst {
						collectEnd = requestBackground(t, end, func() { requestOpen(stopped) })
						receiveWithin(t, stopped.held, 5*time.Second, "ending holds account before commit")
						response = sendInBackground(req)
						requestBlocked(t, w.pool, probe, requestPID(t, endTx), "users")
						requestOpen(stopped)
					} else {
						held := holding(t, w.pool, "SELECT id FROM file_assets WHERE id=$1 FOR SHARE", image)
						defer requestRelease(t, held)
						response = sendInBackground(req)
						pid := requestBlocked(t, w.pool, probe, held.Conn().PgConn().PID(), "file_assets")
						collectEnd = requestBackground(t, end, func() { requestRelease(t, held) })
						requestBlocked(t, w.pool, endTx, pid, "users")
						requestRelease(t, held)
					}
					collectEnd()
					want := 200
					if endingFirst {
						want = 401
					}
					got := imageRaceAnswer(t, w, req, response, want)
					if want == 200 {
						checkImageReply(t, w, string(got.body), field, image, assetTestPNG(t))
					}
					w.send(t, "alice", "GET", path, "", 401)
					requestEventually(t, "credential race releases connections", func(context.Context) bool { return w.requests.appPool.Stat().AcquiredConns() == 0 })
				})
			}
		}
	}
}
