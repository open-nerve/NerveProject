package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	S3AccessKeyIDMaxBytes     = 256
	S3SecretAccessKeyMaxBytes = 4096
	S3SessionTokenMaxBytes    = 16384
)

// S3FilesConfig is explicit deployment configuration, never a request input.
// Files remain unread until bootstrap constructs the active S3 driver.
type S3FilesConfig struct {
	Endpoint            string `koanf:"endpoint"`
	Region              string `koanf:"region"`
	Bucket              string `koanf:"bucket"`
	Prefix              string `koanf:"prefix"`
	PathStyle           bool   `koanf:"path_style"`
	AccessKeyID         string `koanf:"access_key_id"`
	AccessKeyIDFile     string `koanf:"access_key_id_file"`
	SecretAccessKey     string `koanf:"secret_access_key"`
	SecretAccessKeyFile string `koanf:"secret_access_key_file"`
	SessionToken        string `koanf:"session_token"`
	SessionTokenFile    string `koanf:"session_token_file"`
}

func (c S3FilesConfig) validate(fail func(string, string, ...any)) {
	if !validS3Endpoint(c.Endpoint) {
		fail("files.s3.endpoint", "must be an explicit HTTP(S) DNS or IPv4 origin without credentials, path, query or fragment")
	}
	if len(c.Region) < 1 || len(c.Region) > 64 || !asciiToken(c.Region, "-") {
		fail("files.s3.region", "must contain 1 to 64 ASCII letters, digits or hyphens")
	}
	if !validS3Bucket(c.Bucket) {
		fail("files.s3.bucket", "must be a 3 to 63 character ordinary DNS bucket name")
	}
	if !validS3Prefix(c.Prefix) {
		fail("files.s3.prefix", "must be an exclusive 1 to 128 byte ASCII path ending in a slash, without empty or dot segments")
	}
	c.validateCredentials(true, fail)
}

// ValidateCredentials also checks the resolved values after bootstrap reads
// secret files, so file contents and environment values obey the same rule.
func (c S3FilesConfig) ValidateCredentials() error {
	var errs []error
	c.validateCredentials(true, func(key, format string, args ...any) {
		errs = append(errs, fmt.Errorf("%s: %s", key, fmt.Sprintf(format, args...)))
	})
	return errors.Join(errs...)
}

// Validate also serves explicit adapter construction, so it obeys the same
// format rules as deployment configuration. No I/O occurs.
func (c S3FilesConfig) Validate() error {
	var errs []error
	c.validate(func(key, format string, args ...any) {
		errs = append(errs, fmt.Errorf("%s: %s", key, fmt.Sprintf(format, args...)))
	})
	return errors.Join(errs...)
}

func (c S3FilesConfig) validateCredentials(required bool, fail func(string, string, ...any)) {
	for _, s := range []struct {
		key, value, file string
		limit            int
		required         bool
	}{
		{"access_key_id", c.AccessKeyID, c.AccessKeyIDFile, S3AccessKeyIDMaxBytes, true},
		{"secret_access_key", c.SecretAccessKey, c.SecretAccessKeyFile, S3SecretAccessKeyMaxBytes, true},
		{"session_token", c.SessionToken, c.SessionTokenFile, S3SessionTokenMaxBytes, false},
	} {
		key := "files.s3." + s.key
		if s.value != "" && s.file != "" {
			fail(key, "value and _file sources are mutually exclusive")
		}
		if required && s.required && s.value == "" && s.file == "" {
			fail(key, "requires an explicit value or _file source")
		}
		if s.value != "" && (len(s.value) > s.limit || !utf8.ValidString(s.value) || strings.ContainsFunc(s.value, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) })) {
			fail(key, "must contain at most %d bytes without whitespace or control characters", s.limit)
		}
		if s.file != "" && (strings.TrimSpace(s.file) == "" || !utf8.ValidString(s.file) || strings.ContainsFunc(s.file, unicode.IsControl)) {
			fail(key+"_file", "must be a nonempty file path without control characters")
		}
	}
}

func validS3Endpoint(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.Opaque != "" ||
		(u.Path != "" && u.Path != "/") || u.RawPath != "" || strings.ContainsAny(raw, "?#") || strings.ContainsFunc(raw, unicode.IsControl) {
		return false
	}
	if strings.HasSuffix(u.Host, ":") {
		return false
	}
	if port := u.Port(); port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return false
		}
	}
	host := u.Hostname()
	if ip, err := netip.ParseAddr(host); err == nil {
		// CSP host sources cannot express IPv6 literals. DNS names may still
		// resolve to IPv6; reject only a literal before constructing storage.
		return ip.Is4()
	}
	return validDNSName(strings.ToLower(strings.TrimSuffix(host, ".")))
}

func validDNSName(name string) bool {
	if name == "" || len(name) > 253 {
		return false
	}
	for _, label := range strings.Split(name, ".") {
		if label == "" || len(label) > 63 || !asciiToken(label, "-") || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
	}
	return true
}

func validS3Bucket(name string) bool {
	if len(name) < 3 || len(name) > 63 || strings.ToLower(name) != name || !validDNSName(name) {
		return false
	}
	if _, err := netip.ParseAddr(name); err == nil {
		return false
	}
	for _, prefix := range []string{"xn--", "sthree-", "amzn-s3-demo-"} {
		if strings.HasPrefix(name, prefix) {
			return false
		}
	}
	for _, suffix := range []string{"-s3alias", "--ol-s3", ".mrap", "--x-s3", "--table-s3"} {
		if strings.HasSuffix(name, suffix) {
			return false
		}
	}
	return true
}

func validS3Prefix(prefix string) bool {
	if len(prefix) < 1 || len(prefix) > 128 || !strings.HasSuffix(prefix, "/") {
		return false
	}
	for _, part := range strings.Split(strings.TrimSuffix(prefix, "/"), "/") {
		if part == "" || !asciiToken(part, "_-") {
			return false
		}
	}
	return true
}

func asciiToken(value, punctuation string) bool {
	for _, b := range []byte(value) {
		switch {
		case b >= 'a' && b <= 'z', b >= 'A' && b <= 'Z', b >= '0' && b <= '9':
		case strings.IndexByte(punctuation, b) >= 0:
		default:
			return false
		}
	}
	return true
}

func (c S3FilesConfig) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Bool("endpoint_set", c.Endpoint != ""), slog.Bool("region_set", c.Region != ""),
		slog.Bool("bucket_set", c.Bucket != ""), slog.Bool("prefix_set", c.Prefix != ""), slog.Bool("path_style", c.PathStyle),
		slog.Bool("access_key_id_set", c.AccessKeyID != ""), slog.Bool("access_key_id_file_set", c.AccessKeyIDFile != ""),
		slog.Bool("secret_access_key_set", c.SecretAccessKey != ""), slog.Bool("secret_access_key_file_set", c.SecretAccessKeyFile != ""),
		slog.Bool("session_token_set", c.SessionToken != ""), slog.Bool("session_token_file_set", c.SessionTokenFile != ""),
	)
}
