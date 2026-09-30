package bootstrap

import (
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"path/filepath"
	"strings"
	"testing"
)

// packageTests are the names of this package's tests: the tests of each of
// its _test.go files (testsOf). The matrix's completeness check wants the
// test each public exemption names among them.
func packageTests(t *testing.T) map[string]bool {
	t.Helper()
	paths, err := filepath.Glob("*_test.go")
	if err != nil {
		t.Fatal(err)
	}
	fset, tests := token.NewFileSet(), map[string]bool{}
	for _, path := range paths {
		f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		maps.Copy(tests, testsOf(f))
	}
	return tests
}

// testsOf are the tests go test runs of f: each function without a
// receiver, named Test and more, whose one parameter is a *testing.T.
func testsOf(f *ast.File) map[string]bool {
	tests := map[string]bool{}
	for _, d := range f.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || !strings.HasPrefix(fn.Name.Name, "Test") || len(fn.Type.Params.List) != 1 ||
			len(fn.Type.Params.List[0].Names) > 1 {
			continue
		}
		star, ok := fn.Type.Params.List[0].Type.(*ast.StarExpr)
		if !ok {
			continue
		}
		sel, ok := star.X.(*ast.SelectorExpr)
		if !ok {
			continue
		}
		if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "testing" && sel.Sel.Name == "T" {
			tests[fn.Name.Name] = true
		}
	}
	return tests
}

// testsOf finds a test and none of what only looks like one.
func TestTestsOfFindsOnlyTheTestsGoTestRuns(t *testing.T) {
	f, err := parser.ParseFile(token.NewFileSet(), "", `package p
func TestRuns(t *testing.T) {}
func (s suite) TestMethod(t *testing.T) {}
func TestTwo(t *testing.T, n int) {}
func TestPair(t, u *testing.T) {}
func TestBench(b *testing.B) {}
func TestNothing() {}
func TestValue(t testing.T) {}
func TestOther(t *other.T) {}
func helper(t *testing.T) {}
`, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := testsOf(f), map[string]bool{"TestRuns": true}; !maps.Equal(got, want) {
		t.Errorf("testsOf = %v, want %v", got, want)
	}
}
