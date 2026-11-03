package bootstrap

import (
	"fmt"
	"os"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	assetdomain "github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func insertContentPurgeJob(t *testing.T, client *river.Client[pgx.Tx]) int64 {
	t.Helper()
	job, err := client.Insert(t.Context(), purgeArgs{}, &river.InsertOpts{Queue: "periodic"})
	if err != nil {
		t.Fatal(err)
	}
	return job.Job.ID
}

// This uses the existing real child process/River page observer, with content
// files created through APIs instead of profile reverse-reference fixtures.
func TestContentAssetRetentionProcess(t *testing.T) {
	for _, scenario := range []struct {
		name                      string
		expireFirst, pendingPurge bool
	}{
		{"initial_budget_exhausted=false", false, false},
		{"initial_budget_exhausted=true", true, false},
		{"pending_periodic_job", false, true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			testContentAssetRetentionProcess(t, scenario.expireFirst, scenario.pendingPurge)
		})
	}
}

func testContentAssetRetentionProcess(t *testing.T, expireFirst, pendingPurge bool) {
	t.Helper()
	rw := newContentRetentionWorld(t)
	w := rw.issueWorld
	old := newContentProjectTree(t, w, "OLD")
	young := newContentProjectTree(t, w, "YOUNG")
	live := newContentProjectTree(t, w, "LIVE")
	deleteContentProject(t, w, old)
	deleteContentProject(t, w, young)
	rw.stop()
	ageContentTree(t, rw, old, 61)
	ageContentTree(t, rw, young, 59)
	if failed := cleanImageObjects(t, rw, rw.objects); failed != 0 {
		t.Fatal(failed)
	}
	beforeYoung, beforeLive := contentRowsSnapshot(t, w, young.rows), contentRowsSnapshot(t, w, live.rows)
	slices.SortFunc(old.rows, func(a, b assetdomain.Asset) int { return a.ID.Compare(b.ID) })
	first, next := old.rows[0].ID, old.rows[1].ID
	beforeNext := contentRowsSnapshot(t, w, []assetdomain.Asset{old.rows[1]})
	createImagePurgeObservation(t, w.pool)
	if _, err := w.pool.Exec(t.Context(), fmt.Sprintf(`CREATE FUNCTION reject_process_content() RETURNS trigger LANGUAGE plpgsql AS $$
 BEGIN IF OLD.id='%s'::uuid THEN RAISE EXCEPTION 'content page delete failed'; END IF; RETURN OLD; END $$;
 CREATE TRIGGER reject_process_content BEFORE DELETE ON file_assets FOR EACH ROW EXECUTE FUNCTION reject_process_content()`, next)); err != nil {
		t.Fatal(err)
	}
	client := imagePurgeClient(t, w.pool)
	var otherJob int64
	if pendingPurge {
		otherJob = insertContentPurgeJob(t, client)
	}
	job := insertContentPurgeJob(t, client)
	t.Logf("observed purge job=%d preceding purge job=%d", job, otherJob)
	firstPID := imagePurgeProcess(t, rw, "page_failed", job, rw.now, fmt.Sprintf("%s=%t", imageProcessExpireFirst, expireFirst))
	if pendingPurge {
		var snoozes int
		var hasOutput bool
		if err := w.pool.QueryRow(t.Context(), `SELECT coalesce((metadata->>'snoozes')::int,0),metadata ? 'output'
 FROM river_job WHERE id=$1`, otherJob).Scan(&snoozes, &hasOutput); err != nil || snoozes < 1 || hasOutput {
			t.Fatal("unrelated purge was not deferred without executing a page", snoozes, hasOutput, err)
		}
	}
	if expireFirst {
		var initialAttempts, uncommitted int
		if err := w.pool.QueryRow(t.Context(), `SELECT count(*),count(*) FILTER (WHERE committed_after IS NULL)
 FROM image_purge_calls WHERE process=$1 AND after_id=$2`, firstPID, uuid.Nil()).Scan(&initialAttempts, &uncommitted); err != nil || initialAttempts < 2 || uncommitted < 1 {
			t.Fatal("initial budget did not expire before the first committed page", initialAttempts, uncommitted, err)
		}
	}
	result, state := imagePurgeJob(t, w.pool, job)
	cutoff := rw.now.Add(-60 * 24 * time.Hour)
	if result.Complete || result.Cursor.After != first || result.Cursor.Module != "asset" || !result.Cursor.Cutoff.Equal(cutoff) {
		t.Fatal("failed content page lost committed cursor", result, state)
	}
	var failures int
	if err := w.pool.QueryRow(t.Context(), "SELECT count(*) FROM image_purge_calls WHERE failed AND after_id=$1", first).Scan(&failures); err != nil || failures != 1 {
		t.Fatal("second page failure was not observed", failures, err)
	}
	assertContentRowsPresent(t, w, old.rows[:1], false)
	assertContentRowsPresent(t, w, old.rows[1:], true)
	if beforeNext != contentRowsSnapshot(t, w, []assetdomain.Asset{old.rows[1]}) {
		t.Fatal("failed page changed asset audit")
	}
	if _, err := w.pool.Exec(t.Context(), "DROP TRIGGER reject_process_content ON file_assets"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.JobRetry(t.Context(), job); err != nil {
		t.Fatal(err)
	}
	secondPID := imagePurgeProcess(t, rw, "resume", job, rw.now.Add(30*24*time.Hour))
	result, state = imagePurgeJob(t, w.pool, job)
	if secondPID == firstPID || secondPID == os.Getpid() || state != "completed" || !result.Complete || !result.Cursor.Cutoff.Equal(cutoff) {
		t.Fatal("restart lost job/cutoff", firstPID, secondPID, result, state)
	}
	var after uuid.UUID
	if err := w.pool.QueryRow(t.Context(), "SELECT after_id FROM image_purge_calls WHERE process=$1 ORDER BY n LIMIT 1", secondPID).Scan(&after); err != nil || after != first {
		t.Fatal("new process replayed or skipped committed page", after, err)
	}
	var wrong int
	if err := w.pool.QueryRow(t.Context(), "SELECT count(*) FROM image_purge_calls WHERE cutoff<>$1", cutoff).Scan(&wrong); err != nil || wrong != 0 {
		t.Fatal("cutoff drift", wrong, err)
	}
	assertContentRowsPresent(t, w, old.rows, false)
	assertContentProjectPresent(t, w, old, false)
	if beforeYoung != contentRowsSnapshot(t, w, young.rows) || beforeLive != contentRowsSnapshot(t, w, live.rows) {
		t.Fatal("resume changed live or 59-day neighbours")
	}
	for _, row := range append(young.rows, live.rows...) {
		assertContentBytes(t, rw, row)
	}
	t.Logf("same job=%d PIDs=%d->%d resumes after=%s with original cutoff=%s", job, firstPID, secondPID, after, cutoff)
}
