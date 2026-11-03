package bootstrap

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

type protectionKey struct {
	oid                           uint32
	ref                           asset.ProtectedParentReference
	action, parentColumn          string
	columns, parentColumns        int
	deferred, deferrable, indexed bool
}

func protectionKeys(t testing.TB, pool *pgxpool.Pool) []protectionKey {
	t.Helper()
	rows, err := pool.Query(context.Background(), `SELECT c.oid,ns.nspname,child.relname,c.conname,a.attname,parent.relname,
 c.confdeltype::text,b.attname,cardinality(c.conkey),cardinality(c.confkey),c.condeferred,c.condeferrable,
 EXISTS(SELECT 1 FROM pg_index i WHERE i.indrelid=c.conrelid AND i.indisvalid AND i.indpred IS NULL AND i.indexprs IS NULL AND i.indkey[0]=c.conkey[1])
 FROM pg_constraint c JOIN pg_class child ON child.oid=c.conrelid JOIN pg_namespace ns ON ns.oid=child.relnamespace
 JOIN pg_class parent ON parent.oid=c.confrelid
 JOIN pg_attribute a ON a.attrelid=c.conrelid AND a.attnum=c.conkey[1]
 JOIN pg_attribute b ON b.attrelid=c.confrelid AND b.attnum=c.confkey[1]
 WHERE c.contype='f' AND child.relnamespace='public'::regnamespace AND parent.relnamespace='public'::regnamespace ORDER BY c.conname`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []protectionKey
	for rows.Next() {
		var r protectionKey
		if err := rows.Scan(&r.oid, &r.ref.Schema, &r.ref.Table, &r.ref.Constraint, &r.ref.Column, &r.ref.Parent, &r.action, &r.parentColumn, &r.columns, &r.parentColumns, &r.deferred, &r.deferrable, &r.indexed); err != nil {
			t.Fatal(err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func protectedPurgeSteps(pool *pgxpool.Pool) []string {
	refs := protectedAssetReferences()
	return slices.Concat(issue.Provide(pool, refs).Purger.ProtectedTables(), project.Provide(pool, refs).Purger.ProtectedTables(), workspace.Provide(pool, refs).Purger.ProtectedTables())
}

// The entire FK graph is read independently of both the declarations and the
// selected purge sources. NO ACTION is safe only with the exact dependency and
// every transitive CASCADE ancestor assigned to the protected page protocol.
func assetProtectionGaps(t testing.TB, pool *pgxpool.Pool, tables []string, refs []asset.ProtectedParentReference, steps []string) (map[foreignKey]bool, []string) {
	t.Helper()
	all := protectionKeys(t, pool)
	allowed := map[foreignKey]bool{}
	var gaps []string
	declared := map[string]int{}
	ancestors := map[string]bool{}
	for _, ref := range refs {
		declared[ref.Constraint]++
		var found *protectionKey
		for _, row := range all {
			if row.ref == ref {
				copy := row
				found = &copy
				break
			}
		}
		if found == nil || found.action != "a" || found.columns != 1 || found.parentColumns != 1 || found.parentColumn != "id" || found.deferred || found.deferrable || !found.indexed {
			gaps = append(gaps, "invalid protected reference "+ref.Constraint)
			continue
		}
		allowed[foreignKey{ref.Parent, ref.Table, ref.Column}] = true
		if slices.Contains(tables, ref.Parent) {
			ancestors[ref.Parent] = true
		}
	}
	for name, n := range declared {
		if n != 1 {
			gaps = append(gaps, fmt.Sprintf("protected reference %s declared %d times", name, n))
		}
	}
	// All six business parent columns must be declared, including user_id,
	// whose target is intentionally outside the physical purge inventory.
	for _, row := range all {
		if row.ref.Table != "file_assets" || slices.Contains([]string{"created_by_id", "updated_by_id", "deleted_by_id"}, row.ref.Column) {
			continue
		}
		if !slices.Contains(refs, row.ref) {
			gaps = append(gaps, "undeclared asset parent "+row.ref.Constraint)
		}
	}
	for changed := true; changed; {
		changed = false
		for _, edge := range all {
			if edge.action == "c" && ancestors[edge.ref.Table] && slices.Contains(tables, edge.ref.Parent) && !ancestors[edge.ref.Parent] {
				ancestors[edge.ref.Parent] = true
				changed = true
			}
		}
	}
	seen := map[string]int{}
	for _, step := range steps {
		seen[step]++
		if !ancestors[step] {
			gaps = append(gaps, "unexpected protected step "+step)
		}
	}
	for parent := range ancestors {
		if seen[parent] != 1 {
			gaps = append(gaps, fmt.Sprintf("protected ancestor %s has %d steps", parent, seen[parent]))
		}
	}
	return allowed, gaps
}

func TestAssetProtectedCatalogRejectsMissingAncestorsAndConstraints(t *testing.T) {
	pool := openPool(t, pgtest.NewDatabase(t))
	tables := purgeTables(t, pool)
	refs := asset.ProtectedParentReferences()
	steps := protectedPurgeSteps(pool)
	if _, gaps := assetProtectionGaps(t, pool, tables, refs, steps); len(gaps) > 0 {
		t.Fatal(gaps)
	}
	for i, step := range steps {
		copy := append(slices.Clone(steps[:i]), steps[i+1:]...)
		if _, gaps := assetProtectionGaps(t, pool, tables, refs, copy); !strings.Contains(strings.Join(gaps, ";"), "protected ancestor "+step+" has 0 steps") {
			t.Fatalf("accepted missing ancestor %s: %v", step, gaps)
		}
	}
	if !slices.Contains(steps, "states") {
		t.Fatal("states CASCADE ancestor absent")
	}
	for i, ref := range refs {
		copy := append(slices.Clone(refs[:i]), refs[i+1:]...)
		if _, gaps := assetProtectionGaps(t, pool, tables, copy, steps); !strings.Contains(strings.Join(gaps, ";"), "undeclared asset parent "+ref.Constraint) {
			t.Fatalf("accepted missing protected edge %s: %v", ref.Constraint, gaps)
		}
	}
	for _, action := range []string{"CASCADE", "SET NULL", "RESTRICT", "NO ACTION DEFERRABLE"} {
		_, err := pool.Exec(t.Context(), "ALTER TABLE file_assets DROP CONSTRAINT file_assets_issue_fk, ADD CONSTRAINT file_assets_issue_fk FOREIGN KEY(issue_id) REFERENCES issues(id) ON DELETE "+action)
		if err != nil {
			t.Fatal(err)
		}
		if _, gaps := assetProtectionGaps(t, pool, tables, refs, steps); !strings.Contains(strings.Join(gaps, ";"), "invalid protected reference file_assets_issue_fk") {
			t.Fatalf("accepted action %s: %v", action, gaps)
		}
	}
}
