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
	Log      LogConfig      `koanf:"log"`
}

// ServerConfig configures the HTTP server.
type ServerConfig struct {
	Addr              string        `koanf:"addr"`
	ReadHeaderTimeout time.Duration `koanf:"read_header_timeout"`
	ShutdownTimeout   time.Duration `koanf:"shutdown_timeout"`
}

// DatabaseConfig configures the PostgreSQL pool and schema migrations.
type DatabaseConfig struct {
	URL         string `koanf:"url"`
	MaxConns    int32  `koanf:"max_conns"`
	AutoMigrate bool   `koanf:"auto_migrate"`
}

// LogConfig configures the process logger.
type LogConfig struct {
	Level  string `koanf:"level"`  // debug, info, warn or error
	Format string `koanf:"format"` // text or json
}

// LogValue renders the configuration for logs with secrets masked, so the
// effective configuration can be logged at startup.
func (c Config) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("env", c.Env),
		slog.Group("server",
			slog.String("addr", c.Server.Addr),
			slog.Duration("read_header_timeout", c.Server.ReadHeaderTimeout),
			slog.Duration("shutdown_timeout", c.Server.ShutdownTimeout),
		),
		slog.Group("database",
			slog.String("url", redactURL(c.Database.URL)),
			slog.Int("max_conns", int(c.Database.MaxConns)),
			slog.Bool("auto_migrate", c.Database.AutoMigrate),
		),
		slog.Group("log",
			slog.String("level", c.Log.Level),
			slog.String("format", c.Log.Format),
		),
	)
}
