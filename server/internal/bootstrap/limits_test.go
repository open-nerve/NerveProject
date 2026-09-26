package bootstrap

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// Each bucket of the configuration limits the operations it belongs to (M2
// design 3.10). Every bucket has a burst of its own, and each operation is
// refused after exactly its bucket's burst: no bucket is wired in another's
// place. Behind the trusted proxy each case comes from a client of its own,
// so the buckets by client IP share no units across cases.
func TestEachConfiguredBucketLimitsItsOperations(t *testing.T) {
	contract := apitest.Load(t)
	cfg := testConfig(t, pgtest.NewDatabase(t), false)
	cfg.Server.TrustedProxies = []netip.Prefix{netip.MustParsePrefix("127.0.0.0/8")} // the test's own peer
	burst := func(n int) config.BucketConfig { return config.BucketConfig{PerMinute: 1, Burst: n} }
	cfg.RateLimit.Anonymous, cfg.RateLimit.Authenticated, cfg.RateLimit.AuthFailure = burst(9), burst(8), burst(3)
	cfg.RateLimit.LoginIP, cfg.RateLimit.LoginIPEmail = burst(5), burst(2)
	cfg.RateLimit.RegisterIP, cfg.RateLimit.PasswordUser = burst(1), burst(4)
	base := startApp(t, cfg, migrations.FS())
	from := func(client, method, path, token, body string) *http.Request {
		var b []byte
		if body != "" {
			b = []byte(body)
		}
		req := newRequest(t, method, base+path, token, b)
		req.Header.Set("X-Forwarded-For", client)
		return req
	}
	register := func(client, email string) *http.Request {
		return from(client, http.MethodPost, "/api/v0/auth/register", "", `{"email":"`+email+`","password":"Tr0ub4dor&3"}`)
	}
	accessToken := func(client, email string) string {
		res, body := send(t, register(client, email))
		var tokens authTokens
		if res.StatusCode != http.StatusCreated || json.Unmarshal(body, &tokens) != nil {
			t.Fatalf("register %s = %d %s, want 201 with tokens", email, res.StatusCode, body)
		}
		return tokens.AccessToken
	}
	reader, changer := accessToken("198.51.100.101", "reader@example.com"), accessToken("198.51.100.102", "changer@example.com")

	tests := []struct {
		bucket string
		burst  int
		next   func(i int) *http.Request
	}{
		{"anonymous", 9, func(int) *http.Request { return from("198.51.100.1", http.MethodGet, "/api/v0/instance", "", "") }},
		{"authenticated", 8, func(int) *http.Request { return from("198.51.100.2", http.MethodGet, "/api/v0/me", reader, "") }},
		{"auth_failure", 3, func(int) *http.Request { return from("198.51.100.3", http.MethodGet, "/api/v0/me", "forged", "") }},
		{"login_ip", 5, func(i int) *http.Request {
			return from("198.51.100.4", http.MethodPost, "/api/v0/auth/login", "", fmt.Sprintf(`{"email":"nobody%d@example.com","password":"x"}`, i))
		}},
		{"login_ip_email", 2, func(int) *http.Request {
			return from("198.51.100.5", http.MethodPost, "/api/v0/auth/login", "", `{"email":"nobody@example.com","password":"x"}`)
		}},
		{"register_ip", 1, func(i int) *http.Request { return register("198.51.100.6", fmt.Sprintf("new%d@example.com", i)) }},
		{"password_user", 4, func(int) *http.Request {
			return from("198.51.100.7", http.MethodPost, "/api/v0/me/change-password", changer, `{"current_password":"x","new_password":"y"}`)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.bucket, func(t *testing.T) {
			admitted := 0
			for i := range tt.burst + 1 {
				req := tt.next(i)
				res, _ := send(t, req)
				contract.CheckResponse(t, req, res)
				if res.StatusCode == http.StatusTooManyRequests {
					break
				}
				admitted++
			}
			if admitted != tt.burst {
				t.Errorf("%d requests admitted before 429, want the burst %d", admitted, tt.burst)
			}
		})
	}
}
