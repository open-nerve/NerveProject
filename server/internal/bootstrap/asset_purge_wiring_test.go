package bootstrap

import (
	"context"
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
)

// Only tombstone aging uses the observer connection. The running application
// must wire the declared FK identities into the actual periodic parent purge.
func TestAssetProtectedParentDoesNotBlockWiredPurge(t *testing.T) {
	w := newIssueWorldWith(t, func(c *config.Config) { c.Cleanup.PurgeInterval = time.Second; c.Cleanup.PurgeBatch = 1 })
	waitForCompletedKinds(t, w.pool, map[string]int{"asset.cleanup": 1}, &lockedBuffer{})
	a := seedIssueAsset(t, w.pool, w.newWorkItem(t), w.ids["bob"])
	free := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Free", "FREE")
	for _, id := range []uuid.UUID{w.web, free} {
		w.send(t, "alice", http.MethodDelete, "/api/v0/projects/"+id.String(), "", http.StatusNoContent)
		waitProjectDeletion(t, w.pool, id)
	}
	var boundary time.Time
	if err := w.pool.QueryRow(t.Context(), "UPDATE projects SET deleted_at=clock_timestamp()-interval '61 days' WHERE id=ANY($1) RETURNING clock_timestamp()", []uuid.UUID{w.web, free}).Scan(&boundary); err != nil {
		t.Fatal(err)
	}
	requestEventuallyWithin(t, "wired purge skips asset-protected parent and deletes neighbour", 15*time.Second, func(ctx context.Context) bool {
		var protected, unrelated, finished bool
		if err := w.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects WHERE id=$1),EXISTS(SELECT 1 FROM projects WHERE id=$2),
 EXISTS(SELECT 1 FROM river_job WHERE kind='cleanup.purge_deleted' AND state='completed' AND finalized_at>$3 AND (metadata->'output'->>'blocked')::integer>0)`, w.web, free, boundary).Scan(&protected, &unrelated, &finished); err != nil {
			t.Fatal(err)
		}
		if !protected {
			t.Fatal("asset-protected parent disappeared")
		}
		return !unrelated && finished
	})
	var intact bool
	if err := w.pool.QueryRow(t.Context(), "SELECT deleted_at IS NOT NULL AND deleted_at>now()-interval '60 days' FROM file_assets WHERE id=$1", a.ID).Scan(&intact); err != nil || !intact {
		t.Fatalf("purge lost retained asset: %t %v", intact, err)
	}
}
