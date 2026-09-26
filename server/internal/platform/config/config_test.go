package config

import (
	"bytes"
	"log/slog"
	"net/netip"
	"strings"
	"testing"
)

func TestLogValueMasksDatabaseURL(t *testing.T) {
	var buf bytes.Buffer
	slog.New(slog.NewTextHandler(&buf, nil)).Info("configuration loaded", "config", validConfig())

	out := buf.String()
	if strings.Contains(out, "secret") {
		t.Errorf("log output leaks the password: %s", out)
	}
	for _, want := range []string{
		"config.env=test",
		"config.server.addr=:8080",
		"config.server.read_header_timeout=5s",
		"config.server.read_timeout=30s",
		"config.server.write_timeout=1m0s",
		"config.server.shutdown_timeout=20s",
		"config.server.request_timeout=15s",
		"config.server.max_body_bytes=1048576",
		"config.server.addr_file_set=false",
		"config.database.url=xxxxx",
		"config.database.max_conns=10",
		"config.database.commit_timeout=2s",
		"config.auth.signup_enabled=true",
		"config.auth.access_token_ttl=15m0s",
		"config.auth.session_ttl=720h0m0s",
		"config.auth.jwt.private_key_file_set=false",
		"config.auth.password.argon2_memory_kib=19456",
		"config.auth.password.argon2_iterations=2",
		"config.auth.password.argon2_parallelism=1",
		"config.auth.password.max_concurrent_hashes=4",
		"config.auth.password.max_wait=2s",
		"config.auth.refresh_deadline=4s",
		"config.server.trusted_proxies=10.0.0.0/8,2001:db8::/32",
		"config.ratelimit.ipv6_prefix_len=64",
		"config.ratelimit.anonymous.per_minute=600",
		"config.ratelimit.anonymous.burst=100",
		"config.ratelimit.auth_failure.per_minute=60",
		"config.ratelimit.authenticated.burst=200",
		"config.ratelimit.login_ip.per_minute=30",
		"config.ratelimit.login_ip_email.burst=5",
		"config.ratelimit.register_ip.per_minute=10",
		"config.ratelimit.password_user.per_minute=7",
		"config.ratelimit.password_user.burst=3",
		"config.workspace.creation_enabled=false",
		"config.files.size_limit=7340032",
		"config.log.format=json",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("log output lacks %q: %s", want, out)
		}
	}
}

// Every *_file key logs whether it is set, never the path (M0-P2 handoff 4,
// M2 design 3.7): the path of a key file tells where secrets live.
func TestLogValueHidesFilePaths(t *testing.T) {
	cfg := validConfig()
	cfg.Server.TrustedProxies = []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8"), netip.MustParsePrefix("fd00::/8")}
	cfg.Server.AddrFile = "/run/nerve/addr-secret-dir"
	cfg.Auth.JWT.PrivateKeyFile = "/etc/nerve/secret-dir/jwt.pem"
	var buf bytes.Buffer
	slog.New(slog.NewTextHandler(&buf, nil)).Info("configuration loaded", "config", cfg)

	out := buf.String()
	if strings.Contains(out, "secret-dir") {
		t.Errorf("log output shows a file path: %s", out)
	}
	for _, want := range []string{
		"config.server.addr_file_set=true",
		"config.auth.jwt.private_key_file_set=true",
		"config.server.trusted_proxies=10.0.0.0/8,fd00::/8", // not secret: the log should say whom nerve trusts
	} {
		if !strings.Contains(out, want) {
			t.Errorf("log output lacks %q: %s", want, out)
		}
	}
}

// The URL is masked as a whole, whatever its form: pgx parses it with its own
// libpq-compatible grammar, which accepts forms other URL parsers misread.
func TestDatabaseConfigLogValueMasksTheWholeURL(t *testing.T) {
	for _, url := range []string{
		"postgres://nerve:secret@localhost:5432/nerve",
		"postgres://localhost/nerve?password=secret&sslmode=disable",
		"postgres://localhost/nerve?password=secret;more&sslmode=disable", // raw ';': net/url drops the pair, pgx keeps it
		"postgres://localhost/nerve?pass%77ord=secret",
		"postgres://localhost/nerve?sslpassword=secret",
		"postgres://nerve:pa@ss@localhost/nerve?password=a%ZZsecret",
		"host=localhost user=nerve password=secret",
	} {
		var buf bytes.Buffer
		slog.New(slog.NewTextHandler(&buf, nil)).Info("x", "db", DatabaseConfig{URL: url, MaxConns: 10})

		out := buf.String()
		if strings.Contains(out, "secret") || !strings.Contains(out, "db.url=xxxxx ") {
			t.Errorf("DatabaseConfig{URL: %q} logs %s; want db.url=xxxxx and no password", url, out)
		}
		for _, want := range []string{"db.max_conns=10", "db.auto_migrate=false"} {
			if !strings.Contains(out, want) {
				t.Errorf("log output lacks %q: %s", want, out)
			}
		}
	}
}

func TestDatabaseConfigLogValueShowsAnEmptyURL(t *testing.T) {
	var buf bytes.Buffer
	slog.New(slog.NewTextHandler(&buf, nil)).Info("x", "db", DatabaseConfig{})

	if out := buf.String(); !strings.Contains(out, `db.url="" `) {
		t.Errorf("log output = %s, want an empty db.url", out)
	}
}
