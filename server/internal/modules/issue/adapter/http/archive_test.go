package httpadapter_test

import (
	"context"
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeArchive is archiveIssue, or unarchiveIssue: each call is recorded as
// "caller work item"; it answers answer, or err.
type fakeArchive struct {
	calls  []string
	answer domain.Issue
	err    error
}

func (f *fakeArchive) Execute(ctx context.Context, id uuid.UUID) (domain.Issue, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Issue{}, err
	}
	f.calls = append(f.calls, actor.UserID.String()+" "+id.String())
	return f.answer, f.err
}

// POST archive goes to archiveIssue, and POST unarchive to unarchiveIssue,
// for the caller and the path's work item; the answer is 200 with the work
// item the use case answers, as getIssue shows it.
func TestArchiveAndUnarchiveIssue(t *testing.T) {
	for _, tt := range []struct {
		path   string
		answer domain.Issue
		json   string
	}{{"/archive", webSeven, webSevenJSON}, {"/unarchive", webEight, webEightJSON}} {
		archive, unarchive := &fakeArchive{answer: webSeven}, &fakeArchive{answer: webEight}
		h := newServer(t, httpadapter.UseCases{ArchiveIssue: archive, UnarchiveIssue: unarchive})
		res, body := do(t, h, request(http.MethodPost, "/api/v0/issues/"+tt.answer.ID.String()+tt.path, "alice", ""))
		if res.StatusCode != http.StatusOK || body != tt.json+"\n" {
			t.Errorf("POST %s = %d %s, want 200 %s", tt.path, res.StatusCode, body, tt.json)
		}
		called, other := archive, unarchive
		if tt.path == "/unarchive" {
			called, other = unarchive, archive
		}
		if want := []string{aliceID.String() + " " + tt.answer.ID.String()}; !slices.Equal(called.calls, want) || len(other.calls) != 0 {
			t.Errorf("POST %s: calls %q and %q, want %q and none", tt.path, called.calls, other.calls, want)
		}
	}
}

// The use cases' refusals, as the contract declares them, and their
// failure, each through both operations; issue.not_archivable is
// archiveIssue's alone.
func TestArchiveIssueRefusals(t *testing.T) {
	for _, tt := range []struct {
		name   string
		paths  []string
		err    error
		status int
		want   string
	}{
		{"no work item", both, domain.ErrIssueNotFound, http.StatusNotFound,
			`{"status":404,"code":"issue.not_found","title":"Not Found","detail":"The work item does not exist, or you cannot see it."}`},
		{"a project guest", both, shared.Forbidden(), http.StatusForbidden,
			`{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
		{"an archived project", both, domain.ErrProjectArchived, http.StatusConflict,
			`{"status":409,"code":"project.archived","title":"Conflict","detail":"The project is archived; unarchive it to change it."}`},
		{"a work item in progress", []string{"/archive"}, domain.ErrNotArchivable, http.StatusConflict,
			`{"status":409,"code":"issue.not_archivable","title":"Conflict","detail":"Only a completed or cancelled work item can be archived; ` +
				`move it to such a state first."}`},
		{"a busy project", both, shared.ServerBusy(time.Second), http.StatusServiceUnavailable, busy},
		{"a failure", both, errGone, http.StatusInternalServerError,
			`{"status":500,"code":"internal_error","title":"Internal Server Error"}`},
	} {
		for _, path := range tt.paths {
			uc := &fakeArchive{err: tt.err}
			h := newServer(t, httpadapter.UseCases{ArchiveIssue: uc, UnarchiveIssue: uc})
			res, body := do(t, h, request(http.MethodPost, "/api/v0/issues/"+webSeven.ID.String()+path, "alice", ""))
			if res.StatusCode != tt.status || body != tt.want+"\n" || tt.want == busy && res.Header.Get("Retry-After") != "1" {
				t.Errorf("%s: POST %s = %d %s, Retry-After %q; want %d %s", tt.name, path, res.StatusCode, body, res.Header.Get("Retry-After"),
					tt.status, tt.want)
			}
		}
	}
}

// both are the paths of archiveIssue and unarchiveIssue, after a work
// item's.
var both = []string{"/archive", "/unarchive"}
