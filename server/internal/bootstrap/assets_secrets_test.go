package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
)

// Exercise actual files: ignoring conflicts, trimming internal whitespace or
// accepting a truncated oversized read would silently select wrong credentials.
func TestS3CredentialSources(t *testing.T) {
	base := config.S3FilesConfig{AccessKeyID: "access-value", SecretAccessKey: "secret-value"}
	t.Run("explicit values", func(t *testing.T) {
		c := base
		c.SessionToken = "session-value"
		got, err := loadS3Credentials(c)
		if err != nil || got.accessKeyID != "access-value" || got.secretAccessKey != "secret-value" || got.sessionToken != "session-value" {
			t.Fatalf("explicit credential sources not preserved: %v", err)
		}
	})
	t.Run("real files and one terminal newline", func(t *testing.T) {
		for _, suffix := range []string{"", "\n", "\r\n"} {
			c := config.S3FilesConfig{AccessKeyIDFile: writeFile(t, "access-value"+suffix),
				SecretAccessKeyFile: writeFile(t, "secret-value"+suffix), SessionTokenFile: writeFile(t, "session-value"+suffix)}
			got, err := loadS3Credentials(c)
			if err != nil || got.accessKeyID != "access-value" || got.secretAccessKey != "secret-value" || got.sessionToken != "session-value" {
				t.Fatalf("valid file/newline rejected: %v", err)
			}
		}
	})
	t.Run("optional token may be absent", func(t *testing.T) {
		got, err := loadS3Credentials(base)
		if err != nil || got.sessionToken != "" {
			t.Fatalf("optional token: %v", err)
		}
	})
	for _, tc := range []struct {
		name  string
		limit int
		set   func(*config.S3FilesConfig, string)
	}{
		{"access_key_id", 256, func(c *config.S3FilesConfig, p string) { c.AccessKeyID, c.AccessKeyIDFile = "", p }},
		{"secret_access_key", 4096, func(c *config.S3FilesConfig, p string) { c.SecretAccessKey, c.SecretAccessKeyFile = "", p }},
		{"session_token", 16384, func(c *config.S3FilesConfig, p string) { c.SessionToken, c.SessionTokenFile = "", p }},
	} {
		t.Run(tc.name+" bounded reads and invalid contents", func(t *testing.T) {
			c := base
			tc.set(&c, writeFile(t, strings.Repeat("x", tc.limit)+"\r\n"))
			got, err := loadS3Credentials(c)
			if err != nil {
				t.Fatalf("maximum file value rejected: %v", err)
			}
			actual := map[string]string{"access_key_id": got.accessKeyID, "secret_access_key": got.secretAccessKey, "session_token": got.sessionToken}
			if actual[tc.name] != strings.Repeat("x", tc.limit) {
				t.Fatal("maximum file value was truncated or retained a newline")
			}
			for _, data := range []string{"", "\n", "\r", "with space", "first\nsecond", "value\n\n", "value\n\r\n", "value\x00", string([]byte{0xff}), strings.Repeat("x", tc.limit+1), strings.Repeat("x", tc.limit+4)} {
				p := writeFile(t, data)
				tc.set(&c, p)
				_, err := loadS3Credentials(c)
				if err == nil || !strings.Contains(err.Error(), tc.name) || strings.Contains(err.Error(), p) {
					t.Fatalf("invalid file accepted or error exposed its path: %v", err)
				}
			}
		})
	}
	t.Run("conflicts and absent required sources", func(t *testing.T) {
		cases := []config.S3FilesConfig{
			{AccessKeyID: "a", AccessKeyIDFile: "/missing-private-key", SecretAccessKey: "s"},
			{AccessKeyID: "a", SecretAccessKey: "s", SecretAccessKeyFile: "/missing-private-key"},
			{AccessKeyID: "a", SecretAccessKey: "s", SessionToken: "t", SessionTokenFile: "/missing-private-key"},
			{AccessKeyID: "", SecretAccessKey: "s"}, {AccessKeyID: "a", SecretAccessKey: ""},
			{AccessKeyID: "bad access", SecretAccessKey: "s"}, {AccessKeyID: "a", SecretAccessKey: "bad\nsecret"},
		}
		for _, c := range cases {
			if _, err := loadS3Credentials(c); err == nil || strings.Contains(err.Error(), "/missing-private-key") {
				t.Fatalf("invalid sources accepted or leaked path: %v", err)
			}
		}
	})
	t.Run("missing unreadable and nonregular files", func(t *testing.T) {
		denied := writeFile(t, "private-value")
		if err := os.Chmod(denied, 0); err != nil {
			t.Fatal(err)
		}
		for _, path := range []string{filepath.Join(t.TempDir(), "missing"), t.TempDir(), denied} {
			c := base
			c.SecretAccessKey, c.SecretAccessKeyFile = "", path
			if _, err := loadS3Credentials(c); err == nil || strings.Contains(err.Error(), path) || strings.Contains(err.Error(), "private-value") {
				t.Fatalf("file error missing or sensitive: %v", err)
			}
		}
	})
}
