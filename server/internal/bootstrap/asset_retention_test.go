package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/riverqueue/river"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	assetriver "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/river"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
)

// Only tombstone age and the idle objects' safety timestamps are persistence
// fixtures. Upload, confirmation, deletion and both periodic workers are real;
// this test does not claim to prove wall-clock capability expiry.
func TestAssetConfiguredRetentionWired(t *testing.T) {
	for _, tc := range []struct {
		retention, age int
		removed        bool
	}{{90, 61, false}, {1, 2, true}} {
		t.Run(fmt.Sprintf("%dd_retention_%dd_old", tc.retention, tc.age), func(t *testing.T) {
			w, objects := newAssetCompleteWorldWith(t, func(c *config.Config) {
				c.Cleanup.PurgeAfter = time.Duration(tc.retention) * 24 * time.Hour
				c.Cleanup.PurgeInterval = time.Second
			}, func(*assetComponents) {})
			waitForCompletedKinds(t, w.pool, map[string]int{"asset.cleanup": 1}, &lockedBuffer{})
			item := w.newWorkItem(t)
			file := uploadedAssetHTTP(t, w, "bob", "ISSUE_ATTACHMENT", assetScopeBody("issue", item), "retained.txt", "text/plain", []byte("old"))
			row, err := assetpg.New(w.pool).FindAsset(t.Context(), file.Asset.ID)
			if err != nil || row.Metadata.Final == nil {
				t.Fatalf("published file: %+v %v", row, err)
			}
			key := row.Metadata.Final.Key
			w.send(t, "alice", "DELETE", "/api/v0/issues/"+item.String(), "", 204)
			var marker int64
			if err := w.pool.QueryRow(t.Context(), "SELECT coalesce(max(id),0) FROM river_job WHERE kind='asset.cleanup'").Scan(&marker); err != nil {
				t.Fatal(err)
			}
			inserter, err := jobs.NewInserter(w.pool)
			if err != nil {
				t.Fatal(err)
			}
			tx, err := w.pool.Begin(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = tx.Rollback(context.Background()) }()
			at := time.Now().UTC().Add(-time.Duration(tc.age) * 24 * time.Hour)
			if _, err := tx.Exec(t.Context(), "UPDATE issues SET deleted_at=$2 WHERE id=$1", item, at); err != nil {
				t.Fatal(err)
			}
			if _, err := tx.Exec(t.Context(), `UPDATE file_assets SET deleted_at=$2,
 storage_metadata=jsonb_set(jsonb_set(storage_metadata,'{staging,safe_after}',to_jsonb($3::timestamptz)),'{final,safe_after}',to_jsonb($3::timestamptz)),
 storage_revision=storage_revision+1 WHERE id=$1`, file.Asset.ID, at, time.Now().Add(-time.Hour)); err != nil {
				t.Fatal(err)
			}
			if err := inserter.InsertTx(t.Context(), tx, assetriver.CleanupArgs{}, &river.InsertOpts{Queue: "periodic"}); err != nil {
				t.Fatal(err)
			}
			if err := tx.Commit(t.Context()); err != nil {
				t.Fatal(err)
			}
			requestEventuallyWithin(t, "a fresh wired asset sweep", 15*time.Second, func(ctx context.Context) bool {
				var completed bool
				if err := w.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM river_job WHERE kind='asset.cleanup' AND id>$1 AND state='completed')", marker).Scan(&completed); err != nil {
					t.Fatal(err)
				}
				return completed
			})
			_, err = objects.Inspect(t.Context(), key)
			if (err != nil && !errors.Is(err, domain.ErrObjectNotFound)) || errors.Is(err, domain.ErrObjectNotFound) != tc.removed {
				t.Fatalf("configured %dd retention: %dd final removed=%t want=%t: %v", tc.retention, tc.age, errors.Is(err, domain.ErrObjectNotFound), tc.removed, err)
			}
			requestEventuallyWithin(t, "configured asset and parent purge", 15*time.Second, func(ctx context.Context) bool {
				var assetPresent, issuePresent bool
				if err := w.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM file_assets WHERE id=$1),EXISTS(SELECT 1 FROM issues WHERE id=$2)", file.Asset.ID, item).Scan(&assetPresent, &issuePresent); err != nil {
					t.Fatal(err)
				}
				return assetPresent == !tc.removed && issuePresent == !tc.removed
			})
		})
	}
}
