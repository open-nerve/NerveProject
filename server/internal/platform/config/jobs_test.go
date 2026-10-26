package config

import (
	"fmt"
	"strings"
	"testing"
)

func TestJobWorkerConfiguration(t *testing.T) {
	for _, key := range []string{"DEFAULT_WORKERS", "PERIODIC_WORKERS"} {
		for _, value := range []string{"1", "13", "0", "-1", "1.5", "", "null"} {
			t.Run(key+"="+value, func(t *testing.T) {
				cfg, err := Load(Sources{Embedded: embedded(map[string]string{"dev": "database:\n  url: postgres://test\n"}), Environ: []string{"NERVE_JOBS__" + key + "=" + value}})
				if value != "1" && value != "13" {
					if err == nil || !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(key)) {
						t.Fatalf("Load()=%+v,%v; want invalid worker count", cfg.Jobs, err)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				n := cfg.Jobs.DefaultWorkers
				if key == "PERIODIC_WORKERS" {
					n = cfg.Jobs.PeriodicWorkers
				}
				if fmt.Sprint(n) != value {
					t.Fatalf("worker count=%d, want %s", n, value)
				}
			})
		}
	}
}

// YAML counts pass through the same loading and validation as other keys;
// malformed values and YAML null must not silently become defaults.
func TestJobWorkerYAMLConfiguration(t *testing.T) {
	for _, key := range []string{"default_workers", "periodic_workers"} {
		for _, value := range []string{"1", "17", "0", "-1", "1.5", "null"} {
			t.Run(key+"="+value, func(t *testing.T) {
				cfg, err := Load(Sources{Embedded: embedded(map[string]string{"dev": "database:\n  url: postgres://test\njobs:\n  " + key + ": " + value + "\n"})})
				if value != "1" && value != "17" {
					if err == nil || !strings.Contains(err.Error(), key) {
						t.Fatalf("Load()=%+v,%v; want invalid worker count", cfg.Jobs, err)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				n := cfg.Jobs.DefaultWorkers
				if key == "periodic_workers" {
					n = cfg.Jobs.PeriodicWorkers
				}
				if fmt.Sprint(n) != value {
					t.Fatalf("worker count=%d, want %s", n, value)
				}
			})
		}
	}
}
