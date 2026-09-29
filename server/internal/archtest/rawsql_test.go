package archtest

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// TestModulesRunSQLOnlyThroughSQLC holds all of a module's SQL to M2 design
// 3.14, not only the queries sqlc sees (M3 design 6.5; P1 final review M7):
// outside the gen packages that sqlc writes, a module's production code
// neither runs a statement of its own nor holds one. TestSQLCSchemaScope
// sees sqlc's queries only, so a statement run through pgx directly could
// read another module's tables, e.g. a member list that joins users instead
// of asking MemberProfiles. Tests are left out: they seed fixtures with SQL.
func TestModulesRunSQLOnlyThroughSQLC(t *testing.T) {
	registerSources(t)
	files := map[string]string{}
	root := filepath.Join(moduleRoot, "internal", "modules")
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(moduleRoot, path)
		files[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// A walk that finds no store must not pass as "no raw SQL".
	if !slices.ContainsFunc(slices.Collect(maps.Keys(files)), func(p string) bool {
		return strings.HasSuffix(p, "/adapter/postgres/store.go")
	}) {
		t.Fatalf("read %d files under internal/modules and no adapter/postgres/store.go: the rule checks nothing", len(files))
	}
	for _, v := range rawSQLViolations(files) {
		t.Error(v)
	}
}

// statementMethods are the pgx methods that run SQL they are given: pgx's
// Conn, Tx and Batch results, pgxpool's Pool and Conn, and
// platform/postgres's Querier have the first six, and pgconn's PgConn,
// which a pgx.Conn's PgConn() returns, the last three. The call rule counts
// only calls with at least two arguments, a context and the SQL (or the
// batch, the table or the statement's name): a method of one of these names
// that takes fewer, such as url.URL.Query, runs no statement.
var statementMethods = []string{
	"Exec", "Query", "QueryRow", "SendBatch", "CopyFrom", "Prepare",
	"ExecParams", "ExecPrepared", "CopyTo",
}

// sqlText matches a string literal that holds a statement: upper-case SQL,
// as every query of this repository is written. Lower-case prose, such as a
// message that tells a user to "select a workspace from the list", is not
// matched.
var sqlText = regexp.MustCompile(`\b(?:SELECT\b[\s\S]*\bFROM|INSERT\s+INTO|UPDATE\s+\S+\s+SET|DELETE\s+FROM|TRUNCATE|MERGE\s+INTO)\b`)

// rawSQLViolations checks the Go files of internal/modules, by path
// relative to server/: in a file that is neither a test nor in a gen
// package, no call of a statementMethods method with two arguments or more,
// and no string literal that sqlText matches. A file that does not parse is
// reported too, so that it cannot hide either. The rule finds raw SQL
// written by accident, not an evasion: a method value (run := tx.Query) and
// SQL built from lower-case pieces are out of its reach.
func rawSQLViolations(files map[string]string) []string {
	var found []string
	for _, path := range slices.Sorted(maps.Keys(files)) {
		if strings.HasSuffix(path, "_test.go") || slices.Contains(strings.Split(path, "/"), "gen") {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, files[path], parser.SkipObjectResolution)
		if err != nil {
			found = append(found, fmt.Sprintf("%s does not parse: %v", path, err))
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.CallExpr:
				if sel, ok := n.Fun.(*ast.SelectorExpr); ok && slices.Contains(statementMethods, sel.Sel.Name) && len(n.Args) >= 2 {
					found = append(found, fmt.Sprintf("%s: calls %s, which runs SQL outside sqlc's queries", fset.Position(n.Pos()), sel.Sel.Name))
				}
			case *ast.BasicLit:
				if n.Kind != token.STRING {
					return true
				}
				if s, err := strconv.Unquote(n.Value); err == nil && sqlText.MatchString(s) {
					found = append(found, fmt.Sprintf("%s: holds SQL outside sqlc's queries", fset.Position(n.Pos())))
				}
			}
			return true
		})
	}
	return found
}
