package config

import (
	"bytes"
	"log/slog"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/configs"
)

// Missing defaults, a key wired to another field, or an unchecked boundary
// must fail before housekeeping can start.
func TestPurgeConfiguration(t *testing.T) {
	load := func(t *testing.T, env ...string) Config {
		t.Helper()
		cfg, err := Load(Sources{Embedded: configs.FS(), Environ: append([]string{"NERVE_ENV=prod", "NERVE_DATABASE__URL=postgres://test", "NERVE_AUTH__JWT__PRIVATE_KEY_FILE=/test/key"}, env...)})
		if err != nil {
			t.Fatal(err)
		}
		return cfg
	}
	t.Run("defaults", func(t *testing.T) {
		cfg := load(t)
		if cfg.Cleanup != (CleanupConfig{PurgeAfter: 1440 * time.Hour, PurgeInterval: 24 * time.Hour, PurgeBatch: 1000}) || cfg.Issue.TrimVersionsInterval != 24*time.Hour || cfg.Issue.DeleteBatch != 1000 || cfg.Issue.DeleteBudget != 45*time.Second {
			t.Fatalf("defaults: cleanup=%+v issue=%+v", cfg.Cleanup, cfg.Issue)
		}
	})
	t.Run("test profile", func(t *testing.T) {
		cfg := load(t, "NERVE_ENV=test")
		if cfg.Cleanup != (CleanupConfig{PurgeAfter: 1440 * time.Hour, PurgeInterval: 2 * time.Second, PurgeBatch: 1000}) || cfg.Issue.TrimVersionsInterval != 2*time.Second || cfg.Issue.DeleteBatch != 1000 || cfg.Issue.DeleteBudget != 45*time.Second {
			t.Fatalf("test profile: cleanup=%+v issue=%+v", cfg.Cleanup, cfg.Issue)
		}
	})
	t.Run("environment and logs", func(t *testing.T) {
		cfg := load(t, "NERVE_CLEANUP__PURGE_AFTER=1450h", "NERVE_CLEANUP__PURGE_INTERVAL=25h", "NERVE_CLEANUP__PURGE_BATCH=71", "NERVE_ISSUE__TRIM_VERSIONS_INTERVAL=26h", "NERVE_ISSUE__DELETE_BATCH=73", "NERVE_ISSUE__DELETE_BUDGET=47s")
		if cfg.Cleanup != (CleanupConfig{PurgeAfter: 1450 * time.Hour, PurgeInterval: 25 * time.Hour, PurgeBatch: 71}) || cfg.Issue.TrimVersionsInterval != 26*time.Hour || cfg.Issue.DeleteBatch != 73 || cfg.Issue.DeleteBudget != 47*time.Second {
			t.Fatalf("environment: cleanup=%+v issue=%+v", cfg.Cleanup, cfg.Issue)
		}
		var buf bytes.Buffer
		slog.New(slog.NewTextHandler(&buf, nil)).Info("configuration", "config", cfg)
		for _, want := range []string{"config.cleanup.purge_after=1450h0m0s", "config.cleanup.purge_interval=25h0m0s", "config.cleanup.purge_batch=71", "config.issue.trim_versions_interval=26h0m0s", "config.issue.delete_batch=73", "config.issue.delete_budget=47s"} {
			if !strings.Contains(buf.String(), want+" ") {
				t.Errorf("log lacks %q: %s", want, buf.String())
			}
		}
	})
	for _, tt := range []struct {
		key    string
		change func(*Config)
		valid  bool
	}{
		{"cleanup.purge_after", func(c *Config) { c.Cleanup.PurgeAfter = 0 }, false},
		{"cleanup.purge_after", func(c *Config) { c.Cleanup.PurgeAfter = -time.Second }, false},
		{"cleanup.purge_interval", func(c *Config) { c.Cleanup.PurgeInterval = 999 * time.Millisecond }, false},
		{"cleanup.purge_interval", func(c *Config) { c.Cleanup.PurgeInterval = time.Second }, true},
		{"cleanup.purge_batch", func(c *Config) { c.Cleanup.PurgeBatch = 0 }, false},
		{"cleanup.purge_batch", func(c *Config) { c.Cleanup.PurgeBatch = 2147483648 }, false},
		{"cleanup.purge_batch", func(c *Config) { c.Cleanup.PurgeBatch = 1 }, true},
		{"cleanup.purge_batch", func(c *Config) { c.Cleanup.PurgeBatch = 2147483647 }, true},
		{"issue.trim_versions_interval", func(c *Config) { c.Issue.TrimVersionsInterval = 999 * time.Millisecond }, false},
		{"issue.trim_versions_interval", func(c *Config) { c.Issue.TrimVersionsInterval = time.Second }, true},
		{"issue.delete_batch", func(c *Config) { c.Issue.DeleteBatch = 0 }, false},
		{"issue.delete_batch", func(c *Config) { c.Issue.DeleteBatch = 2147483648 }, false},
		{"issue.delete_batch", func(c *Config) { c.Issue.DeleteBatch = 1 }, true},
		{"issue.delete_batch", func(c *Config) { c.Issue.DeleteBatch = 2147483647 }, true},
		{"issue.delete_budget", func(c *Config) { c.Issue.DeleteBudget = 0 }, false},
		{"issue.delete_budget", func(c *Config) { c.Issue.DeleteBudget = -time.Second }, false},
		{"issue.delete_budget", func(c *Config) { c.Issue.DeleteBudget = math.MaxInt64 }, false},
		{"issue.delete_budget", func(c *Config) { c.Issue.DeleteBudget = math.MaxInt64 - 5*time.Second }, true},
	} {
		t.Run(tt.key, func(t *testing.T) {
			cfg := load(t)
			tt.change(&cfg)
			err := cfg.validate()
			if tt.valid {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.HasPrefix(err.Error(), tt.key+":") || strings.Contains(err.Error(), "\n") {
				t.Fatalf("invalid %s: got %v, want only this key rejected", tt.key, err)
			}
		})
	}
}
