package config

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/configs"
)

func loadS3Config(values ...string) (Config, error) {
	env := []string{"NERVE_ENV=test", "NERVE_DATABASE__URL=postgres://test", "NERVE_FILES__DRIVER=s3",
		"NERVE_FILES__S3__ENDPOINT=http://127.0.0.1:9000", "NERVE_FILES__S3__REGION=us-east-1",
		"NERVE_FILES__S3__BUCKET=nerve-test", "NERVE_FILES__S3__PREFIX=instance-a/",
		"NERVE_FILES__S3__ACCESS_KEY_ID=test-access", "NERVE_FILES__S3__SECRET_ACCESS_KEY=test-secret"}
	return Load(Sources{Embedded: configs.FS(), Environ: append(env, values...)})
}

// These cases catch configuration that would send credentials to an ambiguous
// origin or mix another instance's objects into a cleanup listing.
func TestFilesS3Config(t *testing.T) {
	t.Run("explicit configuration and defaults", func(t *testing.T) {
		cfg, err := loadS3Config()
		if err != nil || cfg.Files.Driver != "s3" || !cfg.Files.S3.PathStyle {
			t.Fatalf("explicit S3 config/default path style: %v / %q", err, cfg.Files.Driver)
		}
		cfg, err = loadS3Config("NERVE_FILES__S3__PATH_STYLE=false", "NERVE_FILES__LOCAL__ROOT=")
		if err != nil || cfg.Files.S3.PathStyle {
			t.Fatalf("explicit virtual host/local root unused: %v", err)
		}
	})
	t.Run("local needs no S3 configuration or files", func(t *testing.T) {
		cfg, err := Load(Sources{Embedded: configs.FS(), Environ: []string{"NERVE_ENV=test", "NERVE_DATABASE__URL=postgres://test",
			"NERVE_FILES__S3__SECRET_ACCESS_KEY_FILE=/does-not-exist/unused-secret"}})
		if err != nil || cfg.Files.Driver != "local" {
			t.Fatalf("local config was coupled to S3 credentials: %v", err)
		}
	})
	t.Run("conflicting explicit sources fail even when inactive", func(t *testing.T) {
		_, err := loadS3Config("NERVE_FILES__DRIVER=local", "NERVE_FILES__S3__SECRET_ACCESS_KEY_FILE=/not-read")
		if err == nil || !strings.Contains(err.Error(), "secret_access_key") {
			t.Fatalf("inactive driver hid conflicting explicit sources: %v", err)
		}
	})
	for _, tc := range []struct{ field, value string }{
		{"ENDPOINT", ""}, {"ENDPOINT", "ftp://example.com"}, {"ENDPOINT", "https://user:password@example.com"},
		{"ENDPOINT", "https://example.com/path"}, {"ENDPOINT", "https://example.com?x=1"}, {"ENDPOINT", "https://example.com#fragment"},
		{"ENDPOINT", "https://example.com:0"}, {"ENDPOINT", "https://example.com:65536"}, {"ENDPOINT", "https://example.com:"},
		{"ENDPOINT", "https://example.com\n"}, {"ENDPOINT", "https://*.example.com"}, {"ENDPOINT", "https://"},
		{"REGION", ""}, {"REGION", "bad region"}, {"REGION", strings.Repeat("r", 65)},
		{"BUCKET", "ab"}, {"BUCKET", strings.Repeat("b", 64)}, {"BUCKET", "Upper"}, {"BUCKET", "a..b"},
		{"BUCKET", "a.-b"}, {"BUCKET", "a-.b"}, {"BUCKET", "127.0.0.1"}, {"BUCKET", "-abc"}, {"BUCKET", "abc-"},
		{"BUCKET", "arn:aws:s3:::bucket"}, {"BUCKET", "bucket--x-s3"}, {"BUCKET", "bucket-s3alias"},
		{"BUCKET", "xn--bucket"}, {"BUCKET", "sthree-bucket"}, {"BUCKET", "amzn-s3-demo-bucket"},
		{"BUCKET", "bucket--ol-s3"}, {"BUCKET", "bucket.mrap"}, {"BUCKET", "bucket--table-s3"},
		{"PREFIX", ""}, {"PREFIX", "one"}, {"PREFIX", "/one/"}, {"PREFIX", "one//two/"},
		{"PREFIX", "../one/"}, {"PREFIX", "one/./"}, {"PREFIX", "one%2f/"}, {"PREFIX", "中文/"},
		{"PREFIX", strings.Repeat("a", 128) + "/"}, {"PATH_STYLE", ""}, {"PATH_STYLE", "not-bool"},
		{"UNKNOWN", "no"}, {"SECRET_ACCESS_KEY", "bad\nsecret"}, {"ACCESS_KEY_ID", "bad access"}, {"SESSION_TOKEN", "bad\ttoken"},
	} {
		t.Run("reject "+tc.field+" "+tc.value, func(t *testing.T) {
			_, err := loadS3Config("NERVE_FILES__S3__" + tc.field + "=" + tc.value)
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.field)) {
				t.Fatalf("expected rejection naming field %s, got %v", tc.field, err)
			}
		})
	}
	t.Run("allowed endpoints and maximum nonsecret lengths", func(t *testing.T) {
		for _, endpoint := range []string{"https://s3.example.com", "http://127.0.0.1:9000/", "http://[::1]:9000", "https://example.com:443"} {
			_, err := loadS3Config("NERVE_FILES__S3__ENDPOINT="+endpoint, "NERVE_FILES__S3__REGION="+strings.Repeat("r", 64),
				"NERVE_FILES__S3__BUCKET="+strings.Repeat("b", 63), "NERVE_FILES__S3__PREFIX="+strings.Repeat("a", 127)+"/")
			if err != nil {
				t.Fatalf("valid endpoint/boundaries rejected: %v", err)
			}
		}
	})
	for _, tc := range []struct {
		field string
		limit int
	}{{"ACCESS_KEY_ID", 256}, {"SECRET_ACCESS_KEY", 4096}, {"SESSION_TOKEN", 16384}} {
		t.Run(tc.field+" byte boundary and sources", func(t *testing.T) {
			prefix := "NERVE_FILES__S3__" + tc.field
			if _, err := loadS3Config(prefix + "=" + strings.Repeat("s", tc.limit)); err != nil {
				t.Fatal(err)
			}
			if _, err := loadS3Config(prefix + "=" + strings.Repeat("s", tc.limit+1)); err == nil {
				t.Fatal("oversized credential accepted")
			}
			if _, err := loadS3Config(prefix+"=secret", prefix+"_FILE=/not-read"); err == nil {
				t.Fatal("simultaneous value and file accepted")
			}
			if _, err := loadS3Config(prefix+"=", prefix+"_FILE=/not-read"); err != nil {
				t.Fatalf("file read during pure configuration: %v", err)
			}
		})
	}
	t.Run("inline YAML secrets are rejected before environment overrides", func(t *testing.T) {
		for _, field := range []string{"access_key_id", "secret_access_key", "session_token", "SECRET_ACCESS_KEY"} {
			dir := t.TempDir()
			writeFile(t, dir, "config.yaml", "files:\n  s3:\n    "+field+": forbidden-inline-value\n")
			_, err := loadS3Config("NERVE_CONFIG_DIR=" + dir)
			if err == nil || strings.Contains(err.Error(), "forbidden-inline-value") || !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(field)) {
				t.Fatalf("inline secret was accepted or leaked: %v", err)
			}
		}
	})
	t.Run("S3 configuration logs never reveal values or file paths", func(t *testing.T) {
		cfg, err := loadS3Config("NERVE_FILES__S3__SESSION_TOKEN=private-token-value")
		if err != nil {
			t.Fatal(err)
		}
		cfg.Files.S3.AccessKeyIDFile = "/private/access-file"
		cfg.Files.S3.SecretAccessKeyFile = "/private/secret-file"
		cfg.Files.S3.SessionTokenFile = "/private/token-file"
		var out bytes.Buffer
		slog.New(slog.NewJSONHandler(&out, nil)).Info("configuration", "files", cfg.Files, "s3", cfg.Files.S3)
		for _, value := range []string{"test-access", "test-secret", "private-token-value", "/private/access-file", "/private/secret-file", "/private/token-file"} {
			if strings.Contains(out.String(), value) {
				t.Fatal("configuration log exposed secret material")
			}
		}
		if !strings.Contains(out.String(), `"access_key_id_set":true`) || !strings.Contains(out.String(), `"secret_access_key_file_set":true`) {
			t.Fatal("redacted configuration lost presence information")
		}
	})
}
