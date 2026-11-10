package httpadapter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"testing"
	"time"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeTransfer is transferCycleIssues: each call is recorded as "caller cycle target"; it answers moved, or err.
type fakeTransfer struct {
	calls []string
	moved int
	err   error
}

func (f *fakeTransfer) Execute(ctx context.Context, id, target uuid.UUID) (int, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return 0, err
	}
	f.calls = append(f.calls, actor.UserID.String()+" "+id.String()+" "+target.String())
	return f.moved, f.err
}

// transferPath is the path of a transfer of the cycle id's work items.
func transferPath(id uuid.UUID) string { return "/api/v0/cycles/" + id.String() + "/transfer-issues" }

// POST .../transfer-issues goes to the use case for the caller, the path's cycle and the body's target; the answer is
// 200 with how many it moved, none too.
func TestTransferCycleIssues(t *testing.T) {
	sprint, next := uuid.NewV7(), uuid.NewV7()
	for _, moved := range []int{3, 0} {
		transfer := &fakeTransfer{moved: moved}
		h := newServer(t, httpadapter.UseCases{TransferCycleIssues: transfer})
		res, body := do(t, h, request(http.MethodPost, transferPath(sprint), "alice", `{"target_cycle_id":"`+next.String()+`"}`))
		if want := `{"transferred_issues":` + strconv.Itoa(moved) + "}\n"; res.StatusCode != http.StatusOK || body != want {
			t.Errorf("POST = %d %s, want 200 %s", res.StatusCode, body, want)
		}
		if want := []string{aliceID.String() + " " + sprint.String() + " " + next.String()}; !slices.Equal(transfer.calls, want) {
			t.Errorf("calls %q, want %q", transfer.calls, want)
		}
	}
}

// A body without its target, with null, a value that is no uuid, or a field it may not have: refused as bad_request
// before the use case.
func TestTransferCycleIssuesHoldsTheBodyToItsStructure(t *testing.T) {
	transfer := &fakeTransfer{}
	h := newServer(t, httpadapter.UseCases{TransferCycleIssues: transfer})
	next := uuid.NewV7().String()
	for _, body := range []string{`{}`, `{"target_cycle_id":null}`, `{"target_cycle_id":"next"}`, `{"target_cycle_id":1}`,
		`{"target_cycle_id":"` + next + `","new_cycle_id":"` + next + `"}`, `[]`} {
		res, got := do(t, h, request(http.MethodPost, transferPath(uuid.NewV7()), "alice", body))
		var problem struct{ Code string }
		if err := json.Unmarshal([]byte(got), &problem); err != nil || res.StatusCode != http.StatusBadRequest || problem.Code != "bad_request" {
			t.Errorf("POST %s = %d %s, want 400 bad_request", body, res.StatusCode, got)
		}
	}
	if len(transfer.calls) != 0 {
		t.Errorf("the use case got %q, want nothing", transfer.calls)
	}
}

// The use case's refusals, as the contract declares them, and its failure: a transfer's own messages for the codes
// it shares with other writes.
func TestTransferCycleIssuesRefusals(t *testing.T) {
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"a target not of the project", shared.Invalid(domain.TransferTargetNotAllowed()), http.StatusUnprocessableEntity,
			`{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"target_cycle_id","code":"not_allowed","message":"must be another cycle of the project"}]}`},
		{"no cycle", domain.ErrCycleNotFound, http.StatusNotFound, cycleNotFoundJSON},
		{"a project guest", shared.Forbidden(), http.StatusForbidden, forbiddenJSON},
		{"an archived project", domain.ErrProjectArchived, http.StatusConflict,
			`{"status":409,"code":"project.archived","title":"Conflict","detail":"The project is archived; unarchive it to change it."}`},
		{"cycles turned off", domain.ErrCyclesDisabled, http.StatusConflict,
			`{"status":409,"code":"planning.cycles_disabled","title":"Conflict","detail":"Cycles are turned off in this project; turn them on to change them."}`},
		{"an archived cycle", domain.ErrArchived, http.StatusConflict,
			`{"status":409,"code":"planning.archived","title":"Conflict","detail":"The cycle or module is archived; unarchive it to change it."}`},
		{"a cycle not completed", domain.ErrTransferFromOpenCycle, http.StatusConflict,
			`{"status":409,"code":"planning.cycle_not_completed","title":"Conflict","detail":"Only a completed cycle can transfer its work items."}`},
		{"a completed target", domain.ErrTransferToCompletedCycle, http.StatusConflict,
			`{"status":409,"code":"planning.cycle_completed","title":"Conflict","detail":"A completed cycle cannot take work items."}`},
		{"a busy project", shared.ServerBusy(time.Second), http.StatusServiceUnavailable, busy},
		{"a failure", errGone, http.StatusInternalServerError, internalErrorJSON},
	} {
		h := newServer(t, httpadapter.UseCases{TransferCycleIssues: &fakeTransfer{err: tt.err}})
		res, body := do(t, h, request(http.MethodPost, transferPath(uuid.NewV7()), "alice", `{"target_cycle_id":"`+uuid.NewV7().String()+`"}`))
		if res.StatusCode != tt.status || body != tt.want+"\n" || tt.want == busy && res.Header.Get("Retry-After") != "1" {
			t.Errorf("%s: POST = %d %s; want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}
