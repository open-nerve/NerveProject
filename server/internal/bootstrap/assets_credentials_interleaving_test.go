package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	identitypg "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	identitydomain "github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

func TestAssetCredentialResetInterleaving(t *testing.T) {
	for _, resetFirst := range []bool{true, false} {
		t.Run(fmt.Sprintf("reset first=%t", resetFirst), func(t *testing.T) {
			url := pgtest.NewDatabase(t)
			w := newIssueWorldOnRequests(t, url, openPool(t, url), func(*config.Config) {}, &requestQueries{})
			item := w.newWorkItem(t)
			file := uploadedAssetHTTP(t, w, "bob", "ISSUE_ATTACHMENT", assetScopeBody("issue", item), "x.txt", "text/plain", []byte("secret"))
			path := "/api/v0/assets/" + file.Asset.ID.String() + "/content"
			req := newRequest(t, "GET", w.base+path, w.tokens["bob"], nil)
			probe := w.observe(req)
			resetTx := requestTx(w.pool)
			store := identitypg.New(w.pool)
			gate := newGate()
			var once sync.Once
			release := func() { once.Do(func() { close(gate.open) }) }
			defer release()
			var passwords identityapp.PasswordHashWriter = store
			if resetFirst {
				passwords = gatedPasswords{PasswordHashWriter: store, gate: gate}
			}
			reset := func() error {
				_, err := identityapp.NewResetPassword(identityapp.ResetPasswordDeps{
					Accounts: store, Passwords: passwords, Sessions: store, APITokens: store,
					Hasher: fixedHasher{}, Rules: identitydomain.NewPasswordRules(), Tx: resetTx,
					Clock: clock.System{}, Logger: slog.New(slog.DiscardHandler),
				}).Execute(t.Context(), "bob@example.com", "N3w-Passw0rd!")
				return err
			}
			var response <-chan answer
			var collectReset func()
			if resetFirst {
				collectReset = requestBackground(t, reset, release)
				receiveWithin(t, gate.held, 5*time.Second, "reset owns account before revocation")
				response = sendInBackground(req)
				requestBlocked(t, w.pool, probe, requestPID(t, resetTx), "users")
				release()
			} else {
				held := holding(t, w.pool, "SELECT id FROM file_assets WHERE id=$1 FOR SHARE", file.Asset.ID)
				defer requestRelease(t, held)
				response = sendInBackground(req)
				pid := requestBlocked(t, w.pool, probe, held.Conn().PgConn().PID(), "file_assets")
				collectReset = requestBackground(t, reset, func() { requestRelease(t, held) })
				requestBlocked(t, w.pool, resetTx, pid, "users")
				requestRelease(t, held)
			}
			collectReset()
			got := receiveWithin(t, response, 5*time.Second, "signature concurrent with reset")
			want := 200
			if resetFirst {
				want = 401
			}
			if got.err != nil || got.res.StatusCode != want {
				t.Fatalf("sign=%+v want=%d", got, want)
			}
			w.contract.CheckResponse(t, req, got.res)
			assetAPICall(t, w, "bob", "GET", path, "", 401)
			requestEventually(t, "reset and signer release connections", func(context.Context) bool {
				return w.requests.appPool.Stat().AcquiredConns() == 0
			})
		})
	}
}
