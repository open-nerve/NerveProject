package bootstrap

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

type purgeIncomingKey struct {
	foreignKey
	action string
}

// foreignKeysTo is also used by the row-deletion guard, keeping the physical
// endpoints and the single-column constraint check in one catalog reader.
func foreignKeysTo(t testing.TB, pool *pgxpool.Pool, parent string) []purgeIncomingKey {
	t.Helper()
	rows, err := pool.Query(context.Background(), `SELECT c.conrelid::regclass::text,a.attname,cardinality(c.conkey),c.confdeltype::text
 FROM pg_constraint c JOIN pg_attribute a ON a.attrelid=c.conrelid AND a.attnum=c.conkey[1]
 WHERE c.contype='f' AND c.confrelid=$1::text::regclass ORDER BY 1,2`, parent)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var keys []purgeIncomingKey
	for rows.Next() {
		k := purgeIncomingKey{foreignKey: foreignKey{parent: parent}}
		var columns int
		if err := rows.Scan(&k.table, &k.column, &columns, &k.action); err != nil {
			t.Fatal(err)
		}
		if columns != 1 {
			t.Fatalf("%s references %s with %d columns: find its rows another way", k.foreignKey, parent, columns)
		}
		keys = append(keys, k)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return keys
}

func purgeTables(t *testing.T, pool *pgxpool.Pool) []string {
	t.Helper()
	rows, err := pool.Query(context.Background(), `SELECT c.relname FROM pg_class c JOIN pg_attribute a ON a.attrelid=c.oid
 WHERE c.relnamespace='public'::regnamespace AND c.relkind IN ('r','p') AND a.attname='deleted_at' AND NOT a.attisdropped ORDER BY 1`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		tables = append(tables, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return tables
}

func providedPurgeTables(pool *pgxpool.Pool) []string {
	var tables []string
	tables = append(tables, issue.Provide(pool).Purger.Tables()...)
	tables = append(tables, project.Provide(pool).Purger.Tables()...)
	tables = append(tables, workspace.Provide(pool).Purger.Tables()...)
	return append(tables, identity.Provide(pool).Purger.Tables()...)
}

func inspectedPurgeKeys(t *testing.T, pool *pgxpool.Pool, tables []string) map[foreignKey]string {
	t.Helper()
	keys := map[foreignKey]string{}
	for _, parent := range tables {
		for _, key := range foreignKeysTo(t, pool, parent) {
			keys[key.foreignKey] = key.action
		}
	}
	return keys
}

// The required edge set is read independently, without a source-table filter:
// Non-purge tables are as binding as purge children.
func purgeCatalogGaps(t *testing.T, pool *pgxpool.Pool, tables, provided []string, inspected map[foreignKey]string) []string {
	t.Helper()
	var gaps []string
	seen := map[string]int{}
	for _, table := range provided {
		seen[table]++
		if !slices.Contains(tables, table) {
			gaps = append(gaps, "unexpected cleaner "+table)
		}
	}
	for _, table := range tables {
		if seen[table] != 1 {
			gaps = append(gaps, fmt.Sprintf("%s has %d cleaners", table, seen[table]))
		}
	}
	rows, err := pool.Query(context.Background(), `SELECT c.confrelid::regclass::text,c.conrelid::regclass::text,a.attname,c.confdeltype::text
 FROM pg_constraint c JOIN pg_attribute a ON a.attrelid=c.conrelid AND a.attnum=c.conkey[1]
 WHERE c.contype='f' AND c.confrelid IN (SELECT oid FROM pg_class WHERE relnamespace='public'::regnamespace AND relname=ANY($1)) ORDER BY 1,2,3`, tables)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var k foreignKey
		var action string
		if err := rows.Scan(&k.parent, &k.table, &k.column, &action); err != nil {
			t.Fatal(err)
		}
		observed, ok := inspected[k]
		if !ok {
			gaps = append(gaps, fmt.Sprintf("missing incoming %s -> %s", k, k.parent))
		} else if observed != action {
			gaps = append(gaps, fmt.Sprintf("stale incoming action %s", k))
		}
		if action != "c" && action != "n" {
			gaps = append(gaps, fmt.Sprintf("unsafe incoming %s -> %s: %s", k, k.parent, action))
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return gaps
}

// Dropping a cleaner, skipping the second-level reaction FK or filtering out
// non-purge sources must fail even when every remaining FK is safe.
func TestPurgeCatalogCoversTablesAndIncomingKeys(t *testing.T) {
	pool := openPool(t, pgtest.NewDatabase(t))
	tables := purgeTables(t, pool)
	if len(tables) != 23 {
		t.Fatalf("deleted_at business tables=%d: %v; want 23", len(tables), tables)
	}
	// Check the unmodified business graph before adding the isolated non-purge probe.
	keysTo(t, pool, "issues")
	if _, err := pool.Exec(context.Background(), "CREATE TABLE purge_audit(id uuid PRIMARY KEY,comment_id uuid REFERENCES issue_comments(id) ON DELETE SET NULL)"); err != nil {
		t.Fatal(err)
	}
	inspected := inspectedPurgeKeys(t, pool, tables)
	reaction := foreignKey{"issue_comments", "comment_reactions", "comment_id"}
	if inspected[reaction] != "c" {
		t.Fatalf("second-level comment reaction FK=%q", inspected[reaction])
	}
	nonPurge := 0
	for key := range inspected {
		if !slices.Contains(tables, key.table) {
			nonPurge++
		}
	}
	if nonPurge == 0 {
		t.Fatal("fixture has no non-purge incoming source")
	}
	if gaps := purgeCatalogGaps(t, pool, tables, providedPurgeTables(pool), inspected); len(gaps) > 0 {
		t.Fatal(strings.Join(gaps, "; "))
	}
	t.Logf("catalog tables=%d incoming=%d non-purge-source=%d", len(tables), len(inspected), nonPurge)
}

func TestPurgeCatalogRejectsEachGap(t *testing.T) {
	pool := openPool(t, pgtest.NewDatabase(t))
	if _, err := pool.Exec(context.Background(), "CREATE TABLE purge_audit(id uuid PRIMARY KEY,comment_id uuid REFERENCES issue_comments(id) ON DELETE SET NULL)"); err != nil {
		t.Fatal(err)
	}
	tables := purgeTables(t, pool)
	provided := providedPurgeTables(pool)
	t.Run("missing cleaner", func(t *testing.T) {
		for _, table := range tables {
			copy := slices.Clone(provided)
			i := slices.Index(copy, table)
			if i < 0 {
				t.Errorf("missing existing cleaner %s", table)
				continue
			}
			copy = append(copy[:i], copy[i+1:]...)
			gaps := purgeCatalogGaps(t, pool, tables, copy, inspectedPurgeKeys(t, pool, tables))
			if !strings.Contains(strings.Join(gaps, ";"), table+" has 0 cleaners") {
				t.Fatalf("accepted missing %s: %v", table, gaps)
			}
		}
	})
	t.Run("missing incoming", func(t *testing.T) {
		keys := inspectedPurgeKeys(t, pool, tables)
		for key := range keys {
			copy := inspectedPurgeKeys(t, pool, tables)
			delete(copy, key)
			gaps := purgeCatalogGaps(t, pool, tables, provided, copy)
			if !strings.Contains(strings.Join(gaps, ";"), "missing incoming "+key.String()+" -> "+key.parent) {
				t.Fatalf("accepted missing %s -> %s: %v", key, key.parent, gaps)
			}
		}
	})
	t.Run("non purge source and no action", func(t *testing.T) {
		for _, action := range []string{"CASCADE", "SET NULL", "NO ACTION", "RESTRICT", "SET DEFAULT"} {
			if _, err := pool.Exec(context.Background(), "CREATE TABLE purge_probe(id uuid PRIMARY KEY,comment_id uuid REFERENCES issue_comments(id) ON DELETE "+action+")"); err != nil {
				t.Fatal(err)
			}
			gaps := purgeCatalogGaps(t, pool, tables, provided, inspectedPurgeKeys(t, pool, tables))
			unsafe := action != "CASCADE" && action != "SET NULL"
			if (len(gaps) > 0) != unsafe {
				t.Fatalf("%s: gaps=%v", action, gaps)
			}
			if unsafe && !strings.Contains(strings.Join(gaps, ";"), "unsafe incoming purge_probe.comment_id -> issue_comments") {
				t.Fatalf("wrong violation: %v", gaps)
			}
			if _, err := pool.Exec(context.Background(), "DROP TABLE purge_probe"); err != nil {
				t.Fatal(err)
			}
		}
	})
}
