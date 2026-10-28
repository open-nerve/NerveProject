package migrations_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// A missing index, different key, predicate, schema or uniqueness must fail.
// Discover the tables from the live schema rather than duplicating a directory.
func TestPurgeIndexes(t *testing.T) {
	pool := newPool(t, pgtest.NewDatabase(t))
	ctx := context.Background()
	rows, err := pool.Query(ctx, `SELECT c.relname,coalesce(pg_get_indexdef(i.indexrelid),'')
 FROM pg_class c JOIN pg_attribute a ON a.attrelid=c.oid AND a.attname='deleted_at' AND NOT a.attisdropped
 LEFT JOIN pg_class ix ON ix.relname=c.relname||'_deleted_at_idx' AND ix.relnamespace=c.relnamespace
 LEFT JOIN pg_index i ON i.indexrelid=ix.oid AND i.indrelid=c.oid
 WHERE c.relnamespace='public'::regnamespace AND c.relkind IN ('r','p') ORDER BY 1`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var table, definition string
		if err := rows.Scan(&table, &definition); err != nil {
			t.Fatal(err)
		}
		n++
		want := fmt.Sprintf("CREATE INDEX %s_deleted_at_idx ON public.%s USING btree (deleted_at) WHERE (deleted_at IS NOT NULL)", table, table)
		if definition != want {
			t.Errorf("%s index=%q; want %q", table, definition, want)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if n != 23 {
		t.Fatalf("indexed soft-deleted tables=%d; want 23", n)
	}
}
