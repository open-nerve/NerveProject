package archtest

import (
	"maps"
	"slices"
	"strings"
	"testing"
)

// reachesInfrastructure marks what the pure layers (domain, app,
// internal/shared) must not depend on, even indirectly: net/http,
// database/sql and any module other than this one.
func reachesInfrastructure(path string) bool {
	_, inModule := local(path)
	return !inModule && isInfrastructure(path)
}

func isPure(path string) bool {
	_, layer, ok := moduleOf(path)
	return ok && (layer == "domain" || layer == "app") || inModuleDir(path, "internal/shared")
}

// Rules 2 and 10 judge direct imports only, and some standard library
// packages import net/http themselves (expvar, net/rpc): domain -> expvar
// passes both rules yet links net/http. This walks every dependency instead.
func TestPureLayersReachNoInfrastructure(t *testing.T) {
	g := loadDeps(t, "./...")
	pure := 0
	for _, pkg := range slices.Sorted(maps.Keys(g)) {
		if !isPure(pkg) {
			continue
		}
		pure++
		for _, b := range bannedImports(g, pkg, reachesInfrastructure) {
			t.Errorf("%s must not depend on %s, imported via %s", rel(pkg), b.banned(), b.via())
		}
	}
	// A loader problem must not pass as "nothing reached".
	if pure == 0 {
		t.Fatalf("dependency graph holds no domain, app or internal/shared package; loaded %d packages", len(g))
	}
}

// Infrastructure counts however it is reached: through the standard library
// or through this module's own packages, which are not infrastructure
// themselves.
func TestReachesInfrastructure(t *testing.T) {
	domain := m("internal/modules/issue/domain")
	g := graph{
		domain:                  {"expvar", "fmt", m("internal/shared/id")},
		"expvar":                {"net/http"},
		"fmt":                   {"strconv"},
		m("internal/shared/id"): {"github.com/jackc/pgx/v5/pgtype"},
	}
	var got []string
	for _, b := range bannedImports(g, domain, reachesInfrastructure) {
		got = append(got, b.via())
	}
	want := []string{
		"internal/modules/issue/domain → internal/shared/id → github.com/jackc/pgx/v5/pgtype",
		"internal/modules/issue/domain → expvar → net/http",
	}
	if !slices.Equal(got, want) {
		t.Errorf("bannedImports() =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}
