package apitest

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
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
// Main for module. It reads the _test.go files the build takes
// (go/build.Default.MatchFile: a file its constraints leave out never runs),
// and reports no TestMain function in them, or one whose file does not
// import this package as apitest or whose body is not the one call
// apitest.Main(m, "<module>") with its own *testing.M.
func mainViolations(dir, module string) []string {
	paths, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if err != nil {
		return []string{err.Error()}
	}
	var found []string
	mains := 0
	for _, path := range paths {
		built, err := build.Default.MatchFile(dir, filepath.Base(path))
		if err != nil {
			return []string{err.Error()}
		}
		if !built {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
		if err != nil {
			return []string{err.Error()}
		}
		for _, d := range f.Decls {
			if fn, ok := d.(*ast.FuncDecl); ok && fn.Recv == nil && fn.Name.Name == "TestMain" {
				mains++
				if !importsThisPackage(f) || !callsMain(fn, module) {
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

// importsThisPackage reports whether f imports this package under the name
// apitest, its own or given. Then apitest in f is this package: the name
// of a file's import cannot also be declared in its package.
func importsThisPackage(f *ast.File) bool {
	for _, spec := range f.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err == nil && path == thisPackage() && (spec.Name == nil || spec.Name.Name == "apitest") {
			return true
		}
	}
	return false
}

// thisPackage is this package's import path, as the module names it.
func thisPackage() string {
	return reflect.TypeFor[Contract]().PkgPath()
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
	head := "package p_test\n\nimport (\n\t\"os\"\n\t\"testing\"\n\n\t\"" + thisPackage() + "\"\n)\n\nvar _ = os.Exit\n\n"
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
		{"another package's Main", map[string]string{"handler_test.go": head + "func TestMain(m *testing.M) { other.Main(m, \"project\") }\n"},
			[]string{"DIR/handler_test.go: TestMain is not apitest.Main(m, \"project\")"}},
		{"another function of apitest", map[string]string{"handler_test.go": head + "func TestMain(m *testing.M) { apitest.Run(m, \"project\") }\n"},
			[]string{"DIR/handler_test.go: TestMain is not apitest.Main(m, \"project\")"}},
		{"a method named TestMain only", map[string]string{"handler_test.go": head +
			"type s struct{}\n\nfunc (s) TestMain(m *testing.M) { apitest.Main(m, \"project\") }\n"},
			[]string{"DIR has no TestMain: write func TestMain(m *testing.M) { apitest.Main(m, \"project\") }"}},
		{"another package as apitest, this one under another name", map[string]string{"handler_test.go": "package p_test\n\nimport (\n\t\"testing\"\n\n" +
			"\tat \"" + thisPackage() + "\"\n\tapitest \"x/othertest\"\n)\n\nvar _ = at.Main\n\nfunc TestMain(m *testing.M) { apitest.Main(m, \"project\") }\n"},
			[]string{"DIR/handler_test.go: TestMain is not apitest.Main(m, \"project\")"}},
		{"apitest declared by a sibling file", map[string]string{
			"handler_test.go": "package p_test\n\nimport \"testing\"\n\nfunc TestMain(m *testing.M) { apitest.Main(m, \"project\") }\n",
			"vars_test.go":    "package p_test\n\nimport \"testing\"\n\nvar apitest struct{ Main func(*testing.M, string) }\n"},
			[]string{"DIR/handler_test.go: TestMain is not apitest.Main(m, \"project\")"}},
		{"a TestMain the build leaves out", map[string]string{"handler_test.go": "//go:build never\n\n" + head +
			"func TestMain(m *testing.M) { apitest.Main(m, \"project\") }\n", "other_test.go": head + "func TestOther(t *testing.T) {}\n"},
			[]string{"DIR has no TestMain: write func TestMain(m *testing.M) { apitest.Main(m, \"project\") }"}},
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
