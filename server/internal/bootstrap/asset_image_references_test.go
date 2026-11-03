package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset"
	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	assetapp "github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	assetdomain "github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

type imageReferenceWorld struct {
	pool        *pgxpool.Pool
	user        uuid.UUID
	tree        assetPurgeTree
	images      []assetdomain.Asset
	now, cutoff time.Time
}

// This is a persistence/lock fixture, not a claim that the future public image
// APIs work. T11/T12 create the same references through authenticated APIs.
func newImageReferenceWorld(t *testing.T) imageReferenceWorld {
	t.Helper()
	return seedImageReferenceWorld(t, openPool(t, pgtest.NewDatabase(t)))
}
func seedImageReferenceWorld(t *testing.T, pool *pgxpool.Pool) imageReferenceWorld {
	t.Helper()
	w := imageReferenceWorld{pool: pool, user: uuid.NewV7(), now: time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)}
	w.cutoff = w.now.Add(-60 * 24 * time.Hour)
	if _, err := w.pool.Exec(t.Context(), "INSERT INTO users(id,email,password,display_name) VALUES($1,$2,'x','Images')", w.user, w.user.String()+"@example.com"); err != nil {
		t.Fatal(err)
	}
	w.tree = seedAssetPurgeTree(t, w.pool, w.user)
	old := w.now.Add(-61 * 24 * time.Hour)
	for _, purpose := range []assetdomain.Purpose{assetdomain.UserAvatar, assetdomain.UserCover, assetdomain.WorkspaceLogo, assetdomain.ProjectCover} {
		p := assetdomain.Placement{UserID: w.user, Scope: assetdomain.Scope{Kind: assetdomain.ScopeUser}}
		switch purpose {
		case assetdomain.WorkspaceLogo:
			p.WorkspaceID = w.tree.workspace
			p.Scope = assetdomain.Scope{Kind: assetdomain.ScopeWorkspace, ID: w.tree.workspace}
		case assetdomain.ProjectCover:
			p.WorkspaceID = w.tree.workspace
			p.ProjectID = w.tree.project
			p.Scope = assetdomain.Scope{Kind: assetdomain.ScopeProject, ID: w.tree.project}
		}
		a := assetdomain.Asset{ID: uuid.NewV7(), Purpose: purpose, Placement: p, Name: "image.png", ContentType: "image/png", Size: 3, Uploaded: true, BoundAt: &old, CreatedAt: old, UpdatedAt: old, DeletedAt: &old, DeletedByID: &w.user, CreatedByID: &w.user, UpdatedByID: &w.user,
			Metadata: assetdomain.StorageMetadata{SchemaVersion: 1, Origin: assetdomain.OriginCopy, Candidates: []assetdomain.Candidate{}}}
		got, err := assetpg.New(w.pool).CreateAsset(t.Context(), a)
		if err != nil {
			t.Fatal(err)
		}
		w.images = append(w.images, got)
	}
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{"UPDATE users SET avatar_asset_id=$1,cover_image_asset_id=$2 WHERE id=$3", []any{w.images[0].ID, w.images[1].ID, w.user}},
		{"UPDATE workspaces SET logo_asset_id=$1 WHERE id=$2", []any{w.images[2].ID, w.tree.workspace}},
		{"UPDATE projects SET cover_image_asset_id=$1 WHERE id=$2", []any{w.images[3].ID, w.tree.project}},
	} {
		if _, err := w.pool.Exec(t.Context(), q.sql, q.args...); err != nil {
			t.Fatal(err)
		}
	}
	return w
}

func imageReferencePurger(pool *pgxpool.Pool) *assetapp.Purger {
	return asset.NewPurger(asset.Provide(pool), providedImageReferences(pool), postgres.NewTxManager(pool, time.Second))
}

func (w imageReferenceWorld) references(t *testing.T) []*uuid.UUID {
	t.Helper()
	out := make([]*uuid.UUID, 4)
	if err := w.pool.QueryRow(t.Context(), "SELECT avatar_asset_id,cover_image_asset_id FROM users WHERE id=$1", w.user).Scan(&out[0], &out[1]); err != nil {
		t.Fatal(err)
	}
	if err := w.pool.QueryRow(t.Context(), "SELECT logo_asset_id FROM workspaces WHERE id=$1", w.tree.workspace).Scan(&out[2]); err != nil {
		t.Fatal(err)
	}
	if err := w.pool.QueryRow(t.Context(), "SELECT cover_image_asset_id FROM projects WHERE id=$1", w.tree.project).Scan(&out[3]); err != nil {
		t.Fatal(err)
	}
	return out
}
func (w imageReferenceWorld) audit(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, table := range []string{"users", "workspaces", "projects"} {
		id := w.user
		if table == "workspaces" {
			id = w.tree.workspace
		}
		if table == "projects" {
			id = w.tree.project
		}
		var s string
		if err := w.pool.QueryRow(t.Context(), "SELECT (to_jsonb(r)-'avatar_asset_id'-'cover_image_asset_id'-'logo_asset_id')::text FROM "+table+" r WHERE id=$1", id).Scan(&s); err != nil {
			t.Fatal(err)
		}
		out = append(out, s)
	}
	return out
}
func TestImageReferencePurge(t *testing.T) {
	w := newImageReferenceWorld(t)
	before := w.audit(t)
	for _, q := range []struct {
		sql string
		id  uuid.UUID
	}{
		{"DELETE FROM file_assets WHERE id=$1", w.images[0].ID}, {"DELETE FROM users WHERE id=$1", w.user},
	} {
		_, err := w.pool.Exec(t.Context(), q.sql, q.id)
		var pe *pgconn.PgError
		if !errors.As(err, &pe) || pe.Code != "23503" {
			t.Fatalf("missing two-way FK protection: %v", err)
		}
	}
	p := imageReferencePurger(w.pool)
	count, err := drainPurgeModule(forAssetPurge(p), t.Context(), w.cutoff, 1)
	if err != nil || count != 4 {
		t.Fatalf("image reference purge=%d %v", count, err)
	}
	if !reflect.DeepEqual(w.references(t), make([]*uuid.UUID, 4)) {
		t.Fatal("owner references survived")
	}
	if !reflect.DeepEqual(before, w.audit(t)) {
		t.Fatal("purge rewrote owner audit")
	}
}

func TestImageReferencePurgeEligibility(t *testing.T) {
	w := newImageReferenceWorld(t)
	// Smallest ID still owns an object; middle cover is eligible; last is 59d.
	a := w.images[0]
	key, err := assetdomain.NewObjectKey('o', a.ID, uuid.NewV7())
	if err != nil {
		t.Fatal(err)
	}
	a.Metadata.Final = &assetdomain.ObjectRecord{Key: key, Phase: assetdomain.ObjectActive, SafeAfter: w.now, Size: 3, ContentType: "image/png", ETag: "tag", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	if _, err := assetpg.New(w.pool).SaveAsset(t.Context(), a); err != nil {
		t.Fatal(err)
	}
	if _, err := w.pool.Exec(t.Context(), "UPDATE file_assets SET deleted_at=NULL,is_deleted=false WHERE id=$1", w.images[2].ID); err != nil {
		t.Fatal(err)
	}
	if _, err := w.pool.Exec(t.Context(), "UPDATE file_assets SET deleted_at=$2 WHERE id=$1", w.images[3].ID, w.now.Add(-59*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	count, err := drainPurgeModule(forAssetPurge(imageReferencePurger(w.pool)), t.Context(), w.cutoff, 1)
	if err != nil || count != 1 {
		t.Fatalf("eligibility=%d %v", count, err)
	}
	for i, ref := range w.references(t) {
		if i == 1 {
			if ref != nil {
				t.Fatal("eligible middle cover survived")
			}
		} else if ref == nil || *ref != w.images[i].ID {
			t.Fatalf("ineligible reference %d cleared", i)
		}
	}
}

func TestImageReferencePurgeRollback(t *testing.T) {
	w := newImageReferenceWorld(t)
	before := w.references(t)
	audit := w.audit(t)
	if _, err := w.pool.Exec(t.Context(), `CREATE FUNCTION reject_image_purge() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'purge fault'; END $$;
 CREATE TRIGGER reject_image_purge BEFORE DELETE ON file_assets FOR EACH ROW EXECUTE FUNCTION reject_image_purge()`); err != nil {
		t.Fatal(err)
	}
	cursor := assetapp.PurgeCursor{Table: "file_assets"}
	page, err := imageReferencePurger(w.pool).Page(t.Context(), w.cutoff, 100, cursor)
	var pe *pgconn.PgError
	if !errors.As(err, &pe) || pe.Code != "P0001" || page.Cursor != cursor {
		t.Fatalf("error/cursor=%+v %v", page, err)
	}
	if !reflect.DeepEqual(before, w.references(t)) || !reflect.DeepEqual(audit, w.audit(t)) {
		t.Fatal("failed delete left cleared references/audit")
	}
	for _, a := range w.images {
		if _, err := assetpg.New(w.pool).FindAsset(context.Background(), a.ID); err != nil {
			t.Fatal(err)
		}
	}
}

func providedImageReferences(pool *pgxpool.Pool) assetImageReferences {
	return assetImageReferences{identity.Provide(pool).ImageReferences, workspace.Provide(pool, protectedAssetReferences()).ImageReferences, project.Provide(pool, protectedAssetReferences()).ImageReferences}
}
