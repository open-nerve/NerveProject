package config

import "net/url"

const redacted = "xxxxx"

// redactURL masks the password of a PostgreSQL connection URL, both in the
// user info and in a password query parameter. A value that is not a URL is
// replaced entirely: key/value connection strings can carry a password too.
func redactURL(raw string) string {
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" {
		return redacted
	}
	if q := u.Query(); q.Has("password") {
		q.Set("password", redacted)
		u.RawQuery = q.Encode()
	}
	return u.Redacted()
}
