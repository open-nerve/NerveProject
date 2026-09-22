package archtest

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
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

// loadDeps reads the import graph of cmd/nerve and every package it depends
// on, test files excluded.
func loadDeps(t *testing.T) graph {
	t.Helper()
	registerSources(t)
	cfg := &packages.Config{Mode: packages.NeedName | packages.NeedImports | packages.NeedDeps, Dir: moduleRoot}
	pkgs, err := packages.Load(cfg, "./cmd/nerve")
	if err != nil {
		t.Fatalf("load packages: %v", err)
	}
	g := graph{}
	packages.Visit(pkgs, nil, func(p *packages.Package) {
		for _, e := range p.Errors {
			t.Errorf("load %s: %v", p.PkgPath, e)
		}
		g[p.PkgPath] = slices.Sorted(maps.Keys(p.Imports))
	})
	return g
}

func TestNerveBinaryLinksNoBannedModule(t *testing.T) {
	root := m("cmd/nerve")
	g := loadDeps(t)
	// A loader problem must not pass as "nothing banned".
	for _, want := range []string{root, m("internal/bootstrap"), "net/http"} {
		if _, ok := g[want]; !ok {
			t.Fatalf("dependency graph lacks %s; loaded %d packages", want, len(g))
		}
	}
	for _, b := range bannedImports(g, root, bannedFromBinary()) {
		t.Error(b)
	}
}

// bannedImport is a banned package and the import chain that reaches it.
type bannedImport struct {
	chain []string // from the root to the banned package
}

func (b bannedImport) String() string {
	hops := make([]string, len(b.chain))
	for i, p := range b.chain {
		hops[i] = rel(p)
	}
	return fmt.Sprintf("the nerve binary must not link %s, imported via %s", b.chain[len(b.chain)-1], strings.Join(hops, " → "))
}

// bannedImports walks g from root and returns every banned package that an
// allowed package imports, each with the shortest chain to it. The imports
// of a banned package are not followed: it has to go anyway.
func bannedImports(g graph, root string, banned []string) []bannedImport {
	isBanned := func(path string) bool {
		return slices.ContainsFunc(banned, func(prefix string) bool { return strings.HasPrefix(path, prefix) })
	}
	importer := map[string]string{root: ""}
	var found []bannedImport
	for queue := []string{root}; len(queue) > 0; queue = queue[1:] {
		pkg := queue[0]
		if isBanned(pkg) {
			var chain []string
			for p := pkg; p != ""; p = importer[p] {
				chain = append(chain, p)
			}
			slices.Reverse(chain)
			found = append(found, bannedImport{chain})
			continue
		}
		for _, dep := range g[pkg] {
			if _, seen := importer[dep]; !seen {
				importer[dep] = pkg
				queue = append(queue, dep)
			}
		}
	}
	slices.SortFunc(found, func(a, b bannedImport) int {
		return strings.Compare(a.chain[len(a.chain)-1], b.chain[len(b.chain)-1])
	})
	return found
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
	for _, b := range bannedImports(g, root, bannedFromBinary()) {
		got = append(got, b.String())
	}
	want := []string{
		"the nerve binary must not link github.com/getkin/kin-openapi/openapi3, imported via " +
			"cmd/nerve → internal/bootstrap → internal/platform/httpserver → github.com/getkin/kin-openapi/openapi3",
		"the nerve binary must not link github.com/google/uuid, imported via " +
			"cmd/nerve → internal/bootstrap → internal/modules/issue/adapter/http/gen → github.com/google/uuid",
	}
	if !slices.Equal(got, want) {
		t.Errorf("bannedImports() =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}
