package config

import (
	"strings"
	"testing"
	"time"
)

// The freeze job's interval (M6 P4 spec 2.5): a subsecond one fails before
// River starts, 1s is the valid boundary, and the environment sets it.
func TestFreezeIntervalValidationAndEnvironment(t *testing.T) {
	for _, interval := range []time.Duration{-time.Second, 0, 999 * time.Millisecond, time.Second, time.Hour} {
		cfg := validConfig()
		cfg.Planning.FreezeInterval = interval
		err := cfg.validate()
		if interval < time.Second {
			if err == nil || !strings.Contains(err.Error(), "planning.freeze_interval: must be at least 1s") {
				t.Fatalf("interval %s: %v", interval, err)
			}
		} else if err != nil {
			t.Fatal(err)
		}
	}
	cfg, err := Load(Sources{Embedded: embedded(map[string]string{"dev": ""}), Environ: []string{"NERVE_DATABASE__URL=postgres://test", "NERVE_PLANNING__FREEZE_INTERVAL=7s"}})
	if err != nil || cfg.Planning.FreezeInterval != 7*time.Second {
		t.Fatalf("environment interval=%s err=%v", cfg.Planning.FreezeInterval, err)
	}
}
