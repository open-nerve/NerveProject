// Package archtest enforces the architecture rules of M0 design 3.7 (and the
// platform rules of 3.1) as tests. Each rule is a pure predicate over one
// import edge, so rules are unit-tested on synthetic edges and then applied
// to the real import graph of the module.
package archtest

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

const modulePath = "github.com/open-nerve/NerveProject/server"

// graph maps each package of the module to the import paths it imports.
// Test files are not part of it.
type graph map[string][]string

type rule struct {
	name      string
	forbidden func(from, to string) bool
}

type violation struct {
	from, to, rule string
}

func (v violation) String() string {
	return fmt.Sprintf("%s imports %s: %s", rel(v.from), rel(v.to), v.rule)
}

func rules() []rule {
	return []rule{
		{"module layers point inward: adapter -> app -> domain", layersPointInward},
		{"domain imports only the standard library (not net/http or database/sql), its own module and internal/shared", domainIsPure},
		{"modules do not import each other", modulesAreIsolated},
		{"platform does not import modules or bootstrap", platformIsBusinessFree},
		{"only bootstrap imports modules", onlyBootstrapImportsModules},
		{"generated code is imported only by its module's http adapter", generatedCodeStaysInAdapter},
		{"platform packages do not import each other, except config", platformPackagesAreIndependent},
		{"pgtest is imported only by tests", pgtestOnlyInTests},
	}
}

// check applies every rule to every edge of g, in a stable order.
func check(g graph) []violation {
	var found []violation
	for _, from := range slices.Sorted(maps.Keys(g)) {
		for _, to := range g[from] {
			for _, r := range rules() {
				if r.forbidden(from, to) {
					found = append(found, violation{from, to, r.name})
				}
			}
		}
	}
	return found
}

// local returns the module-relative path of a package of this module.
func local(path string) (string, bool) {
	return strings.CutPrefix(path, modulePath+"/")
}

func rel(path string) string {
	if r, ok := local(path); ok {
		return r
	}
	return path
}

// within reports whether path is dir or inside it.
func within(path, dir string) bool {
	return path == dir || strings.HasPrefix(path, dir+"/")
}

// moduleOf splits internal/modules/<name>/<layer>/... of this module.
func moduleOf(path string) (name, layer string, ok bool) {
	r, ok := local(path)
	if !ok {
		return "", "", false
	}
	rest, ok := strings.CutPrefix(r, "internal/modules/")
	if !ok {
		return "", "", false
	}
	name, sub, _ := strings.Cut(rest, "/")
	layer, _, _ = strings.Cut(sub, "/")
	return name, layer, true
}

// platformOf returns the platform package a path belongs to, e.g. "postgres"
// for internal/platform/postgres/pgtest.
func platformOf(path string) (string, bool) {
	r, ok := local(path)
	if !ok {
		return "", false
	}
	rest, ok := strings.CutPrefix(r, "internal/platform/")
	if !ok {
		return "", false
	}
	name, _, _ := strings.Cut(rest, "/")
	return name, true
}

func isStdlib(path string) bool {
	first, _, _ := strings.Cut(path, "/")
	return !strings.Contains(first, ".")
}

func inModuleDir(path, dir string) bool {
	r, ok := local(path)
	return ok && within(r, dir)
}

// layerRank orders the layers of a module from the inside out; "" is the
// module root (module.go).
func layerRank(layer string) (int, bool) {
	switch layer {
	case "domain":
		return 0, true
	case "app":
		return 1, true
	case "adapter":
		return 2, true
	case "":
		return 3, true
	}
	return 0, false
}

func layersPointInward(from, to string) bool {
	fm, fl, ok := moduleOf(from)
	if !ok {
		return false
	}
	tm, tl, ok := moduleOf(to)
	if !ok || fm != tm {
		return false
	}
	fromRank, fromKnown := layerRank(fl)
	toRank, toKnown := layerRank(tl)
	return fromKnown && toKnown && toRank > fromRank
}

func domainIsPure(from, to string) bool {
	if _, layer, ok := moduleOf(from); !ok || layer != "domain" {
		return false
	}
	if r, ok := local(to); ok {
		_, _, inModule := moduleOf(to) // module imports are covered by the layer and isolation rules
		return !inModule && !within(r, "internal/shared")
	}
	if !isStdlib(to) {
		return true
	}
	return within(to, "net/http") || within(to, "database/sql")
}

func modulesAreIsolated(from, to string) bool {
	fm, _, ok := moduleOf(from)
	if !ok {
		return false
	}
	tm, _, ok := moduleOf(to)
	return ok && fm != tm
}

func platformIsBusinessFree(from, to string) bool {
	return inModuleDir(from, "internal/platform") &&
		(inModuleDir(to, "internal/modules") || inModuleDir(to, "internal/bootstrap"))
}

func onlyBootstrapImportsModules(from, to string) bool {
	return inModuleDir(to, "internal/modules") &&
		!inModuleDir(from, "internal/modules") && !inModuleDir(from, "internal/bootstrap")
}

func generatedCodeStaysInAdapter(from, to string) bool {
	name, _, ok := moduleOf(to)
	if !ok {
		return false
	}
	adapter := "internal/modules/" + name + "/adapter/http"
	gen := adapter + "/gen"
	if !inModuleDir(to, gen) {
		return false
	}
	r, _ := local(from)
	return r != adapter && !within(r, gen)
}

func platformPackagesAreIndependent(from, to string) bool {
	fp, ok := platformOf(from)
	if !ok {
		return false
	}
	tp, ok := platformOf(to)
	return ok && tp != fp && tp != "config"
}

func pgtestOnlyInTests(_, to string) bool {
	// The graph holds no test files, so any importer is production code.
	return inModuleDir(to, "internal/platform/postgres/pgtest")
}
