package domain

import (
	"encoding/json"
	"os"
	"testing"
)

// Each case of api/testdata/identifier-fold.json (M4 design 3.7): the
// project's identifier asked finds the project it says, or none.
func TestMatchIdentifierFindsWhatTheSamplesSay(t *testing.T) {
	data, err := os.ReadFile("../../../../../api/testdata/identifier-fold.json")
	if err != nil {
		t.Fatal(err)
	}
	var samples struct {
		Cases []struct {
			Name        string   `json:"name"`
			Identifiers []string `json:"identifiers"`
			Asked       string   `json:"asked"`
			Want        *string  `json:"want"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(data, &samples); err != nil {
		t.Fatal(err)
	}
	if len(samples.Cases) == 0 {
		t.Fatal("identifier-fold.json has no case")
	}
	for _, c := range samples.Cases {
		i, ok := MatchIdentifier(c.Asked, c.Identifiers)
		switch {
		case c.Want == nil && ok:
			t.Errorf("%s: MatchIdentifier(%q, %q) = %s, want none", c.Name, c.Asked, c.Identifiers, c.Identifiers[i])
		case c.Want != nil && (!ok || c.Identifiers[i] != *c.Want):
			t.Errorf("%s: MatchIdentifier(%q, %q) = %d, %v; want %s", c.Name, c.Asked, c.Identifiers, i, ok, *c.Want)
		}
	}
}

// A work item's identifier is its project's identifier, a hyphen and its
// number, split at the last hyphen; the number is decimal digits without a
// sign or a leading zero, from 1 to the greatest number a work item may
// have. Anything else is no identifier. Identifier writes each identifier
// it reads as it was.
func TestParseIdentifier(t *testing.T) {
	for _, tt := range []struct {
		s       string
		project string
		number  int
		ok      bool
	}{
		{"WEB-12", "WEB", 12, true},
		{"web-1", "web", 1, true},
		{"İŞ-7", "İŞ", 7, true},
		{"A-B-3", "A-B", 3, true},
		{"WEB-2147483647", "WEB", 2147483647, true},
		{"WEB-2147483648", "", 0, false},
		{"WEB-0", "", 0, false},
		{"WEB-012", "", 0, false},
		{"WEB-+12", "", 0, false},
		{"WEB--12", "WEB-", 12, true},
		{"WEB-1x", "", 0, false},
		{"WEB-", "", 0, false},
		{"-12", "", 0, false},
		{"WEB12", "", 0, false},
		{"WEB-١٢", "", 0, false},
		{"", "", 0, false},
	} {
		project, number, ok := ParseIdentifier(tt.s)
		if project != tt.project || number != tt.number || ok != tt.ok {
			t.Errorf("ParseIdentifier(%q) = %q, %d, %v; want %q, %d, %v", tt.s, project, number, ok, tt.project, tt.number, tt.ok)
		}
		if ok && Identifier(project, number) != tt.s {
			t.Errorf("Identifier(%q, %d) = %q, want %q", project, number, Identifier(project, number), tt.s)
		}
	}
}
