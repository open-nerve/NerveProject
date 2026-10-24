package httpadapter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type fakeVersionList struct {
	id     uuid.UUID
	limit  *int
	cursor *string
	calls  int
	page   app.DescriptionVersionPage
	err    error
}

func (f *fakeVersionList) Execute(_ context.Context, id uuid.UUID, limit *int, cursor *string) (app.DescriptionVersionPage, error) {
	f.id, f.limit, f.cursor, f.calls = id, limit, cursor, f.calls+1
	return f.page, f.err
}

type fakeVersionGet struct {
	id      uuid.UUID
	calls   int
	version domain.DescriptionVersion
	err     error
}

func (f *fakeVersionGet) Execute(_ context.Context, id uuid.UUID) (domain.DescriptionVersion, error) {
	f.id, f.calls = id, f.calls+1
	return f.version, f.err
}

var versionsPath = "/api/v0/issues/" + webSixID.String() + "/description-versions"
var savedPath = "/api/v0/issue-description-versions/" + bobComment.String()

func TestDescriptionVersionsHTTP(t *testing.T) {
	for _, query := range []string{"", "?limit=1&cursor=", "?limit=100&cursor=other-position"} {
		f := &fakeVersionList{}
		res, body := do(t, newServer(t, httpadapter.UseCases{ListDescriptionVersions: f}), request(http.MethodGet, versionsPath+query, "alice", ""))
		if res.StatusCode != 200 || body != "{\"data\":[],\"next_cursor\":null}\n" || f.id != webSixID {
			t.Fatal(res.StatusCode, body, f)
		}
		if query == "" {
			if f.limit != nil || f.cursor != nil {
				t.Fatal("absence lost")
			}
		} else if f.limit == nil || f.cursor == nil {
			t.Fatal("presence lost")
		}
	}
	v := domain.DescriptionVersion{ID: bobComment, IssueID: webSixID, OwnedByID: aliceID, HTML: "<p>Saved</p>", Text: "secret", WorkspaceID: acmeID, ProjectID: webID, CreatedByID: aliceID, UpdatedByID: aliceID, LastSavedAt: created, CreatedAt: created, UpdatedAt: created}
	list := &fakeVersionList{page: app.DescriptionVersionPage{Data: []domain.DescriptionVersion{v}, NextCursor: "next"}}
	get := &fakeVersionGet{version: v}
	h := newServer(t, httpadapter.UseCases{ListDescriptionVersions: list, GetDescriptionVersion: get})
	res, body := do(t, h, request(http.MethodGet, versionsPath, "alice", ""))
	var page struct {
		Data []map[string]any
		Next *string `json:"next_cursor"`
	}
	err := json.Unmarshal([]byte(body), &page)
	if err != nil || res.StatusCode != 200 || len(page.Data) != 1 || len(page.Data[0]) != 5 || page.Next == nil || *page.Next != "next" {
		t.Fatal(body, err)
	}
	for _, key := range []string{"id", "issue_id", "owned_by_id", "last_saved_at", "created_at"} {
		if _, ok := page.Data[0][key]; !ok {
			t.Fatal("missing list field", key)
		}
	}
	res, body = do(t, h, request(http.MethodGet, savedPath, "alice", ""))
	var detail map[string]any
	err = json.Unmarshal([]byte(body), &detail)
	if err != nil || res.StatusCode != 200 || len(detail) != 6 || detail["description_html"] != v.HTML || get.id != bobComment {
		t.Fatal(body, err, get)
	}
}

func TestDescriptionVersionsHTTPBinding(t *testing.T) {
	f, g := &fakeVersionList{}, &fakeVersionGet{}
	h := newServer(t, httpadapter.UseCases{ListDescriptionVersions: f, GetDescriptionVersion: g})
	for _, path := range []string{versionsPath, savedPath} {
		res, _ := do(t, h, request(http.MethodGet, path, "", ""))
		if res.StatusCode != 401 {
			t.Fatal(res.StatusCode)
		}
	}
	for _, path := range []string{"/api/v0/issues/bad/description-versions", "/api/v0/issue-description-versions/bad", versionsPath + "?limit=abc"} {
		res, body := do(t, h, request(http.MethodGet, path, "alice", ""))
		if res.StatusCode != 400 {
			t.Fatal(res.StatusCode, body)
		}
	}
	if f.calls != 0 || g.calls != 0 {
		t.Fatal("bad request reached use cases")
	}
}

func TestDescriptionVersionsHTTPRefusals(t *testing.T) {
	for _, tt := range []struct {
		err      error
		status   int
		listOnly bool
	}{
		{shared.InvalidCursor(), 400, true}, {shared.Invalid(shared.FieldError{Field: "limit", Code: shared.FieldOutOfRange, Message: "out of range"}), 422, true}, {domain.ErrIssueNotFound, 404, true}, {domain.ErrDescriptionVersionNotFound, 404, false}, {shared.Forbidden(), 403, false}, {errGone, 500, false},
	} {
		paths := []string{savedPath}
		if tt.listOnly {
			paths = []string{versionsPath}
		} else if tt.status != 404 {
			paths = append(paths, versionsPath)
		}
		for _, path := range paths {
			h := newServer(t, httpadapter.UseCases{ListDescriptionVersions: &fakeVersionList{err: tt.err}, GetDescriptionVersion: &fakeVersionGet{err: tt.err}})
			res, body := do(t, h, request(http.MethodGet, path, "alice", ""))
			if res.StatusCode != tt.status {
				t.Fatal(path, res.StatusCode, body)
			}
		}
	}
}
