package bootstrap

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// testKeyPEM is an Ed25519 key made by `openssl genpkey -algorithm ed25519`
// for these tests only.
const testKeyPEM = "-----BEGIN PRIVATE KEY-----\n" +
	"MC4CAQAwBQYDK2VwBCIEIGqen6oN2FFQjS+yPQPHLVBIW0B2O9faCmNwftOWxqyE\n" +
	"-----END PRIVATE KEY-----\n"

func writeFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "key.pem")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// Registration and GET /me through the wired app and a real database; the
// access token is signed with the key of auth.jwt.private_key_file.
func TestRegisterThenGetMe(t *testing.T) {
	contract := apitest.Load(t)
	cfg := testConfig(t, pgtest.NewDatabase(t), false)
	cfg.Auth.JWT.PrivateKeyFile = writeFile(t, testKeyPEM)
	base := startApp(t, cfg, migrations.FS())

	tokens := registerAccount(t, contract, base, "Alice@Example.com")
	req := newRequest(t, http.MethodGet, base+"/api/v0/me", tokens.AccessToken, nil)
	res, body := send(t, req)

	contract.CheckResponse(t, req, res)
	var me struct {
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
	}
	if err := json.Unmarshal(body, &me); err != nil || res.StatusCode != http.StatusOK ||
		me.Email != "alice@example.com" || me.DisplayName != "alice" {
		t.Errorf("GET /me = %d %s, want 200 alice@example.com", res.StatusCode, body)
	}
	if !signedBy(t, tokens.AccessToken, testKeyPEM) {
		t.Error("the access token is not signed with auth.jwt.private_key_file")
	}
	if !strings.HasPrefix(tokens.RefreshToken, "nrv_rt_") {
		t.Errorf("refresh token %q lacks the nrv_rt_ prefix", tokens.RefreshToken)
	}
}

// A personal access token made through the app authenticates as its
// account until it is revoked (M2 design 3.4, 3.5). It can make another
// token, which the credential lock checks it for, and revoke itself.
func TestAPersonalAccessTokenAuthenticates(t *testing.T) {
	contract := apitest.Load(t)
	base := startApp(t, testConfig(t, pgtest.NewDatabase(t), false), migrations.FS())
	access := registerAccount(t, contract, base, "pat@example.com").AccessToken
	status := func(method, path, token string) int {
		t.Helper()
		req := newRequest(t, method, base+path, token, nil)
		res, _ := send(t, req)
		contract.CheckResponse(t, req, res)
		return res.StatusCode
	}

	first := createPAT(t, contract, base, access)
	second := createPAT(t, contract, base, first.Token)
	me, revoked, after, other := status(http.MethodGet, "/api/v0/me", first.Token),
		status(http.MethodDelete, "/api/v0/api-tokens/"+first.ID, first.Token),
		status(http.MethodGet, "/api/v0/me", first.Token),
		status(http.MethodGet, "/api/v0/me", second.Token)

	if me != http.StatusOK || revoked != http.StatusNoContent || after != http.StatusUnauthorized || other != http.StatusOK {
		t.Errorf("GET /me %d, revoke itself %d, GET /me after %d, the other token %d; want 200, 204, 401, 200", me, revoked, after, other)
	}
}

// A token's expiry reads back as the creation answered it: whatever offset
// and precision the caller sent, both say it in UTC to the microsecond, as
// the database stores it (M2 design 3.13).
func TestATokenExpiryReadsBackAsCreated(t *testing.T) {
	contract, base, token := accountApp(t, "expiry@example.com")

	createStatus, body := call(t, contract, http.MethodPost, base+"/api/v0/me/api-tokens", token, `{"expired_at":"2030-01-01T12:00:00.1234567+02:00"}`)
	listStatus, list := call(t, contract, http.MethodGet, base+"/api/v0/me/api-tokens", token, "")

	type listed struct {
		ID        string `json:"id"`
		ExpiredAt string `json:"expired_at"`
	}
	var created listed
	var page struct {
		Data []listed `json:"data"`
	}
	if createStatus != http.StatusCreated || listStatus != http.StatusOK || json.Unmarshal([]byte(body), &created) != nil || json.Unmarshal([]byte(list), &page) != nil {
		t.Fatalf("POST /me/api-tokens = %d %s, then GET = %d %s; want 201, 200", createStatus, body, listStatus, list)
	}
	if created.ExpiredAt != "2030-01-01T10:00:00.123456Z" {
		t.Errorf("POST /me/api-tokens answered expired_at %q, want 2030-01-01T10:00:00.123456Z", created.ExpiredAt)
	}
	i := slices.IndexFunc(page.Data, func(read listed) bool { return read.ID == created.ID })
	if i < 0 || page.Data[i].ExpiredAt != created.ExpiredAt {
		t.Errorf("GET /me/api-tokens = %s, want the token created with expired_at %q", list, created.ExpiredAt)
	}
}

// pat is the ApiTokenCreated answer.
type pat struct {
	ID    string `json:"id"`
	Token string `json:"token"`
}

// createPAT makes a personal access token with the bearer token given.
func createPAT(t *testing.T, contract *apitest.Contract, base, bearer string) pat {
	t.Helper()
	req := newRequest(t, http.MethodPost, base+"/api/v0/me/api-tokens", bearer, []byte(`{"label":"ci"}`))
	contract.CheckRequest(t, req)
	res, body := send(t, req)
	contract.CheckResponse(t, req, res)
	var created pat
	if err := json.Unmarshal(body, &created); err != nil || res.StatusCode != http.StatusCreated || !strings.HasPrefix(created.Token, "nrv_pat_") {
		t.Fatalf("POST /me/api-tokens = %d %s, want 201 with a token", res.StatusCode, body)
	}
	return created
}

// signedBy reports whether the JWT's EdDSA signature verifies with the
// public half of the PKCS#8 key.
func signedBy(t *testing.T, token, keyPEM string) bool {
	t.Helper()
	block, _ := pem.Decode([]byte(keyPEM))
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	public := key.(ed25519.PrivateKey).Public().(ed25519.PublicKey)
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("token %q is not a JWS", token)
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		t.Fatal(err)
	}
	return ed25519.Verify(public, []byte(parts[0]+"."+parts[1]), sig)
}

// A bad auth.jwt.private_key_file stops the app. The error names the key,
// never the file's path or content (M0-P2 handoff 4).
func TestBadSigningKeyFileStopsTheApp(t *testing.T) {
	const secret = "top-secret-content"
	missing := filepath.Join(t.TempDir(), "secret-dir", "missing.pem")
	tests := []struct {
		name, path, want string
	}{
		{"missing", missing, "auth.jwt.private_key_file: no such file or directory"},
		{"not a key", writeFile(t, secret), `auth.jwt.private_key_file: not a PEM "PRIVATE KEY" block (PKCS#8)`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testConfig(t, unreachableDB, false)
			cfg.Auth.JWT.PrivateKeyFile = tt.path

			_, err := newApp(context.Background(), cfg, slog.New(slog.DiscardHandler), fstest.MapFS{}, testWebUI)

			if err == nil || err.Error() != tt.want {
				t.Errorf("newApp() error = %v, want %q", err, tt.want)
			}
		})
	}
}

// Without auth.jwt.private_key_file the key is ephemeral, and the app says so.
func TestEphemeralSigningKeyIsAWarning(t *testing.T) {
	var logs bytes.Buffer
	a, err := newApp(context.Background(), testConfig(t, unreachableDB, false),
		slog.New(slog.NewTextHandler(&logs, nil)), fstest.MapFS{}, testWebUI)
	if err != nil {
		t.Fatal(err)
	}
	defer a.close()

	if !strings.Contains(logs.String(), `level=WARN msg="auth.jwt.private_key_file is not set: signing with an ephemeral key;`) {
		t.Errorf("logs lack the ephemeral key warning:\n%s", logs.String())
	}
}

// Outside prod, listening beyond loopback is warned about once at startup
// (M2 design 6.1).
func TestWarnIfExposed(t *testing.T) {
	tests := []struct {
		env, addr string
		warn      bool
	}{
		{config.EnvDev, "127.0.0.1:8080", false},
		{config.EnvDev, "[::1]:8080", false},
		{config.EnvDev, "localhost:8080", false},
		{config.EnvDev, "0.0.0.0:8080", true},
		{config.EnvDev, ":8080", true},
		{config.EnvTest, "[::]:0", true},
		{config.EnvDev, "nerve.example.com:8080", true},
		{config.EnvProd, "0.0.0.0:8080", false},
	}
	for _, tt := range tests {
		var logs bytes.Buffer
		cfg := config.Config{Env: tt.env, Server: config.ServerConfig{Addr: tt.addr}}

		warnIfExposed(context.Background(), slog.New(slog.NewTextHandler(&logs, nil)), cfg)

		if got := strings.Contains(logs.String(), "level=WARN msg=\"not running as prod but listening beyond this machine"); got != tt.warn {
			t.Errorf("%s on %s: warned = %v, want %v\n%s", tt.env, tt.addr, got, tt.warn, logs.String())
		}
	}
}
