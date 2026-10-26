package httpadapter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type fakeLinkCreate struct {
	id, actor uuid.UUID
	input     domain.NewLink
	calls     int
	err       error
}

func (f *fakeLinkCreate) Execute(ctx context.Context, id uuid.UUID, input domain.NewLink) (domain.Link, error) {
	a, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Link{}, err
	}
	f.id, f.actor, f.input, f.calls = id, a.UserID, input, f.calls+1
	return domain.Link{ID: bobComment, IssueID: id, URL: input.URL, Title: input.Title, Metadata: domain.LinkMetadata{}, CreatedByID: &a.UserID, CreatedAt: created, UpdatedAt: created}, f.err
}

type fakeLinkList struct {
	id    uuid.UUID
	calls int
	rows  []domain.Link
	err   error
}

func (f *fakeLinkList) Execute(_ context.Context, id uuid.UUID) ([]domain.Link, error) {
	f.id, f.calls = id, f.calls+1
	return f.rows, f.err
}

var linksPath = "/api/v0/issues/" + webSixID.String() + "/links"

func TestCreateIssueLinkHTTP(t *testing.T) {
	for _, title := range []string{"", `,"title":null`, `,"title":""`, `,"title":"  Mine  "`} {
		f := &fakeLinkCreate{}
		res, body := do(t, newServer(t, httpadapter.UseCases{CreateIssueLink: f}), request(http.MethodPost, linksPath, "alice", `{"url":"http://169.254.169.254/P"`+title+`}`))
		var got map[string]any
		if err := json.Unmarshal([]byte(body), &got); err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != 201 || f.id != webSixID || f.actor != aliceID || f.input.URL != "http://169.254.169.254/P" || len(got) != 8 || !reflect.DeepEqual(got["metadata"], map[string]any{}) || got["created_by_id"] != aliceID.String() {
			t.Fatal(res.StatusCode, body, f)
		}
		if title == "" || title == `,"title":null` {
			if f.input.Title != nil || got["title"] != nil {
				t.Fatal("null lost", body)
			}
		} else if f.input.Title == nil || got["title"] != *f.input.Title {
			t.Fatal("title lost", body)
		}
	}
}
func TestListIssueLinksHTTP(t *testing.T) {
	for _, rows := range [][]domain.Link{nil, {{ID: bobComment, IssueID: webSixID, URL: "http://169.254.169.254/P", Metadata: domain.LinkMetadata{"title": nil, "favicon": linkString("data:image/png;base64,aA=="), "url": linkString("http://169.254.169.254/P"), "favicon_url": nil, "error": nil}, CreatedAt: created, UpdatedAt: created}}} {
		f := &fakeLinkList{rows: rows}
		res, body := do(t, newServer(t, httpadapter.UseCases{ListIssueLinks: f}), request(http.MethodGet, linksPath, "alice", ""))
		if res.StatusCode != 200 || f.id != webSixID {
			t.Fatal(res.StatusCode, body)
		}
		if len(rows) == 0 {
			if body != "{\"data\":[]}\n" {
				t.Fatal(body)
			}
		} else if !strings.Contains(body, `"title":null`) || !strings.Contains(body, `"created_by_id":null`) || !strings.Contains(body, `"error":null`) {
			t.Fatal("nullable shape", body)
		}
	}
}
func TestCreateIssueLinkHTTPBinding(t *testing.T) {
	f := &fakeLinkCreate{}
	h := newServer(t, httpadapter.UseCases{CreateIssueLink: f})
	for _, body := range []string{`{}`, `{"url":null}`, `{"url":1}`, `{"url":"http://169.254.169.254","title":1}`, `{"url":"http://169.254.169.254","metadata":{}}`, `{"url":"http://169.254.169.254","created_by_id":null}`} {
		res, got := do(t, h, request(http.MethodPost, linksPath, "alice", body))
		if res.StatusCode != 400 {
			t.Fatal(body, res.StatusCode, got)
		}
	}
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		res, _ := do(t, h, request(method, linksPath, "", `{"url":"x"}`))
		if res.StatusCode != 401 {
			t.Fatal(res.StatusCode)
		}
	}
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		res, _ := do(t, h, request(method, "/api/v0/issues/no-id/links", "alice", `{"url":"x"}`))
		if res.StatusCode != 400 {
			t.Fatal(res.StatusCode)
		}
	}
	if f.calls != 0 {
		t.Fatal("bad input reached use case")
	}
}
func TestCreateIssueLinkHTTPRefusals(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
		post   bool
	}{
		{shared.Invalid(shared.FieldError{Field: "url", Code: shared.FieldNotAllowed, Message: "invalid URL"}), 422, true},
		{domain.ErrIssueNotFound, 404, false}, {shared.Forbidden(), 403, false}, {domain.ErrLinkExists, 409, true}, {domain.ErrProjectArchived, 409, true}, {domain.ErrIssueArchived, 409, true}, {shared.ServerBusy(time.Second), 503, true}, {errGone, 500, false},
	} {
		methods := []string{http.MethodPost}
		if !tc.post {
			methods = append(methods, http.MethodGet)
		}
		for _, method := range methods {
			f, g := &fakeLinkCreate{err: tc.err}, &fakeLinkList{err: tc.err}
			res, body := do(t, newServer(t, httpadapter.UseCases{CreateIssueLink: f, ListIssueLinks: g}), request(method, linksPath, "alice", `{"url":"http://169.254.169.254/P"}`))
			if res.StatusCode != tc.status {
				t.Fatal(method, tc.err, res.StatusCode, body)
			}
			if tc.status == 503 && res.Header.Get("Retry-After") != "1" {
				t.Fatal("Retry-After missing")
			}
		}
	}
}

func linkString(s string) *string { return &s }
