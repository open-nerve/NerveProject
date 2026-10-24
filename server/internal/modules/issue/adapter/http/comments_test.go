package httpadapter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type fakeCommentCreate struct {
	actor, id uuid.UUID
	html      string
	calls     int
	err       error
}

func (f *fakeCommentCreate) Execute(ctx context.Context, id uuid.UUID, html string) (domain.Comment, error) {
	a, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Comment{}, err
	}
	f.actor, f.id, f.html, f.calls = a.UserID, id, html, f.calls+1
	return domain.Comment{ID: bobComment, IssueID: id, ActorID: a.UserID, HTML: html, Text: "secret", CreatedAt: created, UpdatedAt: created}, f.err
}

type fakeCommentList struct {
	limit  *int
	cursor *string
	id     uuid.UUID
	calls  int
	err    error
	page   app.CommentPage
}

func (f *fakeCommentList) Execute(_ context.Context, id uuid.UUID, limit *int, cursor *string) (app.CommentPage, error) {
	f.id, f.limit, f.cursor, f.calls = id, limit, cursor, f.calls+1
	return f.page, f.err
}

var bobComment = uuid.MustParse("0199a2b4-0000-7000-8000-0000000000c1")
var commentsPath = "/api/v0/issues/" + webSixID.String() + "/comments"

func TestCommentsHTTP(t *testing.T) {
	f := &fakeCommentCreate{}
	res, body := do(t, newServer(t, httpadapter.UseCases{CreateIssueComment: f}), request(http.MethodPost, commentsPath, "alice", `{"comment_html":"<p>Hello</p>"}`))
	var got map[string]any
	decode := json.Unmarshal([]byte(body), &got)
	if decode != nil || res.StatusCode != 201 || f.actor != aliceID || f.id != webSixID || f.html != "<p>Hello</p>" || len(got) != 8 || got["edited_at"] != nil || !slices.Equal(got["reactions"].([]any), []any{}) || got["comment_html"] != "<p>Hello</p>" {
		t.Fatal(res.StatusCode, body, f)
	}
	for _, query := range []string{"", "?limit=1&cursor=", "?limit=100&cursor=some"} {
		f := &fakeCommentList{}
		res, body := do(t, newServer(t, httpadapter.UseCases{ListIssueComments: f}), request(http.MethodGet, commentsPath+query, "alice", ""))
		if res.StatusCode != 200 || body != "{\"data\":[],\"next_cursor\":null}\n" || f.id != webSixID {
			t.Fatal(res.StatusCode, body)
		}
		if query == "" {
			if f.limit != nil || f.cursor != nil {
				t.Fatal("absence lost")
			}
		} else if f.limit == nil || f.cursor == nil {
			t.Fatal("presence lost")
		}
	}
	fList := &fakeCommentList{page: app.CommentPage{Data: []domain.Comment{{ID: bobComment, IssueID: webSixID, ActorID: aliceID, HTML: "hi", CreatedAt: created, UpdatedAt: created}}, NextCursor: "next"}}
	res, body = do(t, newServer(t, httpadapter.UseCases{ListIssueComments: fList}), request(http.MethodGet, commentsPath, "alice", ""))
	if res.StatusCode != 200 || !strings.Contains(body, `"next_cursor":"next"`) || !strings.Contains(body, `"reactions":[]`) {
		t.Fatal(body)
	}
}

func TestCommentsHTTPBinding(t *testing.T) {
	f := &fakeCommentCreate{}
	h := newServer(t, httpadapter.UseCases{CreateIssueComment: f})
	for _, body := range []string{`{}`, `{"comment_html":null}`, `{"comment_html":5}`, `{"comment_html":"x","actor_id":"` + aliceID.String() + `"}`, `{"comment_html":"x","access":"INTERNAL"}`, `{"comment_html":"x","parent_id":null}`, `{"comment_html":"x","comment_json":{}}`, `{"comment_html":"x","attachments":[]}`, `{"comment_html":"x","description_id":null}`} {
		res, got := do(t, h, request(http.MethodPost, commentsPath, "alice", body))
		if res.StatusCode != 400 {
			t.Fatal(body, res.StatusCode, got)
		}
	}
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		res, _ := do(t, h, request(method, commentsPath, "", `{"comment_html":"hi"}`))
		if res.StatusCode != 401 {
			t.Fatal(res.StatusCode)
		}
	}
	if f.calls != 0 {
		t.Fatal("bad input reached use case")
	}
	for _, path := range []string{"/api/v0/issues/not-a-uuid/comments", commentsPath + "?limit=abc"} {
		res, _ := do(t, newServer(t, httpadapter.UseCases{}), request(http.MethodGet, path, "alice", ""))
		if res.StatusCode != 400 {
			t.Fatal(res.StatusCode)
		}
	}
}

func TestCommentsHTTPRefusals(t *testing.T) {
	for _, tt := range []struct {
		err      error
		status   int
		postOnly bool
	}{
		{shared.InvalidCursor(), 400, false}, {shared.Invalid(shared.FieldError{Field: "limit", Code: shared.FieldOutOfRange, Message: "out of range"}), 422, false},
		{domain.RequiredComment(), 422, true}, {domain.ErrIssueNotFound, 404, false}, {shared.Forbidden(), 403, false}, {domain.ErrProjectArchived, 409, true}, {domain.ErrIssueArchived, 409, true}, {shared.ServerBusy(time.Second), 503, true}, {errGone, 500, false},
	} {
		methods := []string{http.MethodPost}
		if !tt.postOnly {
			methods = append(methods, http.MethodGet)
		}
		for _, method := range methods {
			if method == http.MethodPost && tt.status == 400 {
				continue
			}
			f, g := &fakeCommentCreate{err: tt.err}, &fakeCommentList{err: tt.err}
			res, body := do(t, newServer(t, httpadapter.UseCases{CreateIssueComment: f, ListIssueComments: g}), request(method, commentsPath, "alice", `{"comment_html":"hi"}`))
			if res.StatusCode != tt.status {
				t.Fatal(tt.err, res.StatusCode, body)
			}
			if tt.status == 503 && res.Header.Get("Retry-After") != "1" {
				t.Fatal("retry after")
			}
		}
	}
}
