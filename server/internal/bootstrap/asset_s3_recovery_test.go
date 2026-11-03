package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
	"uuid"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/platform/s3test"
)

func TestAssetS3Recovery(t *testing.T) {
	service := s3test.New(t)
	t.Run("final DB commit failure retains sealed candidate", func(t *testing.T) { s3FinalCommitRecovery(t, service.NewBucket(t)) })
	for _, change := range []string{"member revoke", "issue delete"} {
		t.Run(change, func(t *testing.T) { s3LateOwnerChange(t, service.NewBucket(t), change) })
	}
}
func s3FinalCommitRecovery(t *testing.T, b *s3test.Bucket) {
	w := newAssetS3World(t, b, func(*config.Config) {}, func(*assetComponents) {})
	item := w.newWorkItem(t)
	data := []byte("S3 sealed before deferred commit")
	file := prepareAssetHTTP(t, w.issueWorld, "bob", "ISSUE_ATTACHMENT", assetScopeBody("issue", item), "x.txt", "text/plain", len(data))
	putAssetS3HTTP(t, w, file, data)
	_, err := w.pool.Exec(t.Context(), `CREATE FUNCTION reject_s3_publication() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'S3 publication rejected'; END $$;
CREATE CONSTRAINT TRIGGER reject_s3_publication AFTER UPDATE ON file_assets DEFERRABLE INITIALLY DEFERRED FOR EACH ROW WHEN (NEW.is_uploaded) EXECUTE FUNCTION reject_s3_publication();`)
	if err != nil {
		t.Fatal(err)
	}
	assetAPICall(t, w.issueWorld, "bob", "POST", completeAssetPath(file.Asset.ID), `{}`, 500)
	rows := assetpg.New(w.pool)
	row, err := rows.FindAsset(t.Context(), file.Asset.ID)
	if err != nil || row.Uploaded || row.Metadata.Final != nil || len(row.Metadata.Candidates) != 1 {
		t.Fatalf("rollback forgot candidate: %+v %v", row, err)
	}
	old := row.Metadata.Candidates[0].Key
	if _, err := w.files.Inspect(t.Context(), old); err != nil {
		t.Fatal("sealed candidate bytes lost", err)
	}
	if _, err := w.pool.Exec(t.Context(), "DROP TRIGGER reject_s3_publication ON file_assets"); err != nil {
		t.Fatal(err)
	}
	assetAPICall(t, w.issueWorld, "bob", "POST", completeAssetPath(file.Asset.ID), `{}`, 200)
	after, err := rows.FindAsset(t.Context(), file.Asset.ID)
	if err != nil || !after.Uploaded || after.Metadata.Final == nil || after.Metadata.Final.Key == old || len(after.Metadata.Candidates) != 1 || after.Metadata.Candidates[0].Key != old {
		t.Fatalf("retry reused/forgot candidate: %+v %v", after, err)
	}
	_, got := readAssetS3HTTP(t, w, "bob", file.Asset.ID)
	if !bytes.Equal(got, data) {
		t.Fatal("retry published different S3 bytes")
	}
	reopened := restartAssetS3Cleanup(t, w, 0)
	if _, err := reopened.Inspect(t.Context(), old); !errors.Is(err, domain.ErrObjectNotFound) {
		t.Fatalf("app recovery missed failed candidate: %v", err)
	}
	final, err := reopened.Inspect(t.Context(), after.Metadata.Final.Key)
	if err != nil || final.ETag != after.Metadata.Final.ETag {
		t.Fatalf("app recovery lost published final: %+v %v", final, err)
	}
	recovered, err := rows.FindAsset(t.Context(), file.Asset.ID)
	if err != nil || !recovered.Uploaded || recovered.Metadata.Final == nil || recovered.Metadata.Final.Key != after.Metadata.Final.Key || len(recovered.Metadata.Candidates) != 0 || recovered.Metadata.Staging != nil {
		t.Fatalf("app recovery failed to converge: %+v %v", recovered, err)
	}
}

func s3LateOwnerChange(t *testing.T, b *s3test.Bucket, change string) {
	attempts := make(chan *assetSealAttempt, 1)
	w := newAssetS3World(t, b, func(c *config.Config) {
		c.Issue.WriteGate = config.WriteGateConfig{PerProject: 1, MaxQueue: 0, MaxWait: time.Second}
	}, func(c *assetComponents) { c.sealing.Sealer = controlledAssetSeal(c.store, attempts) })
	waitForCompletedKinds(t, w.pool, map[string]int{"asset.cleanup": 1, "cleanup.purge_deleted": 1}, &lockedBuffer{})
	w.send(t, "alice", "PATCH", "/api/v0/projects/"+w.web.String(), `{"network":0}`, 200)
	item := w.newWorkItem(t)
	file := prepareAssetHTTP(t, w.issueWorld, "bob", "ISSUE_ATTACHMENT", assetScopeBody("issue", item), "late.txt", "text/plain", 3)
	putAssetS3HTTP(t, w, file, []byte("old"))
	req := newRequest(t, "POST", w.base+completeAssetPath(file.Asset.ID), w.tokens["bob"], []byte(`{}`))
	response := sendInBackground(req)
	attempt := awaitAssetSeal(t, attempts, response)
	defer attempt.resume(nil)
	requestEventually(t, "S3 external I/O releases app connection", func(context.Context) bool { return w.requests.appPool.Stat().AcquiredConns() == 0 })
	tx, err := w.requests.appPool.Begin(pgtest.Soon(t))
	if err != nil {
		t.Fatal(err)
	}
	defer requestRelease(t, tx)
	for _, lock := range []struct {
		table string
		id    uuid.UUID
	}{{"users", w.ids["bob"]}, {"workspaces", w.acme}, {"projects", w.web}, {"file_assets", file.Asset.ID}} {
		if _, err := tx.Exec(pgtest.Soon(t), "SELECT id FROM "+lock.table+" WHERE id=$1 FOR UPDATE NOWAIT", lock.id); err != nil {
			t.Fatalf("S3 bytes held %s row lock: %v", lock.table, err)
		}
	}
	requestRelease(t, tx)
	w.send(t, "bob", "PATCH", "/api/v0/issues/"+item.String(), `{"name":"same gate stays available during S3"}`, 200)
	switch change {
	case "member revoke":
		member := queryIDs(t, w.pool, "SELECT id FROM project_members WHERE project_id=$1 AND member_id=$2 AND is_active AND deleted_at IS NULL", w.web, w.ids["bob"])[0]
		w.send(t, "alice", "DELETE", "/api/v0/project-members/"+member.String(), "", 204)
	case "issue delete":
		w.send(t, "alice", "DELETE", "/api/v0/issues/"+item.String(), "", 204)
	}
	attempt.resume(nil)
	got := receiveWithin(t, response, 5*time.Second, "S3 publication after owner change")
	if got.err != nil || got.res.StatusCode != 404 || problemCode(t, got.body) != "asset.not_found" {
		t.Fatalf("late S3 publication after %s: %+v", change, got)
	}
	w.contract.CheckResponse(t, req, got.res)
	row, err := assetpg.New(w.pool).FindAsset(t.Context(), file.Asset.ID)
	if err != nil || row.Uploaded || row.Metadata.Final != nil || len(row.Metadata.Candidates) != 1 || row.Metadata.Candidates[0].Key != attempt.key {
		t.Fatalf("late S3 recovery lost: %+v %v", row, err)
	}
	if _, err := w.files.Inspect(t.Context(), attempt.key); err != nil {
		t.Fatal(err)
	}
	refused := assetAPICall(t, w.issueWorld, "bob", "GET", fmt.Sprintf("/api/v0/assets/%s/content", file.Asset.ID), "", 404)
	if problemCode(t, []byte(refused)) != "asset.not_found" {
		t.Fatal(refused)
	}
}
