package httpadapter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type fakeReactionAdd struct {
	actor, id uuid.UUID
	value     string
	calls     int
	err       error
}

func (f *fakeReactionAdd) Execute(ctx context.Context, id uuid.UUID, value string) (domain.IssueReaction, error) {
	a, e := shared.RequireActor(ctx)
	if e != nil {
		return domain.IssueReaction{}, e
	}
	f.actor, f.id, f.value, f.calls = a.UserID, id, value, f.calls+1
	return domain.IssueReaction{ID: bobComment, WorkspaceID: webSixID, ProjectID: webSixID, IssueID: id, ActorID: a.UserID, Reaction: value, CreatedAt: created}, f.err
}

type fakeReactionRemove struct {
	id    uuid.UUID
	value string
	calls int
	err   error
}

func (f *fakeReactionRemove) Execute(_ context.Context, id uuid.UUID, value string) error {
	f.id, f.value, f.calls = id, value, f.calls+1
	return f.err
}

type fakeReactionList struct {
	id    uuid.UUID
	calls int
	rows  []domain.IssueReaction
	err   error
}

func (f *fakeReactionList) Execute(_ context.Context, id uuid.UUID) ([]domain.IssueReaction, error) {
	f.id, f.calls = id, f.calls+1
	return f.rows, f.err
}

var reactionsPath = "/api/v0/issues/" + webSixID.String() + "/reactions"

func TestIssueReactionsHTTP(t *testing.T) {
	add, remove, list := &fakeReactionAdd{}, &fakeReactionRemove{}, &fakeReactionList{}
	h := newServer(t, httpadapter.UseCases{AddIssueReaction: add, RemoveIssueReaction: remove, ListIssueReactions: list})
	res, body := do(t, h, request(http.MethodPost, reactionsPath, "alice", `{"reaction":"10084-65039"}`))
	var got map[string]any
	if e := json.Unmarshal([]byte(body), &got); e != nil || res.StatusCode != 201 || len(got) != 5 || got["reaction"] != "10084-65039" || got["issue_id"] != webSixID.String() || got["actor_id"] != aliceID.String() || add.actor != aliceID || add.id != webSixID || add.value != "10084-65039" {
		t.Fatal(res.StatusCode, body, add)
	}
	res, body = do(t, h, request(http.MethodGet, reactionsPath, "alice", ""))
	if res.StatusCode != 200 || body != "{\"data\":[]}\n" || list.id != webSixID {
		t.Fatal(res.StatusCode, body, list)
	}
	list.rows = []domain.IssueReaction{{ID: bobComment, IssueID: webSixID, ActorID: aliceID, Reaction: "128077", CreatedAt: created}}
	res, body = do(t, h, request(http.MethodGet, reactionsPath, "alice", ""))
	got = nil
	if e := json.Unmarshal([]byte(body), &got); e != nil || res.StatusCode != 200 || len(got) != 1 || len(got["data"].([]any)) != 1 || !strings.Contains(body, `"reaction":"128077"`) {
		t.Fatal(res.StatusCode, body)
	}
	res, body = do(t, h, request(http.MethodDelete, reactionsPath+"/10084-65039", "alice", ""))
	if res.StatusCode != 204 || body != "" || remove.id != webSixID || remove.value != "10084-65039" {
		t.Fatal(res.StatusCode, body, remove)
	}
}
func TestIssueReactionsHTTPBinding(t *testing.T) {
	add, remove, list := &fakeReactionAdd{}, &fakeReactionRemove{}, &fakeReactionList{}
	h := newServer(t, httpadapter.UseCases{AddIssueReaction: add, RemoveIssueReaction: remove, ListIssueReactions: list})
	for _, body := range []string{`{}`, `{"reaction":null}`, `{"reaction":7}`, `{"reaction":"1","actor_id":"` + aliceID.String() + `"}`, `{"reaction":"1","extra":true}`} {
		res, got := do(t, h, request(http.MethodPost, reactionsPath, "alice", body))
		if res.StatusCode != 400 {
			t.Fatal(body, res.StatusCode, got)
		}
	}
	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodDelete} {
		path := reactionsPath
		if method == http.MethodDelete {
			path += "/128077"
		}
		res, _ := do(t, h, request(method, path, "", `{"reaction":"128077"}`))
		if res.StatusCode != 401 {
			t.Fatal(res.StatusCode)
		}
	}
	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodDelete} {
		path := "/api/v0/issues/not-a-uuid/reactions"
		if method == http.MethodDelete {
			path += "/128077"
		}
		res, _ := do(t, h, request(method, path, "alice", `{"reaction":"128077"}`))
		if res.StatusCode != 400 {
			t.Fatal(res.StatusCode)
		}
	}
	if add.calls != 0 || remove.calls != 0 || list.calls != 0 {
		t.Fatal("invalid request reached use case")
	}
}
func TestIssueReactionsHTTPRefusals(t *testing.T) {
	for _, tt := range []struct {
		err     error
		status  int
		methods []string
	}{
		{domain.ErrIssueNotFound, 404, []string{"GET", "POST", "DELETE"}}, {shared.Forbidden(), 403, []string{"GET", "POST", "DELETE"}}, {errGone, 500, []string{"GET", "POST", "DELETE"}},
		{domain.ErrReactionExists, 409, []string{"POST"}}, {domain.ErrReactionNotFound, 404, []string{"DELETE"}},
		{domain.ErrProjectArchived, 409, []string{"POST", "DELETE"}}, {domain.ErrIssueArchived, 409, []string{"POST", "DELETE"}}, {shared.ServerBusy(time.Second), 503, []string{"POST", "DELETE"}},
		{shared.Invalid(shared.FieldError{Field: "reaction", Code: shared.FieldInvalidFormat, Message: "invalid code"}), 422, []string{"POST", "DELETE"}},
	} {
		for _, method := range tt.methods {
			h := newServer(t, httpadapter.UseCases{AddIssueReaction: &fakeReactionAdd{err: tt.err}, RemoveIssueReaction: &fakeReactionRemove{err: tt.err}, ListIssueReactions: &fakeReactionList{err: tt.err}})
			path := reactionsPath
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
