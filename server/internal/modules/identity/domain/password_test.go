package domain

import (
	"slices"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

var rules = NewPasswordRules()

func checkCode(password, email string) string {
	if f := rules.Check("password", password, email); f != nil {
		return f.Code
	}
	return ""
}

func TestPasswordComposition(t *testing.T) {
	tests := []struct{ name, password, want string }{
		{"empty", "", shared.FieldRequired},
		{"7 characters", "Xq7!vbn", shared.FieldWeakPassword},
		{"8 characters", "Xq7!vbnz", ""},
		{"128 characters", "Xq7!" + strings.Repeat("vbnz", 31), ""},
		{"129 characters", "Xq7!" + strings.Repeat("vbnz", 31) + "k", shared.FieldWeakPassword},
		{"no upper-case letter", "xq7!vbnzk", shared.FieldWeakPassword},
		{"no lower-case letter", "XQ7!VBNZK", shared.FieldWeakPassword},
		{"no digit", "Xqa!vbnzk", shared.FieldWeakPassword},
		{"no special character", "Xq7avbnzk", shared.FieldWeakPassword},
		{"a special outside the set", "Xq7~vbnzk", shared.FieldWeakPassword},
		{"non-ASCII letters are not upper or lower case", "ÄÖ7!äöüß", shared.FieldWeakPassword},
		{"non-ASCII beside the classes", "Xq7!vbnzé", ""},
		// Length is UTF-16 code units, like password.length in the web app.
		{"two emoji make 8 units", "Xq7!😀😀", ""},
		{"one emoji makes 7 units", "Xq7!v😀", shared.FieldWeakPassword},
		{"64 emoji are 128 units", "Xq7!" + strings.Repeat("😀", 62), ""},
		{"65 emoji are 130 units", "Xq7!" + strings.Repeat("😀", 63), shared.FieldWeakPassword},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checkCode(tt.password, "someone@example.com"); got != tt.want {
				t.Errorf("Check(%q) = %q, want %q (%d UTF-16 units)", tt.password, got, tt.want, len(utf16.Encode([]rune(tt.password))))
			}
		})
	}
}

// M2 design 3.8 lists these: the common words that the composition rules
// let through, and the seven spellings that got past the second draft.
func TestCommonPasswords(t *testing.T) {
	common := []string{
		"Password1!", "Summer2024!", "Qwerty123!", "Welcome1!", "P@ssw0rd1", "Dragon#2026", "Zxcvbnm1!",
		"Password1!~", "Password1! ", "~Password1!", "Summer2024!`", `Welcome1!\`, "Qwerty123!~", "Dragon#2026 ",
	}
	for _, p := range common {
		if got := checkCode(p, "someone@example.com"); got != shared.FieldCommonPassword {
			t.Errorf("Check(%q) = %q, want common_password", p, got)
		}
	}
	for _, p := range []string{"Tr0ub4dor&3", "Correct-Horse-9", "Nerve2026!"} {
		if got := checkCode(p, "someone@example.com"); got != "" {
			t.Errorf("Check(%q) = %q, want accepted", p, got)
		}
	}
}

func TestPasswordWhoseCoreIsTheEmailsLocalPart(t *testing.T) {
	if got := checkCode("Liuwei123!", "liuwei@example.com"); got != shared.FieldCommonPassword {
		t.Errorf("Check(Liuwei123!, liuwei@…) = %q, want common_password", got)
	}
	if got := checkCode("Liuwei123!", "someone@example.com"); got != "" {
		t.Errorf("Check(Liuwei123!, someone@…) = %q, want accepted", got)
	}
	// Only the ends are trimmed: the dot inside stays in both cores.
	if got := checkCode("~Zhang.San9", "zhang.san@example.com"); got != shared.FieldCommonPassword {
		t.Errorf("Check(~Zhang.San9, zhang.san@…) = %q, want common_password", got)
	}
}

func TestCore(t *testing.T) {
	for in, want := range map[string]string{
		"Password1!~":  "password",
		"~Password1!":  "password",
		"Password1! ":  "password",
		"12!Pass-word": "pass-word",
		"Ünïcödé9!":    "ünïcödé",
		"2024!!":       "",
	} {
		if got := core(in); got != want {
			t.Errorf("core(%q) = %q, want %q", in, got, want)
		}
	}
}

// The list is what tools/password-blocklist/build.mjs writes: the header
// names the source and its licence, the entries are sorted, lowercase, and
// each meets one of the build filters. Checking the filters here keeps the
// JavaScript and Go core rules in step (M2 design 3.8).
func TestCommonPasswordList(t *testing.T) {
	header, _, _ := strings.Cut(commonPasswordsFile, "\n\n")
	for _, want := range []string{
		"SHA-256 c2e5696882c603b76bb67a47ee970897e5a76fc4c3f5547abe3d0ca340c576e0",
		"Contains public sector information licensed under the Open Government Licence v3.0:",
		"https://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/",
	} {
		if !strings.Contains(header, want) {
			t.Errorf("the list's header lacks %q", want)
		}
	}
	list := rules.common
	if len(list) != 33887 {
		t.Errorf("the list has %d entries, want 33887", len(list))
	}
	if !slices.IsSorted(list) || len(slices.Compact(slices.Clone(list))) != len(list) {
		t.Error("the list is not sorted by bytes without duplicates")
	}
	for _, e := range list {
		units := len(utf16.Encode([]rune(e)))
		inFull := units >= 8 && units <= 128 && asciiLetters(e) >= 2 &&
			strings.ContainsAny(e, "0123456789") && strings.ContainsAny(e, passwordSpecials)
		asCore := e != "" && units <= 126 && core(e) == e && asciiLetters(e) >= 2
		if kept := inFull || asCore; strings.ToLower(e) != e || !kept {
			t.Errorf("entry %q is not lowercase or meets no build filter", e)
		}
	}
}

func asciiLetters(s string) int {
	n := 0
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			n++
		}
	}
	return n
}
