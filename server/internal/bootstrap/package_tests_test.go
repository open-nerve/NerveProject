package bootstrap

import (
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// packageTests are the names of the tests go test runs in dir: the tests
// of each of its _test.go files (testsOf) that the build includes, not
// those its build constraints leave out (a //go:build line, or a GOOS or
// GOARCH suffix of the name). The matrix's completeness check wants the
// test each public exemption names among this package's.
func packageTests(t *testing.T, dir string) map[string]bool {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	fset, tests := token.NewFileSet(), map[string]bool{}
	for _, path := range paths {
		built, err := build.Default.MatchFile(dir, filepath.Base(path))
		if err != nil {
			t.Fatal(err)
		}
		if !built {
			continue
		}
		f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		maps.Copy(tests, testsOf(f))
	}
	return tests
}

// packageTests leaves out the tests of a file the build does not include:
// of a directory with one file built and one a build constraint excludes,
// only the first's test.
func TestPackageTestsSkipsFilesTheBuildExcludes(t *testing.T) {
	dir := t.TempDir()
	for name, src := range map[string]string{
		"built_test.go":    "package p\n\nimport \"testing\"\n\nfunc TestBuilt(t *testing.T) {}\n",
		"excluded_test.go": "//go:build never\n\npackage p\n\nimport \"testing\"\n\nfunc TestExcluded(t *testing.T) {}\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if got, want := packageTests(t, dir), map[string]bool{"TestBuilt": true}; !maps.Equal(got, want) {
		t.Errorf("packageTests = %v, want %v", got, want)
	}
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
