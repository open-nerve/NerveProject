package bootstrap

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

type storageContract struct {
	Routes []string `json:"routes"`
	Codes  []string `json:"x-problem-codes"`
}

func rawStorageContract(t *testing.T) storageContract {
	t.Helper()
	data, err := os.ReadFile("../../../api/storage-local.json")
	if err != nil {
		t.Fatal(err)
	}
	var contract storageContract
	if err := json.Unmarshal(data, &contract); err != nil {
		t.Fatal(err)
	}
	if len(contract.Routes) == 0 || len(contract.Codes) == 0 {
		t.Fatal("raw storage contract is empty")
	}
	return contract
}

// The non-JSON routes have their own exact contract. Keep its owned codes equal
// to production literals so the frontend's bilingual catalog cannot drift.
func TestRawStorageErrorCatalog(t *testing.T) {
	files, err := filepath.Glob("../modules/asset/adapter/http/*.go")
	if err != nil {
		t.Fatal(err)
	}
	codes := map[string]bool{}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			literal, ok := node.(*ast.BasicLit)
			if ok && literal.Kind == token.STRING {
				value, err := strconv.Unquote(literal.Value)
				if err == nil && strings.HasPrefix(value, "asset.") {
					codes[value] = true
				}
			}
			return true
		})
	}
	want := rawStorageContract(t).Codes
	slices.Sort(want)
	got := make([]string, 0, len(codes))
	for code := range codes {
		got = append(got, code)
	}
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Fatalf("raw error codes %q differ from contract %q", got, want)
	}
}
