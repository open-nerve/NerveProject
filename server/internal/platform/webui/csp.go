package webui

import (
	"crypto/sha256"
	"encoding/base64"
	"regexp"
	"slices"
	"strings"
)

// Every script element of a page, with its attributes and its text. A script's
// text is raw: it ends at the first "</script", as the browser ends it, and
// React Router escapes "<" in the data it writes inline.
var (
	scriptElement = regexp.MustCompile(`(?is)<script\b([^>]*)>(.*?)</script\s*>`)
	srcAttribute  = regexp.MustCompile(`(?i)(?:^|\s)src\s*=`)
)

// contentSecurityPolicy is the CSP of the app's pages (M2 design 8.3), for
// the page index: scripts come from nerve only, or are the page's own inline
// scripts, each allowed by the SHA-256 of its text, so a rebuilt frontend
// needs no change here. Styles may be inline (React's style attributes and
// popper's positions); nothing else is allowed from anywhere but nerve.
func contentSecurityPolicy(index []byte) string {
	scripts := []string{"'self'"}
	for _, m := range scriptElement.FindAllSubmatch(index, -1) {
		if srcAttribute.Match(m[1]) {
			continue // a script file: 'self' covers it
		}
		sum := sha256.Sum256(m[2])
		source := "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
		if !slices.Contains(scripts, source) {
			scripts = append(scripts, source)
		}
	}
	return strings.Join([]string{
		"default-src 'self'",
		"script-src " + strings.Join(scripts, " "),
		"style-src 'self' 'unsafe-inline'",
		"img-src 'self' data: blob:",
		"font-src 'self' data:",
		"connect-src 'self'",
		"object-src 'none'",
		"base-uri 'none'",
		"form-action 'self'",
		"frame-ancestors 'none'",
	}, "; ")
}
