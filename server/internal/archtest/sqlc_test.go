package archtest

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/knadh/koanf/parsers/yaml"
)

// TestSQLCSchemaScope holds server/sqlc.yaml to M2 design 3.14: a module's
// queries see only its own migrations, so a query can touch only its own
// tables; sqlc itself then rejects the rest.
func TestSQLCSchemaScope(t *testing.T) {
	registerSources(t)
	entries := readSQLCEntries(t, filepath.Join(moduleRoot, "sqlc.yaml"))
	migrations := readMigrations(t, filepath.Join(moduleRoot, "migrations", "sql"))
	queryModules, err := filepath.Glob(filepath.Join(moduleRoot, "internal", "modules", "*", "adapter", "postgres", "queries"))
	if err != nil {
		t.Fatal(err)
	}
	var modules []string
	for _, dir := range queryModules {
		modules = append(modules, filepath.Base(filepath.Dir(filepath.Dir(filepath.Dir(dir)))))
	}
	if len(entries) == 0 || len(migrations) == 0 || len(modules) == 0 {
		t.Fatalf("read %d sqlc entries, %d migrations, %d modules with queries; want some of each", len(entries), len(migrations), len(modules))
	}
	for _, v := range sqlcScopeViolations(entries, migrations, modules) {
		t.Error(v)
	}
}

// sqlcEntry is one sql entry of sqlc.yaml, paths relative to server/.
type sqlcEntry struct {
	schema  []string
	queries string
	out     string
}

func readSQLCEntries(t *testing.T, path string) []sqlcEntry {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := yaml.Parser().Unmarshal(data)
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	raw, _ := doc["sql"].([]any)
	var entries []sqlcEntry
	for i, item := range raw {
		m, _ := item.(map[string]any)
		gen, _ := m["gen"].(map[string]any)
		goGen, _ := gen["go"].(map[string]any)
		e := sqlcEntry{queries: fmt.Sprint(m["queries"]), out: fmt.Sprint(goGen["out"])}
		schema, ok := m["schema"].([]any)
		if !ok {
			t.Fatalf("%s: sql[%d].schema is not a list of migration files", path, i)
		}
		for _, s := range schema {
			e.schema = append(e.schema, fmt.Sprint(s))
		}
		entries = append(entries, e)
	}
	return entries
}

// migrationFile is one migration: its file name and SQL.
type migrationFile struct {
	name string
	sql  string
}

func readMigrations(t *testing.T, dir string) []migrationFile {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		t.Fatal(err)
	}
	var out []migrationFile
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, migrationFile{name: filepath.Base(f), sql: string(data)})
	}
	return out
}

var (
	migrationFileName = regexp.MustCompile(`^\d{5}_([a-z][a-z0-9]*)_[a-z0-9_]+\.sql$`)
	sqlComment        = regexp.MustCompile(`--[^\n]*`)
	createTable       = regexp.MustCompile(`(?i)\bCREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(?:public\.)?([a-z_][a-z0-9_]*)`)
	alterTable        = regexp.MustCompile(`(?i)\bALTER\s+TABLE\s+(?:IF\s+EXISTS\s+)?(?:ONLY\s+)?(?:public\.)?([a-z_][a-z0-9_]*)`)
	moduleQueries     = regexp.MustCompile(`^internal/modules/([a-z][a-z0-9]*)/adapter/postgres/queries$`)
)

// sqlcScopeViolations checks, in M2 design 3.14's words:
//   - migration files are named <version>_<module>_<content>.sql;
//   - the target of every ALTER TABLE is created by the file name's module,
//     so a migration belongs to the module that owns the table it changes;
//   - each sqlc entry is a module's (queries in its adapter/postgres) and
//     lists exactly that module's migrations: no other module's, and never
//     River's, which no module queries;
//   - every module with queries has an entry.
func sqlcScopeViolations(entries []sqlcEntry, migrations []migrationFile, queryModules []string) []string {
	var found []string
	report := func(format string, args ...any) { found = append(found, fmt.Sprintf(format, args...)) }

	moduleOf := map[string]string{}   // migration file → module
	byModule := map[string][]string{} // module → its migrations, as sqlc.yaml lists them
	owner := map[string]string{}      // table → module that creates it
	for _, m := range migrations {
		match := migrationFileName.FindStringSubmatch(m.name)
		if match == nil {
			report("migration %s is not named <version>_<module>_<content>.sql", m.name)
			continue
		}
		moduleOf[m.name] = match[1]
		byModule[match[1]] = append(byModule[match[1]], "migrations/sql/"+m.name)
		for _, c := range createTable.FindAllStringSubmatch(sqlComment.ReplaceAllString(m.sql, ""), -1) {
			table := strings.ToLower(c[1])
			if prev, ok := owner[table]; ok && prev != match[1] {
				report("tables: %s is created by both %s and %s", table, prev, match[1])
				continue
			}
			owner[table] = match[1]
		}
	}
	for _, m := range migrations {
		module, ok := moduleOf[m.name]
		if !ok {
			continue
		}
		var altered []string // once per table: Up and Down both alter it
		for _, a := range alterTable.FindAllStringSubmatch(sqlComment.ReplaceAllString(m.sql, ""), -1) {
			if table := strings.ToLower(a[1]); !slices.Contains(altered, table) {
				altered = append(altered, table)
			}
		}
		for _, table := range altered {
			switch tableOwner, known := owner[table]; {
			case !known:
				report("migration %s alters %s, which no migration creates", m.name, table)
			case tableOwner != module:
				report("migration %s alters %s, which module %s creates: the migration belongs to %s", m.name, table, tableOwner, tableOwner)
			}
		}
	}

	covered := map[string]bool{}
	for _, e := range entries {
		match := moduleQueries.FindStringSubmatch(e.queries)
		if match == nil {
			report("sqlc entry %s: queries must be internal/modules/<module>/adapter/postgres/queries", e.queries)
			continue
		}
		module := match[1]
		covered[module] = true
		if want := "internal/modules/" + module + "/adapter/postgres/gen"; e.out != want {
			report("sqlc entry %s: out is %s, want %s", module, e.out, want)
		}
		for _, s := range e.schema {
			if other, ok := moduleOf[filepath.Base(s)]; ok && other != module {
				report("sqlc entry %s lists %s, a migration of %s", module, s, other)
			} else if !ok {
				report("sqlc entry %s lists %s, which is not a migration", module, s)
			}
		}
		for _, own := range byModule[module] {
			if !slices.Contains(e.schema, own) {
				report("sqlc entry %s does not list its migration %s", module, own)
			}
		}
	}
	for _, module := range slices.Sorted(slices.Values(queryModules)) {
		if !covered[module] {
			report("module %s has adapter/postgres/queries but no sqlc entry", module)
		}
	}
	return found
}
