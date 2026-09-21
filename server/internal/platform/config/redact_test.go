package config

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestRedactURL(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"empty stays empty", "", ""},
		{"password in user info", "postgres://nerve:secret@localhost:5432/nerve", "postgres://nerve:xxxxx@localhost:5432/nerve"},
		{"password query parameter", "postgres://localhost/nerve?password=secret&sslmode=disable", "postgres://localhost/nerve?password=xxxxx&sslmode=disable"},
		{"password query parameter in any case", "postgres://localhost/nerve?PassWord=secret&sslmode=disable", "postgres://localhost/nerve?PassWord=xxxxx&sslmode=disable"},
		{"no password is unchanged", "postgres://nerve@localhost/nerve?sslmode=disable", "postgres://nerve@localhost/nerve?sslmode=disable"},
		{"key/value string is masked entirely", "host=localhost user=nerve password=secret", "xxxxx"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := redactURL(tt.in); got != tt.want {
				t.Errorf("redactURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestLogValueMasksDatabasePassword(t *testing.T) {
	var buf bytes.Buffer
	slog.New(slog.NewTextHandler(&buf, nil)).Info("configuration loaded", "config", validConfig())

	out := buf.String()
	if strings.Contains(out, "secret") {
		t.Errorf("log output leaks the password: %s", out)
	}
	for _, want := range []string{
		"config.env=test",
		"config.server.addr=:8080",
		"config.server.shutdown_timeout=20s",
		"config.database.url=postgres://nerve:xxxxx@localhost:5432/nerve",
		"config.database.max_conns=10",
		"config.log.format=json",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("log output lacks %q: %s", want, out)
		}
	}
}
