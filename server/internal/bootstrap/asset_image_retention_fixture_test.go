package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset"
	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	assetapp "github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	assetdomain "github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

type imageRetentionWorld struct {
	issueWorld
	objects assetStore
	stop    func()
	dbURL   string
	root    string
	now     time.Time
}

func newImageRetentionWorld(t *testing.T) imageRetentionWorld {
	t.Helper()
	db := pgtest.NewDatabase(t)
	var objects assetStore
	assets := func(ctx context.Context, c config.FilesConfig, keys *identity.Keys, logger *slog.Logger) (assetComponents, error) {
		out, err := newAssets(ctx, c, keys, logger)
		objects = out.store
		return out, err
	}
	cfg := testConfig(t, db, false)
	w, stop := startImageAPIWorld(t, cfg, appFactories{pool: postgres.NewPool, assets: assets})
	return imageRetentionWorld{issueWorld: w, objects: objects, stop: stop, dbURL: db, root: cfg.Files.Local.Root, now: time.Now().UTC().Truncate(time.Microsecond)}
}

// Every positive reference, including the full project graph, is made through
// real upload/completion/owner APIs. SQL below only ages or corrupts persistence
// after the completed request; it cannot make an image binding succeed.
func (w imageRetentionWorld) group(t *testing.T, name string, deleted bool) imageReferenceWorld {
	t.Helper()
	w.tokens[name] = registerAccount(t, w.contract, w.base, name+"@example.com").AccessToken
	w.ids[name] = accountID(t, w.contract, w.base, w.tokens[name])
	var made struct{ ID uuid.UUID }
	decodeAnswer(t, w.send(t, name, "POST", "/api/v0/workspaces", fmt.Sprintf(`{"name":%q,"slug":%q}`, name, name), 201), &made)
	g := imageReferenceWorld{pool: w.pool, user: w.ids[name], now: w.now, cutoff: w.now.Add(-60 * 24 * time.Hour)}
	g.tree.workspace = made.ID
	g.tree.project = createdProject(t, w.contract, w.base, w.tokens[name], name, "Images", "IMG")
	g.tree.issue = w.newWorkItemBy(t, name, g.tree.project, `{"name":"Retained tree"}`)
	g.tree.state = stateID(t, w.pool, g.tree.project, "Todo")
	for _, tc := range []struct{ purpose, scope, path, field string }{
		{"USER_AVATAR", `{"kind":"user"}`, "/api/v0/me", "avatar_asset_id"},
		{"USER_COVER", `{"kind":"user"}`, "/api/v0/me", "cover_image_asset_id"},
		{"WORKSPACE_LOGO", assetScopeBody("workspace", g.tree.workspace), "/api/v0/workspaces/" + name, "logo_asset_id"},
		{"PROJECT_COVER", assetScopeBody("project", g.tree.project), "/api/v0/projects/" + g.tree.project.String(), "cover_image_asset_id"},
	} {
		id := uploadedAssetHTTP(t, w.issueWorld, name, tc.purpose, tc.scope, "image.png", "image/png", assetTestPNG(t)).Asset.ID
		w.send(t, name, "PATCH", tc.path, fmt.Sprintf(`{%q:%q}`, tc.field, id), 200)
		a, err := assetpg.New(w.pool).FindAsset(t.Context(), id)
		if err != nil || a.BoundAt == nil || a.Metadata.Final == nil {
			t.Fatal("API did not bind the real image", a, err)
		}
		g.images = append(g.images, a)
	}
	if deleted {
		w.send(t, name, "DELETE", "/api/v0/workspaces/"+name, "", 204)
		waitWorkspaceDeletion(t, w.pool, g.tree.workspace)
	}
	return g
}

func (w imageRetentionWorld) age(t *testing.T, g imageReferenceWorld, days int, empty bool) {
	t.Helper()
	at := w.now.Add(-time.Duration(days) * 24 * time.Hour)
	for _, original := range g.images {
		a, err := assetpg.New(w.pool).FindAsset(t.Context(), original.ID)
		if err != nil {
			t.Fatal(err)
		}
		// User images ordinarily lose the old reference on replacement. Keeping
		// it while tombstoning here is an explicit stale-reference fault, covering
		// both reverse FKs without inventing a public user-deletion operation.
		a.DeletedAt, a.DeletedByID = &at, &g.user
		for _, record := range []*assetdomain.ObjectRecord{a.Metadata.Staging, a.Metadata.Final} {
			if record == nil {
				continue
			}
			record.SafeAfter = w.now.Add(-time.Hour)
			if empty {
				if err := w.objects.Delete(t.Context(), record.Key); err != nil {
					t.Fatal(err)
				}
			}
		}
		if len(a.Metadata.Candidates) != 0 {
			t.Fatal("unexpected unfinished publication", a.ID)
		}
		if empty {
			a.Metadata.Staging, a.Metadata.Final = nil, nil
		}
		if _, err := assetpg.New(w.pool).SaveAsset(t.Context(), a); err != nil {
			t.Fatal(err)
		}
	}
	for _, table := range []string{"workspaces", "projects", "issues"} {
		if _, err := w.pool.Exec(t.Context(), "UPDATE "+table+" SET deleted_at=$2 WHERE id=$1 AND deleted_at IS NOT NULL", g.tree.parent(table), at); err != nil {
			t.Fatal(err)
		}
	}
}

func imagePurgeModules(pool *pgxpool.Pool) []purgeModule {
	return []purgeModule{
		{"asset", forAssetPurge(asset.NewPurger(asset.Provide(pool), providedImageReferences(pool), postgres.NewTxManager(pool, time.Second)))},
		{"issue", forIssuePurge(issue.Provide(pool, protectedAssetReferences()).Purger)},
		{"planning", forPlanningPurge(planning.Provide(pool).Purger)},
		{"project", forProjectPurge(project.Provide(pool, protectedAssetReferences()).Purger)},
		{"workspace", forWorkspacePurge(workspace.Provide(pool, protectedAssetReferences()).Purger)},
		{"identity", forIdentityPurge(identity.Provide(pool).Purger)},
	}
}

func purgeImageTree(t *testing.T, w imageRetentionWorld) purgeResult {
	t.Helper()
	uc := newPurgeDeleted(imagePurgeModules(w.pool), clockAt(w.now), time.Now, 60*24*time.Hour, 1, 5*time.Second)
	var cursor purgeCursor
	var total purgeResult
	for range 100 {
		result, err := uc.Execute(t.Context(), cursor)
		if err != nil {
			t.Fatal(err)
		}
		if !result.Cursor.Cutoff.Equal(w.now.Add(-60 * 24 * time.Hour)) {
			t.Fatal("cutoff drift", result)
		}
		total.Scanned += result.Scanned
		total.Deleted += result.Deleted
		total.Blocked += result.Blocked
		if result.Complete {
			return total
		}
		cursor = result.Cursor
	}
	t.Fatal("image tree purge did not finish")
	return total
}

var errImageDelete = errors.New("specific image key unavailable")

type imageDeleteFault struct {
	assetStore
	keys map[assetdomain.ObjectKey]bool
}

func (s imageDeleteFault) Delete(ctx context.Context, key assetdomain.ObjectKey) error {
	if s.keys[key] {
		return errImageDelete
	}
	return s.assetStore.Delete(ctx, key)
}

func cleanImageObjects(t *testing.T, w imageRetentionWorld, objects assetStore, allowed ...error) int {
	t.Helper()
	rows := assetpg.New(w.pool)
	parents := assetParentReader{workspaces: workspace.Provide(w.pool, protectedAssetReferences()).AssetParents, projects: project.Provide(w.pool, protectedAssetReferences()).AssetParents, issues: issue.Provide(w.pool, protectedAssetReferences()).AssetParents}
	uc := assetapp.NewCleanup(assetapp.CleanupDeps{Rows: rows, Locks: rows, Writer: rows, Parents: parents, Objects: objects, Inventory: objects, Temporary: objects,
		Tx: postgres.NewTxManager(w.pool, time.Second), Clock: clockAt(w.now), RuntimeNow: time.Now, Batch: 1, Budget: time.Second, OperationTimeout: time.Second, PurgeAfter: 60 * 24 * time.Hour})
	var cursor assetapp.SweepCursor
	failed := 0
	for range 100 {
		result, err := uc.Execute(t.Context(), cursor)
		accepted := errors.Is(err, errImageDelete)
		for _, expected := range allowed {
			accepted = accepted || errors.Is(err, expected)
		}
		if err != nil && !accepted {
			t.Fatal(err)
		}
		failed += result.Failed
		if result.Complete {
			return failed
		}
		cursor = result.Cursor
	}
	t.Fatal("image cleanup did not converge")
	return 0
}
