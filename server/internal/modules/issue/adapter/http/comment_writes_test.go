package httpadapter_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type fakeCommentDelete struct {
	id, actor uuid.UUID
	calls     int
	err       error
}

func (f *fakeCommentDelete) Execute(ctx context.Context, id uuid.UUID) error {
	a, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	f.id, f.actor, f.calls = id, a.UserID, f.calls+1
	return f.err
}

var commentWritePath = "/api/v0/issue-comments/" + bobComment.String()

func TestCommentWritesHTTP(t *testing.T) {
	f, d := &fakeCommentCreate{}, &fakeCommentDelete{}
	h := newServer(t, httpadapter.UseCases{UpdateIssueComment: f, DeleteIssueComment: d})
	res, body := do(t, h, request(http.MethodPatch, commentWritePath, "alice", `{"comment_html":"<p>Updated</p>"}`))
	if res.StatusCode != 200 || f.actor != aliceID || f.id != bobComment || f.html != "<p>Updated</p>" || !strings.Contains(body, `"reactions":[]`) || strings.Contains(body, "secret") {
		t.Fatal(res.StatusCode, body, f)
	}
	res, body = do(t, h, request(http.MethodDelete, commentWritePath, "alice", ""))
	if res.StatusCode != 204 || body != "" || d.actor != aliceID || d.id != bobComment {
		t.Fatal(res.StatusCode, body, d)
	}
}
func TestCommentWritesHTTPBinding(t *testing.T) {
	f, d := &fakeCommentCreate{}, &fakeCommentDelete{}
	h := newServer(t, httpadapter.UseCases{UpdateIssueComment: f, DeleteIssueComment: d})
	for _, body := range []string{`{}`, `{"comment_html":null}`, `{"comment_html":4}`, `{"comment_html":"x","actor_id":"x"}`, `{"comment_html":"x","edited_at":null}`, `{"comment_html":"x","reactions":[]}`, `{"comment_html":"x","comment_json":{}}`} {
		res, got := do(t, h, request(http.MethodPatch, commentWritePath, "alice", body))
		if res.StatusCode != 400 {
			t.Fatal(res.StatusCode, got)
		}
	}
	for _, method := range []string{http.MethodPatch, http.MethodDelete} {
		res, _ := do(t, h, request(method, commentWritePath, "", `{"comment_html":"x"}`))
		if res.StatusCode != 401 {
			t.Fatal(res.StatusCode)
		}
		res, _ = do(t, h, request(method, "/api/v0/issue-comments/not-a-uuid", "alice", `{"comment_html":"x"}`))
		if res.StatusCode != 400 {
			t.Fatal(res.StatusCode)
		}
	}
	if f.calls != 0 || d.calls != 0 {
		t.Fatal("invalid input reached use cases")
	}
}
func TestCommentWritesHTTPRefusals(t *testing.T) {
	for _, tt := range []struct {
		err    error
		status int
	}{
		{domain.RequiredComment(), 422}, {domain.ErrCommentNotFound, 404}, {domain.ErrNotAuthor, 403}, {shared.Forbidden(), 403}, {domain.ErrProjectArchived, 409}, {domain.ErrIssueArchived, 409}, {shared.ServerBusy(time.Second), 503}, {errGone, 500},
	} {
		for _, method := range []string{http.MethodPatch, http.MethodDelete} {
			if method == http.MethodDelete && tt.status == 422 {
				continue
			}
			f, d := &fakeCommentCreate{err: tt.err}, &fakeCommentDelete{err: tt.err}
			res, body := do(t, newServer(t, httpadapter.UseCases{UpdateIssueComment: f, DeleteIssueComment: d}), request(method, commentWritePath, "alice", `{"comment_html":"hi"}`))
			if res.StatusCode != tt.status {
				t.Fatal(tt.err, res.StatusCode, body)
			}
			if tt.status == 503 && res.Header.Get("Retry-After") != "1" {
				t.Fatal("retry after")
			}
		}
	}
}
