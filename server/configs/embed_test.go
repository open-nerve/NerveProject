package configs_test

import (
	"net/netip"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/configs"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
)

// The default buckets (M2 design 3.10), and the test profile's, raised so
// that one IP can sign many accounts up and in.
var (
	defaultLimits = config.RateLimitConfig{
		IPv6PrefixLen: 64,
		Anonymous:     config.BucketConfig{PerMinute: 600, Burst: 100},
		AuthFailure:   config.BucketConfig{PerMinute: 60, Burst: 60},
		Authenticated: config.BucketConfig{PerMinute: 1200, Burst: 200},
		LoginIP:       config.BucketConfig{PerMinute: 30, Burst: 10},
		LoginIPEmail:  config.BucketConfig{PerMinute: 10, Burst: 5},
		RegisterIP:    config.BucketConfig{PerMinute: 10, Burst: 5},
	}
	high       = config.BucketConfig{PerMinute: 600000, Burst: 100000}
	testLimits = config.RateLimitConfig{
		IPv6PrefixLen: 64, Anonymous: high, AuthFailure: high, Authenticated: high, LoginIP: high, LoginIPEmail: high, RegisterIP: high,
	}
)

func TestBuiltInProfiles(t *testing.T) {
	const devURL = "postgres://nerve:nerve@localhost:55432/nerve?sslmode=disable"
	tests := []struct {
		env         string
		addr        string // expected server.addr
		url         string // expected database.url
		autoMigrate bool
		signup      bool   // auth.signup_enabled: closed in prod unless overridden (M2 design decision 2)
		argon2      uint32 // auth.password.argon2_memory_kib
		iterations  uint32 // auth.password.argon2_iterations
		keyFile     string // auth.jwt.private_key_file
		limits      config.RateLimitConfig
		level       string
		format      string
	}{
		{env: "dev", addr: "127.0.0.1:8080", url: devURL, autoMigrate: true, signup: true, argon2: 19456, iterations: 2, limits: defaultLimits, level: "debug", format: "text"},
		{env: "test", addr: ":8080", url: "postgres://from-env", autoMigrate: true, signup: true, argon2: 64, iterations: 1, limits: testLimits, level: "warn", format: "text"},
		{env: "prod", addr: ":8080", url: "postgres://from-env", autoMigrate: false, signup: false, argon2: 19456, iterations: 2, keyFile: "/etc/nerve/jwt.pem", limits: defaultLimits, level: "info", format: "json"},
	}
	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			environ := []string{"NERVE_ENV=" + tt.env}
			if tt.env != "dev" {
				environ = append(environ, "NERVE_DATABASE__URL=postgres://from-env")
			}
			if tt.keyFile != "" {
				environ = append(environ, "NERVE_AUTH__JWT__PRIVATE_KEY_FILE="+tt.keyFile)
			}
			cfg, err := config.Load(config.Sources{Embedded: configs.FS(), Environ: environ})
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			want := config.Config{
				Env: tt.env,
				Server: config.ServerConfig{
					Addr:              tt.addr,
					ReadHeaderTimeout: 5 * time.Second,
					ReadTimeout:       30 * time.Second,
					WriteTimeout:      60 * time.Second,
					ShutdownTimeout:   20 * time.Second,
					RequestTimeout:    15 * time.Second,
					MaxBodyBytes:      1 << 20,
					TrustedProxies:    []netip.Prefix{}, // none: the client is the connection's peer
				},
				Database: config.DatabaseConfig{URL: tt.url, MaxConns: 10, AutoMigrate: tt.autoMigrate, CommitTimeout: 2 * time.Second},
				Auth: config.AuthConfig{
					SignupEnabled:   tt.signup,
					AccessTokenTTL:  15 * time.Minute,
					SessionTTL:      30 * 24 * time.Hour,
					RefreshDeadline: 4 * time.Second,
					JWT:             config.JWTConfig{PrivateKeyFile: tt.keyFile},
					Password: config.PasswordConfig{
						Argon2MemoryKiB:     tt.argon2,
						Argon2Iterations:    tt.iterations,
						Argon2Parallelism:   1,
						MaxConcurrentHashes: 4,
						MaxWait:             2 * time.Second,
					},
				},
				RateLimit: tt.limits,
				Log:       config.LogConfig{Level: tt.level, Format: tt.format},
			}
			if !reflect.DeepEqual(cfg, want) {
				t.Errorf("Load() =\n%+v\nwant\n%+v", cfg, want)
			}
		})
	}
}

func TestProdRequiresDatabaseURLAndSigningKey(t *testing.T) {
	_, err := config.Load(config.Sources{Embedded: configs.FS(), Environ: []string{"NERVE_ENV=prod"}})
	if err == nil {
		t.Fatal("Load() error = nil, want database.url and auth.jwt.private_key_file to be required")
	}
	for _, key := range []string{"database.url: is required", "auth.jwt.private_key_file: is required in prod"} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("Load() error = %v, want it to report %q", err, key)
		}
	}
}
