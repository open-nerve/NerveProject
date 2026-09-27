package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/netip"
	"sync"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
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

// unusablePasswordWarning is the hasher's warning about a stored password
// in another format, which it logs while it holds its slot.
const unusablePasswordWarning = "a stored password hash is not an argon2id PHC string: no password matches it"

// slotHolder is a log handler that keeps a password hashing slot taken: it
// returns from the hasher's warning about an unusable password only once
// the test releases it.
type slotHolder struct {
	slog.Handler
	held    chan struct{}
	release chan struct{}
}

func (h *slotHolder) Handle(ctx context.Context, r slog.Record) error {
	if r.Message == unusablePasswordWarning {
		h.held <- struct{}{}
		select {
		case <-h.release:
		case <-time.After(10 * time.Second): // never released: let go, so the test fails instead of hanging
		}
	}
	return h.Handler.Handle(ctx, r)
}

// auth.password.max_concurrent_hashes and auth.password.max_wait reach the
// hasher (M2 design 3.8), two settings of each. While a login holds the only
// slot, a second login waits max_wait and gets 503 server_busy; with two
// slots it signs in at once. The first login's account has an unusable
// password, so the hasher logs its warning while it holds the slot, and
// the test's log handler keeps it there. The hasher's own tests cover the
// waiting itself; this one covers the wiring.
func TestPasswordHashingIsLimitedAsConfigured(t *testing.T) {
	tests := []struct {
		name       string
		concurrent int
		wait       time.Duration
		want       int // the second login's status
	}{
		{"one slot, a short wait", 1, 250 * time.Millisecond, http.StatusServiceUnavailable},
		{"one slot, a long wait", 1, 1500 * time.Millisecond, http.StatusServiceUnavailable},
		{"two slots", 2, 250 * time.Millisecond, http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			url := pgtest.NewDatabase(t)
			cfg := testConfig(t, url, false)
			cfg.Auth.Password.MaxConcurrentHashes, cfg.Auth.Password.MaxWait = tt.concurrent, tt.wait
			holder := &slotHolder{Handler: slog.NewTextHandler(io.Discard, nil), held: make(chan struct{}, 1), release: make(chan struct{})}
			base := startAppLogging(t, cfg, migrations.FS(), slog.New(holder))
			var once sync.Once
			release := func() { once.Do(func() { close(holder.release) }) }
			t.Cleanup(release) // before the app stops: cleanups run last first
			contract := apitest.Load(t)
			registerAccount(t, contract, base, "holder@example.com")
			registerAccount(t, contract, base, "second@example.com")
			if _, err := openPool(t, url).Exec(context.Background(), "UPDATE users SET password = '!' WHERE email = 'holder@example.com'"); err != nil {
				t.Fatal(err)
			}
			login := func(email string) *http.Request {
				return newRequest(t, http.MethodPost, base+"/api/v0/auth/login", "", []byte(`{"email":"`+email+`","password":"Tr0ub4dor&3"}`))
			}

			holding := login("holder@example.com")
			first := make(chan int, 1)
			go func() {
				res, err := client.Do(holding)
				if err != nil {
					first <- 0
					return
				}
				_ = res.Body.Close()
				first <- res.StatusCode
			}()
			receiveWithin(t, holder.held, 5*time.Second, "slot held by the first login")

			req := login("second@example.com")
			sent := time.Now()
			res, body := send(t, req)
			took := time.Since(sent)
			release()
			contract.CheckResponse(t, req, res)

			switch {
			case res.StatusCode != tt.want:
				t.Errorf("the second login = %d after %s, want %d", res.StatusCode, took, tt.want)
			case tt.want == http.StatusServiceUnavailable:
				if code := problemCode(t, body); code != shared.CodeServerBusy || res.Header.Get("Retry-After") != "1" ||
					took < tt.wait || took > tt.wait+time.Second {
					t.Errorf("the second login = %s, Retry-After %q, after %s; want server_busy with Retry-After 1 after max_wait %s",
						code, res.Header.Get("Retry-After"), took, tt.wait)
				}
			}
			if status := receiveWithin(t, first, 5*time.Second, "answer to the first login"); status != http.StatusUnauthorized {
				t.Errorf("the first login = %d, want 401: its password is unusable", status)
			}
		})
	}
}
