package domain

import (
	"strings"
	"testing"
)

// The expectations are what Plane's contains_url answers for the same
// strings (Python 3, plane/apps/api/plane/utils/url.py).
func TestContainsURLAsPlane(t *testing.T) {
	a := strings.Repeat
	tests := []struct {
		name, s string
		want    bool
	}{
		{"https address", "https://x", true},
		{"scheme only", "http://", false},
		{"scheme then a no-break space", "http://\u00a0x", false},
		{"scheme then a vertical tab", "http://\vx", false},
		{"scheme then an information separator", "http://\x1cx", false},
		{"scheme then a next line", "http://\u0085x", false},
		{"scheme then an Ogham space", "http://\u1680x", false},
		{"scheme then an en quad", "http://\u2000x", false},
		{"scheme then a hair space", "http://\u200ax", false},
		{"scheme then a line separator", "http://\u2028x", false},
		{"scheme then a paragraph separator", "http://\u2029x", false},
		{"scheme then a narrow no-break space", "http://\u202fx", false},
		{"scheme then a medium mathematical space", "http://\u205fx", false},
		{"scheme then an ideographic space", "http://\u3000x", false},
		{"scheme then a zero-width space", "http://\u200bx", true},
		{"upper case", "HTTPS://X", true},
		{"ftp", "ftp://x", false},
		{"dotted name", "John.Smith", true},
		{"plain name", "Ann", false},
		{"hyphen", "Mary-Ann", false},
		{"apostrophe", "O'Brien", false},
		{"www host", "www.x", true},
		{"upper-case www", "WWW.Example", true},
		{"one-letter TLD", "a.b", false},
		{"two-letter TLD", "a.bc", true},
		{"label ending in a hyphen", "a-.bc", false},
		{"TLD with a hyphen", "a.b-c", false},
		{"leading hyphen", "-a.bc", true},
		{"IPv4", "1.2.3.4", true},
		{"IPv4 inside a longer number", "999.1.1.1", true},
		{"decimal", "3.14", false},
		{"three numbers", "1.2.3", false},
		{"dotted capital I", "\u0130.ab", true},
		{"dotless i", "\u0131.ab", true},
		{"long s", "\u017f.ab", true},
		{"Kelvin sign", "\u212a.ab", true},
		{"e acute", "\u00e9.ab", false},
		{"letters before an address", "\u00e9l\u00e8ve.com", true},
		{"after an Ogham space", "\u1680http://x", true},
		{"second line", "x\nhttps://y", true},
		{"after a line separator", "x\u2028https://y", true},
		{"no-break space after it", "x.co\u00a0", true},
		{"address ending at character 500", a("a", 490) + " https://x", true},
		{"address cut at character 500", a("a", 491) + " https://x", false},
		{"1000 characters", "x.io\n" + a("a", 995), true},
		{"1001 characters", "x.io\n" + a("a", 996), false},
		{"1000 characters, not bytes", "x.io\n" + a("\u00e9", 995), true},
		{"address ending at character 500, not byte 500", a("\u00e9", 490) + " https://x", true},
		{"address cut at character 500, not byte 500", a("\u00e9", 491) + " https://x", false},
		{"address within character 500, beyond byte 500", a("\u00e9", 300) + " https://x" + a("a", 300), true},
		{"a long first line, an address on the second", a("a", 450) + "\n" + a("b", 60) + " x.io", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := containsURL(tt.s); got != tt.want {
				t.Errorf("containsURL(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}
