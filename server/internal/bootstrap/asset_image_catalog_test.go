package bootstrap

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

type imageReferenceProof struct {
	allowed     map[foreignKey]bool
	constraints []uint32
}

// This independent inventory describes reference edges, not ownership edges.
// It permits no extra NO ACTION relationship or missing owner capability.
func imageReferenceCatalog(t testing.TB, pool *pgxpool.Pool, steps []imageReferenceStep) (imageReferenceProof, []string) {
	t.Helper()
	required := map[string][]imageReferenceDeclaration{
		"identity":  {{"public", "users", "users_avatar_asset_fk", "avatar_asset_id", "file_assets"}, {"public", "users", "users_cover_image_asset_fk", "cover_image_asset_id", "file_assets"}},
		"workspace": {{"public", "workspaces", "workspaces_logo_asset_fk", "logo_asset_id", "file_assets"}},
		"project":   {{"public", "projects", "projects_cover_image_asset_fk", "cover_image_asset_id", "file_assets"}},
	}
	proof := imageReferenceProof{allowed: map[foreignKey]bool{}}
	var gaps []string
	var names []string
	var refs []imageReferenceDeclaration
	for _, step := range steps {
		names = append(names, step.owner)
		if step.detach == nil || !slices.Equal(step.references, required[step.owner]) {
			gaps = append(gaps, "invalid image detacher "+step.owner)
		}
		refs = append(refs, step.references...)
	}
	if !slices.Equal(names, []string{"identity", "workspace", "project"}) {
		gaps = append(gaps, "invalid image detacher order/count")
	}
	all := protectionKeys(t, pool)
	for owner, want := range required {
		for _, ref := range want {
			if !slices.Contains(refs, ref) {
				gaps = append(gaps, "undeclared image reference "+owner+"."+ref.Column)
			}
			found := false
			for _, row := range all {
				actual := imageReferenceDeclaration{row.ref.Schema, row.ref.Table, row.ref.Constraint, row.ref.Column, row.ref.Parent}
				if actual != ref {
					continue
				}
				found = true
				if row.action != "a" || row.columns != 1 || row.parentColumns != 1 || row.parentColumn != "id" || row.deferred || row.deferrable || !row.indexed {
					gaps = append(gaps, "invalid image reference "+ref.Constraint)
					continue
				}
				proof.allowed[foreignKey{ref.Parent, ref.Table, ref.Column}] = true
				proof.constraints = append(proof.constraints, row.oid)
			}
			if !found {
				gaps = append(gaps, "missing image reference "+ref.Constraint)
			}
		}
	}
	for _, row := range all {
		if row.ref.Parent != "file_assets" {
			continue
		}
		actual := imageReferenceDeclaration{row.ref.Schema, row.ref.Table, row.ref.Constraint, row.ref.Column, row.ref.Parent}
		known := false
		for _, want := range required {
			if slices.Contains(want, actual) {
				known = true
			}
		}
		if !known {
			gaps = append(gaps, "unknown incoming image reference "+row.ref.Table+"."+row.ref.Constraint)
		}
	}
	return proof, gaps
}

func validatedImageReferenceConstraints(t testing.TB, pool *pgxpool.Pool) []uint32 {
	t.Helper()
	proof, gaps := imageReferenceCatalog(t, pool, providedImageReferences(pool).steps())
	if len(gaps) > 0 {
		t.Fatalf("%s", strings.Join(gaps, "; "))
	}
	return proof.constraints
}

func TestImageReferenceCatalog(t *testing.T) {
	pool := openPool(t, pgtest.NewDatabase(t))
	provided := providedImageReferences(pool)
	steps := provided.steps()
	proof, gaps := imageReferenceCatalog(t, pool, steps)
	if len(gaps) != 0 || len(proof.allowed) != 4 || len(proof.constraints) != 4 {
		t.Fatalf("image proof=%+v gaps=%v", proof, gaps)
	}
	for i := range steps {
		missing := append(slices.Clone(steps[:i]), steps[i+1:]...)
		if _, gaps := imageReferenceCatalog(t, pool, missing); len(gaps) == 0 {
			t.Fatalf("accepted missing step %s", steps[i].owner)
		}
		for j := range steps[i].references {
			altered := provided.steps()
			altered[i].references = append(altered[i].references[:j], altered[i].references[j+1:]...)
			if _, gaps := imageReferenceCatalog(t, pool, altered); len(gaps) == 0 {
				t.Fatal("accepted undeclared reference")
			}
		}
	}
	slices.Reverse(steps)
	if _, gaps := imageReferenceCatalog(t, pool, steps); len(gaps) == 0 {
		t.Fatal("accepted reversed owner order")
	}
	steps = provided.steps()
	steps[0].references[0].Column = "wrong"
	if _, gaps := imageReferenceCatalog(t, pool, provided.steps()); len(gaps) != 0 {
		t.Fatal("declarations expose shared mutable slice")
	}
	if _, err := pool.Exec(t.Context(), "CREATE TABLE image_probe(id uuid PRIMARY KEY, asset_id uuid CONSTRAINT users_avatar_asset_fk REFERENCES file_assets(id))"); err != nil {
		t.Fatal(err)
	}
	if _, gaps := imageReferenceCatalog(t, pool, provided.steps()); !strings.Contains(strings.Join(gaps, ";"), "unknown incoming image reference image_probe.users_avatar_asset_fk") {
		t.Fatal("accepted names-only unknown reference", gaps)
	}
}

func TestImageReferenceCatalogRejectsUnsafeEdges(t *testing.T) {
	for _, change := range []string{"CASCADE", "SET NULL", "RESTRICT", "NO ACTION DEFERRABLE", "missing index"} {
		t.Run(change, func(t *testing.T) {
			pool := openPool(t, pgtest.NewDatabase(t))
			sql := "ALTER TABLE users DROP CONSTRAINT users_avatar_asset_fk,ADD CONSTRAINT users_avatar_asset_fk FOREIGN KEY(avatar_asset_id) REFERENCES file_assets(id) ON DELETE " + change
			if change == "missing index" {
				sql = "DROP INDEX users_avatar_asset_id_idx"
			}
			if _, err := pool.Exec(context.Background(), sql); err != nil {
				t.Fatal(err)
			}
			_, gaps := imageReferenceCatalog(t, pool, providedImageReferences(pool).steps())
			if !strings.Contains(strings.Join(gaps, ";"), "invalid image reference users_avatar_asset_fk") {
				t.Fatalf("accepted %s: %v", change, gaps)
			}
			for _, parent := range []string{"workspaces", "projects", "issues"} {
				failed := fatalOf(func(tb testing.TB) { keysTo(tb, pool, parent) })
				if !strings.Contains(failed, "invalid image reference") {
					t.Fatal("recursive deletion guard ignored invalid reference", fmt.Sprint(parent, failed))
				}
			}
		})
	}
}
