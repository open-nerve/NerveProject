package config

import (
	"net/url"
	"strings"
)

const redacted = "xxxxx"

// redactURL masks the password of a PostgreSQL connection URL, both in the
// user info and in every query parameter whose name contains "password" in any
// case (password, sslpassword). Only values starting with postgres:// or
// postgresql:// (the URL prefixes pgx recognises, matched in any case here)
// are treated as URLs; anything else is replaced entirely, since key/value
// connection strings and opaque values can carry a password too.
func redactURL(raw string) string {
	if raw == "" {
		return ""
	}
	lower := strings.ToLower(raw)
	if !strings.HasPrefix(lower, "postgres://") && !strings.HasPrefix(lower, "postgresql://") {
		return redacted
	}
	u, err := url.Parse(raw)
	if err != nil {
		return redacted
	}
	q := u.Query()
	masked := false
	for key := range q {
		if strings.Contains(strings.ToLower(key), "password") {
			q.Set(key, redacted)
			masked = true
		}
	}
	if masked {
		u.RawQuery = q.Encode()
	}
	return u.Redacted()
}
