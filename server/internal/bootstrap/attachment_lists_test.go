package bootstrap

import (
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

type attachmentListRow struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	ContentType string     `json:"content_type"`
	Size        int64      `json:"size"`
	CreatedAt   time.Time  `json:"created_at"`
	CreatedByID *uuid.UUID `json:"created_by_id"`
}
type attachmentListPage struct {
	Data       []attachmentListRow `json:"data"`
	NextCursor *string             `json:"next_cursor"`
}

func attachmentListRead(t *testing.T, w issueWorld, who, path string, status int) attachmentListPage {
	t.Helper()
	req := newRequest(t, "GET", w.base+path, w.tokens[who], nil)
	res, body := send(t, req)
	if res.StatusCode != status {
		t.Fatalf("attachment list status=%d want=%d body=%s", res.StatusCode, status, body)
	}
	w.contract.CheckResponse(t, req, res)
	var page attachmentListPage
	if status == 200 {
		if err := json.Unmarshal(body, &page); err != nil {
			t.Fatal(err)
		}
		var raw struct {
			Data []map[string]any `json:"data"`
		}
		if err := json.Unmarshal(body, &raw); err != nil {
			t.Fatal(err)
		}
		for _, row := range raw.Data {
			if len(row) != 6 {
				t.Fatalf("metadata exposed unexpected fields: %v", row)
			}
		}
	}
	return page
}

func TestAttachmentLists(t *testing.T) {
	w := newIssueWorld(t)
	item := w.newWorkItem(t)
	comment := w.newComment(t, "bob", item)
	draft := createDraftHTTP(t, w, "bob", fmt.Sprintf(`{"project_id":%q,"payload":{}}`, w.web))
	for _, tc := range []struct {
		kind, path, reader string
		id                 uuid.UUID
		count              int
	}{
		{"issue", "/api/v0/issues/" + item.String() + "/attachments", "alice", item, 101},
		{"comment", "/api/v0/issue-comments/" + comment.ID.String() + "/attachments", "alice", comment.ID, 2},
		{"draft", "/api/v0/drafts/" + draft.ID.String() + "/attachments", "bob", draft.ID, 2},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			purpose := map[string]string{"issue": "ISSUE_ATTACHMENT", "comment": "COMMENT_ATTACHMENT", "draft": "DRAFT_ISSUE_ATTACHMENT"}[tc.kind]
			var ids []uuid.UUID
			for n := range tc.count {
				p := uploadedAssetHTTP(t, w, "bob", purpose, attachmentScope(tc.kind, tc.id), fmt.Sprintf("same-%03d.txt", tc.count-n), "text/plain", []byte("abc"))
				ids = append(ids, p.Asset.ID)
			}
			slices.SortFunc(ids, uuid.UUID.Compare)
			p := attachmentListRead(t, w, tc.reader, tc.path, 200)
			if len(p.Data) != min(50, tc.count) {
				t.Fatalf("default page size=%d", len(p.Data))
			}
			limit := 1
			if tc.kind == "issue" {
				limit = 100
			}
			var got []uuid.UUID
			next := tc.path + fmt.Sprintf("?limit=%d", limit)
			pages := 0
			for next != "" {
				page := attachmentListRead(t, w, tc.reader, next, 200)
				pages++
				for _, row := range page.Data {
					if row.CreatedByID == nil || *row.CreatedByID != w.ids["bob"] || row.Size != 3 || row.ContentType != "text/plain" || row.CreatedAt.IsZero() {
						t.Fatalf("metadata lost origin: %+v", row)
					}
					got = append(got, row.ID)
				}
				next = ""
				if page.NextCursor != nil {
					next = tc.path + fmt.Sprintf("?limit=%d&cursor=", limit) + url.QueryEscape(*page.NextCursor)
				}
				if pages > 3 {
					t.Fatal("pagination did not terminate")
				}
			}
			if pages != 2 || !slices.Equal(got, ids) {
				t.Fatalf("pagination lost/duplicated/ordered files: pages=%d got=%v want=%v", pages, got, ids)
			}
			beyond, err := domain.EncodeAttachmentCursor(tc.kind, tc.id, uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff"))
			if err != nil {
				t.Fatal(err)
			}
			end := attachmentListRead(t, w, tc.reader, tc.path+"?cursor="+url.QueryEscape(beyond), 200)
			if len(end.Data) != 0 || end.NextCursor != nil {
				t.Fatal("cursor beyond end did not return an empty page", end)
			}
			for _, host := range []struct {
				kind string
				id   uuid.UUID
			}{{tc.kind, uuid.NewV7()}, {"other", tc.id}} {
				kind := host.kind
				if kind == "other" {
					kind = "issue"
					if tc.kind == "issue" {
						kind = "draft"
					}
				}
				wrong, err := domain.EncodeAttachmentCursor(kind, host.id, ids[0])
				if err != nil {
					t.Fatal(err)
				}
				attachmentListRead(t, w, tc.reader, tc.path+"?cursor="+url.QueryEscape(wrong), 400)
			}
			if tc.kind == "draft" {
				attachmentListRead(t, w, "alice", tc.path, 404)
			}
			attachmentListRead(t, w, tc.reader, tc.path+"?unknown=true", 400)
			attachmentListRead(t, w, tc.reader, tc.path+"?limit=1&limit=2", 400)
			attachmentListRead(t, w, tc.reader, tc.path+"?limit=101", 422)
			attachmentListRead(t, w, "olga", tc.path, 404)
		})
	}
}
