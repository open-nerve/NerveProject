package config

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/configs"
)

func TestFileLifecycleConfig(t *testing.T) {
	load := func(values ...string) (Config, error) {
		return Load(Sources{Embedded: configs.FS(), Environ: append([]string{"NERVE_ENV=test", "NERVE_DATABASE__URL=postgres://test"}, values...)})
	}
	c, err := load()
	if err != nil {
		t.Fatal(err)
	}
	if c.Files.UploadTTL != 15*time.Minute || c.Files.OperationTimeout != 10*time.Second || c.Files.CleanupInterval != time.Hour || c.Files.CleanupBatch != 100 || c.Files.CleanupBudget != 10*time.Second {
		t.Fatalf("wrong lifecycle defaults: %+v", c.Files)
	}
	for _, setting := range []string{
		"UPLOAD_TTL=9.999999999s", "UPLOAD_TTL=15m1ns", "UPLOAD_TTL=9223372036854775808ns",
		"OPERATION_TIMEOUT=999.999999ms", "OPERATION_TIMEOUT=10s1ns",
		"CLEANUP_INTERVAL=59.999999999s", "CLEANUP_INTERVAL=24h1ns",
		"CLEANUP_BATCH=0", "CLEANUP_BATCH=1001", "CLEANUP_BATCH=9223372036854775808", "CLEANUP_BATCH=1.5",
		"CLEANUP_BUDGET=999.999999ms", "CLEANUP_BUDGET=30s1ns", "CLEANUP_BUDGET=9223372036854775808ns",
		"CLEANUP_BUDET=10s",
	} {
		key, _, _ := strings.Cut(setting, "=")
		if _, err := load("NERVE_FILES__" + setting); err == nil || !strings.Contains(err.Error(), strings.ToLower(key)) {
			t.Errorf("%s = %v", setting, err)
		}
	}
	for _, settings := range [][]string{
		{"NERVE_FILES__UPLOAD_TTL=10s", "NERVE_FILES__OPERATION_TIMEOUT=1s", "NERVE_FILES__CLEANUP_INTERVAL=1m", "NERVE_FILES__CLEANUP_BATCH=1", "NERVE_FILES__CLEANUP_BUDGET=1s"},
		{"NERVE_FILES__UPLOAD_TTL=15m", "NERVE_FILES__OPERATION_TIMEOUT=10s", "NERVE_FILES__CLEANUP_INTERVAL=24h", "NERVE_FILES__CLEANUP_BATCH=1000", "NERVE_FILES__CLEANUP_BUDGET=30s"},
	} {
		if _, err := load(settings...); err != nil {
			t.Fatal(err)
		}
	}
	for _, value := range []string{"cleanup_batch: 1.5", "cleanup_batch: 9223372036854775808", "operation_timeout: 10", "upload_ttl: null"} {
		dir := t.TempDir()
		writeFile(t, dir, "config.yaml", "files:\n  "+value+"\n")
		if _, err := load("NERVE_CONFIG_DIR=" + dir); err == nil {
			t.Fatalf("invalid YAML accepted: %s", value)
		}
	}
	c, err = load("NERVE_FILES__UPLOAD_TTL=12m", "NERVE_FILES__OPERATION_TIMEOUT=7s", "NERVE_FILES__CLEANUP_INTERVAL=2h", "NERVE_FILES__CLEANUP_BATCH=31", "NERVE_FILES__CLEANUP_BUDGET=19s", "NERVE_FILES__LOCAL__ROOT=/secret/lifecycle-root")
	if err != nil {
		t.Fatal(err)
	}
	var log bytes.Buffer
	slog.New(slog.NewTextHandler(&log, nil)).Info("configuration", "files", c.Files)
	for _, value := range []string{"files.upload_ttl=12m0s", "files.operation_timeout=7s", "files.cleanup_interval=2h0m0s", "files.cleanup_batch=31", "files.cleanup_budget=19s", "files.local.root_set=true"} {
		if !strings.Contains(log.String(), value) {
			t.Errorf("log missing %s", value)
		}
	}
	if strings.Contains(log.String(), "lifecycle-root") {
		t.Fatal("root exposed in log")
	}
}
