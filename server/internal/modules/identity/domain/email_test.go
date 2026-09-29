package domain

import "testing"

func TestDisplayNameFromEmail(t *testing.T) {
	for in, want := range map[string]string{
		"alice@corp.com":          "alice",
		`"a@b"@example.com`:       `"a`, // Plane's email.split("@")[0]
		"élodie@exämple.com":      "élodie",
		"first.last+tag@corp.com": "first.last+tag",
	} {
		if got := DisplayNameFromEmail(in); got != want {
			t.Errorf("DisplayNameFromEmail(%q) = %q, want %q", in, got, want)
		}
	}
}
