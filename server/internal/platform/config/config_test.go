package config

import (
	"bytes"
	"log/slog"
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
	cfg.Server.AddrFile = "/run/nerve/addr-secret-dir"
	cfg.Auth.JWT.PrivateKeyFile = "/etc/nerve/secret-dir/jwt.pem"
	var buf bytes.Buffer
	slog.New(slog.NewTextHandler(&buf, nil)).Info("configuration loaded", "config", cfg)

	out := buf.String()
	if strings.Contains(out, "secret-dir") {
		t.Errorf("log output shows a file path: %s", out)
	}
	for _, want := range []string{"config.server.addr_file_set=true", "config.auth.jwt.private_key_file_set=true"} {
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
