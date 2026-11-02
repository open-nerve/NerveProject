package bootstrap

import (
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/local"
	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/storagetest"
	assetapp "github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	assetdomain "github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

func recoverAssetRows(t *testing.T, w issueWorld, at time.Time) {
	t.Helper()
	objects, err := local.New(local.Config{Root: t.TempDir(), SizeLimit: assetdomain.MaxObjectSize}, local.Deps{UploadMAC: storagetest.MAC("upload"), ReadMAC: storagetest.MAC("read"), RuntimeNow: time.Now})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := objects.Close(); err != nil {
			t.Error(err)
		}
	}()
	rows := assetpg.New(w.pool)
	parents := assetParentReader{workspaces: workspace.Provide(w.pool, protectedAssetReferences()).AssetParents, projects: project.Provide(w.pool, protectedAssetReferences()).AssetParents, issues: issue.Provide(w.pool, protectedAssetReferences()).AssetParents}
	uc := assetapp.NewCleanup(assetapp.CleanupDeps{Rows: rows, Locks: rows, Writer: rows, Parents: parents, Objects: objects, Inventory: objects, Temporary: objects, Tx: postgres.NewTxManager(w.pool, time.Second), Clock: clockAt(at), RuntimeNow: time.Now, Batch: 2, Budget: time.Second, OperationTimeout: time.Second})
	var cursor assetapp.SweepCursor
	for page := 0; page < 100; page++ {
		r, err := uc.Execute(t.Context(), cursor)
		if err != nil {
			t.Fatal(err)
		}
		if r.Complete {
			return
		}
		cursor = r.Cursor
	}
	t.Fatal("asset recovery did not converge")
}
