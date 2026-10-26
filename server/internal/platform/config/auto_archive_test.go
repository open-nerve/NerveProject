package config

import (
	"strings"
	"testing"
	"time"
)

// A subsecond interval must fail before River starts; 1s is the valid boundary.
func TestAutoArchiveIntervalValidationAndEnvironment(t *testing.T) {
	for _, interval := range []time.Duration{-time.Second, 0, 999 * time.Millisecond, time.Second, 24 * time.Hour} {
		cfg := validConfig()
		cfg.Issue.AutoArchiveInterval = interval
		err := cfg.validate()
		if interval < time.Second {
			if err == nil || !strings.Contains(err.Error(), "issue.auto_archive_interval: must be at least 1s") {
				t.Fatalf("interval %s: %v", interval, err)
			}
		} else if err != nil {
			t.Fatal(err)
		}
	}
	cfg, err := Load(Sources{Embedded: embedded(map[string]string{"dev": ""}), Environ: []string{"NERVE_DATABASE__URL=postgres://test", "NERVE_ISSUE__AUTO_ARCHIVE_INTERVAL=7s"}})
	if err != nil || cfg.Issue.AutoArchiveInterval != 7*time.Second {
		t.Fatalf("environment interval=%s err=%v", cfg.Issue.AutoArchiveInterval, err)
	}
}
