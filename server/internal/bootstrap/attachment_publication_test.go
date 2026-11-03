package bootstrap

import (
	"fmt"
	"reflect"
	"testing"
	"uuid"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
)

func attachmentScope(kind string, id uuid.UUID) string {
	return fmt.Sprintf(`{"kind":%q,"id":%q}`, kind, id)
}

func TestAttachmentPublication(t *testing.T) {
	w := newIssueWorld(t)
	item := w.newWorkItemBy(t, "bob", w.web, `{"name":"Attachment publication"}`)
	var comment answeredComment
	decodeAnswer(t, w.send(t, "bob", "POST", commentsOf(item), `{"comment_html":"<p>Files</p>"}`, 201), &comment)
	draft := createDraftHTTP(t, w, "bob", fmt.Sprintf(`{"project_id":%q,"payload":{}}`, w.web))
	for _, tc := range []struct {
		purpose, kind string
		id            uuid.UUID
		bind          bool
	}{
		{"ISSUE_ATTACHMENT", "issue", item, true},
		{"COMMENT_ATTACHMENT", "comment", comment.ID, true},
		{"DRAFT_ISSUE_ATTACHMENT", "draft", draft.ID, true},
		{"COMMENT_ATTACHMENT", "new_comment", item, false},
		{"ISSUE_DESCRIPTION", "issue", item, false},
		{"COMMENT_DESCRIPTION", "comment", comment.ID, false},
		{"DRAFT_ISSUE_DESCRIPTION", "draft", draft.ID, false},
		{"COMMENT_DESCRIPTION", "new_comment", item, false},
	} {
		t.Run(tc.purpose+"/"+tc.kind, func(t *testing.T) {
			data := assetTestPNG(t)
			prepared := prepareAssetHTTP(t, w, "bob", tc.purpose, attachmentScope(tc.kind, tc.id), "private-file.png", "image/png", len(data))
			putAssetHTTP(t, w, prepared, data, 204)
			body := assetAPICall(t, w, "bob", "POST", completeAssetPath(prepared.Asset.ID), `{}`, 200)
			got := prepared.Asset
			decodeAnswer(t, body, &got)
			row, err := assetpg.New(w.pool).FindAsset(t.Context(), got.ID)
			if err != nil || got.IsBound != tc.bind || (row.BoundAt != nil) != tc.bind || !row.Uploaded {
				t.Fatalf("publication binding=%v DB=%+v error=%v want=%v", got.IsBound, row, err, tc.bind)
			}
			if tc.bind && (!row.BoundAt.Equal(row.UpdatedAt) || row.UpdatedByID == nil || *row.UpdatedByID != w.ids["bob"]) {
				t.Fatal("binding did not use final publication stamp")
			}
			if again := assetAPICall(t, w, "bob", "POST", completeAssetPath(got.ID), `{}`, 200); again != body {
				t.Fatal("repeat complete changed public result")
			}
			after, err := assetpg.New(w.pool).FindAsset(t.Context(), got.ID)
			if err != nil || !reflect.DeepEqual(after, row) {
				t.Fatalf("repeat complete changed binding, audit or recovery: %+v %v", after, err)
			}
		})
	}
	if attachmentActivityCount(t, w, item) != 1 {
		t.Fatal("comment, draft or description assets emitted issue attachment actions")
	}
}

func TestAttachmentPublicationDuplicate(t *testing.T) {
	w := newIssueWorld(t)
	item := w.newWorkItem(t)
	prepared := prepareAssetHTTP(t, w, "alice", "ISSUE_ATTACHMENT", attachmentScope("issue", item), "source.txt", "text/plain", 3)
	putAssetHTTP(t, w, prepared, []byte("abc"), 204)
	assetAPICall(t, w, "alice", "POST", completeAssetPath(prepared.Asset.ID), `{}`, 200)
	for _, kind := range []string{"issue", "new_comment"} {
		purpose := "ISSUE_ATTACHMENT"
		if kind == "new_comment" {
			purpose = "COMMENT_ATTACHMENT"
		}
		body := fmt.Sprintf(`{"scope":%s,"purpose":%q}`, attachmentScope(kind, item), purpose)
		raw := assetAPICall(t, w, "alice", "POST", "/api/v0/assets/"+prepared.Asset.ID.String()+"/duplicate", body, 201)
		got := prepared.Asset
		decodeAnswer(t, raw, &got)
		if got.ID == prepared.Asset.ID || got.IsBound != (kind == "issue") {
			t.Fatalf("duplicate binding=%s", raw)
		}
		_, data := readAssetHTTP(t, w, "alice", got.ID)
		if string(data) != "abc" {
			t.Fatal("duplicate lost actual bytes")
		}
	}
	if attachmentActivityCount(t, w, item) != 2 {
		t.Fatal("copy did not emit once for the issue attachment only")
	}
}
