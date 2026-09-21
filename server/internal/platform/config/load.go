package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"
)

const (
	// Control variables: they steer loading and are never configuration keys.
	envProfile   = "NERVE_ENV"
	envConfigDir = "NERVE_CONFIG_DIR"

	envPrefix = "NERVE_"
	envKeySep = "__" // separates levels: NERVE_DATABASE__MAX_CONNS -> database.max_conns
	baseFile  = "config.yaml"
)

// Sources tells Load where configuration comes from.
type Sources struct {
	// Embedded holds the built-in config.yaml and config.<env>.yaml files.
	Embedded fs.FS
	// Environ is the process environment in os.Environ form.
	Environ []string
	// LocalFile is the personal override file. It is read only in the dev
	// profile, and only when it exists.
	LocalFile string
}

// Load builds the effective configuration. Each layer overrides the ones
// before it:
//
//  1. the built-in config.yaml, which lists every key with its default;
//  2. the built-in config.<env>.yaml;
//  3. config.yaml, then config.<env>.yaml, in $NERVE_CONFIG_DIR, when set and present;
//  4. LocalFile, in the dev profile only, when present;
//  5. NERVE_<SECTION>__<KEY> environment variables.
//
// The result is validated; the error lists every invalid key.
func Load(src Sources) (Config, error) {
	profile := lookupEnv(src.Environ, envProfile)
	if profile == "" {
		profile = EnvDev
	}
	if !slices.Contains([]string{EnvDev, EnvTest, EnvProd}, profile) {
		return Config{}, fmt.Errorf("%s must be one of dev, test, prod, got %q", envProfile, profile)
	}
	profileFile := "config." + profile + ".yaml"

	k := koanf.New(".")
	for _, name := range []string{baseFile, profileFile} {
		data, err := fs.ReadFile(src.Embedded, name)
		if err != nil {
			return Config{}, fmt.Errorf("read built-in %s: %w", name, err)
		}
		if err := k.Load(rawbytes.Provider(data), yaml.Parser()); err != nil {
			return Config{}, fmt.Errorf("parse built-in %s: %w", name, err)
		}
	}
	if dir := lookupEnv(src.Environ, envConfigDir); dir != "" {
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			return Config{}, fmt.Errorf("%s=%s is not a directory", envConfigDir, dir)
		}
		for _, name := range []string{baseFile, profileFile} {
			if err := loadFileIfExists(k, filepath.Join(dir, name)); err != nil {
				return Config{}, err
			}
		}
	}
	if profile == EnvDev && src.LocalFile != "" {
		if err := loadFileIfExists(k, src.LocalFile); err != nil {
			return Config{}, err
		}
	}
	environ := env.Provider(".", env.Opt{
		Prefix:        envPrefix,
		TransformFunc: envKey,
		EnvironFunc:   func() []string { return src.Environ },
	})
	if err := k.Load(environ, nil); err != nil {
		return Config{}, fmt.Errorf("read environment: %w", err)
	}

	cfg := Config{Env: profile}
	if err := decode(k, &cfg); err != nil {
		return Config{}, fmt.Errorf("invalid configuration:\n%w", err)
	}
	if err := cfg.validate(); err != nil {
		return Config{}, fmt.Errorf("invalid configuration:\n%w", err)
	}
	return cfg, nil
}

func lookupEnv(environ []string, name string) string {
	var value string
	for _, kv := range environ {
		if k, v, ok := strings.Cut(kv, "="); ok && k == name {
			value = v
		}
	}
	return value
}

// loadFileIfExists merges the YAML file at path into k and skips a missing
// file. It reads the file itself rather than through koanf's file provider,
// which would link fsnotify into the binary for a watch nerve never uses.
func loadFileIfExists(k *koanf.Koanf, path string) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("load %s: %w", path, err)
	}
	if err := k.Load(rawbytes.Provider(data), yaml.Parser()); err != nil {
		return fmt.Errorf("load %s: %w", path, err)
	}
	return nil
}

// envKey maps NERVE_DATABASE__MAX_CONNS to database.max_conns. Variables
// without the "__" separator (NERVE_ENV, NERVE_CONFIG_DIR, NERVE_DEV_DB_PORT
// and the like) are not configuration keys and are skipped: every key lives in
// a section, so a real key always has a separator.
func envKey(name, value string) (string, any) {
	key := strings.TrimPrefix(name, envPrefix)
	if !strings.Contains(key, envKeySep) {
		return "", nil
	}
	return strings.ToLower(strings.ReplaceAll(key, envKeySep, ".")), value
}

// decode copies the merged layers into cfg. Unknown keys are errors, so a typo
// never silently falls back to a default.
func decode(k *koanf.Koanf, cfg *Config) error {
	return k.UnmarshalWithConf("", cfg, koanf.UnmarshalConf{
		DecoderConfig: &mapstructure.DecoderConfig{
			DecodeHook:       durationHook,
			ErrorUnused:      true,
			WeaklyTypedInput: true, // environment values are strings
		},
	})
}

// durationHook decodes Go duration strings such as "5s". Bare numbers are
// rejected: a YAML 5 would otherwise silently mean 5ns.
func durationHook(_ reflect.Type, to reflect.Type, data any) (any, error) {
	if to != reflect.TypeFor[time.Duration]() {
		return data, nil
	}
	s, ok := data.(string)
	if !ok {
		return nil, fmt.Errorf("must be a duration such as \"5s\", got %v", data)
	}
	return time.ParseDuration(s)
}
