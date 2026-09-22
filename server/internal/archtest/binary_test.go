package archtest

import (
	"slices"
	"strings"
	"testing"
)

// bannedFromBinary lists the import path prefixes that must not reach the
// nerve binary: the test-only kin-openapi (apitest), testcontainers and
// docker (pgtest), and google/uuid, which the standard library's uuid
// replaces. Rule 8 keeps the test helpers themselves out, but generated code
// (an embedded spec, an unmapped format: uuid) or any other import could
// still pull these in, and depguard does not look at generated files.
func bannedFromBinary() []string {
	return []string{
		"github.com/getkin/kin-openapi",
		"github.com/testcontainers/",
		"github.com/google/uuid",
		"github.com/docker/",
	}
}

func isBannedFromBinary(path string) bool {
	return slices.ContainsFunc(bannedFromBinary(), func(prefix string) bool { return strings.HasPrefix(path, prefix) })
}

func TestNerveBinaryLinksNoBannedModule(t *testing.T) {
	root := m("cmd/nerve")
	g := loadDeps(t, "./cmd/nerve")
	// A loader problem must not pass as "nothing banned".
	for _, want := range []string{root, m("internal/bootstrap"), "net/http"} {
		if _, ok := g[want]; !ok {
			t.Fatalf("dependency graph lacks %s; loaded %d packages", want, len(g))
		}
	}
	for _, b := range bannedImports(g, root, isBannedFromBinary) {
		t.Errorf("the nerve binary must not link %s, imported via %s", b.banned(), b.via())
	}
}

func TestBannedImports(t *testing.T) {
	root := m("cmd/nerve")
	g := graph{
		root:                                         {"github.com/spf13/cobra", m("internal/bootstrap")},
		"github.com/spf13/cobra":                     {"github.com/spf13/pflag"},
		m("internal/bootstrap"):                      {m("internal/modules/issue/adapter/http/gen"), m("internal/platform/httpserver")},
		m("internal/platform/httpserver"):            {"github.com/getkin/kin-openapi/openapi3", "net/http"},
		"github.com/getkin/kin-openapi/openapi3":     {"github.com/google/uuid"},
		m("internal/modules/issue/adapter/http/gen"): {"github.com/google/uuid"},
		// Not reachable from the root: test helpers stay out of the walk.
		m("internal/platform/postgres/pgtest"): {"github.com/testcontainers/testcontainers-go"},
	}
	var got []string
	for _, b := range bannedImports(g, root, isBannedFromBinary) {
		got = append(got, b.via())
	}
	want := []string{
		"cmd/nerve → internal/bootstrap → internal/platform/httpserver → github.com/getkin/kin-openapi/openapi3",
		"cmd/nerve → internal/bootstrap → internal/modules/issue/adapter/http/gen → github.com/google/uuid",
	}
	if !slices.Equal(got, want) {
		t.Errorf("bannedImports() =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}
