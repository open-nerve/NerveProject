package config

import (
	"net/netip"
	"reflect"
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
			TrustedProxies:    []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8"), netip.MustParsePrefix("2001:db8::/32")},
		},
		Database: DatabaseConfig{URL: "postgres://nerve:secret@localhost:5432/nerve", MaxConns: 10, CommitTimeout: 2 * time.Second},
		Auth: AuthConfig{
			SignupEnabled:   true,
			AccessTokenTTL:  15 * time.Minute,
			SessionTTL:      720 * time.Hour,
			RefreshDeadline: 4 * time.Second,
			Password: PasswordConfig{
				Argon2MemoryKiB:     19456,
				Argon2Iterations:    2,
				Argon2Parallelism:   1,
				MaxConcurrentHashes: 4,
				MaxWait:             2 * time.Second,
			},
		},
		RateLimit: RateLimitConfig{
			IPv6PrefixLen: 64,
			Anonymous:     BucketConfig{PerMinute: 600, Burst: 100},
			AuthFailure:   BucketConfig{PerMinute: 60, Burst: 60},
			Authenticated: BucketConfig{PerMinute: 1200, Burst: 200},
			LoginIP:       BucketConfig{PerMinute: 30, Burst: 10},
			LoginIPEmail:  BucketConfig{PerMinute: 10, Burst: 5},
			RegisterIP:    BucketConfig{PerMinute: 10, Burst: 5},
			PasswordUser:  BucketConfig{PerMinute: 7, Burst: 3},
		},
		// Unlike the defaults and unlike auth.signup_enabled, so that a key
		// logged from the wrong field shows.
		Workspace: WorkspaceConfig{CreationEnabled: false},
		Files:     FilesConfig{SizeLimit: 7340032},
		Log:       LogConfig{Level: "info", Format: "json"},
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
		"auth.refresh_deadline: must be positive, got 0s",
		"ratelimit.ipv6_prefix_len: must be from 1 to 128, got 0",
		"ratelimit.anonymous.per_minute: must be at least 1, got 0",
		"ratelimit.anonymous.burst: must be at least 1, got 0",
		"ratelimit.auth_failure.per_minute: must be at least 1, got 0",
		"ratelimit.auth_failure.burst: must be at least 1, got 0",
		"ratelimit.authenticated.per_minute: must be at least 1, got 0",
		"ratelimit.authenticated.burst: must be at least 1, got 0",
		"ratelimit.login_ip.per_minute: must be at least 1, got 0",
		"ratelimit.login_ip.burst: must be at least 1, got 0",
		"ratelimit.login_ip_email.per_minute: must be at least 1, got 0",
		"ratelimit.login_ip_email.burst: must be at least 1, got 0",
		"ratelimit.register_ip.per_minute: must be at least 1, got 0",
		"ratelimit.register_ip.burst: must be at least 1, got 0",
		"ratelimit.password_user.per_minute: must be at least 1, got 0",
		"ratelimit.password_user.burst: must be at least 1, got 0",
		"files.size_limit: must be at least 1, got 0",
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
		{
			// The server must be done with a refresh before the web client
			// gives up on it after 8 s (M2 design 3.5, 7.1).
			name:   "refresh deadline and commit timeout reach the web client's timeout",
			change: func(c *Config) { c.Auth.RefreshDeadline = 6 * time.Second },
			want:   "auth.refresh_deadline: plus database.commit_timeout (2s) must be less than 8s, the web client's refresh timeout, got 6s",
		},
		{
			// The client IP is unmapped before it is compared (M2 design 3.10).
			name: "IPv4-mapped trusted proxy",
			change: func(c *Config) {
				c.Server.TrustedProxies = append(c.Server.TrustedProxies, netip.MustParsePrefix("::ffff:10.0.0.0/104"))
			},
			want: "server.trusted_proxies: ::ffff:10.0.0.0/104 is an IPv4-mapped IPv6 prefix, which no address matches: write the IPv4 prefix",
		},
		{
			// Every client could then choose its own IP (M2 design 3.10).
			name: "every IPv4 address trusted",
			change: func(c *Config) {
				c.Server.TrustedProxies = append(c.Server.TrustedProxies, netip.MustParsePrefix("0.0.0.0/0"))
			},
			want: "server.trusted_proxies: 0.0.0.0/0 trusts every address, so any client could choose its own IP; list only your proxies' addresses",
		},
		{
			name: "every IPv6 address trusted",
			change: func(c *Config) {
				c.Server.TrustedProxies = append(c.Server.TrustedProxies, netip.MustParsePrefix("::/0"))
			},
			want: "server.trusted_proxies: ::/0 trusts every address, so any client could choose its own IP; list only your proxies' addresses",
		},
		{
			name:   "negative file size limit",
			change: func(c *Config) { c.Files.SizeLimit = -1 },
			want:   "files.size_limit: must be at least 1, got -1",
		},
		{
			name:   "IPv6 prefix longer than an address",
			change: func(c *Config) { c.RateLimit.IPv6PrefixLen = 129 },
			want:   "ratelimit.ipv6_prefix_len: must be from 1 to 128, got 129",
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

// Every bucket in RateLimitConfig is validated from its own settings, under
// its own key: with only one setting of one bucket invalid, validate reports
// that key and nothing else. The buckets are found by reflection, so a bucket
// added to the struct but not to validate fails here too.
func TestValidateChecksEachBucketUnderItsOwnKey(t *testing.T) {
	limits := reflect.TypeFor[RateLimitConfig]()
	buckets := 0
	for i := range limits.NumField() {
		field := limits.Field(i)
		if field.Type != reflect.TypeFor[BucketConfig]() {
			continue
		}
		buckets++
		for _, setting := range []struct {
			key  string
			zero func(*BucketConfig)
		}{
			{"per_minute", func(b *BucketConfig) { b.PerMinute = 0 }},
			{"burst", func(b *BucketConfig) { b.Burst = 0 }},
		} {
			key := "ratelimit." + field.Tag.Get("koanf") + "." + setting.key
			t.Run(key, func(t *testing.T) {
				cfg := validConfig()
				setting.zero(reflect.ValueOf(&cfg.RateLimit).Elem().Field(i).Addr().Interface().(*BucketConfig))
				want := key + ": must be at least 1, got 0"
				if err := cfg.validate(); err == nil || err.Error() != want {
					t.Errorf("validate() = %v, want only %q", err, want)
				}
			})
		}
	}
	if buckets == 0 {
		t.Fatal("RateLimitConfig has no BucketConfig field; want the buckets to be found")
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
