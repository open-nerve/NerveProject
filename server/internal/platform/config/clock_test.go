package config

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/configs"
)

// Removing profile validation or environment precedence must reject or accept
// the wrong real Load input, rather than merely changing a struct validator.
func TestClockConfig(t *testing.T) {
	for _, profile := range []string{EnvDev, EnvProd, EnvTest} {
		for _, value := range []string{"0s", "25h", "-25h"} {
			t.Run(profile+"/"+value, func(t *testing.T) {
				cfg, err := Load(Sources{Embedded: configs.FS(), Environ: []string{"NERVE_ENV=" + profile, "NERVE_DATABASE__URL=postgres://test", "NERVE_AUTH__JWT__PRIVATE_KEY_FILE=/test/key", "NERVE_CLOCK__OFFSET=" + value}})
				if profile != EnvTest && value != "0s" {
					if err == nil || !strings.Contains(err.Error(), "clock.offset") {
						t.Fatalf("Load() = %+v, %v; want clock.offset rejection", cfg, err)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				want, _ := time.ParseDuration(value)
				if cfg.Clock.Offset != want {
					t.Fatalf("offset = %s, want %s", cfg.Clock.Offset, want)
				}
			})
		}
	}
	t.Run("default", func(t *testing.T) {
		cfg, err := Load(Sources{Embedded: configs.FS(), Environ: []string{"NERVE_ENV=test", "NERVE_DATABASE__URL=postgres://test"}})
		if err != nil || cfg.Clock.Offset != 0 {
			t.Fatalf("Load() = %+v, %v; want zero offset", cfg.Clock, err)
		}
	})
	for _, tt := range []struct {
		name, yaml, env string
		want            time.Duration
		invalid         bool
	}{
		{name: "yaml", yaml: "-49h", want: -49 * time.Hour},
		{name: "env overrides yaml", yaml: "-49h", env: "26h", want: 26 * time.Hour},
		{name: "env resets yaml", yaml: "-49h", env: "0s"},
		{name: "malformed", env: "tomorrow", invalid: true},
		{name: "empty", env: "", yaml: "1h", invalid: true},
		{name: "yaml number", yaml: "42", invalid: true},
		{name: "yaml null", yaml: "null", invalid: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			profile := ""
			if tt.yaml != "" {
				profile = "clock:\n  offset: " + tt.yaml + "\n"
			}
			env := []string{"NERVE_ENV=test", "NERVE_DATABASE__URL=postgres://test"}
			if tt.env != "" || tt.name == "empty" {
				env = append(env, "NERVE_CLOCK__OFFSET="+tt.env)
			}
			cfg, err := Load(Sources{Embedded: embedded(map[string]string{"test": profile}), Environ: env})
			if tt.invalid {
				if err == nil || !strings.Contains(err.Error(), "clock.offset") {
					t.Fatalf("Load() = %+v, %v; want invalid clock.offset", cfg.Clock, err)
				}
				return
			}
			if err != nil || cfg.Clock.Offset != tt.want {
				t.Fatalf("Load() = %+v, %v; want %s", cfg.Clock, err, tt.want)
			}
			var logs bytes.Buffer
			slog.New(slog.NewTextHandler(&logs, nil)).Info("config", "config", cfg)
			if !strings.Contains(logs.String(), "config.clock.offset="+tt.want.String()) {
				t.Fatalf("effective offset missing: %s", logs.String())
			}
		})
	}
}
