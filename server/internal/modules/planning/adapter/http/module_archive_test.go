package httpadapter_test

import (
	"context"
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeArchiveModule is archiveModule or unarchiveModule: each call is recorded as "caller module"; it answers answer,
// or err.
type fakeArchiveModule struct {
	calls  []string
	answer domain.Module
	err    error
}

func (f *fakeArchiveModule) Execute(ctx context.Context, id uuid.UUID) (domain.Module, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Module{}, err
	}
	f.calls = append(f.calls, actor.UserID.String()+" "+id.String())
	return f.answer, f.err
}

// POST .../archive and .../unarchive go to their own use case for the caller and the path's module; the answer is
// 200 with every field of the module it answers.
func TestArchiveModule(t *testing.T) {
	billing, auth := modules(t)
	for _, tt := range []struct {
		path   string
		answer domain.Module
		want   string
	}{{"/archive", billing, billingJSON}, {"/unarchive", auth, authJSON}} {
		archive, unarchive := &fakeArchiveModule{answer: billing}, &fakeArchiveModule{answer: auth}
		h := newServer(t, httpadapter.UseCases{ArchiveModule: archive, UnarchiveModule: unarchive})
		res, body := do(t, h, request(http.MethodPost, "/api/v0/modules/"+tt.answer.ID.String()+tt.path, "alice", ""))
		if res.StatusCode != http.StatusOK || body != tt.want+"\n" {
			t.Errorf("POST %s = %d %s, want 200 %s", tt.path, res.StatusCode, body, tt.want)
		}
		called, idle := archive, unarchive
		if tt.path == "/unarchive" {
			called, idle = unarchive, archive
		}
		if want := []string{aliceID.String() + " " + tt.answer.ID.String()}; !slices.Equal(called.calls, want) || len(idle.calls) != 0 {
			t.Errorf("POST %s: calls %q, the other %q; want %q", tt.path, called.calls, idle.calls, want)
		}
	}
}

// The refusals of a module's archive, unarchive and deletion as the contract declares them, and their failure;
// planning.module_not_closed is archiveModule's alone. A deletion answers 204 without a body.
func TestArchiveAndDeleteModuleRefusals(t *testing.T) {
	id := uuid.NewV7()
	del := &fakeDelete{}
	res, body := do(t, newServer(t, httpadapter.UseCases{DeleteModule: del}), request(http.MethodDelete, "/api/v0/modules/"+id.String(), "alice", ""))
	if res.StatusCode != http.StatusNoContent || body != "" || !slices.Equal(del.calls, []string{aliceID.String() + " " + id.String()}) {
		t.Errorf("DELETE = %d %q, calls %q; want 204", res.StatusCode, body, del.calls)
	}
	for _, tt := range []struct {
		name    string
		err     error
		status  int
		want    string
		archive bool // archiveModule's alone
	}{
		{"no module", domain.ErrModuleNotFound, http.StatusNotFound, moduleNotFoundJSON, false},
		{"a role refused", shared.Forbidden(), http.StatusForbidden, forbiddenJSON, false},
		{"an archived project", domain.ErrProjectArchived, http.StatusConflict,
			`{"status":409,"code":"project.archived","title":"Conflict","detail":"The project is archived; unarchive it to change it."}`, false},
		{"modules turned off", domain.ErrModulesDisabled, http.StatusConflict,
			`{"status":409,"code":"planning.modules_disabled","title":"Conflict","detail":"Modules are turned off in this project; turn them on to change them."}`, false},
		{"not closed", domain.ErrModuleNotClosed, http.StatusConflict,
			`{"status":409,"code":"planning.module_not_closed","title":"Conflict","detail":"Only a completed or cancelled module can be archived."}`, true},
		{"a busy project", shared.ServerBusy(time.Second), http.StatusServiceUnavailable, busy, false},
		{"a failure", errGone, http.StatusInternalServerError, internalErrorJSON, false},
	} {
		h := newServer(t, httpadapter.UseCases{ArchiveModule: &fakeArchiveModule{err: tt.err}, UnarchiveModule: &fakeArchiveModule{err: tt.err},
			DeleteModule: &fakeDelete{err: tt.err}})
		requests := []*http.Request{request(http.MethodPost, "/api/v0/modules/"+id.String()+"/archive", "alice", "")}
		if !tt.archive {
			requests = append(requests, request(http.MethodPost, "/api/v0/modules/"+id.String()+"/unarchive", "alice", ""),
				request(http.MethodDelete, "/api/v0/modules/"+id.String(), "alice", ""))
		}
		for _, req := range requests {
			path := req.Method + " " + req.URL.Path
			res, body := do(t, h, req)
			if res.StatusCode != tt.status || body != tt.want+"\n" || tt.want == busy && res.Header.Get("Retry-After") != "1" {
				t.Errorf("%s: %s = %d %s; want %d %s", tt.name, path, res.StatusCode, body, tt.status, tt.want)
			}
		}
	}
}
