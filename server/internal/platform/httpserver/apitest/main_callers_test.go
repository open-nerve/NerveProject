package apitest

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"
)

// Every module with a file in api/modules/ runs the tests of its HTTP
// adapter through Main, under its own name (M3/P1 review, M6): a module
// whose tests ran without it, or with another module's name, would never
// have its declared problem codes checked.
func TestEveryModuleRunsItsHTTPTestsThroughMain(t *testing.T) {
	names, err := moduleNames()
	if err != nil || len(names) == 0 {
		t.Fatalf("module files = %q, %v; want at least one", names, err)
	}
	for _, name := range names {
		for _, v := range mainViolations(adapterDir(name), name) {
			t.Error(v)
		}
	}
}

// adapterDir is server/internal/modules/<module>/adapter/http, where Main's
// doc comment says the module's HTTP tests are.
func adapterDir(module string) string {
	return filepath.Join(apiDir(), "..", "server", "internal", "modules", module, "adapter", "http")
}

// mainViolations reports what keeps the tests in dir from running through
// Main for module: no TestMain in its _test.go files, or one whose body is
// not the one call Main(m, "<module>") with its own *testing.M.
func mainViolations(dir, module string) []string {
	paths, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if err != nil {
		return []string{err.Error()}
	}
	var found []string
	mains := 0
	for _, path := range paths {
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
		if err != nil {
			return []string{err.Error()}
		}
		for _, d := range f.Decls {
			if fn, ok := d.(*ast.FuncDecl); ok && fn.Recv == nil && fn.Name.Name == "TestMain" {
				mains++
				if !callsMain(fn, module) {
					found = append(found, fmt.Sprintf("%s: TestMain is not apitest.Main(m, %q)", path, module))
				}
			}
		}
	}
	if mains == 0 {
		found = append(found, fmt.Sprintf("%s has no TestMain: write func TestMain(m *testing.M) { apitest.Main(m, %q) }", dir, module))
	}
	return found
}

// callsMain reports whether fn's body is the one statement
// apitest.Main(<its parameter>, "<module>").
func callsMain(fn *ast.FuncDecl, module string) bool {
	params := fn.Type.Params.List
	if len(params) != 1 || len(params[0].Names) != 1 || fn.Body == nil || len(fn.Body.List) != 1 {
		return false
	}
	stmt, ok := fn.Body.List[0].(*ast.ExprStmt)
	if !ok {
		return false
	}
	call, ok := stmt.X.(*ast.CallExpr)
	if !ok || len(call.Args) != 2 {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	arg, isIdent := call.Args[0].(*ast.Ident)
	name, isLit := call.Args[1].(*ast.BasicLit)
	return ok && pkg.Name == "apitest" && sel.Sel.Name == "Main" && isIdent && arg.Name == params[0].Names[0].Name && isLit &&
		name.Kind == token.STRING && name.Value == strconv.Quote(module)
}

// Each check of mainViolations fails on its counterexample, and a package
// that runs through Main passes.
func TestMainViolationsCatchesEachGap(t *testing.T) {
	const head = "package p_test\n\nimport (\n\t\"os\"\n\t\"testing\"\n\n\t\"x/apitest\"\n)\n\nvar _ = os.Exit\n\n"
	tests := []struct {
		name  string
		files map[string]string
		want  []string
	}{
		{"through Main", map[string]string{"handler_test.go": head + "func TestMain(m *testing.M) { apitest.Main(m, \"project\") }\n",
			"other_test.go": head + "func TestOther(t *testing.T) {}\n"}, nil},
		{"no TestMain", map[string]string{"handler_test.go": head + "func TestOther(t *testing.T) {}\n"},
			[]string{"DIR has no TestMain: write func TestMain(m *testing.M) { apitest.Main(m, \"project\") }"}},
		{"a plain TestMain", map[string]string{"handler_test.go": head + "func TestMain(m *testing.M) { os.Exit(m.Run()) }\n"},
			[]string{"DIR/handler_test.go: TestMain is not apitest.Main(m, \"project\")"}},
		{"another module's name", map[string]string{"handler_test.go": head + "func TestMain(m *testing.M) { apitest.Main(m, \"workspace\") }\n"},
			[]string{"DIR/handler_test.go: TestMain is not apitest.Main(m, \"project\")"}},
		{"Main and more", map[string]string{"handler_test.go": head +
			"func TestMain(m *testing.M) {\n\tapitest.Main(m, \"project\")\n\tos.Exit(0)\n}\n"},
			[]string{"DIR/handler_test.go: TestMain is not apitest.Main(m, \"project\")"}},
		{"another M", map[string]string{"handler_test.go": head + "var other *testing.M\n\nfunc TestMain(m *testing.M) { apitest.Main(other, \"project\") }\n"},
			[]string{"DIR/handler_test.go: TestMain is not apitest.Main(m, \"project\")"}},
	}
	for _, tt := range tests {
		dir := t.TempDir()
		for name, src := range tt.files {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		var want []string
		for _, w := range tt.want {
			want = append(want, filepath.Clean(dir)+w[len("DIR"):])
		}
		if got := mainViolations(dir, "project"); !slices.Equal(got, want) {
			t.Errorf("%s: %q, want %q", tt.name, got, want)
		}
	}
}
