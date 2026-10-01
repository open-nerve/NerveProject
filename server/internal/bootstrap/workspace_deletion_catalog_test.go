package bootstrap

import (
	"context"
	"strings"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// The catalog reading of the deletions' guards (workspace_deletion_test.go,
// project_deletion_test.go): the foreign keys to the deleted row's table,
// workspaces or projects, which fail on a table under it without one of its
// own (keysTo, with its counterexample), and each key's rows under a
// deleted row: undeleted, unstamped by the deletion, or all of them as
// text.

// foreignKey is a column that references parent, table as SQL names it.
type foreignKey struct{ parent, table, column string }

func (k foreignKey) String() string { return k.table + "." + k.column }

// keysTo are the parent row itself, as <parent>.id, then every foreign key
// to parent in the catalog. A table under parent only through others'
// foreign keys, followed from parent however far, fails the test, named
// with the tables it hangs from: no check here would see its rows, nor
// would a cascade that deletes by the parent's key column.
func keysTo(t testing.TB, pool *pgxpool.Pool, parent string) []foreignKey {
	t.Helper()
	rows, err := pool.Query(context.Background(), `
		SELECT c.conrelid::regclass::text, a.attname, cardinality(c.conkey)
		FROM pg_constraint c JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = c.conkey[1]
		WHERE c.contype = 'f' AND c.confrelid = $1::text::regclass
		ORDER BY 1, 2`, parent)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	keys := []foreignKey{{parent, parent, "id"}}
	for rows.Next() {
		k := foreignKey{parent: parent}
		var columns int
		if err := rows.Scan(&k.table, &k.column, &columns); err != nil {
			t.Fatal(err)
		}
		if columns != 1 {
			t.Fatalf("%s references %s with %d columns: find its rows another way", k, parent, columns)
		}
		keys = append(keys, k)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(keys) == 1 {
		t.Fatalf("no foreign key to %s in the catalog: the test would check the %s row only", parent, parent)
	}
	rows, err = pool.Query(context.Background(), `
		WITH RECURSIVE under (tab, via) AS (
			SELECT conrelid, confrelid FROM pg_constraint WHERE contype = 'f' AND confrelid = $1::text::regclass
			UNION
			SELECT c.conrelid, c.confrelid FROM pg_constraint c JOIN under u ON c.confrelid = u.tab WHERE c.contype = 'f')
		SELECT tab::regclass::text || ' (through ' || string_agg(DISTINCT via::regclass::text, ', ' ORDER BY via::regclass::text) || ')'
		FROM under
		WHERE NOT EXISTS (SELECT 1 FROM pg_constraint w WHERE w.contype = 'f' AND w.conrelid = under.tab AND w.confrelid = $1::text::regclass)
		GROUP BY tab ORDER BY 1`, parent)
	if err != nil {
		t.Fatal(err)
	}
	unkeyed, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	if len(unkeyed) > 0 {
		t.Fatalf("%s: under %s with no foreign key to %s of its own, so no check here sees its rows: add %s_id, or extend the guard",
			strings.Join(unkeyed, "; "), parent, parent, strings.TrimSuffix(parent, "s"))
	}
	return keys
}

// undeleted are the ids of k's undeleted rows under the parent row id.
func (k foreignKey) undeleted(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) []uuid.UUID {
	t.Helper()
	rows, err := pool.Query(context.Background(), "SELECT id FROM "+k.table+" WHERE "+pgx.Identifier{k.column}.Sanitize()+
		" = $1 AND deleted_at IS NULL ORDER BY id", id)
	if err != nil {
		t.Fatalf("%s: %v", k, err)
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		t.Fatalf("%s: %v", k, err)
	}
	return ids
}

// unstamped is those of k's rows ids that the deletion of the parent row id
// did not stamp, each as text: a deleted_at other than the parent row's, or
// an updated_by_id other than by. Every table under a workspace or a
// project has both columns.
func (k foreignKey) unstamped(t *testing.T, pool *pgxpool.Pool, ids []uuid.UUID, id, by uuid.UUID) string {
	t.Helper()
	var s string
	if err := pool.QueryRow(context.Background(), "SELECT coalesce(string_agg(r::text, E'\\n' ORDER BY r::text), '') FROM "+k.table+
		" r WHERE r.id = ANY ($1) AND (r.deleted_at IS DISTINCT FROM (SELECT deleted_at FROM "+k.parent+" WHERE id = $2)"+
		" OR r.updated_by_id IS DISTINCT FROM $3)", ids, id, by).Scan(&s); err != nil {
		t.Fatalf("%s: %v", k, err)
	}
	return s
}

// rows is k's rows under the parent row id, deleted ones too, each as text,
// in order; "" when there is none.
func (k foreignKey) rows(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) string {
	t.Helper()
	var s string
	if err := pool.QueryRow(context.Background(), "SELECT coalesce(string_agg(r::text, E'\\n' ORDER BY r::text), '') FROM "+k.table+
		" r WHERE r."+pgx.Identifier{k.column}.Sanitize()+" = $1", id).Scan(&s); err != nil {
		t.Fatalf("%s: %v", k, err)
	}
	return s
}

// keysTo fails on a table under workspaces through projects only, and on
// one under that one, and names both with the tables they hang from; and,
// for projects, on the one under projects only through the other: the
// deletions' checks would see neither's rows.
func TestKeysRefuseATableWithoutItsParent(t *testing.T) {
	pool := openPool(t, pgtest.NewDatabase(t))
	for _, sql := range []string{
		"CREATE TABLE widgets (id uuid PRIMARY KEY, project_id uuid NOT NULL REFERENCES projects, deleted_at timestamptz)",
		"CREATE TABLE gadgets (id uuid PRIMARY KEY, widget_id uuid NOT NULL REFERENCES widgets, deleted_at timestamptz)",
	} {
		if _, err := pool.Exec(context.Background(), sql); err != nil {
			t.Fatal(err)
		}
	}
	for parent, want := range map[string]string{
		"workspaces": "gadgets (through widgets); widgets (through projects): under workspaces with no foreign key to workspaces of its own, " +
			"so no check here sees its rows: add workspace_id, or extend the guard",
		"projects": "gadgets (through widgets): under projects with no foreign key to projects of its own, " +
			"so no check here sees its rows: add project_id, or extend the guard",
	} {
		if failed := fatalOf(func(tb testing.TB) { keysTo(tb, pool, parent) }); failed != want {
			t.Errorf("keysTo(%s) with widgets and gadgets: failed with %q, want %q", parent, failed, want)
		}
	}
}
