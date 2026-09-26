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
// (an embedded spec) or any other import could still pull these in, and
// depguard does not look at generated files.
func bannedFromBinary() []string {
	return []string{
		"github.com/getkin/kin-openapi",
		"github.com/testcontainers/",
		"github.com/google/uuid",
		"github.com/docker/",
	}
}

// oapiRuntime is the module that generated code imports to bind parameters.
// It imports google/uuid itself (runtime/types/uuid.go and the runtime
// package's styleparam.go, v1.7.0), so its packages, and only they, may
// (M2 design 3.12). That generated code uses the standard library's uuid is
// checked directly, by TestGeneratedCodeUsesTheStandardUUID.
const oapiRuntime = "github.com/oapi-codegen/runtime"

// isBannedFromBinary judges the import of path by importer.
func isBannedFromBinary(importer, path string) bool {
	if strings.HasPrefix(path, "github.com/google/uuid") && (importer == oapiRuntime || strings.HasPrefix(importer, oapiRuntime+"/")) {
		return false
	}
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

// The binary embeds the time zone database (M2 design 4.2): which
// user_timezone is accepted, and the offsets of GET /api/v0/timezones, must
// not depend on the zones of the host, which a container may lack.
func TestNerveBinaryEmbedsTheTimeZoneDatabase(t *testing.T) {
	g := loadDeps(t, "./cmd/nerve")
	if !slices.Contains(g[m("cmd/nerve")], "time/tzdata") {
		t.Errorf("cmd/nerve imports %q, not time/tzdata", g[m("cmd/nerve")])
	}
}

// google/uuid is reached first through oapi-codegen/runtime, which may
// import it; every other import of it is still reported, each on its own.
func TestBannedImports(t *testing.T) {
	root := m("cmd/nerve")
	gen := m("internal/modules/issue/adapter/http/gen")
	g := graph{
		root:                                        {"github.com/spf13/cobra", m("internal/bootstrap")},
		"github.com/spf13/cobra":                    {"github.com/spf13/pflag"},
		m("internal/bootstrap"):                     {gen, m("internal/platform/httpserver")},
		gen:                                         {"github.com/oapi-codegen/runtime", "github.com/oapi-codegen/runtime-extra", "net/http"},
		"github.com/oapi-codegen/runtime":           {"github.com/google/uuid", "github.com/oapi-codegen/runtime/types"},
		"github.com/oapi-codegen/runtime/types":     {"github.com/google/uuid"},
		"github.com/oapi-codegen/runtime-extra":     {"github.com/google/uuid"},
		m("internal/platform/httpserver"):           {"github.com/getkin/kin-openapi/openapi3", m("internal/platform/httpserver/bodyshape"), "net/http"},
		"github.com/getkin/kin-openapi/openapi3":    {"github.com/google/uuid"},
		m("internal/platform/httpserver/bodyshape"): {"github.com/google/uuid"},
		// Not reachable from the root: test helpers stay out of the walk.
		m("internal/platform/postgres/pgtest"): {"github.com/testcontainers/testcontainers-go"},
	}
	var got []string
	for _, b := range bannedImports(g, root, isBannedFromBinary) {
		got = append(got, b.via())
	}
	want := []string{
		"cmd/nerve → internal/bootstrap → internal/platform/httpserver → github.com/getkin/kin-openapi/openapi3",
		"cmd/nerve → internal/bootstrap → internal/modules/issue/adapter/http/gen → github.com/oapi-codegen/runtime-extra → github.com/google/uuid",
		"cmd/nerve → internal/bootstrap → internal/platform/httpserver → internal/platform/httpserver/bodyshape → github.com/google/uuid",
	}
	if !slices.Equal(got, want) {
		t.Errorf("bannedImports() =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}
