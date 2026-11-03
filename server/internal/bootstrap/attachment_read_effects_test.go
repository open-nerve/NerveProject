package bootstrap

import (
	"fmt"
	"testing"
	"uuid"
)

func TestAttachmentListsNoWritesOrStorageIO(t *testing.T) {
	var storage *imageStorageCalls
	w, _ := newAssetCompleteWorld(t, func(c *assetComponents) {
		storage = &imageStorageCalls{assetStore: c.store}
		c.store = storage
		c.sealing.Inspector, c.sealing.Reader, c.sealing.Sealer = storage, storage, storage
	})
	item := w.newWorkItem(t)
	comment := w.newComment(t, "bob", item)
	draft := createDraftHTTP(t, w, "bob", `{"payload":{}}`)
	targets := []struct {
		kind, path, purpose string
		id                  uuid.UUID
	}{{"issue", "/api/v0/issues/" + item.String(), "ISSUE_ATTACHMENT", item}, {"comment", commentPath(comment.ID), "COMMENT_ATTACHMENT", comment.ID}, {"draft", draftPath(draft.ID), "DRAFT_ISSUE_ATTACHMENT", draft.ID}}
	for _, q := range targets {
		uploadedAssetHTTP(t, w, "bob", q.purpose, attachmentScope(q.kind, q.id), "file.txt", "text/plain", []byte("abc"))
	}
	waitForCompletedKinds(t, w.pool, map[string]int{"asset.cleanup": 1, "cleanup.purge_deleted": 1}, &lockedBuffer{})
	if storage.io.Load() == 0 {
		t.Fatal("upload bypassed storage instrumentation")
	}
	if _, err := w.pool.Exec(t.Context(), `CREATE TABLE attachment_read_audit(op text);
 CREATE FUNCTION attachment_read_write() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN INSERT INTO attachment_read_audit VALUES(TG_OP); RETURN NULL; END $$`); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"file_assets", "issues", "issue_comments", "draft_issues", "issue_activities"} {
		if _, err := w.pool.Exec(t.Context(), fmt.Sprintf("CREATE TRIGGER attachment_read_write AFTER INSERT OR UPDATE OR DELETE ON %s FOR EACH STATEMENT EXECUTE FUNCTION attachment_read_write()", table)); err != nil {
			t.Fatal(err)
		}
	}
	beforeIO, beforeSigns := storage.io.Load(), storage.signs.Load()
	for _, q := range targets {
		page := attachmentListRead(t, w, "bob", q.path+"/attachments", 200)
		if len(page.Data) != 1 {
			t.Fatal(page)
		}
		attachmentListRead(t, w, "olga", q.path+"/attachments", 404)
	}
	if storage.io.Load() != beforeIO || storage.signs.Load() != beforeSigns {
		t.Fatal("list performed storage I/O or signed URLs")
	}
	var writes int
	if err := w.pool.QueryRow(t.Context(), "SELECT count(*) FROM attachment_read_audit").Scan(&writes); err != nil || writes != 0 {
		t.Fatal("read wrote rows", writes, err)
	}
	// Deliberately unavailable asset storage table: owner refusals must still
	// happen first, while authorized reads expose an opaque error, not []/200.
	if _, err := w.pool.Exec(t.Context(), "ALTER TABLE file_assets RENAME TO unavailable_file_assets"); err != nil {
		t.Fatal(err)
	}
	for _, q := range targets {
		attachmentListRead(t, w, "olga", q.path+"/attachments", 404)
		attachmentListRead(t, w, "bob", q.path+"/attachments", 500)
	}
}
