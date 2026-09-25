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
			RequestTimeout:    15 * time.Second,
			MaxBodyBytes:      1 << 20,
		},
		Database: DatabaseConfig{URL: "postgres://nerve:secret@localhost:5432/nerve", MaxConns: 10, CommitTimeout: 2 * time.Second},
		Auth: AuthConfig{
			SignupEnabled:  true,
			AccessTokenTTL: 15 * time.Minute,
			SessionTTL:     720 * time.Hour,
			Password: PasswordConfig{
				Argon2MemoryKiB:     19456,
				Argon2Iterations:    2,
				Argon2Parallelism:   1,
				MaxConcurrentHashes: 4,
				MaxWait:             2 * time.Second,
			},
		},
		Log: LogConfig{Level: "info", Format: "json"},
	}
}

func TestValidateAcceptsValidConfig(t *testing.T) {
	if err := validConfig().validate(); err != nil {
		t.Fatalf("validate() = %v, want nil", err)
	}
}

func TestValidateReportsEveryInvalidKey(t *testing.T) {
	cfg := Config{
		Env:      EnvProd,
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
		"server.request_timeout: must be positive, got 0s",
		"server.max_body_bytes: must be at least 1, got 0",
		"database.url: is required",
		"database.max_conns: must be at least 1, got 0",
		"database.commit_timeout: must be positive, got 0s",
		"auth.access_token_ttl: must be positive, got 0s",
		"auth.session_ttl: must be positive, got 0s",
		"auth.jwt.private_key_file: is required in prod: a PKCS#8 PEM Ed25519 private key, e.g. from openssl genpkey -algorithm ed25519",
		"auth.password.argon2_iterations: must be at least 1, got 0",
		"auth.password.argon2_parallelism: must be at least 1, got 0",
		"auth.password.argon2_memory_kib: must be at least 8 per lane (8), got 0",
		"auth.password.max_concurrent_hashes: must be at least 1, got 0",
		"auth.password.max_wait: must be positive, got 0s",
		`log.level: must be one of debug, info, warn, error, got "verbose"`,
		`log.format: must be text or json, got "xml"`,
	}
	if got := err.Error(); got != strings.Join(want, "\n") {
		t.Errorf("validate() errors:\n%s\nwant:\n%s", got, strings.Join(want, "\n"))
	}
}

func TestValidateCrossKeyRules(t *testing.T) {
	tests := []struct {
		name   string
		change func(*Config)
		want   string
	}{
		{
			// read_timeout is the budget for the whole request, so the headers
			// alone may not be allowed longer.
			name:   "header timeout above read timeout",
			change: func(c *Config) { c.Server.ReadHeaderTimeout = 40 * time.Second },
			want:   "server.read_header_timeout: must not exceed server.read_timeout (30s), got 40s",
		},
		{
			name:   "request timeout not below write timeout",
			change: func(c *Config) { c.Server.RequestTimeout = 60 * time.Second },
			want:   "server.request_timeout: must be less than server.write_timeout (1m0s), got 1m0s",
		},
		{
			name:   "session not longer than an access token",
			change: func(c *Config) { c.Auth.SessionTTL = 15 * time.Minute },
			want:   "auth.session_ttl: must be longer than auth.access_token_ttl (15m0s), got 15m0s",
		},
		{
			name: "argon2 memory below 8 KiB per lane",
			change: func(c *Config) {
				c.Auth.Password.Argon2Parallelism = 4
				c.Auth.Password.Argon2MemoryKiB = 31
			},
			want: "auth.password.argon2_memory_kib: must be at least 8 per lane (32), got 31",
		},
		{
			name:   "prod without a signing key",
			change: func(c *Config) { c.Env = EnvProd },
			want:   "auth.jwt.private_key_file: is required in prod: a PKCS#8 PEM Ed25519 private key, e.g. from openssl genpkey -algorithm ed25519",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.change(&cfg)
			if err := cfg.validate(); err == nil || err.Error() != tt.want {
				t.Errorf("validate() = %v, want %q", err, tt.want)
			}
		})
	}
}

// Only prod requires the signing key file; dev and test generate a key.
func TestValidateAcceptsProdWithASigningKeyFile(t *testing.T) {
	cfg := validConfig()
	cfg.Env = EnvProd
	cfg.Auth.JWT.PrivateKeyFile = "/etc/nerve/jwt.pem"

	if err := cfg.validate(); err != nil {
		t.Errorf("validate() = %v, want nil", err)
	}
}
