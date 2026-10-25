package httpadapter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type fakeCommentReactionAdd struct {
	actor, id uuid.UUID
	value     string
	calls     int
	err       error
}

func (f *fakeCommentReactionAdd) Execute(ctx context.Context, id uuid.UUID, value string) (domain.CommentReaction, error) {
	a, e := shared.RequireActor(ctx)
	if e != nil {
		return domain.CommentReaction{}, e
	}
	f.actor, f.id, f.value, f.calls = a.UserID, id, value, f.calls+1
	return domain.CommentReaction{ID: bobComment, WorkspaceID: webSixID, ProjectID: webSixID, CommentID: id, ActorID: a.UserID, Reaction: value, CreatedAt: created}, f.err
}

type fakeCommentReactionRemove struct {
	id    uuid.UUID
	value string
	calls int
	err   error
}

func (f *fakeCommentReactionRemove) Execute(_ context.Context, id uuid.UUID, value string) error {
	f.id, f.value, f.calls = id, value, f.calls+1
	return f.err
}

var commentReactionsPath = "/api/v0/issue-comments/" + webSixID.String() + "/reactions"

func TestCommentReactionsHTTP(t *testing.T) {
	add, remove := &fakeCommentReactionAdd{}, &fakeCommentReactionRemove{}
	h := newServer(t, httpadapter.UseCases{AddCommentReaction: add, RemoveCommentReaction: remove})
	res, body := do(t, h, request(http.MethodPost, commentReactionsPath, "alice", `{"reaction":"10084-65039"}`))
	var got map[string]any
	if e := json.Unmarshal([]byte(body), &got); e != nil || res.StatusCode != 201 || len(got) != 5 || got["reaction"] != "10084-65039" || got["comment_id"] != webSixID.String() || got["actor_id"] != aliceID.String() || add.actor != aliceID || add.id != webSixID || add.value != "10084-65039" {
		t.Fatal(res.StatusCode, body, add)
	}
	res, body = do(t, h, request(http.MethodDelete, commentReactionsPath+"/10084-65039", "alice", ""))
	if res.StatusCode != 204 || body != "" || remove.id != webSixID || remove.value != "10084-65039" {
		t.Fatal(res.StatusCode, body, remove)
	}
}
func TestCommentReactionsHTTPBinding(t *testing.T) {
	add, remove := &fakeCommentReactionAdd{}, &fakeCommentReactionRemove{}
	h := newServer(t, httpadapter.UseCases{AddCommentReaction: add, RemoveCommentReaction: remove})
	for _, body := range []string{`{}`, `{"reaction":null}`, `{"reaction":7}`, `{"reaction":"1","actor_id":"` + aliceID.String() + `"}`, `{"reaction":"1","extra":true}`} {
		res, got := do(t, h, request(http.MethodPost, commentReactionsPath, "alice", body))
		if res.StatusCode != 400 {
			t.Fatal(body, res.StatusCode, got)
		}
	}
	for _, method := range []string{http.MethodPost, http.MethodDelete} {
		path := commentReactionsPath
		if method == http.MethodDelete {
			path += "/128077"
		}
		res, _ := do(t, h, request(method, path, "", `{"reaction":"128077"}`))
		if res.StatusCode != 401 {
			t.Fatal(res.StatusCode)
		}
	}
	for _, method := range []string{http.MethodPost, http.MethodDelete} {
		path := "/api/v0/issue-comments/not-a-uuid/reactions"
		if method == http.MethodDelete {
			path += "/128077"
		}
		res, _ := do(t, h, request(method, path, "alice", `{"reaction":"128077"}`))
		if res.StatusCode != 400 {
			t.Fatal(res.StatusCode)
		}
	}
	if add.calls != 0 || remove.calls != 0 {
		t.Fatal("invalid request reached use case")
	}
}
func TestCommentReactionsHTTPRefusals(t *testing.T) {
	for _, tt := range []struct {
		err     error
		status  int
		methods []string
	}{
		{domain.ErrCommentNotFound, 404, []string{"POST", "DELETE"}}, {shared.Forbidden(), 403, []string{"POST", "DELETE"}}, {errGone, 500, []string{"POST", "DELETE"}},
		{domain.ErrReactionExists, 409, []string{"POST"}}, {domain.ErrReactionNotFound, 404, []string{"DELETE"}},
		{domain.ErrProjectArchived, 409, []string{"POST", "DELETE"}}, {domain.ErrIssueArchived, 409, []string{"POST", "DELETE"}}, {shared.ServerBusy(time.Second), 503, []string{"POST", "DELETE"}},
		{shared.Invalid(shared.FieldError{Field: "reaction", Code: shared.FieldInvalidFormat, Message: "invalid code"}), 422, []string{"POST", "DELETE"}},
	} {
		for _, method := range tt.methods {
			h := newServer(t, httpadapter.UseCases{AddCommentReaction: &fakeCommentReactionAdd{err: tt.err}, RemoveCommentReaction: &fakeCommentReactionRemove{err: tt.err}})
			path := commentReactionsPath
			if method == http.MethodDelete {
				path += "/128077"
			}
			res, body := do(t, h, request(method, path, "alice", `{"reaction":"128077"}`))
			if res.StatusCode != tt.status {
				t.Fatal(method, tt.err, res.StatusCode, body)
			}
			if tt.status == 503 && res.Header.Get("Retry-After") != "1" {
				t.Fatal("retry after")
			}
		}
	}
}
