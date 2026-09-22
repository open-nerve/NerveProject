package config

import (
	"strings"
	"testing"
	"time"
)

func validConfig() Config {
	return Config{
		Env: EnvTest,
		Server: ServerConfig{
			Addr:              ":8080",
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      60 * time.Second,
			ShutdownTimeout:   20 * time.Second,
		},
		Database: DatabaseConfig{URL: "postgres://nerve:secret@localhost:5432/nerve", MaxConns: 10},
		Log:      LogConfig{Level: "info", Format: "json"},
	}
}

func TestValidateAcceptsValidConfig(t *testing.T) {
	if err := validConfig().validate(); err != nil {
		t.Fatalf("validate() = %v, want nil", err)
	}
}

func TestValidateReportsEveryInvalidKey(t *testing.T) {
	cfg := Config{
		Server:   ServerConfig{Addr: "8080", ReadHeaderTimeout: 0, ReadTimeout: 0, WriteTimeout: -time.Second, ShutdownTimeout: -time.Second},
		Database: DatabaseConfig{URL: "", MaxConns: 0},
		Log:      LogConfig{Level: "verbose", Format: "xml"},
	}
	err := cfg.validate()
	if err == nil {
		t.Fatal("validate() = nil, want errors")
	}
	want := []string{
		`server.addr: must be host:port, e.g. ":8080", got "8080"`,
		"server.read_header_timeout: must be positive, got 0s",
		"server.read_timeout: must be positive, got 0s",
		"server.write_timeout: must be positive, got -1s",
		"server.shutdown_timeout: must be positive, got -1s",
		"database.url: is required",
		"database.max_conns: must be at least 1, got 0",
		`log.level: must be one of debug, info, warn, error, got "verbose"`,
		`log.format: must be text or json, got "xml"`,
	}
	if got := err.Error(); got != strings.Join(want, "\n") {
		t.Errorf("validate() errors:\n%s\nwant:\n%s", got, strings.Join(want, "\n"))
	}
}

// read_timeout is the budget for the whole request, so the headers alone may
// not be allowed longer.
func TestValidateRejectsHeaderTimeoutAboveReadTimeout(t *testing.T) {
	cfg := validConfig()
	cfg.Server.ReadHeaderTimeout = 40 * time.Second

	want := "server.read_header_timeout: must not exceed server.read_timeout (30s), got 40s"
	if err := cfg.validate(); err == nil || err.Error() != want {
		t.Errorf("validate() = %v, want %q", err, want)
	}
}
