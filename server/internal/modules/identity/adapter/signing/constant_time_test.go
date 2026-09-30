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
// can time it, so this one reads the code: (*MAC).Verify is there, and every
// method Verify of mac.go has no branch and no comparison, so every path
// runs to its end, and every return answers crypto/hmac's Equal, which
// compares every byte.
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

// constantTimeViolations is what in file's methods Verify could answer
// before crypto/hmac's Equal compares every byte, and a missing
// (*MAC).Verify. Every method Verify is checked, whatever its receiver, so
// that no other one stands in for (*MAC).Verify.
func constantTimeViolations(fset *token.FileSet, file *ast.File) []string {
	uses := identifierUses(fset, file)
	var found []string
	onMAC := false
	for _, d := range file.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || fn.Name.Name != "Verify" {
			continue
		}
		receiver := types.ExprString(fn.Recv.List[0].Type)
		onMAC = onMAC || receiver == "*MAC"
		found = append(found, verifyViolations("("+receiver+").Verify", fn.Body, uses)...)
	}
	if !onMAC {
		found = append(found, "there is no method (*MAC).Verify")
	}
	return found
}

// verifyViolations is what in body, the body of the method named method,
// could answer before crypto/hmac's Equal compares every byte: a branching
// statement, a comparison, a return of anything but one call of
// crypto/hmac's Equal, and no return at all.
func verifyViolations(method string, body *ast.BlockStmt, uses map[*ast.Ident]types.Object) []string {
	var found []string
	returns := 0
	ast.Inspect(body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.IfStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SelectStmt, *ast.BranchStmt:
			found = append(found, fmt.Sprintf("%s branches: %T", method, n))
		case *ast.BinaryExpr:
			switch n.Op {
			case token.EQL, token.NEQ, token.LSS, token.GTR, token.LEQ, token.GEQ:
				found = append(found, method+" compares with "+n.Op.String())
			}
		case *ast.ReturnStmt:
			returns++
			if len(n.Results) != 1 || !callsHMACEqual(n.Results[0], uses) {
				found = append(found, method+" returns something other than crypto/hmac's Equal(…)")
			}
		}
		return true
	})
	if returns == 0 {
		found = append(found, method+" has no return")
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
	clean := " Verify(message []byte, tag [16]byte) bool { want := m.Tag(message); return hmac.Equal(want[:], tag[:]) }\n"
	shortcut := " Verify(message []byte, tag [16]byte) bool { return m.Tag(message) == tag }\n"
	for _, tt := range []struct {
		name, funcs string
		want        int
	}{
		{"a function Verify, not a method", "func" + clean, 1},
		{"only another type's Verify", "func (m *other)" + clean, 1},
		{"Verify on MAC, not *MAC", "func (m MAC)" + clean, 1},
		{"a shortcut in (*MAC).Verify, another Verify after it", "func (m *MAC)" + shortcut + "func (m *other)" + clean, 2},
		{"a shortcut in another Verify, after (*MAC).Verify", "func (m *MAC)" + clean + "func (m *other)" + shortcut, 2},
	} {
		if got := violationsOf(t, "package signing\nimport \"crypto/hmac\"\n"+tt.funcs); len(got) != tt.want {
			t.Errorf("%s: %q, want %d violations", tt.name, got, tt.want)
		}
	}
}
