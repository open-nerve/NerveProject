package domain

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// Widening complete identifiers, accepting non-ASCII or overflowing sequence
// numbers, or losing literal text must change these observable search inputs.
func TestSearchIdentifierAndNumeric(t *testing.T) {
	for _, tc := range []struct {
		text, identifier string
		number, sequence int
	}{
		{"WEB-12", "WEB", 12, 0},
		{"WEB-2147483647", "WEB", 2147483647, 0},
		{"A-B-3", "A-B", 3, 0},
		{"WEB--12", "WEB-", 12, 0},
		{"WEB-012", "", 0, 0},
		{"WEB-2147483648", "", 0, 0},
		{"WEB-0", "", 0, 0},
		{"12", "", 0, 12},
		{"00012", "", 0, 12},
		{"2147483647", "", 0, 2147483647},
		{"0", "", 0, 0},
		{"000", "", 0, 0},
		{"2147483648", "", 0, 0},
		{strings.Repeat("9", 100), "", 0, 0},
		{"+12", "", 0, 0},
		{"-12", "", 0, 0},
		{"١٢", "", 0, 0},
		{"１２", "", 0, 0},
		{"12x", "", 0, 0},
		{"100% done", "", 0, 0},
		{"under_score", "", 0, 0},
		{`a\b`, "", 0, 0},
	} {
		t.Run(tc.text, func(t *testing.T) {
			got, err := ParseSearchQuery(map[string][]string{"q": {" \t" + tc.text + "\n"}})
			if err != nil {
				t.Fatal(err)
			}
			if got.Text != tc.text || got.Identifier != tc.identifier {
				t.Fatalf("text/identifier = %q/%q; want %q/%q", got.Text, got.Identifier, tc.text, tc.identifier)
			}
			assertSearchNumber(t, "identifier", got.IdentifierNumber, tc.number)
			assertSearchNumber(t, "sequence", got.SequenceNumber, tc.sequence)
		})
	}
	t.Run("fold_fixture", func(t *testing.T) {
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
		for _, sample := range samples.Cases {
			t.Run(sample.Name, func(t *testing.T) {
				got, err := ParseSearchQuery(map[string][]string{"q": {sample.Asked + "-12"}})
				if err != nil {
					t.Fatal(err)
				}
				if got.Identifier != sample.Asked || got.Text != sample.Asked+"-12" {
					t.Fatalf("query = %+v", got)
				}
				assertSearchNumber(t, "identifier", got.IdentifierNumber, 12)
				index, ok := MatchIdentifier(got.Identifier, sample.Identifiers)
				if sample.Want == nil {
					if ok {
						t.Fatalf("matched %s; want no match", sample.Identifiers[index])
					}
				} else if !ok || sample.Identifiers[index] != *sample.Want {
					t.Fatalf("match = %d, %v; want %s", index, ok, *sample.Want)
				}
			})
		}
	})
}

func assertSearchNumber(t *testing.T, name string, got *int, want int) {
	t.Helper()
	if want == 0 {
		if got != nil {
			t.Fatalf("%s = %d; want no numeric branch", name, *got)
		}
	} else if got == nil || *got != want {
		t.Fatalf("%s = %v; want %d", name, got, want)
	}
}
