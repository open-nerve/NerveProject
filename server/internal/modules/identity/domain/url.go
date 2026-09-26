package domain

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// pySpace is what Python's \s matches in a str pattern (str.isspace()):
// Go's \s is ASCII only and lacks \v.
const pySpace = `\t-\r\x1c-\x20\x85\xa0\x{1680}\x{2000}-\x{200a}\x{2028}\x{2029}\x{202f}\x{205f}\x{3000}`

// pyLetters is [a-zA-Z] under Python's re.IGNORECASE, which also matches
// İ, ı, ſ and K (Python's re documentation). Go's (?i) adds ſ and K by case
// folding; İ and ı are listed.
const pyLetters = `a-zA-Z\x{0130}\x{0131}`

// urlPattern is Plane's URL_PATTERN (plane/apps/api/plane/utils/url.py:12-23)
// for RE2: an http(s) address, a www. host, a dotted host name with a TLD
// of 2–6 letters, or an IPv4 address, anywhere in the text.
var urlPattern = regexp.MustCompile(`(?i)(?:` +
	`https?://[^` + pySpace + `]+` +
	`|www\.[` + pyLetters + `0-9](?:[` + pyLetters + `0-9-]{0,61}[` + pyLetters + `0-9])?(?:\.[` + pyLetters + `0-9](?:[` + pyLetters + `0-9-]{0,61}[` + pyLetters + `0-9])?)*` +
	`|(?:[` + pyLetters + `0-9](?:[` + pyLetters + `0-9-]{0,61}[` + pyLetters + `0-9])?\.)+[` + pyLetters + `]{2,6}` +
	`|(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)` +
	`)`)

// containsURL is Plane's contains_url (plane/apps/api/plane/utils/url.py:
// 26-53), which it applies to first and last names: text of more than 1000
// characters is not looked at, and each line only up to its 500th
// character.
func containsURL(s string) bool {
	if utf8.RuneCountInString(s) > 1000 {
		return false
	}
	for line := range strings.SplitSeq(s, "\n") {
		if runes := []rune(line); len(runes) > 500 {
			line = string(runes[:500])
		}
		if urlPattern.MatchString(line) {
			return true
		}
	}
	return false
}
