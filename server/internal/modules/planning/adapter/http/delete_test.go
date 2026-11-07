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

// fakeDelete is deleteCycle: each call is recorded as "caller cycle"; it answers err.
type fakeDelete struct {
	calls []string
	err   error
}

func (f *fakeDelete) Execute(ctx context.Context, id uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	f.calls = append(f.calls, actor.UserID.String()+" "+id.String())
	return f.err
}

// DELETE goes to the use case for the caller and the path's cycle; the answer is 204 without a body. Its refusals, as
// the contract declares them, and its failure.
func TestDeleteCycle(t *testing.T) {
	id := uuid.NewV7()
	del := &fakeDelete{}
	res, body := do(t, newServer(t, httpadapter.UseCases{DeleteCycle: del}), request(http.MethodDelete, "/api/v0/cycles/"+id.String(), "alice", ""))
	if res.StatusCode != http.StatusNoContent || body != "" || !slices.Equal(del.calls, []string{aliceID.String() + " " + id.String()}) {
		t.Errorf("DELETE = %d %q, calls %q; want 204", res.StatusCode, body, del.calls)
	}
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"no cycle", domain.ErrCycleNotFound, http.StatusNotFound, cycleNotFoundJSON},
		{"another's cycle", shared.Forbidden(), http.StatusForbidden, forbiddenJSON},
		{"an archived project", domain.ErrProjectArchived, http.StatusConflict,
			`{"status":409,"code":"project.archived","title":"Conflict","detail":"The project is archived; unarchive it to change it."}`},
		{"cycles turned off", domain.ErrCyclesDisabled, http.StatusConflict,
			`{"status":409,"code":"planning.cycles_disabled","title":"Conflict","detail":"Cycles are turned off in this project; turn them on to change them."}`},
		{"a busy project", shared.ServerBusy(time.Second), http.StatusServiceUnavailable, busy},
		{"a failure", errGone, http.StatusInternalServerError, internalErrorJSON},
	} {
		h := newServer(t, httpadapter.UseCases{DeleteCycle: &fakeDelete{err: tt.err}})
		res, body := do(t, h, request(http.MethodDelete, "/api/v0/cycles/"+uuid.NewV7().String(), "alice", ""))
		if res.StatusCode != tt.status || body != tt.want+"\n" || tt.want == busy && res.Header.Get("Retry-After") != "1" {
			t.Errorf("%s: DELETE = %d %s; want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}
