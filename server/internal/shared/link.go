package shared

import (
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// LinkMetadata represents pending previews with an empty object and preview
// fields with nullable strings. Stored keys are title, favicon, url,
// favicon_url and error, constrained by the schema of each table that stores
// links.
type LinkMetadata map[string]*string

// CheckLinkURL normalizes only surrounding whitespace and an omitted HTTP
// scheme, preserving the exact URL otherwise. Anything but an HTTP or HTTPS
// URL without credentials, zones or control characters, of at most 2048
// characters, is a not_allowed problem on url.
func CheckLinkURL(raw string) (string, *FieldError) {
	refused := &FieldError{Field: "url", Code: FieldNotAllowed, Message: "must be an HTTP or HTTPS URL without credentials, zones or control characters, of at most 2048 characters"}
	value := strings.TrimSpace(raw)
	if len(value) >= 7 && strings.EqualFold(value[:7], "http://") || len(value) >= 8 && strings.EqualFold(value[:8], "https://") {
		// Parse a copy below; the caller's casing remains the uniqueness key.
	} else if prefix, _, found := strings.Cut(value, "://"); found && linkScheme(prefix) {
		return "", refused
	} else {
		value = "http://" + value
	}
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) > 2048 || strings.ContainsFunc(value, unicode.IsControl) {
		return "", refused
	}
	u, err := url.Parse(value)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || strings.Contains(u.Hostname(), "%") {
		return "", refused
	}
	if strings.Contains(u.Hostname(), ":") || strings.HasPrefix(u.Host, "[") {
		ip, err := netip.ParseAddr(u.Hostname())
		if err != nil || !ip.Is6() || !strings.HasPrefix(u.Host, "[") {
			return "", refused
		}
	}
	if strings.HasSuffix(u.Host, ":") {
		return "", refused
	}
	if port := u.Port(); port != "" {
		if _, err := strconv.ParseUint(port, 10, 16); err != nil {
			return "", refused
		}
	}
	return value, nil
}

// CheckLinkTitle refuses a title of more than 255 characters or with a NUL
// character on title; a nil title is no title and passes.
func CheckLinkTitle(title *string) *FieldError {
	if title == nil {
		return nil
	}
	if utf8.RuneCountInString(*title) > 255 {
		return &FieldError{Field: "title", Code: FieldTooLong, Message: "must be at most 255 characters"}
	}
	if strings.ContainsRune(*title, 0) {
		return &FieldError{Field: "title", Code: FieldInvalidFormat, Message: "must not contain a NUL character"}
	}
	return nil
}

// linkScheme recognizes only a leading RFC scheme; path/query :// is data.
func linkScheme(s string) bool {
	for n, c := range s {
		letter := c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
		rest := c >= '0' && c <= '9' || c == '+' || c == '-' || c == '.'
		if !letter && (n == 0 || !rest) {
			return false
		}
	}
	return s != ""
}
