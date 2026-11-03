package httpadapter_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type fakeAttachmentList struct {
	calls  int
	id     uuid.UUID
	limit  *int
	cursor *string
	page   app.AttachmentPage
	err    error
}

func (f *fakeAttachmentList) Execute(_ context.Context, id uuid.UUID, limit *int, cursor *string) (app.AttachmentPage, error) {
	f.calls++
	f.id, f.limit, f.cursor = id, limit, cursor
	return f.page, f.err
}
func TestAttachmentListsHTTP(t *testing.T) {
	for _, host := range []string{"issues", "issue-comments", "drafts"} {
		t.Run(host, func(t *testing.T) {
			path := "/api/v0/" + host + "/" + webSixID.String() + "/attachments"
			f := &fakeAttachmentList{}
			h := newServer(t, httpadapter.UseCases{GetIssueAttachments: f, GetCommentAttachments: f, GetDraftAttachments: f})
			res, body := do(t, h, request("GET", path, "alice", ""))
			if res.StatusCode != 200 || body != "{\"data\":[],\"next_cursor\":null}\n" || f.id != webSixID || f.limit != nil || f.cursor != nil {
				t.Fatal(res.StatusCode, body, f)
			}
			f.page = app.AttachmentPage{Data: []app.Attachment{{ID: aliceID, Name: "x.txt", ContentType: "text/plain", Size: 3, CreatedAt: created}}, NextCursor: "next"}
			res, body = do(t, h, request("GET", path+"?limit=1&cursor=position", "alice", ""))
			var got struct {
				Data       []map[string]any `json:"data"`
				NextCursor string           `json:"next_cursor"`
			}
			if json.Unmarshal([]byte(body), &got) != nil || res.StatusCode != 200 || len(got.Data) != 1 || len(got.Data[0]) != 6 || got.Data[0]["created_by_id"] != nil || got.NextCursor != "next" || f.limit == nil || *f.limit != 1 || f.cursor == nil || *f.cursor != "position" {
				t.Fatal(body, f)
			}
			calls := f.calls
			for _, query := range []string{"?unknown=1", "?limit=1&limit=2", "?cursor=a&cursor=b", "?limit=x", "?cursor=%ZZ", "?cursor=a;other=b"} {
				res, body := do(t, h, request("GET", path+query, "alice", ""))
				if res.StatusCode != 400 || f.calls != calls {
					t.Fatal(query, res.StatusCode, body)
				}
			}
			res, _ = do(t, h, request("GET", path, "", ""))
			if res.StatusCode != 401 || f.calls != calls {
				t.Fatal("anonymous reached use case")
			}
			missing := map[string]error{"issues": domain.ErrIssueNotFound, "issue-comments": domain.ErrCommentNotFound, "drafts": domain.ErrDraftNotFound}[host]
			for _, tc := range []struct {
				err    error
				status int
			}{{missing, 404}, {shared.Forbidden(), 403}, {shared.InvalidCursor(), 400}, {shared.Invalid(shared.FieldError{Field: "limit", Code: shared.FieldOutOfRange, Message: "out of range"}), 422}, {errGone, 500}} {
				f.err = tc.err
				res, body = do(t, h, request("GET", path, "alice", ""))
				if res.StatusCode != tc.status || strings.Contains(body, errGone.Error()) {
					t.Fatal(tc.err, res.StatusCode, body)
				}
			}
		})
	}
}
