package bootstrap

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

const imageProcessMode = "NERVE_TEST_IMAGE_PURGE_MODE"

func TestImagePurgeProcess(t *testing.T) {
	if mode := os.Getenv(imageProcessMode); mode != "" {
		runImagePurgeChild(t, mode)
		return
	}
	for _, mode := range []string{"page_committed", "page_failed"} {
		t.Run(mode, func(t *testing.T) {
			w := newImageRetentionWorld(t)
			old := w.group(t, "processold", true)
			young := w.group(t, "processyoung", true)
			live := w.group(t, "processlive", false)
			w.stop()
			w.age(t, old, 61, true)
			w.age(t, young, 59, true)
			beforeYoung, beforeLive := imageGroupSnapshot(t, young), imageGroupSnapshot(t, live)
			if _, err := w.pool.Exec(t.Context(), `CREATE TABLE image_purge_calls (
 n bigint GENERATED ALWAYS AS IDENTITY, process integer, mode text, after_id uuid, cutoff timestamptz,
 committed_after uuid, failed boolean)`); err != nil {
				t.Fatal(err)
			}
			if mode == "page_failed" {
				if _, err := w.pool.Exec(t.Context(), fmt.Sprintf(`CREATE FUNCTION reject_process_image() RETURNS trigger LANGUAGE plpgsql AS $$
 BEGIN IF OLD.id='%s'::uuid THEN RAISE EXCEPTION 'image page delete failed'; END IF; RETURN OLD; END $$;
 CREATE TRIGGER reject_process_image BEFORE DELETE ON file_assets FOR EACH ROW EXECUTE FUNCTION reject_process_image()`, old.images[1].ID)); err != nil {
					t.Fatal(err)
				}
			}
			client := imagePurgeClient(t, w.pool)
			job, err := client.Insert(t.Context(), purgeArgs{}, &river.InsertOpts{Queue: "periodic"})
			if err != nil {
				t.Fatal(err)
			}
			firstPID := imagePurgeProcess(t, w, mode, job.Job.ID, w.now)
			result, state := imagePurgeJob(t, w.pool, job.Job.ID)
			if result.Complete || result.Cursor.Version != 1 || result.Cursor.Module != "asset" || result.Cursor.Table != "file_assets" || result.Cursor.After != old.images[0].ID || !result.Cursor.Cutoff.Equal(old.cutoff) {
				t.Fatal("first process lost committed page position", state, result)
			}
			refs := old.references(t)
			for i, ref := range refs {
				if i == 0 {
					if ref != nil {
						t.Fatal("committed first page retained avatar reference")
					}
				} else if ref == nil || *ref != old.images[i].ID {
					t.Fatal("uncommitted page detached a reference", i)
				}
			}
			var present bool
			if err := w.pool.QueryRow(t.Context(), "SELECT EXISTS(SELECT 1 FROM file_assets WHERE id=$1)", old.images[0].ID).Scan(&present); err != nil || present {
				t.Fatal("first page was not committed", present, err)
			}
			if mode == "page_failed" {
				var failures int
				if err := w.pool.QueryRow(t.Context(), "SELECT count(*) FROM image_purge_calls WHERE failed AND after_id=$1", old.images[0].ID).Scan(&failures); err != nil || failures != 1 {
					t.Fatal("real second-page failure did not execute", failures, err)
				}
				if _, err := w.pool.Exec(t.Context(), "DROP TRIGGER reject_process_image ON file_assets"); err != nil {
					t.Fatal(err)
				}
			}
			// Use River's retry operation, not a hand-written metadata/cursor reset.
			if _, err := client.JobRetry(t.Context(), job.Job.ID); err != nil {
				t.Fatal(err)
			}
			secondPID := imagePurgeProcess(t, w, "resume", job.Job.ID, w.now.Add(30*24*time.Hour))
			if firstPID == secondPID || firstPID == os.Getpid() || secondPID == os.Getpid() {
				t.Fatal("purge did not switch OS processes", firstPID, secondPID)
			}
			result, state = imagePurgeJob(t, w.pool, job.Job.ID)
			if state != "completed" || !result.Complete || !result.Cursor.Cutoff.Equal(old.cutoff) {
				t.Fatal("same River job did not finish with original cutoff", state, result)
			}
			var after uuid.UUID
			var badCutoffs int
			if err := w.pool.QueryRow(t.Context(), "SELECT after_id FROM image_purge_calls WHERE process=$1 ORDER BY n LIMIT 1", secondPID).Scan(&after); err != nil || after != old.images[0].ID {
				t.Fatal("new process replayed or skipped a committed position", after, err)
			}
			if err := w.pool.QueryRow(t.Context(), "SELECT count(*) FROM image_purge_calls WHERE cutoff<>$1", old.cutoff).Scan(&badCutoffs); err != nil || badCutoffs != 0 {
				t.Fatal("page cutoff drift", badCutoffs, err)
			}
			assertImageRows(t, old, false)
			assertImageParents(t, old, false)
			if imageGroupSnapshot(t, young) != beforeYoung || imageGroupSnapshot(t, live) != beforeLive {
				t.Fatal("restart changed young/live rows or references")
			}
			t.Logf("mode=%s real PIDs=%d→%d same River job=%d resumes after=%s cutoff=%s", mode, firstPID, secondPID, job.Job.ID, after, result.Cursor.Cutoff)
		})
	}
}

func imagePurgeClient(t *testing.T, pool *pgxpool.Pool) *river.Client[pgx.Tx] {
	t.Helper()
	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func imagePurgeJob(t *testing.T, pool *pgxpool.Pool, id int64) (purgeResult, string) {
	t.Helper()
	var state string
	var metadata []byte
	if err := pool.QueryRow(t.Context(), "SELECT state,metadata FROM river_job WHERE id=$1", id).Scan(&state, &metadata); err != nil {
		t.Fatal(err)
	}
	result, present, err := decodePurgeOutput(metadata)
	if err != nil || !present {
		t.Fatal("missing real River output", state, string(metadata), err)
	}
	return result, state
}

func imagePurgeProcess(t *testing.T, w imageRetentionWorld, mode string, id int64, now time.Time) int {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, "-test.run=^TestImagePurgeProcess$", "-test.timeout=25s", "-test.v")
	cmd.Env = append(os.Environ(), imageProcessMode+"="+mode, "NERVE_TEST_IMAGE_PURGE_DB="+w.dbURL,
		"NERVE_TEST_IMAGE_PURGE_JOB="+strconv.FormatInt(id, 10), "NERVE_TEST_IMAGE_PURGE_NOW="+now.Format(time.RFC3339Nano))
	output, err := cmd.CombinedOutput()
	if err != nil || ctx.Err() != nil {
		t.Fatalf("image purge child %s failed: %v\n%s", mode, err, output)
	}
	t.Logf("%s", output)
	return cmd.Process.Pid
}
