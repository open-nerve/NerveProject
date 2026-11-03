package bootstrap

import (
	"testing"
	"time"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
)

func TestAttachmentEventsKeepPublicationTransactionAndLocks(t *testing.T) {
	w := newObservedIssueWorld(t)
	item := w.newWorkItem(t)
	p := prepareAssetHTTP(t, w, "bob", "ISSUE_ATTACHMENT", attachmentScope("issue", item), "atomic.txt", "text/plain", 3)
	putAssetHTTP(t, w, p, []byte("abc"), 204)
	blocker, err := w.pool.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer requestRelease(t, blocker)
	if _, err := blocker.Exec(t.Context(), "LOCK TABLE issue_activities IN SHARE MODE"); err != nil {
		t.Fatal(err)
	}
	req := newRequest(t, "POST", w.base+completeAssetPath(p.Asset.ID), w.tokens["bob"], []byte(`{}`))
	probe := w.observe(req)
	response := sendInBackground(req)
	requestTableBlocked(t, w.pool, probe, blocker.Conn().PgConn().PID(), "issue_activities")
	for _, row := range []struct {
		from, want string
		id         any
	}{
		{"projects WHERE id=$1", "FOR NO KEY UPDATE", w.web},
		{"file_assets WHERE id=$1", "FOR UPDATE", p.Asset.ID},
	} {
		if got := requestRowLock(t, w.pool, w.pool, probe, row.from, row.id); got != row.want {
			t.Fatalf("publication consumer lost lock: %s=%s want %s", row.from, got, row.want)
		}
	}
	outside, err := assetpg.New(w.pool).FindAsset(t.Context(), p.Asset.ID)
	if err != nil || outside.Uploaded || outside.BoundAt != nil || len(outside.Metadata.Candidates) != 1 || attachmentActivityCount(t, w, item) != 0 {
		t.Fatalf("publication escaped blocked subscriber transaction: %+v %v", outside, err)
	}
	requestRelease(t, blocker)
	result := receiveWithin(t, response, 5*time.Second, "attachment publication transaction")
	if result.err != nil || result.res.StatusCode != 200 {
		t.Fatalf("publication=%+v", result)
	}
	w.contract.CheckResponse(t, req, result.res)
	if attachmentActivityCount(t, w, item) != 1 {
		t.Fatal("publication did not commit exactly one activity")
	}
}
