// Package config loads, validates and describes nerve's configuration.
package config

import (
	"log/slog"
	"time"
)

// Profiles, selected with NERVE_ENV.
const (
	EnvDev  = "dev"
	EnvTest = "test"
	EnvProd = "prod"
)

// Config is the effective configuration of a nerve process.
type Config struct {
	// Env is the profile the configuration was loaded for. It comes from
	// NERVE_ENV and is not a configuration key.
	Env      string         `koanf:"-"`
	Server   ServerConfig   `koanf:"server"`
	Database DatabaseConfig `koanf:"database"`
	Auth     AuthConfig     `koanf:"auth"`
	Log      LogConfig      `koanf:"log"`
}

// ServerConfig configures the HTTP server. The timeouts bound the reads and
// writes on a connection: reading the request headers, reading the whole
// request (headers and body), and writing the response; idle keep-alive
// connections have a fixed timeout in httpserver. RequestTimeout bounds each
// API request's context, which write_timeout does not cancel.
type ServerConfig struct {
	Addr              string        `koanf:"addr"`
	ReadHeaderTimeout time.Duration `koanf:"read_header_timeout"`
	ReadTimeout       time.Duration `koanf:"read_timeout"`
	WriteTimeout      time.Duration `koanf:"write_timeout"`
	ShutdownTimeout   time.Duration `koanf:"shutdown_timeout"`
	RequestTimeout    time.Duration `koanf:"request_timeout"`
	MaxBodyBytes      int64         `koanf:"max_body_bytes"`
	// AddrFile, when set, receives the address the server listens on once it
	// does, e.g. for addr ":0".
	AddrFile string `koanf:"addr_file"`
}

// DatabaseConfig configures the PostgreSQL pool and schema migrations.
type DatabaseConfig struct {
	URL         string `koanf:"url"`
	MaxConns    int32  `koanf:"max_conns"`
	AutoMigrate bool   `koanf:"auto_migrate"`
	// CommitTimeout bounds COMMIT and ROLLBACK, which the request deadline
	// does not cancel.
	CommitTimeout time.Duration `koanf:"commit_timeout"`
}

// AuthConfig configures accounts and credentials (M2 design 6.5).
type AuthConfig struct {
	SignupEnabled  bool           `koanf:"signup_enabled"`
	AccessTokenTTL time.Duration  `koanf:"access_token_ttl"`
	SessionTTL     time.Duration  `koanf:"session_ttl"`
	JWT            JWTConfig      `koanf:"jwt"`
	Password       PasswordConfig `koanf:"password"`
}

// JWTConfig locates the Ed25519 signing key.
type JWTConfig struct {
	// PrivateKeyFile is a PKCS#8 PEM Ed25519 private key. Required in prod;
	// empty elsewhere means an ephemeral key generated at startup.
	PrivateKeyFile string `koanf:"private_key_file"`
}

// PasswordConfig configures argon2id and how many hashes run at once.
type PasswordConfig struct {
	Argon2MemoryKiB     uint32        `koanf:"argon2_memory_kib"`
	Argon2Iterations    uint32        `koanf:"argon2_iterations"`
	Argon2Parallelism   uint8         `koanf:"argon2_parallelism"`
	MaxConcurrentHashes int           `koanf:"max_concurrent_hashes"`
	MaxWait             time.Duration `koanf:"max_wait"`
}

// LogConfig configures the process logger.
type LogConfig struct {
	Level  string `koanf:"level"`  // debug, info, warn or error
	Format string `koanf:"format"` // text or json
}

// LogValue renders the configuration for logs with secrets masked, so the
// effective configuration can be logged at startup. Only the keys listed here
// reach the log; every *_file key logs whether it is set, never its path.
func (c Config) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("env", c.Env),
		slog.Group("server",
			slog.String("addr", c.Server.Addr),
			slog.Duration("read_header_timeout", c.Server.ReadHeaderTimeout),
			slog.Duration("read_timeout", c.Server.ReadTimeout),
			slog.Duration("write_timeout", c.Server.WriteTimeout),
			slog.Duration("shutdown_timeout", c.Server.ShutdownTimeout),
			slog.Duration("request_timeout", c.Server.RequestTimeout),
			slog.Int64("max_body_bytes", c.Server.MaxBodyBytes),
			slog.Bool("addr_file_set", c.Server.AddrFile != ""),
		),
		slog.Any("database", c.Database),
		slog.Group("auth",
			slog.Bool("signup_enabled", c.Auth.SignupEnabled),
			slog.Duration("access_token_ttl", c.Auth.AccessTokenTTL),
			slog.Duration("session_ttl", c.Auth.SessionTTL),
			slog.Group("jwt",
				slog.Bool("private_key_file_set", c.Auth.JWT.PrivateKeyFile != ""),
			),
			slog.Group("password",
				slog.Uint64("argon2_memory_kib", uint64(c.Auth.Password.Argon2MemoryKiB)),
				slog.Uint64("argon2_iterations", uint64(c.Auth.Password.Argon2Iterations)),
				slog.Uint64("argon2_parallelism", uint64(c.Auth.Password.Argon2Parallelism)),
				slog.Int("max_concurrent_hashes", c.Auth.Password.MaxConcurrentHashes),
				slog.Duration("max_wait", c.Auth.Password.MaxWait),
			),
		),
		slog.Group("log",
			slog.String("level", c.Log.Level),
			slog.String("format", c.Log.Format),
		),
	)
}

// redacted stands in for a secret in log output.
const redacted = "xxxxx"

// LogValue renders the database settings with the URL masked as a whole, so
// they are safe to log on their own too. pgx parses the URL with its own
// libpq-compatible grammar, which accepts forms that other parsers read
// differently, so masking only the password another parser finds could leak
// the rest; log the target from the parsed pool configuration instead.
func (d DatabaseConfig) LogValue() slog.Value {
	url := ""
	if d.URL != "" {
		url = redacted
	}
	return slog.GroupValue(
		slog.String("url", url),
		slog.Int("max_conns", int(d.MaxConns)),
		slog.Bool("auto_migrate", d.AutoMigrate),
		slog.Duration("commit_timeout", d.CommitTimeout),
	)
}
