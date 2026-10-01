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

// The catalog reading of the deletion's guard (workspace_deletion_test.go):
// the foreign keys to workspaces, which fail on a table under workspaces
// without one of its own (workspaceKeys, with its counterexample), and each
// key's rows under a workspace: undeleted, unstamped by the deletion, or
// all of them as text.

// foreignKey is a column that references workspaces, table as SQL names it.
type foreignKey struct{ table, column string }

func (k foreignKey) String() string { return k.table + "." + k.column }

// workspaceKeys are the workspace row itself, as workspaces.id, then every
// foreign key to workspaces in the catalog. A table under workspaces only
// through others' foreign keys, followed from workspaces however far, fails
// the test, named with the tables it hangs from: no check here would see
// its rows, nor would a cascade that deletes by workspace_id.
func workspaceKeys(t testing.TB, pool *pgxpool.Pool) []foreignKey {
	t.Helper()
	rows, err := pool.Query(context.Background(), `
		SELECT c.conrelid::regclass::text, a.attname, cardinality(c.conkey)
		FROM pg_constraint c JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = c.conkey[1]
		WHERE c.contype = 'f' AND c.confrelid = 'workspaces'::regclass
		ORDER BY 1, 2`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	keys := []foreignKey{{"workspaces", "id"}}
	for rows.Next() {
		var k foreignKey
		var columns int
		if err := rows.Scan(&k.table, &k.column, &columns); err != nil {
			t.Fatal(err)
		}
		if columns != 1 {
			t.Fatalf("%s references workspaces with %d columns: find its rows another way", k, columns)
		}
		keys = append(keys, k)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(keys) == 1 {
		t.Fatal("no foreign key to workspaces in the catalog: the test would check the workspace row only")
	}
	rows, err = pool.Query(context.Background(), `
		WITH RECURSIVE under (tab, via) AS (
			SELECT conrelid, confrelid FROM pg_constraint WHERE contype = 'f' AND confrelid = 'workspaces'::regclass
			UNION
			SELECT c.conrelid, c.confrelid FROM pg_constraint c JOIN under u ON c.confrelid = u.tab WHERE c.contype = 'f')
		SELECT tab::regclass::text || ' (through ' || string_agg(DISTINCT via::regclass::text, ', ' ORDER BY via::regclass::text) || ')'
		FROM under
		WHERE NOT EXISTS (SELECT 1 FROM pg_constraint w WHERE w.contype = 'f' AND w.conrelid = under.tab AND w.confrelid = 'workspaces'::regclass)
		GROUP BY tab ORDER BY 1`)
	if err != nil {
		t.Fatal(err)
	}
	unkeyed, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	if len(unkeyed) > 0 {
		t.Fatalf("%s: under workspaces with no foreign key to workspaces of its own, so no check here sees its rows: "+
			"add workspace_id, or extend the guard", strings.Join(unkeyed, "; "))
	}
	return keys
}

// undeleted are the ids of k's undeleted rows under the workspace id.
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

// unstamped is those of k's rows ids that the deletion of the workspace id
// did not stamp, each as text: a deleted_at other than the workspace's, or
// an updated_by_id other than by. Every table under a workspace has both
// columns.
func (k foreignKey) unstamped(t *testing.T, pool *pgxpool.Pool, ids []uuid.UUID, id, by uuid.UUID) string {
	t.Helper()
	var s string
	if err := pool.QueryRow(context.Background(), "SELECT coalesce(string_agg(r::text, E'\\n' ORDER BY r::text), '') FROM "+k.table+
		" r WHERE r.id = ANY ($1) AND (r.deleted_at IS DISTINCT FROM (SELECT deleted_at FROM workspaces WHERE id = $2)"+
		" OR r.updated_by_id IS DISTINCT FROM $3)", ids, id, by).Scan(&s); err != nil {
		t.Fatalf("%s: %v", k, err)
	}
	return s
}

// rows is k's rows under the workspace id, deleted ones too, each as text,
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

// workspaceKeys fails on a table under workspaces through projects only,
// and on one under that one, and names both with the tables they hang
// from: the deletion's checks would see neither's rows.
func TestWorkspaceKeysRefuseATableWithoutItsWorkspace(t *testing.T) {
	pool := openPool(t, pgtest.NewDatabase(t))
	for _, sql := range []string{
		"CREATE TABLE widgets (id uuid PRIMARY KEY, project_id uuid NOT NULL REFERENCES projects, deleted_at timestamptz)",
		"CREATE TABLE gadgets (id uuid PRIMARY KEY, widget_id uuid NOT NULL REFERENCES widgets, deleted_at timestamptz)",
	} {
		if _, err := pool.Exec(context.Background(), sql); err != nil {
			t.Fatal(err)
		}
	}
	want := "gadgets (through widgets); widgets (through projects): under workspaces with no foreign key to workspaces of its own, " +
		"so no check here sees its rows: add workspace_id, or extend the guard"
	if failed := fatalOf(func(tb testing.TB) { workspaceKeys(tb, pool) }); failed != want {
		t.Errorf("workspaceKeys with widgets and gadgets: failed with %q, want %q", failed, want)
	}
}
