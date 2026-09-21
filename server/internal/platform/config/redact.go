package config

import (
	"net/url"
	"strings"
)

const redacted = "xxxxx"

// redactURL masks the password of a PostgreSQL connection URL, both in the
// user info and in a password query parameter (matched in any case). A value
// that is not a URL is replaced entirely: key/value connection strings can
// carry a password too.
func redactURL(raw string) string {
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" {
		return redacted
	}
	q := u.Query()
	masked := false
	for key := range q {
		if strings.EqualFold(key, "password") {
			q.Set(key, redacted)
			masked = true
		}
	}
	if masked {
		u.RawQuery = q.Encode()
	}
	return u.Redacted()
}
