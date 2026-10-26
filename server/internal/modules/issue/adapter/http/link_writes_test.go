package httpadapter_test

import (
	"context"
	"net/http"
	"reflect"
	"testing"
	"time"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type fakeLinkUpdate struct {
	id, actor uuid.UUID
	patch     domain.LinkPatch
	calls     int
	err       error
}

func (f *fakeLinkUpdate) Execute(ctx context.Context, id uuid.UUID, p domain.LinkPatch) (domain.Link, error) {
	a, _ := shared.RequireActor(ctx)
	f.id, f.actor, f.patch, f.calls = id, a.UserID, p, f.calls+1
	return domain.Link{ID: id, IssueID: webSixID, URL: "http://169.254.169.254/P", Title: p.Title, Metadata: domain.LinkMetadata{}, CreatedAt: created, UpdatedAt: created}, f.err
}

type fakeLinkDelete struct {
	id, actor uuid.UUID
	calls     int
	err       error
}

func (f *fakeLinkDelete) Execute(ctx context.Context, id uuid.UUID) error {
	a, _ := shared.RequireActor(ctx)
	f.id, f.actor, f.calls = id, a.UserID, f.calls+1
	return f.err
}

var linkWritePath = "/api/v0/issue-links/" + bobComment.String()

func TestUpdateIssueLinkHTTP(t *testing.T) {
	for _, tc := range []struct {
		body  string
		patch domain.LinkPatch
	}{{`{}`, domain.LinkPatch{}}, {`{"title":null}`, domain.LinkPatch{SetTitle: true}}, {`{"title":""}`, domain.LinkPatch{SetTitle: true, Title: linkString("")}}, {`{"url":"169.254.169.254/P","title":"  Named  "}`, domain.LinkPatch{URL: linkString("169.254.169.254/P"), SetTitle: true, Title: linkString("  Named  ")}}} {
		f := &fakeLinkUpdate{}
		res, body := do(t, newServer(t, httpadapter.UseCases{UpdateIssueLink: f}), request(http.MethodPatch, linkWritePath, "alice", tc.body))
		if res.StatusCode != 200 || f.id != bobComment || f.actor != aliceID || !reflect.DeepEqual(f.patch, tc.patch) {
			t.Fatal(res.StatusCode, body, f)
		}
	}
	f := &fakeLinkDelete{}
	res, body := do(t, newServer(t, httpadapter.UseCases{DeleteIssueLink: f}), request(http.MethodDelete, linkWritePath, "alice", ""))
	if res.StatusCode != 204 || body != "" || f.id != bobComment || f.actor != aliceID {
		t.Fatal(res.StatusCode, body, f)
	}
}
func TestUpdateIssueLinkHTTPBinding(t *testing.T) {
	f, d := &fakeLinkUpdate{}, &fakeLinkDelete{}
	h := newServer(t, httpadapter.UseCases{UpdateIssueLink: f, DeleteIssueLink: d})
	for _, body := range []string{`{"url":null}`, `{"url":1}`, `{"title":1}`, `{"metadata":{}}`, `{"created_by_id":null}`} {
		res, got := do(t, h, request(http.MethodPatch, linkWritePath, "alice", body))
		if res.StatusCode != 400 {
			t.Fatal(res.StatusCode, got)
		}
	}
	for _, method := range []string{http.MethodPatch, http.MethodDelete} {
		res, _ := do(t, h, request(method, linkWritePath, "", `{}`))
		if res.StatusCode != 401 {
			t.Fatal(res.StatusCode)
		}
		res, _ = do(t, h, request(method, "/api/v0/issue-links/no-id", "alice", `{}`))
		if res.StatusCode != 400 {
			t.Fatal(res.StatusCode)
		}
	}
	if f.calls != 0 || d.calls != 0 {
		t.Fatal("invalid request reached use case")
	}
}
func TestUpdateIssueLinkHTTPRefusals(t *testing.T) {
	for _, tc := range []struct {
		err       error
		status    int
		patchOnly bool
	}{
		{shared.Invalid(shared.FieldError{Field: "url", Code: shared.FieldNotAllowed, Message: "invalid URL"}), 422, true},
		{domain.ErrLinkNotFound, 404, false}, {shared.Forbidden(), 403, false}, {domain.ErrLinkExists, 409, true}, {domain.ErrProjectArchived, 409, false}, {domain.ErrIssueArchived, 409, false}, {shared.ServerBusy(time.Second), 503, false}, {errGone, 500, false},
	} {
		methods := []string{http.MethodPatch}
		if !tc.patchOnly {
			methods = append(methods, http.MethodDelete)
		}
		for _, method := range methods {
			f, d := &fakeLinkUpdate{err: tc.err}, &fakeLinkDelete{err: tc.err}
			res, body := do(t, newServer(t, httpadapter.UseCases{UpdateIssueLink: f, DeleteIssueLink: d}), request(method, linkWritePath, "alice", `{}`))
			if res.StatusCode != tc.status {
				t.Fatal(method, tc.err, res.StatusCode, body)
			}
			if tc.status == 503 && res.Header.Get("Retry-After") != "1" {
				t.Fatal("missing Retry-After")
			}
		}
	}
}
