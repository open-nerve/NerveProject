package httpadapter_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
)

func TestCommentAssetsBodyShape(t *testing.T) {
	for _, tc := range []struct{ body, field string }{
		{`{"comment_html":"x","attachment_ids":null}`, "attachment_ids"},
		{`{"comment_html":"x","attachment_ids":{}}`, "attachment_ids"},
		{`{"comment_html":"x","attachment_ids":[null]}`, "attachment_ids[0]"},
		{`{"comment_html":"x","attachment_ids":[4]}`, "attachment_ids[0]"},
		{`{"comment_html":"x","attachment_ids":["private-invalid"]}`, "attachment_ids[0]"},
		{`{"comment_html":"x","attachment_ids":["0199A2B4-7C3E-7D2A-9F10-2B3C4D5E6F70"]}`, "attachment_ids[0]"},
		{`{"comment_html":"x","attachment_ids":["0199a2b47c3e7d2a9f102b3c4d5e6f70"]}`, "attachment_ids[0]"},
		{`{"comment_html":"x","attachment_ids":[],"attachment_ids":[]}`, "attachment_ids"},
	} {
		t.Run(tc.body, func(t *testing.T) {
			f := &fakeCommentCreate{}
			res, body := do(t, newServer(t, httpadapter.UseCases{CreateIssueComment: f}), request("POST", commentsPath, "alice", tc.body))
			if res.StatusCode != 400 || f.calls != 0 || !strings.Contains(body, `"field":"`+tc.field+`"`) || strings.Contains(body, "private-invalid") {
				t.Fatal("unstable structural rejection", res.StatusCode, body, f.calls)
			}
		})
	}
	for _, body := range []string{`{"comment_html":"x"}`, `{"comment_html":"x","attachment_ids":[]}`} {
		f := &fakeCommentCreate{}
		res, got := do(t, newServer(t, httpadapter.UseCases{CreateIssueComment: f}), request("POST", commentsPath, "alice", body))
		if res.StatusCode != 201 || f.calls != 1 {
			t.Fatal("absent/empty attachment list rejected", res.StatusCode, got)
		}
	}
	f := &fakeCommentUpdate{}
	res, body := do(t, newServer(t, httpadapter.UseCases{UpdateIssueComment: f}), request("PATCH", commentWritePath, "alice", `{"comment_html":"x","attachment_ids":[]}`))
	if res.StatusCode != 400 || f.calls != 0 {
		t.Fatal("PATCH accepted replacement attachment list", res.StatusCode, body)
	}
}

func TestCommentAssetsHTTPInput(t *testing.T) {
	f := &fakeCommentCreate{}
	body := fmt.Sprintf(`{"comment_html":"x","attachment_ids":[%q,%q]}`, bobComment, webSixID)
	res, got := do(t, newServer(t, httpadapter.UseCases{CreateIssueComment: f}), request("POST", commentsPath, "alice", body))
	if res.StatusCode != 201 || f.actor != aliceID || f.id != webSixID || !slices.Equal(f.attachments, []uuid.UUID{bobComment, webSixID}) {
		t.Fatal("attachment identities lost in adapter", res.StatusCode, got, f)
	}
}
