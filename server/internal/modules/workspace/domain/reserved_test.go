package domain

import (
	"slices"
	"strings"
	"testing"
)

// The list parses into three sections that do not overlap, every name is
// spelled as a slug, and the held names are exactly the four of M3 design
// 3.10 (9.1). Which names the app and the server sections hold is checked
// against the routes: the server's by bootstrap, the app's by the web app.
func TestTheReservedListIsWellFormed(t *testing.T) {
	r := Reserved()
	for name, section := range map[string][]string{"app": r.App, "server": r.Server, "reserved": r.Reserved} {
		if len(section) == 0 {
			t.Errorf("section [%s] is empty", name)
		}
	}
	seen := map[string]bool{}
	for _, slug := range r.All() {
		if seen[slug] {
			t.Errorf("%q is listed twice", slug)
		}
		seen[slug] = true
		if !slugPattern.MatchString(slug) || len(slug) > maxSlugLength {
			t.Errorf("%q is not spelled as a slug", slug)
		}
		if CheckSlug(slug) != SlugReserved {
			t.Errorf("CheckSlug(%q) = %q, want reserved", slug, CheckSlug(slug))
		}
	}
	if want := []string{"admin", "docs", "help", "static"}; !slices.Equal(r.Reserved, want) {
		t.Errorf("section [reserved] = %q, want %q", r.Reserved, want)
	}
}

func TestParseReserved(t *testing.T) {
	got, err := parseReserved("# a comment\n\n[server]\napi\n  healthz  \n[app]\nsign-up\n# another\n[reserved]\nhelp\n[app]\nicons\n")
	want := ReservedSlugs{App: []string{"sign-up", "icons"}, Server: []string{"api", "healthz"}, Reserved: []string{"help"}}
	if err != nil || !slices.Equal(got.App, want.App) || !slices.Equal(got.Server, want.Server) || !slices.Equal(got.Reserved, want.Reserved) {
		t.Errorf("parseReserved() = %+v, %v; want %+v", got, err, want)
	}
	// Each refusal, on a list that is otherwise whole.
	const rest = "[app]\nsign-up\n[server]\napi\n[reserved]\nhelp\n"
	for _, tt := range []struct{ name, text, want string }{
		{"a name before every section", "api\n" + rest, `line 1: "api" is in no section`},
		{"an unknown header", rest + "[apps]\nicons\n", `line 7: "[apps]" is not [app], [server] or [reserved]`},
		{"a header in capitals", "[APP]\n" + rest, `line 1: "[APP]" is not [app], [server] or [reserved]`},
		{"a name with a comment after it", rest + "docs # later\n", `line 7: "docs # later" is not spelled as a slug`},
		{"a name in capitals", rest + "Docs\n", `line 7: "Docs" is not spelled as a slug`},
		{"a name of 49 characters", rest + strings.Repeat("x", 49) + "\n", `line 7: "` + strings.Repeat("x", 49) + `" is not spelled as a slug`},
		{"an empty section", "[app]\nsign-up\n[server]\n[reserved]\nhelp\n", "section [server] is empty"},
		{"a missing section", "[app]\nsign-up\n[server]\napi\n", "section [reserved] is empty"},
		{"a name twice in one section", rest + "help\n", `line 7: "help" is listed twice`},
		{"a name in two sections", rest + "[app]\napi\n", `line 8: "api" is listed twice`},
	} {
		if _, err := parseReserved(tt.text); err == nil || err.Error() != tt.want {
			t.Errorf("%s: parseReserved() = %v, want %q", tt.name, err, tt.want)
		}
	}
	// A name of 48 characters is a slug: the limit is not one less.
	if _, err := parseReserved(rest + strings.Repeat("x", 48) + "\n"); err != nil {
		t.Errorf("a name of 48 characters: %v, want it listed", err)
	}
}

// Reserved returns a copy: a caller that changes it changes no answer.
func TestReservedReturnsACopy(t *testing.T) {
	want := Reserved().All()
	r := Reserved()
	for _, section := range [][]string{r.App, r.Server, r.Reserved} {
		section[0] = "not-reserved"
	}
	if got := Reserved().All(); !slices.Equal(got, want) || CheckSlug("not-reserved") != "" {
		t.Errorf("changing Reserved()'s result changed the list: %q, want %q", got, want)
	}
}
