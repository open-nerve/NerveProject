package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// sessionApp runs the app on a database of its own and returns its base URL
// and a pool on the same database, for the assertions.
func sessionApp(t *testing.T) (string, *pgxpool.Pool) {
	t.Helper()
	url := pgtest.NewDatabase(t)
	base := startApp(t, testConfig(t, url, false), migrations.FS())
	return base, openPool(t, url)
}

// openPool opens a pool on the database at url, for the assertions, and
// closes it when the test ends.
func openPool(t *testing.T, url string) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// postTokens posts {"refresh_token": token} to the auth path and returns
// the status and, on 200, the tokens.
func postTokens(t *testing.T, contract *apitest.Contract, base, path, token string) (int, authTokens) {
	t.Helper()
	req := newRequest(t, http.MethodPost, base+path, "", []byte(`{"refresh_token":"`+token+`"}`))
	res, body := send(t, req)
	contract.CheckResponse(t, req, res)
	var tokens authTokens
	if res.StatusCode == http.StatusOK && json.Unmarshal(body, &tokens) != nil {
		t.Fatalf("POST %s = %s, want tokens", path, body)
	}
	return res.StatusCode, tokens
}

// revocation reads the revoke_reason of the only session of email; "" while
// it is live.
func revocation(t *testing.T, pool *pgxpool.Pool, email string) string {
	t.Helper()
	var reason *string
	err := pool.QueryRow(context.Background(), `SELECT s.revoke_reason FROM auth_sessions s JOIN users u ON u.id = s.user_id
		WHERE u.email = $1`, email).Scan(&reason)
	if err != nil {
		t.Fatal(err)
	}
	if reason == nil {
		return ""
	}
	return *reason
}

// A retired refresh token revokes its session, and the revocation commits
// although the answer is an error: afterwards the latest refresh token and
// the latest access token fail too (M2 design 3.5).
func TestRefreshReuseRevokesTheSession(t *testing.T) {
	contract := apitest.Load(t)
	base, pool := sessionApp(t)
	first := registerAccount(t, contract, base, "reuse@example.com")

	status, second := postTokens(t, contract, base, "/api/v0/auth/refresh", first.RefreshToken)
	if status != http.StatusOK {
		t.Fatalf("first refresh = %d, want 200", status)
	}
	reused, _ := postTokens(t, contract, base, "/api/v0/auth/refresh", first.RefreshToken)
	latest, _ := postTokens(t, contract, base, "/api/v0/auth/refresh", second.RefreshToken)
	me, _ := send(t, newRequest(t, http.MethodGet, base+"/api/v0/me", second.AccessToken, nil))

	if reused != http.StatusUnauthorized || revocation(t, pool, "reuse@example.com") != "reuse_detected" {
		t.Errorf("the retired token = %d, session revoked for %q; want 401 and reuse_detected", reused, revocation(t, pool, "reuse@example.com"))
	}
	if latest != http.StatusUnauthorized || me.StatusCode != http.StatusUnauthorized {
		t.Errorf("after the reuse: the latest refresh token = %d, GET /me = %d; want 401 for both", latest, me.StatusCode)
	}
}

// Two refreshes with one token at once: one rotates, the other sees an
// older generation that the session issued and revokes it, whichever way
// the two transactions interleave (M2 design 3.5).
func TestConcurrentRefreshesOfOneToken(t *testing.T) {
	contract := apitest.Load(t)
	base, pool := sessionApp(t)
	for i := range 5 {
		email := fmt.Sprintf("race%d@example.com", i)
		token := registerAccount(t, contract, base, email).RefreshToken
		statuses := make([]int, 2)
		var wg sync.WaitGroup
		for j := range statuses {
			wg.Go(func() { statuses[j], _ = postTokens(t, contract, base, "/api/v0/auth/refresh", token) })
		}
		wg.Wait()

		slices.Sort(statuses)
		if !slices.Equal(statuses, []int{http.StatusOK, http.StatusUnauthorized}) || revocation(t, pool, email) != "reuse_detected" {
			t.Errorf("round %d: statuses %v, session revoked for %q; want 200 and 401, reuse_detected", i, statuses, revocation(t, pool, email))
		}
	}
}

// login posts email and password and returns the status and how long the
// answer took.
func login(t *testing.T, base, email, password string) (int, time.Duration) {
	t.Helper()
	req := newRequest(t, http.MethodPost, base+"/api/v0/auth/login", "", []byte(`{"email":"`+email+`","password":"`+password+`"}`))
	start := time.Now()
	res, _ := send(t, req)
	return res.StatusCode, time.Since(start)
}

func median(d []time.Duration) time.Duration {
	s := slices.Clone(d)
	slices.Sort(s)
	return s[len(s)/2]
}

// With the default argon2id parameters, a login for an unknown address
// takes as long as one with a wrong password for a known one: both verify
// one hash (M2 design 3.9). The two kinds alternate, and their medians
// must lie within a quarter of each other.
func TestLoginTakesAsLongForAnUnknownAddress(t *testing.T) {
	cfg := testConfig(t, pgtest.NewDatabase(t), false)
	cfg.Auth.Password.Argon2MemoryKiB, cfg.Auth.Password.Argon2Iterations = 19456, 2
	base := startApp(t, cfg, migrations.FS())
	registerAccount(t, apitest.Load(t), base, "timing@example.com")

	var known, unknown []time.Duration
	for range 15 {
		statusKnown, k := login(t, base, "timing@example.com", "Wr0ng-password")
		statusUnknown, u := login(t, base, "nobody@example.com", "Wr0ng-password")
		if statusKnown != http.StatusUnauthorized || statusUnknown != http.StatusUnauthorized {
			t.Fatalf("logins = %d, %d; want 401 for both", statusKnown, statusUnknown)
		}
		known, unknown = append(known, k), append(unknown, u)
	}

	mk, mu := median(known), median(unknown)
	t.Logf("median login: known address %v, unknown address %v", mk, mu)
	if diff := max(mk, mu) - min(mk, mu); diff*4 > max(mk, mu) {
		t.Errorf("median login: known address %v, unknown address %v; want them within a quarter of each other", mk, mu)
	}
}

// Behind a trusted proxy, a login's session records the client that
// X-Forwarded-For names, not the proxy: bootstrap hands server.trusted_proxies
// to the API (M2 design 3.10).
func TestLoginThroughATrustedProxyRecordsTheClient(t *testing.T) {
	url := pgtest.NewDatabase(t)
	cfg := testConfig(t, url, false)
	cfg.Server.TrustedProxies = []netip.Prefix{netip.MustParsePrefix("127.0.0.0/8")} // the test's own peer
	base := startApp(t, cfg, migrations.FS())
	pool := openPool(t, url)
	registerAccount(t, apitest.Load(t), base, "proxied@example.com")

	req := newRequest(t, http.MethodPost, base+"/api/v0/auth/login", "", []byte(`{"email":"proxied@example.com","password":"Tr0ub4dor&3"}`))
	req.Header.Set("X-Forwarded-For", "198.51.100.23")
	res, _ := send(t, req)
	var ip string
	if err := pool.QueryRow(context.Background(), `SELECT host(ip) FROM auth_sessions ORDER BY created_at DESC LIMIT 1`).Scan(&ip); err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != http.StatusOK || ip != "198.51.100.23" {
		t.Errorf("login through the proxy = %d, the new session's ip %s; want 200 and 198.51.100.23", res.StatusCode, ip)
	}
}
