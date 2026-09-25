package domain

import (
	_ "embed"
	"slices"
	"strings"
	"unicode"
	"unicode/utf16"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Password lengths, in UTF-16 code units like the web app's password.length.
const (
	MinPasswordLength = 8
	MaxPasswordLength = 128
)

// passwordSpecials are the special characters of the composition rules,
// the same as getPasswordStrength in web/packages/utils/src/auth.ts.
const passwordSpecials = `!@#$%^&*()-_+=[]{}|;:'",.<>?/`

//go:embed common_passwords.txt
var commonPasswordsFile string

// PasswordRules are the server's password rules (M2 design 3.8): the
// composition rules the web app shows, and the common-password list.
type PasswordRules struct {
	common []string // sorted: binary search
}

// NewPasswordRules parses the embedded common-password list: a header, a
// blank line, then one lowercased entry per line, sorted by bytes.
func NewPasswordRules() *PasswordRules {
	_, list, _ := strings.Cut(commonPasswordsFile, "\n\n")
	return &PasswordRules{common: strings.Split(strings.TrimSuffix(list, "\n"), "\n")}
}

// Check returns the field error for a new password of the account with the
// given normalized e-mail address, or nil when it is acceptable.
//
//   - weak_password: not 8–128 characters, or it lacks an upper-case letter,
//     a lower-case letter, a digit or a special character (ASCII classes);
//   - common_password: the lowercased password or its core is on the list,
//     or its core is the core of the address's local part.
func (p *PasswordRules) Check(field, password, email string) *shared.FieldError {
	switch {
	case password == "":
		return &shared.FieldError{Field: field, Code: shared.FieldRequired, Message: "is required"}
	case !composed(password):
		return &shared.FieldError{Field: field, Code: shared.FieldWeakPassword,
			Message: "must be 8-128 characters with an upper-case letter, a lower-case letter, a digit and one of " + passwordSpecials}
	case p.isCommon(password, email):
		return &shared.FieldError{Field: field, Code: shared.FieldCommonPassword, Message: "is too common"}
	}
	return nil
}

func composed(password string) bool {
	n := len(utf16.Encode([]rune(password)))
	return n >= MinPasswordLength && n <= MaxPasswordLength &&
		strings.ContainsFunc(password, func(r rune) bool { return r >= 'A' && r <= 'Z' }) &&
		strings.ContainsFunc(password, func(r rune) bool { return r >= 'a' && r <= 'z' }) &&
		strings.ContainsFunc(password, func(r rune) bool { return r >= '0' && r <= '9' }) &&
		strings.ContainsAny(password, passwordSpecials)
}

func (p *PasswordRules) isCommon(password, email string) bool {
	c := core(password)
	local, _, _ := strings.Cut(email, "@")
	return p.listed(strings.ToLower(password)) || p.listed(c) || (c != "" && c == core(local))
}

func (p *PasswordRules) listed(s string) bool {
	_, found := slices.BinarySearch(p.common, s)
	return found
}

// core lowercases s and trims every character that is not a letter
// (unicode.IsLetter, \p{L}) from both ends: Password1!~, ~Password1! and
// "Password1! " all have the core "password". tools/password-blocklist
// applies the same rule when it builds the list.
func core(s string) string {
	return strings.TrimFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) })
}
