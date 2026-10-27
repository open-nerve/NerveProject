package config

import (
	"bytes"
	"log/slog"
	"slices"
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/configs"
)

// Removing the profile boundary, accepting malformed lists, or ignoring bad
// prefixes must change the result of the real layered Load path.
func TestOutboundAllowCIDRsConfig(t *testing.T) {
	for _, profile := range []string{EnvDev, EnvProd, EnvTest} {
		t.Run("built in/"+profile, func(t *testing.T) {
			cfg, err := Load(Sources{Embedded: configs.FS(), Environ: []string{
				"NERVE_ENV=" + profile, "NERVE_DATABASE__URL=postgres://test", "NERVE_AUTH__JWT__PRIVATE_KEY_FILE=/test/key",
			}})
			var want []string
			if profile == EnvTest {
				want = []string{"127.0.0.1/32"}
			}
			if err != nil || !slices.Equal(cfg.Outbound.AllowCIDRs, want) {
				t.Fatalf("Load() allow CIDRs = %v, %v; want %v", cfg.Outbound.AllowCIDRs, err, want)
			}
		})
		for _, input := range []string{"127.0.0.1/32", "::1/128"} {
			t.Run("environment/"+profile+"/"+input, func(t *testing.T) {
				cfg, err := Load(Sources{Embedded: embedded(map[string]string{profile: ""}), Environ: []string{
					"NERVE_ENV=" + profile, "NERVE_DATABASE__URL=postgres://test", "NERVE_AUTH__JWT__PRIVATE_KEY_FILE=/test/key",
					"NERVE_OUTBOUND__ALLOW_CIDRS=" + input,
				}})
				if profile != EnvTest {
					if err == nil || !strings.Contains(err.Error(), "outbound.allow_cidrs") {
						t.Fatalf("Load() error = %v; want outbound.allow_cidrs rejection", err)
					}
					return
				}
				if err != nil || !slices.Equal(cfg.Outbound.AllowCIDRs, []string{input}) {
					t.Fatalf("Load() allow CIDRs = %v, %v; want [%s]", cfg.Outbound.AllowCIDRs, err, input)
				}
			})
		}
	}
	for _, tt := range []struct {
		name, yaml string
		env        []string
		want       []string
		invalid    bool
	}{
		{name: "empty YAML", yaml: "[]"},
		{name: "IPv4 YAML", yaml: "[127.0.0.1/32]", want: []string{"127.0.0.1/32"}},
		{name: "IPv6 YAML", yaml: "['::1/128']", want: []string{"::1/128"}},
		{name: "comma environment overrides YAML", yaml: "[192.0.2.0/24]", env: []string{"NERVE_OUTBOUND__ALLOW_CIDRS= 127.0.0.1/32 , ::1/128 "}, want: []string{"127.0.0.1/32", "::1/128"}},
		{name: "empty environment clears YAML", yaml: "[127.0.0.1/32]", env: []string{"NERVE_OUTBOUND__ALLOW_CIDRS="}},
		{name: "bad CIDR YAML", yaml: "[127.0.0.1/32, broken]", invalid: true},
		{name: "missing prefix", env: []string{"NERVE_OUTBOUND__ALLOW_CIDRS=127.0.0.1"}, invalid: true},
		{name: "bad IPv6 prefix", env: []string{"NERVE_OUTBOUND__ALLOW_CIDRS=::1/129"}, invalid: true},
		{name: "JSON array environment", env: []string{`NERVE_OUTBOUND__ALLOW_CIDRS=["127.0.0.1/32"]`}, invalid: true},
		{name: "empty JSON array environment", env: []string{"NERVE_OUTBOUND__ALLOW_CIDRS=[]"}, invalid: true},
		{name: "JSON object environment", env: []string{"NERVE_OUTBOUND__ALLOW_CIDRS={}"}, invalid: true},
		{name: "null environment", env: []string{"NERVE_OUTBOUND__ALLOW_CIDRS=null"}, invalid: true},
		{name: "indexed environment", env: []string{"NERVE_OUTBOUND__ALLOW_CIDRS__0=127.0.0.1/32"}, invalid: true},
		{name: "empty element environment", env: []string{"NERVE_OUTBOUND__ALLOW_CIDRS=127.0.0.1/32,"}, invalid: true},
		{name: "null YAML", yaml: "null", invalid: true},
		{name: "null YAML element", yaml: "[127.0.0.1/32, null]", invalid: true},
		{name: "number YAML", yaml: "42", invalid: true},
		{name: "object YAML", yaml: "{cidr: 127.0.0.1/32}", invalid: true},
		{name: "empty object YAML", yaml: "{}", invalid: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			profile := ""
			if tt.yaml != "" {
				profile = "outbound:\n  allow_cidrs: " + tt.yaml + "\n"
			}
			cfg, err := Load(Sources{Embedded: embedded(map[string]string{EnvTest: profile}), Environ: append([]string{
				"NERVE_ENV=test", "NERVE_DATABASE__URL=postgres://test",
			}, tt.env...)})
			if tt.invalid {
				if err == nil || !strings.Contains(err.Error(), "outbound.allow_cidrs") {
					t.Fatalf("Load() error = %v; want invalid outbound.allow_cidrs", err)
				}
				return
			}
			if err != nil || !slices.Equal(cfg.Outbound.AllowCIDRs, tt.want) {
				t.Fatalf("Load() allow CIDRs = %v, %v; want %v", cfg.Outbound.AllowCIDRs, err, tt.want)
			}
		})
	}
}

// Adding the effective outbound settings must keep the established slog
// whitelist: database credentials and secret-bearing file paths stay masked.
func TestOutboundConfigLogsRedacted(t *testing.T) {
	cfg := validConfig()
	cfg.Outbound.AllowCIDRs = []string{"127.0.0.1/32", "::1/128"}
	cfg.Database.URL = "postgres://user:db-password-sentinel@localhost/db?password=query-secret-sentinel"
	cfg.Server.AddrFile = "/private/addr-path-sentinel"
	cfg.Auth.JWT.PrivateKeyFile = "/private/key-path-sentinel"
	for _, format := range []string{"text", "json"} {
		t.Run(format, func(t *testing.T) {
			var logs bytes.Buffer
			var handler slog.Handler = slog.NewTextHandler(&logs, nil)
			if format == "json" {
				handler = slog.NewJSONHandler(&logs, nil)
			}
			slog.New(handler).Info("configuration loaded", "config", cfg)
			out := logs.String()
			for _, secret := range []string{"db-password-sentinel", "query-secret-sentinel", "addr-path-sentinel", "key-path-sentinel"} {
				if strings.Contains(out, secret) {
					t.Errorf("configuration log leaks %s: %s", secret, out)
				}
			}
			for _, value := range []string{"allow_cidrs", "127.0.0.1/32", "::1/128", "xxxxx", "private_key_file_set", "addr_file_set"} {
				if !strings.Contains(out, value) {
					t.Errorf("configuration log lacks %s: %s", value, out)
				}
			}
		})
	}
}
