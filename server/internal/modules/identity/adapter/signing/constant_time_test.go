package signing

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// Verify answers hmac.Equal of the two tags and nothing else: a comparison
// that stops at the first difference tells a forger, by its time, how much
// of a tag is right (M2 design 3.9, M3 design 8.1). No test can time it, so
// this one reads the code: every return of Verify returns hmac.Equal(…), and
// nothing in it compares with == or !=, so no path answers before
// hmac.Equal has compared every byte.
func TestMACVerifyComparesInConstantTime(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "mac.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var verify *ast.FuncDecl
	for _, d := range file.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok && fn.Name.Name == "Verify" && fn.Recv != nil {
			verify = fn
		}
	}
	if verify == nil {
		t.Fatal("mac.go has no method Verify")
	}
	for _, v := range constantTimeViolations(verify.Body) {
		t.Error(v)
	}
}

// constantTimeViolations is what in body could answer before hmac.Equal
// compares every byte: a return of anything but one hmac.Equal call, a
// comparison with == or !=, and no return at all.
func constantTimeViolations(body *ast.BlockStmt) []string {
	var found []string
	returns := 0
	ast.Inspect(body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.ReturnStmt:
			returns++
			if len(n.Results) != 1 || !isCallOf(n.Results[0], "hmac", "Equal") {
				found = append(found, "Verify returns something other than hmac.Equal(…)")
			}
		case *ast.BinaryExpr:
			if n.Op == token.EQL || n.Op == token.NEQ {
				found = append(found, "Verify compares with "+n.Op.String())
			}
		}
		return true
	})
	if returns == 0 {
		found = append(found, "Verify has no return")
	}
	return found
}

// Each check of constantTimeViolations fails on its counterexample.
func TestConstantTimeViolationsCatchesEachShortcut(t *testing.T) {
	for _, tt := range []struct {
		name, body string
		want       int
	}{
		{"the real body", "want := m.Tag(message); return hmac.Equal(want[:], tag[:])", 0},
		{"== answered", "want := m.Tag(message); return want == tag", 2},
		{"an early return on a mismatch", "want := m.Tag(message); if want != tag { return false }; return hmac.Equal(want[:], tag[:])", 2},
		{"== in a dead branch", "want := m.Tag(message); if ok := want == tag; true { return ok }; return hmac.Equal(want[:], tag[:])", 2},
		{"no return", "for {}", 1},
	} {
		f, err := parser.ParseFile(token.NewFileSet(), "", "package p\nfunc (m *MAC) Verify(message []byte, tag [16]byte) bool {"+tt.body+"}", 0)
		if err != nil {
			t.Fatal(err)
		}
		if got := constantTimeViolations(f.Decls[0].(*ast.FuncDecl).Body); len(got) != tt.want {
			t.Errorf("%s: %q, want %d violations", tt.name, got, tt.want)
		}
	}
}

// isCallOf reports whether e calls pkg.name.
func isCallOf(e ast.Expr, pkg, name string) bool {
	call, ok := e.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	x, ok := sel.X.(*ast.Ident)
	return ok && x.Name == pkg && sel.Sel.Name == name
}
