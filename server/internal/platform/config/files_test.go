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

func TestFilesConfigurationBudgets(t *testing.T) {
	load := func(values ...string) (Config, error) {
		return Load(Sources{Embedded: configs.FS(), Environ: append([]string{"NERVE_ENV=test", "NERVE_DATABASE__URL=postgres://test"}, values...)})
	}
	t.Run("default budget", func(t *testing.T) {
		cfg, err := load()
		if err != nil {
			t.Fatal(err)
		}
		want := FilesConfig{SizeLimit: 5242880, Driver: "local", Local: LocalFilesConfig{Root: "./data/files"},
			MinTransferBytesPerSecond: 131072, TransferGrace: 15 * time.Second, TransferIdleTimeout: 30 * time.Second,
			MaxTransferDuration: 15 * time.Minute, MaxConcurrentTransfers: 16, TransferRate: BucketConfig{PerMinute: 60, Burst: 20}}
		if cfg.Files != want {
			t.Fatalf("default files configuration = %+v; want %+v", cfg.Files, want)
		}
		for _, tc := range []struct {
			size int64
			want time.Duration
		}{
			{1, 16 * time.Second}, {131072, 16 * time.Second}, {131073, 17 * time.Second}, {5242880, 55 * time.Second},
		} {
			got, err := cfg.Files.TransferBudget(tc.size)
			if err != nil || got != tc.want {
				t.Errorf("budget(%d) = %v, %v; want %s", tc.size, got, err, tc.want)
			}
		}
		for _, size := range []int64{-1, 0, 104857601, 9223372036854775807} {
			if _, err := cfg.Files.TransferBudget(size); err == nil {
				t.Errorf("budget accepts size %d", size)
			}
		}
		for _, change := range []func(*FilesConfig){
			func(c *FilesConfig) { c.SizeLimit = math.MaxInt64 },
			func(c *FilesConfig) { c.MinTransferBytesPerSecond = 0 },
			func(c *FilesConfig) { c.MinTransferBytesPerSecond = math.MaxInt64 },
			func(c *FilesConfig) { c.TransferGrace = time.Duration(math.MaxInt64) },
			func(c *FilesConfig) { c.MaxTransferDuration = time.Duration(math.MaxInt64) },
			func(c *FilesConfig) { c.MaxTransferDuration = 54 * time.Second },
		} {
			c := cfg.Files
			change(&c)
			if _, err := c.TransferBudget(5242880); err == nil {
				t.Errorf("directly constructed invalid config admitted a budget: %+v", c)
			}
		}
	})
	t.Run("older objects retain bounded read budgets after a lower upload limit", func(t *testing.T) {
		cfg, err := load("NERVE_FILES__SIZE_LIMIT=1")
		if err != nil {
			t.Fatal(err)
		}
		if budget, err := cfg.Files.TransferBudget(5242880); err != nil || budget != 55*time.Second {
			t.Fatalf("existing object budget = %s, %v", budget, err)
		}
		cfg.Files.MaxTransferDuration = 16 * time.Second
		if budget, err := cfg.Files.TransferBudget(5242880); err != nil || budget != 16*time.Second {
			t.Fatalf("existing object capped budget = %s, %v", budget, err)
		}
	})
	t.Run("YAML values retain their numeric types", func(t *testing.T) {
		for _, value := range []string{"size_limit: 1.5", "size_limit: 9223372036854775808", "max_concurrent_transfers: 1.5", "transfer_grace: 15"} {
			dir := t.TempDir()
			writeFile(t, dir, "config.yaml", "files:\n  "+value+"\n")
			_, err := load("NERVE_CONFIG_DIR=" + dir)
			key, _, _ := strings.Cut(value, ":")
			if err == nil || !strings.Contains(err.Error(), key) {
				t.Fatalf("Load(%s) = %v; want type rejection", value, err)
			}
		}
	})
	t.Run("load rejects invalid settings", func(t *testing.T) {
		for _, tc := range []struct{ value, key string }{
			{"SIZE_LIMIT=0", "size_limit"}, {"SIZE_LIMIT=-1", "size_limit"}, {"SIZE_LIMIT=104857601", "size_limit"},
			{"SIZE_LIMIT=1.5", "size_limit"}, {"SIZE_LIMIT=9223372036854775808", "size_limit"},
			{"DRIVER=s3", "driver"}, {"LOCAL__ROOT=", "local.root"},
			{"MIN_TRANSFER_BYTES_PER_SECOND=1023", "min_transfer_bytes_per_second"},
			{"MIN_TRANSFER_BYTES_PER_SECOND=1073741825", "min_transfer_bytes_per_second"},
			{"MIN_TRANSFER_BYTES_PER_SECOND=1.5", "min_transfer_bytes_per_second"},
			{"TRANSFER_GRACE=0s", "transfer_grace"}, {"TRANSFER_GRACE=60.000000001s", "transfer_grace"},
			{"TRANSFER_IDLE_TIMEOUT=-1s", "transfer_idle_timeout"}, {"TRANSFER_IDLE_TIMEOUT=61s", "transfer_idle_timeout"},
			{"MAX_TRANSFER_DURATION=0s", "max_transfer_duration"}, {"MAX_TRANSFER_DURATION=1h1ns", "max_transfer_duration"},
			{"MAX_TRANSFER_DURATION=54.999999999s", "max_transfer_duration"},
			{"MAX_CONCURRENT_TRANSFERS=0", "max_concurrent_transfers"}, {"MAX_CONCURRENT_TRANSFERS=257", "max_concurrent_transfers"},
			{"TRANSFER_RATE__PER_MINUTE=0", "transfer_rate.per_minute"}, {"TRANSFER_RATE__BURST=-1", "transfer_rate.burst"},
		} {
			t.Run(tc.value, func(t *testing.T) {
				_, err := load("NERVE_FILES__" + tc.value)
				if err == nil || !strings.Contains(err.Error(), tc.key) {
					t.Fatalf("Load = %v; want %s rejection", err, tc.key)
				}
			})
		}
	})
	t.Run("inclusive boundaries and independent global limits", func(t *testing.T) {
		for _, values := range [][]string{
			{"NERVE_FILES__MAX_TRANSFER_DURATION=55s"},
			{"NERVE_FILES__SIZE_LIMIT=1", "NERVE_FILES__MIN_TRANSFER_BYTES_PER_SECOND=1024", "NERVE_FILES__TRANSFER_GRACE=1ns", "NERVE_FILES__TRANSFER_IDLE_TIMEOUT=1ns", "NERVE_FILES__MAX_CONCURRENT_TRANSFERS=1"},
			{"NERVE_FILES__SIZE_LIMIT=104857600", "NERVE_FILES__MIN_TRANSFER_BYTES_PER_SECOND=1073741824", "NERVE_FILES__TRANSFER_GRACE=60s", "NERVE_FILES__TRANSFER_IDLE_TIMEOUT=60s", "NERVE_FILES__MAX_TRANSFER_DURATION=1h", "NERVE_FILES__MAX_CONCURRENT_TRANSFERS=256"},
		} {
			cfg, err := load(values...)
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Server.ReadTimeout != 30*time.Second || cfg.Server.RequestTimeout != 15*time.Second || cfg.Server.MaxBodyBytes != 1048576 {
				t.Fatal("file configuration widened ordinary requests")
			}
		}
	})
	t.Run("logs hide root including standalone value", func(t *testing.T) {
		cfg, err := load("NERVE_FILES__LOCAL__ROOT=/private/secret-file-root")
		if err != nil {
			t.Fatal(err)
		}
		for _, value := range []any{cfg, cfg.Files} {
			var buf bytes.Buffer
			slog.New(slog.NewJSONHandler(&buf, nil)).Info("configuration", "config", value)
			if out := buf.String(); strings.Contains(out, "secret-file-root") || !strings.Contains(out, `"root_set":true`) {
				t.Errorf("unsafe or incomplete log: %s", out)
			}
		}
	})
	t.Run("unknown file key", func(t *testing.T) {
		_, err := load("NERVE_FILES__MAX_TRANSFER_DURATON=1h")
		if err == nil || !strings.Contains(err.Error(), "max_transfer_duraton") {
			t.Fatalf("unknown key = %v", err)
		}
	})
}
