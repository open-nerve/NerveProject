package signing

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path"
	"testing"
)

// Verify answers crypto/hmac's Equal of the two tags and nothing else: a
// comparison that stops at the first difference tells a forger, by its
// time, how much of a tag is right (M2 design 3.9, M3 design 8.1). No test
// can time it, so this one reads the code: Verify has no branch and no
// comparison, so every path runs to its end, and every return answers
// crypto/hmac's Equal, which compares every byte.
func TestMACVerifyComparesInConstantTime(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "mac.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range constantTimeViolations(fset, file) {
		t.Error(v)
	}
}

// constantTimeViolations is what in file's method Verify could answer
// before crypto/hmac's Equal compares every byte: a branching statement, a
// comparison, a return of anything but one call of crypto/hmac's Equal, and
// no return at all.
func constantTimeViolations(fset *token.FileSet, file *ast.File) []string {
	var verify *ast.FuncDecl
	for _, d := range file.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok && fn.Name.Name == "Verify" && fn.Recv != nil {
			verify = fn
		}
	}
	if verify == nil {
		return []string{"there is no method Verify"}
	}
	uses := identifierUses(fset, file)
	var found []string
	returns := 0
	ast.Inspect(verify.Body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.IfStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SelectStmt, *ast.BranchStmt:
			found = append(found, fmt.Sprintf("Verify branches: %T", n))
		case *ast.BinaryExpr:
			switch n.Op {
			case token.EQL, token.NEQ, token.LSS, token.GTR, token.LEQ, token.GEQ:
				found = append(found, "Verify compares with "+n.Op.String())
			}
		case *ast.ReturnStmt:
			returns++
			if len(n.Results) != 1 || !callsHMACEqual(n.Results[0], uses) {
				found = append(found, "Verify returns something other than crypto/hmac's Equal(…)")
			}
		}
		return true
	})
	if returns == 0 {
		found = append(found, "Verify has no return")
	}
	return found
}

// identifierUses is what each identifier of file denotes. file is
// type-checked on its own, its imports empty packages: the errors that
// leaves (their members are undefined) are ignored, for a package's name
// still denotes its import and a local name its declaration.
func identifierUses(fset *token.FileSet, file *ast.File) map[*ast.Ident]types.Object {
	info := &types.Info{Uses: map[*ast.Ident]types.Object{}}
	conf := types.Config{Importer: emptyImports{}, Error: func(error) {}}
	_, _ = conf.Check(file.Name.Name, fset, []*ast.File{file}, info)
	return info.Uses
}

// emptyImports imports every path as an empty package named after the
// path's last element.
type emptyImports struct{}

func (emptyImports) Import(importPath string) (*types.Package, error) {
	pkg := types.NewPackage(importPath, path.Base(importPath))
	pkg.MarkComplete()
	return pkg, nil
}

// callsHMACEqual reports whether e calls Equal of the package crypto/hmac,
// under whatever name the file imports it.
func callsHMACEqual(e ast.Expr, uses map[*ast.Ident]types.Object) bool {
	call, ok := e.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Equal" {
		return false
	}
	x, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	pkg, ok := uses[x].(*types.PkgName)
	return ok && pkg.Imported().Path() == "crypto/hmac"
}

// violationsOf is constantTimeViolations of the file src.
func violationsOf(t *testing.T, src string) []string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	return constantTimeViolations(fset, file)
}

// Each check of constantTimeViolations fails on its counterexample. A
// counterexample imports bytes and crypto/hmac unless it names its imports;
// it needs only to parse.
func TestConstantTimeViolationsCatchesEachShortcut(t *testing.T) {
	for _, tt := range []struct {
		name, imports, body string
		want                int
	}{
		{"the real body", "", "want := m.Tag(message); return hmac.Equal(want[:], tag[:])", 0},
		{"crypto/hmac imported as h", `h "crypto/hmac"`, "want := m.Tag(message); return h.Equal(want[:], tag[:])", 0},
		{"== answered", "", "want := m.Tag(message); return want == tag", 2},
		{"an early return on a mismatch", "", "want := m.Tag(message); if want != tag { return false }; return hmac.Equal(want[:], tag[:])", 3},
		{"== in a dead branch", "", "want := m.Tag(message); if ok := want == tag; true { return ok }; return hmac.Equal(want[:], tag[:])", 3},
		{"an early return per byte with < and >", "", "want := m.Tag(message); for i := range want { if want[i] < tag[i] || want[i] > tag[i] { return hmac.Equal(nil, tag[:]) } }; return hmac.Equal(want[:], tag[:])", 4},
		{"an early return through switch", "", "want := m.Tag(message); switch want { case tag: default: return hmac.Equal(nil, tag[:]) }; return hmac.Equal(want[:], tag[:])", 1},
		{"<= and >=", "", "want := m.Tag(message); _ = want[0] <= tag[0] || want[0] >= tag[0]; return hmac.Equal(want[:], tag[:])", 2},
		{"a type switch", "", "want := m.Tag(message); switch any(want).(type) { case [16]byte: }; return hmac.Equal(want[:], tag[:])", 1},
		{"a select", "", "want := m.Tag(message); select { default: }; return hmac.Equal(want[:], tag[:])", 1},
		{"a goto", "", "want := m.Tag(message); goto done; done: return hmac.Equal(want[:], tag[:])", 1},
		{"no return", "", "for {}", 2},
		{"bytes.Equal answered", "", "want := m.Tag(message); return bytes.Equal(want[:], tag[:])", 1},
		{"another package imported as hmac", `hmac "bytes"`, "want := m.Tag(message); return hmac.Equal(want[:], tag[:])", 1},
		{"hmac shadowed by a local", "", "want := m.Tag(message); hmac := struct{ Equal func(a, b []byte) bool }{bytes.Equal}; return hmac.Equal(want[:], tag[:])", 1},
		{"crypto/hmac's New answered", "", "return hmac.New(nil, nil)", 1},
	} {
		imports := tt.imports
		if imports == "" {
			imports = `"bytes"; "crypto/hmac"`
		}
		src := "package signing\nimport (" + imports + ")\nfunc (m *MAC) Verify(message []byte, tag [16]byte) bool {" + tt.body + "}"
		if got := violationsOf(t, src); len(got) != tt.want {
			t.Errorf("%s: %q, want %d violations", tt.name, got, tt.want)
		}
	}
	src := "package signing\nimport \"crypto/hmac\"\nfunc Verify(want, tag [16]byte) bool { return hmac.Equal(want[:], tag[:]) }"
	if got := violationsOf(t, src); len(got) != 1 {
		t.Errorf("a function Verify, not a method: %q, want 1 violation", got)
	}
}
