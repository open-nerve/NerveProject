package bootstrap

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	assetdomain "github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	issuepg "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

func TestContentAssetRetention(t *testing.T) {
	t.Run("trimmed history and original preparation age survive publication", func(t *testing.T) {
		rw := newContentRetentionWorld(t)
		w := rw.issueWorld
		// Join the initial cleanup before introducing the explicit age fixture;
		// the configured hourly sweep must not expire it before publication.
		waitForCompletedKinds(t, w.pool, map[string]int{"issue.trim_description_versions": 1, "asset.cleanup": 1, "cleanup.purge_deleted": 1}, &lockedBuffer{})
		d := createDraftHTTP(t, w, "alice", fmt.Sprintf(`{"project_id":%q,"payload":{"name":"Published history"}}`, w.web))
		history := draftAsset(t, w, d.ID, true)
		ordinary := draftAsset(t, w, d.ID, false)
		unused := draftAsset(t, w, d.ID, true)
		young := draftAsset(t, w, d.ID, true)
		draftRequest(t, w, "alice", "PATCH", draftPath(d.ID), draftPayloadBody(t, map[string]any{"name": "Published history", "description_html": issueImageHTML(history.Asset.ID)}), 200)
		// Only age is a persistence fixture. Publication must preserve it;
		// changing updated_at during migration must not renew a preparation.
		for _, id := range []uuid.UUID{history.Asset.ID, ordinary.Asset.ID, unused.Asset.ID, young.Asset.ID} {
			days := 8
			if id == young.Asset.ID {
				days = 6
			}
			if _, err := w.pool.Exec(t.Context(), "UPDATE file_assets SET created_at=$2 WHERE id=$1", id, rw.now.Add(-time.Duration(days)*24*time.Hour)); err != nil {
				t.Fatal(err)
			}
		}
		before, err := assetpg.New(w.pool).FindAsset(t.Context(), history.Asset.ID)
		if err != nil {
			t.Fatal(err)
		}
		var published storedIssue
		decodeAnswer(t, draftRequest(t, w, "alice", "POST", publishPath(d.ID), "", 201), &published)
		first := savedVersion(t, w, published.ID)
		for i := range 23 {
			actor := "bob"
			if i%2 == 1 {
				actor = "alice"
			}
			saveIssueImage(t, w, actor, published.ID, fmt.Sprintf("<p>Revision %d without images</p>", i), false, 200)
		}
		if n := len(trimVersionIDs(t, w.pool, published.ID)); n != 24 {
			t.Fatal("API edits did not produce distinct versions", n)
		}
		if _, err := retentionJob(w, postgres.NewTxManager(w.pool, time.Second), issuepg.New(w.pool)).Execute(t.Context()); err != nil {
			t.Fatal(err)
		}
		versions := trimVersionIDs(t, w.pool, published.ID)
		if len(versions) != 20 || slices.Contains(versions, first.ID) {
			t.Fatal("real trim did not remove image-bearing history", versions)
		}
		_, data := readAssetHTTP(t, w, "bob", history.Asset.ID)
		if !bytes.Equal(data, assetTestPNG(t)) {
			t.Fatal("trimmed image lost readable bytes")
		}
		state := contentAssetState(t, w)
		assetAPICall(t, w, "alice", "DELETE", "/api/v0/assets/"+history.Asset.ID.String(), "", 409)
		if !reflect.DeepEqual(state, contentAssetState(t, w)) {
			t.Fatal("bound image delete wrote rows")
		}
		rw.stop()
		if _, err := w.pool.Exec(t.Context(), "UPDATE draft_issues SET deleted_at=$2 WHERE id=$1", d.ID, rw.now.Add(-61*24*time.Hour)); err != nil {
			t.Fatal(err)
		}
		purgeImageTree(t, rw)
		if ids := queryIDs(t, w.pool, "SELECT id FROM draft_issues WHERE id=$1", d.ID); len(ids) != 0 {
			t.Fatal("consumed draft still referenced by migrated files")
		}
		if failures := cleanImageObjects(t, rw, rw.objects); failures != 0 {
			t.Fatal(failures)
		}
		for _, id := range []uuid.UUID{history.Asset.ID, ordinary.Asset.ID, unused.Asset.ID, young.Asset.ID} {
			a, err := assetpg.New(w.pool).FindAsset(t.Context(), id)
			if err != nil {
				t.Fatal(err)
			}
			if a.Placement.DraftID != uuid.Nil() || a.Placement.IssueID != published.ID {
				t.Fatal("file retained dead draft host", a)
			}
			days := 8
			if id == young.Asset.ID {
				days = 6
			}
			if !a.CreatedAt.Equal(rw.now.Add(-time.Duration(days) * 24 * time.Hour)) {
				t.Fatal("migration renewed preparation age", a.CreatedAt)
			}
			if id == unused.Asset.ID {
				if a.DeletedAt == nil || !a.DeletedAt.Equal(rw.now) || a.DeletedByID != nil || a.BoundAt != nil {
					t.Fatal("old unreferenced image did not expire as system action", a)
				}
			} else if a.DeletedAt != nil {
				t.Fatal("bound or six-day file expired", a)
			}
			if id == history.Asset.ID && (!reflect.DeepEqual(a.BoundAt, before.BoundAt) || a.Purpose != assetdomain.IssueDescription) {
				t.Fatal("historical binding changed", a)
			}
			if id == ordinary.Asset.ID {
				assertContentBytes(t, rw, a, []byte("draft bytes"))
			} else {
				assertContentBytes(t, rw, a)
			}
		}
		t.Log("24 actual API versions trimmed to 20; removed image stays bound/readable; consumed draft purged; migrated unused 8d expires while 6d and bound files survive")
	})
	t.Run("physical failure preserves parent and another process recovers", func(t *testing.T) {
		rw := newContentRetentionWorld(t)
		w := rw.issueWorld
		blocked := newContentProjectTree(t, w, "BLOCKED")
		old := newContentProjectTree(t, w, "OLD")
		young := newContentProjectTree(t, w, "YOUNG")
		live := newContentProjectTree(t, w, "LIVE")
		for _, g := range []contentAssetTree{blocked, old, young} {
			deleteContentProject(t, w, g)
		}
		rw.stop()
		ageContentTree(t, rw, blocked, 61)
		ageContentTree(t, rw, old, 61)
		ageContentTree(t, rw, young, 59)
		beforeLive := contentRowsSnapshot(t, w, live.rows)
		for i, row := range young.rows {
			var err error
			young.rows[i], err = assetpg.New(w.pool).FindAsset(t.Context(), row.ID)
			if err != nil {
				t.Fatal(err)
			}
		}
		original, err := assetpg.New(w.pool).FindAsset(t.Context(), blocked.rows[0].ID)
		if err != nil {
			t.Fatal(err)
		}
		key := blocked.rows[0].Metadata.Final.Key
		dir := filepath.Join(rw.root, "objects", string(key))
		if err := os.Chmod(dir, 0500); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Chmod(dir, 0700) }()
		if failed := cleanImageObjects(t, rw, rw.objects, fs.ErrPermission); failed != 1 {
			t.Fatal("real physical Delete did not fail exactly once", failed)
		}
		failed, err := assetpg.New(w.pool).FindAsset(t.Context(), blocked.rows[0].ID)
		if err != nil || failed.Metadata.Final == nil || failed.Metadata.Final.Key != key || failed.Metadata.Final.Phase != assetdomain.ObjectReclaiming || failed.DeletedAt == nil || !failed.DeletedAt.Equal(rw.now.Add(-61*24*time.Hour)) || failed.DeletedByID == nil || *failed.DeletedByID != w.ids["alice"] {
			t.Fatal("file failure lost recovery identity or original audit", failed, err)
		}
		audit := failed
		audit.Metadata, audit.StorageRevision = original.Metadata, original.StorageRevision
		if !reflect.DeepEqual(audit, original) {
			t.Fatal("recovery rewrote original audit or binding", audit, original)
		}
		assertContentBytes(t, rw, failed)
		if result := purgeImageTree(t, rw); result.Blocked == 0 {
			t.Fatal("recoverable file failed to protect parent", result)
		}
		assertContentRowsPresent(t, w, old.rows, false)
		assertContentProjectPresent(t, w, old, false)
		assertContentRowsPresent(t, w, []assetdomain.Asset{failed}, true)
		assertContentProjectPresent(t, w, blocked, true)
		// Elapsed staging leases may be reclaimed even for young tombstones.
		// Take their stable, post-cleanup recovery snapshot before restart.
		afterYoung := contentRowsSnapshot(t, w, young.rows)
		for _, before := range young.rows {
			got, err := assetpg.New(w.pool).FindAsset(t.Context(), before.ID)
			if err != nil || before.Metadata.Staging == nil {
				t.Fatal("missing staged object fixture", err)
			}
			before.Metadata.Staging = nil
			before.StorageRevision += 2
			if !reflect.DeepEqual(got, before) {
				t.Fatal("young cleanup changed more than staging", got, before)
			}
		}
		if beforeLive != contentRowsSnapshot(t, w, live.rows) {
			t.Fatal("live current files changed")
		}
		if err := os.Chmod(dir, 0700); err != nil {
			t.Fatal(err)
		}
		createImagePurgeObservation(t, w.pool)
		client := imagePurgeClient(t, w.pool)
		job := insertContentPurgeJob(t, client)
		pid := imagePurgeProcess(t, rw, "resume", job, rw.now, "NERVE_TEST_CONTENT_CLEANUP_ROOT="+rw.root)
		if pid == os.Getpid() {
			t.Fatal("recovery did not enter another process")
		}
		assertContentRowsPresent(t, w, blocked.rows, false)
		assertContentProjectPresent(t, w, blocked, false)
		assertContentRowsPresent(t, w, young.rows, true)
		assertContentProjectPresent(t, w, young, true)
		if afterYoung != contentRowsSnapshot(t, w, young.rows) || beforeLive != contentRowsSnapshot(t, w, live.rows) {
			t.Fatal("new process changed retained neighbours")
		}
		for _, a := range append(young.rows, live.rows...) {
			assertContentBytes(t, rw, a)
		}
		t.Logf("real permission failure retained final key and NO ACTION parent; later candidate progressed; child PID=%d recovered six-purpose graph", pid)
	})
}
