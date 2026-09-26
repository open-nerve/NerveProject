package bootstrap

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// accountApp is the wired app on a new database, with a personal access
// token of a new account: every operation that needs a token works with one
// (M2 design 12, P3).
func accountApp(t *testing.T, email string) (contract *apitest.Contract, base, token string) {
	t.Helper()
	contract = apitest.Load(t)
	base = startApp(t, testConfig(t, pgtest.NewDatabase(t), false), migrations.FS())
	return contract, base, createPAT(t, contract, base, registerAccount(t, contract, base, email).AccessToken).Token
}

// call sends a request that the contract allows and returns its status and
// body.
func call(t *testing.T, contract *apitest.Contract, method, url, token, body string) (int, string) {
	t.Helper()
	var b []byte
	if body != "" {
		b = []byte(body)
	}
	req := newRequest(t, method, url, token, b)
	contract.CheckRequest(t, req)
	res, out := send(t, req)
	contract.CheckResponse(t, req, res)
	return res.StatusCode, string(out)
}

// Changing the password with a personal access token revokes every
// session, the token keeps working, and only the new password signs in (M2
// design 3.5, story A7).
func TestChangingThePasswordWithAPersonalAccessToken(t *testing.T) {
	base, pool := sessionApp(t)
	contract := apitest.Load(t)
	session := registerAccount(t, contract, base, "change@example.com")
	token := createPAT(t, contract, base, session.AccessToken).Token

	changed, body := call(t, contract, http.MethodPost, base+"/api/v0/me/change-password", token,
		`{"current_password":"Tr0ub4dor&3","new_password":"N3w-Passw0rd!"}`)
	reason := revocation(t, pool, "change@example.com")
	refreshed, _ := postTokens(t, contract, base, "/api/v0/auth/refresh", session.RefreshToken)
	me, _ := call(t, contract, http.MethodGet, base+"/api/v0/me", token, "")
	oldLogin, _ := login(t, base, "change@example.com", "Tr0ub4dor&3")
	newLogin, _ := login(t, base, "change@example.com", "N3w-Passw0rd!")

	if changed != http.StatusNoContent || reason != "password_changed" || refreshed != http.StatusUnauthorized || me != http.StatusOK {
		t.Errorf("change = %d %s; the session revoked for %q, its refresh %d; the token's GET /me %d; want 204, password_changed, 401, 200",
			changed, body, reason, refreshed, me)
	}
	if oldLogin != http.StatusUnauthorized || newLogin != http.StatusOK {
		t.Errorf("login with the old password %d, with the new one %d; want 401, 200", oldLogin, newLogin)
	}
}

// Deactivating with a personal access token: the account is inactive with
// its password, every session is revoked, onboarding starts over, and the
// token stays but authenticates no more; the right password answers
// identity.account_deactivated (M2 design 3.5, story A12).
func TestDeactivatingWithAPersonalAccessToken(t *testing.T) {
	base, pool := sessionApp(t)
	contract := apitest.Load(t)
	token := createPAT(t, contract, base, registerAccount(t, contract, base, "leaving@example.com").AccessToken).Token
	if status, body := call(t, contract, http.MethodPatch, base+"/api/v0/me/profile", token,
		`{"onboarding_step":{"profile_complete":true},"is_onboarded":true}`); status != http.StatusOK {
		t.Fatalf("PATCH /me/profile = %d %s", status, body)
	}

	deactivated, body := call(t, contract, http.MethodPost, base+"/api/v0/me/deactivate", token, "")
	reason := revocation(t, pool, "leaving@example.com")
	me, _ := call(t, contract, http.MethodGet, base+"/api/v0/me", token, "")
	signIn, _ := login(t, base, "leaving@example.com", "Tr0ub4dor&3")

	if deactivated != http.StatusNoContent || reason != "deactivated" || me != http.StatusUnauthorized || signIn != http.StatusForbidden {
		t.Errorf("deactivate = %d %s; the session revoked for %q; the token's GET /me %d; login %d; want 204, deactivated, 401, 403",
			deactivated, body, reason, me, signIn)
	}
	var active, startsOver bool
	var revokedTokens int
	err := pool.QueryRow(context.Background(), `SELECT u.is_active,
		p.onboarding_step = '{"profile_complete": false, "workspace_create": false, "workspace_invite": false, "workspace_join": false}'
			AND NOT p.is_onboarded,
		(SELECT count(*) FROM api_tokens t WHERE t.user_id = u.id AND t.deleted_at IS NOT NULL)
		FROM users u JOIN profiles p ON p.user_id = u.id WHERE u.email = 'leaving@example.com'`).Scan(&active, &startsOver, &revokedTokens)
	if err != nil || active || !startsOver || revokedTokens != 0 {
		t.Errorf("active %v, onboarding started over %v, tokens revoked %d (%v); want false, true, 0", active, startsOver, revokedTokens, err)
	}
}

func TestTheAccountAndItsPreferencesWithAPersonalAccessToken(t *testing.T) {
	contract, base, token := accountApp(t, "account@example.com")

	meStatus, me := call(t, contract, http.MethodPatch, base+"/api/v0/me", token, `{"first_name":"Ann","user_timezone":"Asia/Shanghai"}`)
	patchStatus, _ := call(t, contract, http.MethodPatch, base+"/api/v0/me/profile", token, `{"theme":"dark","onboarding_step":{"profile_complete":true}}`)
	getStatus, profile := call(t, contract, http.MethodGet, base+"/api/v0/me/profile", token, "")

	if meStatus != http.StatusOK || !strings.Contains(me, `"first_name":"Ann"`) || !strings.Contains(me, `"user_timezone":"Asia/Shanghai"`) {
		t.Errorf("PATCH /me = %d %s, want 200 with the new name and time zone", meStatus, me)
	}
	if patchStatus != http.StatusOK || getStatus != http.StatusOK || !strings.Contains(profile, `"theme":"dark"`) ||
		!strings.Contains(profile, `"onboarding_step":{"profile_complete":true,"workspace_create":false,`) {
		t.Errorf("PATCH /me/profile %d, then GET = %d %s; want 200 and the new theme with the step merged", patchStatus, getStatus, profile)
	}
}
