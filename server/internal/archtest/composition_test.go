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

// The command line's compositions, bootstrap.Users and bootstrap.Workspaces,
// are a pool and the modules' administrator use cases (M2 design 3.17, M3
// design 6.6): nothing they call builds a module's HTTP side or the
// Authorizer (a module's New), the HTTP server, a rate limiter or a jobs
// client. The rule follows the static calls from each; the commands are
// func values it calls dynamically, so they are not followed: they only
// receive the composition. Reaching the module's NewAdmin shows the walk
// sees the composition at all.
func TestCommandsComposeNoServerAndNoJobs(t *testing.T) {
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
	graph := static.CallGraph(prog)
	for _, c := range []struct{ root, admin string }{
		{"Users", m("internal/modules/identity") + ".NewAdmin"},
		{"Workspaces", m("internal/modules/workspace") + ".NewAdmin"},
	} {
		root := built[0].Func(c.root)
		if root == nil {
			t.Fatalf("bootstrap.%s not found: the rule checks nothing", c.root)
		}
		reached, banned := walkCalls(graph, root, composesMore)
		if !slices.ContainsFunc(reached, func(chain []*ssa.Function) bool {
			return chain[len(chain)-1].String() == c.admin
		}) {
			var names []string
			for _, chain := range reached {
				names = append(names, funcName(chain[len(chain)-1]))
			}
			t.Errorf("bootstrap.%s does not reach %s, so the rule checks nothing; it reaches:\n%s",
				c.root, c.admin, strings.Join(names, "\n"))
		}
		for _, chain := range banned {
			names := make([]string, len(chain))
			for i, f := range chain {
				names[i] = funcName(f)
			}
			t.Errorf("bootstrap.%s builds more than the pool and the administrator use cases: %s", c.root, strings.Join(names, " → "))
		}
	}
}

// composesMore reports whether f builds what the command line must not: a
// module's HTTP side or the Authorizer (New in a module's root package:
// identity.New, workspace.New, access.New and those to come), the HTTP
// server (platform/httpserver), a rate limiter or a jobs client
// (platform/jobs, River).
func composesMore(f *ssa.Function) bool {
	if f.Pkg == nil {
		return false
	}
	path := f.Pkg.Pkg.Path()
	if module, ok := strings.CutPrefix(path, m("internal/modules")+"/"); ok && !strings.Contains(module, "/") && f.Name() == "New" {
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
