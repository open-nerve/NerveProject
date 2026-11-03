package bootstrap

import (
	"context"
	"encoding/json"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	assetriver "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/river"
	assetapp "github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

func holdAssetScopeDeletion(t *testing.T, pool *pgxpool.Pool, project uuid.UUID) pgx.Tx {
	t.Helper()
	var id uuid.UUID
	if err := pool.QueryRow(t.Context(), "SELECT id FROM file_assets WHERE project_id=$1 ORDER BY id LIMIT 1", project).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return holding(t, pool, "SELECT 1 FROM file_assets WHERE id=$1 FOR UPDATE", id)
}

// Observe the real committed job and every original argument, not merely the
// eventual row state, which a separate periodic recovery could also produce.
func waitAssetScopeDeletion(t *testing.T, pool *pgxpool.Pool, scope string, id uuid.UUID) {
	t.Helper()
	var at time.Time
	var by *uuid.UUID
	table := "projects"
	if scope == "workspace" {
		table = "workspaces"
	}
	if err := pool.QueryRow(t.Context(), "SELECT deleted_at,updated_by_id FROM "+table+" WHERE id=$1", id).Scan(&at, &by); err != nil {
		t.Fatal(err)
	}
	var jobID int64
	requestEventuallyWithin(t, "asset scope deletion completed", 15*time.Second, func(ctx context.Context) bool {
		rows, err := pool.Query(ctx, "SELECT id,args,state::text,attempt,coalesce(cardinality(errors),0),queue,max_attempts FROM river_job WHERE kind='asset.delete_scope' AND args->>'scope'=$1 AND args->>'id'=$2", scope, id.String())
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		count := 0
		complete := false
		for rows.Next() {
			var gotID int64
			var raw []byte
			var state, queue string
			var attempt, failures, maxAttempts int
			if err := rows.Scan(&gotID, &raw, &state, &attempt, &failures, &queue, &maxAttempts); err != nil {
				t.Fatal(err)
			}
			var args assetriver.DeleteScopeArgs
			if err := json.Unmarshal(raw, &args); err != nil {
				t.Fatal(err)
			}
			if jobID != 0 && gotID != jobID {
				t.Fatal("asset job identity changed")
			}
			jobID = gotID
			if string(args.Scope) != scope || args.ID != id || !args.At.Equal(at) || (args.By == nil) != (by == nil) || (by != nil && *args.By != *by) || queue != "default" || maxAttempts != 25 {
				t.Fatalf("asset job lost original context: %+v", args)
			}
			if failures != 0 || attempt > 1 || state == "cancelled" || state == "discarded" {
				t.Fatalf("asset job %d failed: %s attempt=%d failures=%d", jobID, state, attempt, failures)
			}
			complete = state == "completed" && attempt == 1
			count++
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("asset scope deletion has %d jobs", count)
		}
		return complete
	})
	t.Logf("asset scope job %d completed with original %s/%s stamp", jobID, scope, id)
}

func assetDeletionJobForTest(pool *pgxpool.Pool, cfg config.Config) jobs.Job {
	rows := assetpg.New(pool)
	uc := assetapp.NewDeleteScope(assetapp.DeleteScopeDeps{Rows: rows, Locks: rows, Writer: rows, Tx: postgres.NewTxManager(pool, cfg.Database.CommitTimeout), Batch: cfg.Files.CleanupBatch, Budget: cfg.Files.CleanupBudget})
	return assetriver.DeleteScopeJob(uc, cfg.Files.CleanupBudget)
}
