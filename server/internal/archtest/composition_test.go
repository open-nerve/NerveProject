package archtest

import (
	"slices"
	"strings"
	"testing"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/static"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// The command line's composition, bootstrap.Users, is a pool and identity's
// administrator use cases (M2 design 3.17): nothing it calls builds the HTTP
// side, a rate limiter or a jobs client. The rule follows the static calls
// from bootstrap.Users; the commands are func values it calls dynamically,
// so they are not followed: they only receive the composition. Reaching
// identity.NewAdmin shows the walk sees the composition at all.
func TestUsersComposeNoServerAndNoJobs(t *testing.T) {
	registerSources(t)
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedDeps |
			packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedTypesSizes,
		Dir: moduleRoot,
	}
	pkgs, err := packages.Load(cfg, "./internal/bootstrap")
	if err != nil {
		t.Fatalf("load packages: %v", err)
	}
	if packages.PrintErrors(pkgs) > 0 {
		t.Fatal("the packages have errors")
	}
	prog, built := ssautil.AllPackages(pkgs, 0)
	prog.Build()
	users := built[0].Func("Users")
	if users == nil {
		t.Fatal("bootstrap.Users not found: the rule checks nothing")
	}
	reached, banned := walkCalls(static.CallGraph(prog), users, composesMore)
	if !slices.ContainsFunc(reached, func(chain []*ssa.Function) bool {
		return chain[len(chain)-1].String() == m("internal/modules/identity")+".NewAdmin"
	}) {
		var names []string
		for _, chain := range reached {
			names = append(names, funcName(chain[len(chain)-1]))
		}
		t.Fatalf("bootstrap.Users does not reach identity.NewAdmin, so the rule checks nothing; it reaches:\n%s",
			strings.Join(names, "\n"))
	}
	for _, chain := range banned {
		names := make([]string, len(chain))
		for i, f := range chain {
			names[i] = funcName(f)
		}
		t.Errorf("bootstrap.Users builds more than the pool and NewAdmin: %s", strings.Join(names, " → "))
	}
}

// composesMore reports whether f builds what the command line must not: the
// HTTP side (identity.New, platform/httpserver), a rate limiter or a jobs
// client (platform/jobs, River).
func composesMore(f *ssa.Function) bool {
	if f.Pkg == nil {
		return false
	}
	path := f.Pkg.Pkg.Path()
	if path == m("internal/modules/identity") && f.Name() == "New" {
		return true
	}
	return slices.ContainsFunc([]string{
		m("internal/platform/httpserver"), m("internal/platform/ratelimit"), m("internal/platform/jobs"), "github.com/riverqueue/river",
	}, func(dir string) bool { return within(path, dir) })
}

// walkCalls follows the static calls from root, breadth first. It returns
// the call chain to every function reached, and the chain to each call of a
// function stop accepts, whose own calls it does not follow.
func walkCalls(graph *callgraph.Graph, root *ssa.Function, stop func(*ssa.Function) bool) (reached, stopped [][]*ssa.Function) {
	start := graph.Nodes[root]
	chains := map[*callgraph.Node][]*ssa.Function{start: {root}}
	for queue := []*callgraph.Node{start}; len(queue) > 0; queue = queue[1:] {
		caller := queue[0]
		for _, e := range caller.Out {
			chain := append(slices.Clip(chains[caller]), e.Callee.Func)
			if stop(e.Callee.Func) {
				stopped = append(stopped, chain)
				continue
			}
			if _, seen := chains[e.Callee]; !seen {
				chains[e.Callee] = chain
				reached = append(reached, chain)
				queue = append(queue, e.Callee)
			}
		}
	}
	return reached, stopped
}

// funcName is f's full name, this module's paths shortened.
func funcName(f *ssa.Function) string {
	return strings.ReplaceAll(f.String(), modulePath+"/", "")
}
