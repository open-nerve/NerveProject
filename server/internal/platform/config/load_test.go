package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

const testBase = `
server:
  addr: ":8080"
  read_header_timeout: 5s
  shutdown_timeout: 20s
database:
  url: ""
  max_conns: 10
  auto_migrate: true
log:
  level: info
  format: json
`

func embedded(profiles map[string]string) fstest.MapFS {
	fsys := fstest.MapFS{"config.yaml": {Data: []byte(testBase)}}
	for name, content := range profiles {
		fsys["config."+name+".yaml"] = &fstest.MapFile{Data: []byte(content)}
	}
	return fsys
}

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadAppliesLayersInOrder(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", "log:\n  format: text\ndatabase:\n  max_conns: 20\n")
	writeFile(t, dir, "config.dev.yaml", "database:\n  max_conns: 30\nserver:\n  shutdown_timeout: 30s\n")
	local := writeFile(t, t.TempDir(), "config.local.yaml", "server:\n  shutdown_timeout: 40s\n  read_header_timeout: 7s\n")

	cfg, err := Load(Sources{
		Embedded: embedded(map[string]string{"dev": "database:\n  url: postgres://embedded-dev\nlog:\n  level: debug\n"}),
		Environ: []string{
			"NERVE_CONFIG_DIR=" + dir,
			"NERVE_SERVER__READ_HEADER_TIMEOUT=9s",
			"NERVE_DATABASE__AUTO_MIGRATE=false",
		},
		LocalFile: local,
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := Config{
		Env: EnvDev, // NERVE_ENV defaults to dev
		Server: ServerConfig{
			Addr:              ":8080",         // built-in config.yaml
			ReadHeaderTimeout: 9 * time.Second, // environment beats config.local.yaml
			ShutdownTimeout:   40 * time.Second,
		},
		Database: DatabaseConfig{
			URL:         "postgres://embedded-dev", // built-in config.dev.yaml
			MaxConns:    30,                        // config dir: config.dev.yaml beats config.yaml
			AutoMigrate: false,                     // environment
		},
		Log: LogConfig{Level: "debug", Format: "text"},
	}
	if cfg != want {
		t.Errorf("Load() =\n%+v\nwant\n%+v", cfg, want)
	}
}

func TestLoadSelectsProfile(t *testing.T) {
	local := writeFile(t, t.TempDir(), "config.local.yaml", "log:\n  level: error\n")
	cfg, err := Load(Sources{
		Embedded: embedded(map[string]string{"test": "log:\n  level: warn\n"}),
		Environ:  []string{"NERVE_ENV=test", "NERVE_DATABASE__URL=postgres://from-env"},
		// Only the dev profile reads the local file.
		LocalFile: local,
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Env != EnvTest || cfg.Log.Level != "warn" || cfg.Database.URL != "postgres://from-env" {
		t.Errorf("Load() = %+v, want test profile, level warn and the URL from the environment", cfg)
	}
}

func TestLoadIgnoresMissingOptionalFiles(t *testing.T) {
	_, err := Load(Sources{
		Embedded:  embedded(map[string]string{"dev": "database:\n  url: postgres://embedded-dev\n"}),
		Environ:   []string{"NERVE_CONFIG_DIR=" + t.TempDir()},
		LocalFile: filepath.Join(t.TempDir(), "config.local.yaml"),
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestLoadSkipsVariablesThatAreNotKeys(t *testing.T) {
	cfg, err := Load(Sources{
		Embedded: embedded(map[string]string{"prod": ""}),
		Environ: []string{
			"NERVE_ENV=prod",
			"NERVE_CONFIG_DIR=" + t.TempDir(),
			"NERVE_DEV_DB_PORT=55433",
			"NERVE_DATABASE__URL=postgres://from-env",
			"OTHER__VAR=ignored",
		},
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Database.URL != "postgres://from-env" {
		t.Errorf("database.url = %q, want the value of NERVE_DATABASE__URL", cfg.Database.URL)
	}
}

func TestLoadErrors(t *testing.T) {
	tests := []struct {
		name    string
		environ []string
		dirFile string // content of config.yaml in NERVE_CONFIG_DIR, if any
		want    string
	}{
		{
			name:    "unknown profile",
			environ: []string{"NERVE_ENV=staging"},
			want:    `NERVE_ENV must be one of dev, test, prod, got "staging"`,
		},
		{
			name:    "config dir does not exist",
			environ: []string{"NERVE_ENV=test", "NERVE_CONFIG_DIR=/does/not/exist"},
			want:    "NERVE_CONFIG_DIR=/does/not/exist is not a directory",
		},
		{
			name:    "required key missing",
			environ: []string{"NERVE_ENV=test"},
			want:    "invalid configuration:\ndatabase.url: is required",
		},
		{
			name:    "unknown key in the environment",
			environ: []string{"NERVE_ENV=test", "NERVE_DATABASE__URL=postgres://x", "NERVE_DATABSE__URL=postgres://x"},
			want:    "has invalid keys: databse",
		},
		{
			name:    "unknown key in a file",
			environ: []string{"NERVE_ENV=test", "NERVE_DATABASE__URL=postgres://x"},
			dirFile: "database:\n  max_con: 5\n",
			want:    "'database' has invalid keys: max_con",
		},
		{
			name:    "malformed duration",
			environ: []string{"NERVE_ENV=test", "NERVE_DATABASE__URL=postgres://x", "NERVE_SERVER__SHUTDOWN_TIMEOUT=soon"},
			want:    `'server.shutdown_timeout' time: invalid duration "soon"`,
		},
		{
			name:    "number where a duration is expected",
			environ: []string{"NERVE_ENV=test", "NERVE_DATABASE__URL=postgres://x"},
			dirFile: "server:\n  shutdown_timeout: 20\n",
			want:    `'server.shutdown_timeout' must be a duration such as "5s", got 20`,
		},
		{
			name:    "malformed number",
			environ: []string{"NERVE_ENV=test", "NERVE_DATABASE__URL=postgres://x", "NERVE_DATABASE__MAX_CONNS=many"},
			want:    "'database.max_conns'",
		},
		{
			name:    "malformed YAML",
			environ: []string{"NERVE_ENV=test"},
			dirFile: "server: [",
			want:    "config.yaml: yaml:",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			environ := tt.environ
			if tt.dirFile != "" {
				dir := t.TempDir()
				writeFile(t, dir, "config.yaml", tt.dirFile)
				environ = append(environ, "NERVE_CONFIG_DIR="+dir)
			}
			_, err := Load(Sources{Embedded: embedded(map[string]string{"test": ""}), Environ: environ})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Load() error = %v, want it to contain %q", err, tt.want)
			}
		})
	}
}
