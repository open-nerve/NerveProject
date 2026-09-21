package configs_test

import (
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/configs"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
)

func TestBuiltInProfiles(t *testing.T) {
	const devURL = "postgres://nerve:nerve@localhost:55432/nerve?sslmode=disable"
	tests := []struct {
		env         string
		url         string // expected database.url
		autoMigrate bool
		level       string
		format      string
	}{
		{env: "dev", url: devURL, autoMigrate: true, level: "debug", format: "text"},
		{env: "test", url: "postgres://from-env", autoMigrate: true, level: "warn", format: "text"},
		{env: "prod", url: "postgres://from-env", autoMigrate: false, level: "info", format: "json"},
	}
	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			environ := []string{"NERVE_ENV=" + tt.env}
			if tt.env != "dev" {
				environ = append(environ, "NERVE_DATABASE__URL=postgres://from-env")
			}
			cfg, err := config.Load(config.Sources{Embedded: configs.FS(), Environ: environ})
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			want := config.Config{
				Env: tt.env,
				Server: config.ServerConfig{
					Addr:              ":8080",
					ReadHeaderTimeout: 5 * time.Second,
					ShutdownTimeout:   20 * time.Second,
				},
				Database: config.DatabaseConfig{URL: tt.url, MaxConns: 10, AutoMigrate: tt.autoMigrate},
				Log:      config.LogConfig{Level: tt.level, Format: tt.format},
			}
			if cfg != want {
				t.Errorf("Load() =\n%+v\nwant\n%+v", cfg, want)
			}
		})
	}
}

func TestProdRequiresDatabaseURL(t *testing.T) {
	_, err := config.Load(config.Sources{Embedded: configs.FS(), Environ: []string{"NERVE_ENV=prod"}})
	if err == nil {
		t.Fatal("Load() error = nil, want database.url to be required")
	}
}
