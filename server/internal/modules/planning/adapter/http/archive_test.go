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

// fakeArchive is archiveCycle or unarchiveCycle: each call is recorded as "caller cycle"; it answers answer, or err.
type fakeArchive struct {
	calls  []string
	answer domain.ReadCycle
	err    error
}

func (f *fakeArchive) Execute(ctx context.Context, id uuid.UUID) (domain.ReadCycle, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.ReadCycle{}, err
	}
	f.calls = append(f.calls, actor.UserID.String()+" "+id.String())
	return f.answer, f.err
}

// POST .../archive and .../unarchive go to their own use case for the caller and the path's cycle; the answer is
// 200 with the cycle it answers.
func TestArchiveCycle(t *testing.T) {
	sprint, backlog := cycles(t)
	for _, tt := range []struct {
		path   string
		answer domain.ReadCycle
		want   string
	}{{"/archive", sprint, sprintJSON}, {"/unarchive", backlog, backlogJSON}} {
		archive, unarchive := &fakeArchive{answer: sprint}, &fakeArchive{answer: backlog}
		h := newServer(t, httpadapter.UseCases{ArchiveCycle: archive, UnarchiveCycle: unarchive})
		res, body := do(t, h, request(http.MethodPost, "/api/v0/cycles/"+tt.answer.ID.String()+tt.path, "alice", ""))
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

// The use cases' refusals, as the contract declares them, and their failure; planning.cycle_not_completed is
// archiveCycle's alone.
func TestArchiveCycleRefusals(t *testing.T) {
	for _, tt := range []struct {
		name    string
		err     error
		status  int
		want    string
		archive bool // archiveCycle's alone
	}{
		{"no cycle", domain.ErrCycleNotFound, http.StatusNotFound, cycleNotFoundJSON, false},
		{"a project guest", shared.Forbidden(), http.StatusForbidden, forbiddenJSON, false},
		{"an archived project", domain.ErrProjectArchived, http.StatusConflict,
			`{"status":409,"code":"project.archived","title":"Conflict","detail":"The project is archived; unarchive it to change it."}`, false},
		{"cycles turned off", domain.ErrCyclesDisabled, http.StatusConflict,
			`{"status":409,"code":"planning.cycles_disabled","title":"Conflict","detail":"Cycles are turned off in this project; turn them on to change them."}`, false},
		{"not completed", domain.ErrCycleNotCompleted, http.StatusConflict,
			`{"status":409,"code":"planning.cycle_not_completed","title":"Conflict","detail":"Only a completed cycle can be archived."}`, true},
		{"a busy project", shared.ServerBusy(time.Second), http.StatusServiceUnavailable, busy, false},
		{"a failure", errGone, http.StatusInternalServerError, internalErrorJSON, false},
	} {
		h := newServer(t, httpadapter.UseCases{ArchiveCycle: &fakeArchive{err: tt.err}, UnarchiveCycle: &fakeArchive{err: tt.err}})
		paths := []string{"/archive", "/unarchive"}
		if tt.archive {
			paths = paths[:1]
		}
		for _, path := range paths {
			res, body := do(t, h, request(http.MethodPost, "/api/v0/cycles/"+uuid.NewV7().String()+path, "alice", ""))
			if res.StatusCode != tt.status || body != tt.want+"\n" || tt.want == busy && res.Header.Get("Retry-After") != "1" {
				t.Errorf("%s: POST %s = %d %s; want %d %s", tt.name, path, res.StatusCode, body, tt.status, tt.want)
			}
		}
	}
}
