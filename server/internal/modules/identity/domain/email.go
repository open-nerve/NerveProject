package domain

import (
	"net/netip"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// MaxEmailLength is the length of users.email, varchar(255), in characters.
const MaxEmailLength = 255

// NormalizeEmail is what Plane does before it validates or looks up an
// e-mail address (email.strip().lower(), plane/apps/api/plane/
// authentication/views/app/email.py:73).
func NormalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// ValidEmail reports whether a normalized address is acceptable (M2 design
// 4.2): at most 255 characters, no white space or control character
// anywhere, and valid by Django's EmailValidator, which Plane uses
// (plane/apps/api/plane/authentication/adapter/base.py:79).
func ValidEmail(email string) bool {
	if email == "" || utf8.RuneCountInString(email) > MaxEmailLength || !utf8.ValidString(email) {
		return false
	}
	for _, r := range email {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return false
		}
	}
	return djangoEmail(email)
}

// The regular expressions of Django 5.2's EmailValidator
// (django/core/validators.py), with every look-around rewritten for RE2 into
// the equivalent explicit form: a domain label of 1–63 characters neither
// starts nor ends with a hyphen.
//
// Django compiles them with re.IGNORECASE on str patterns, under which
// [A-Z] and [a-z] also match four non-ASCII letters (Python's re
// documentation): İ, ı, ſ and K. After lower-casing only ı (U+0131) and
// ſ (U+017F) are left, so the local part's classes add those two.
const (
	djangoUL          = `\x{00a1}-\x{ffff}` // Django's "Unicode letters" range
	djangoLabelChar   = `[a-z` + djangoUL + `0-9]`
	djangoLabel       = djangoLabelChar + `(?:[a-z` + djangoUL + `0-9-]{0,61}` + djangoLabelChar + `)?`
	djangoTLDChar     = `[a-z` + djangoUL + `]`
	djangoTLD         = `\.(?:` + djangoTLDChar + `[a-z` + djangoUL + `-]{0,61}` + djangoTLDChar + `|xn--[a-z0-9]{1,59})`
	djangoAtom        = "[-!#$%&'*+/=?^_" + "`" + `{}|~0-9a-z\x{0131}\x{017f}]+`
	djangoQuotedChars = `[\x01-\x08\x0b\x0c\x0e-\x1f!#-\[\]-\x7f\x{0131}\x{017f}]|\\[\x01-\x09\x0b\x0c\x0e-\x7f\x{0131}\x{017f}]`
)

var (
	djangoUser    = regexp.MustCompile(`(?i)^(?:` + djangoAtom + `(?:\.` + djangoAtom + `)*|"(?:` + djangoQuotedChars + `)*")$`)
	djangoDomain  = regexp.MustCompile(`(?i)^` + djangoLabel + `(?:\.` + djangoLabel + `)*` + djangoTLD + `$`)
	djangoLiteral = regexp.MustCompile(`(?i)^\[([a-f0-9:.]+)\]$`)
)

// djangoEmail is EmailValidator.__call__ with the default allow list, except
// for the length, which ValidEmail bounds more tightly.
func djangoEmail(email string) bool {
	at := strings.LastIndexByte(email, '@')
	if at < 0 {
		return false
	}
	user, domain := email[:at], email[at+1:]
	if !djangoUser.MatchString(user) {
		return false
	}
	if domain == "localhost" || djangoDomain.MatchString(domain) {
		return true
	}
	// A literal address: validate_ipv46_address.
	m := djangoLiteral.FindStringSubmatch(domain)
	if m == nil {
		return false
	}
	_, err := netip.ParseAddr(m[1])
	return err == nil
}

// DisplayNameFromEmail is the display name a new account gets: the part of
// the address before its first @, as Plane's User.save() does
// (plane/apps/api/plane/db/models/user.py:169-187). It is never empty for a
// valid address.
func DisplayNameFromEmail(email string) string {
	name, _, _ := strings.Cut(email, "@")
	return name
}
