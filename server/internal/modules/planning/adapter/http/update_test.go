package httpadapter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeUpdate is updateCycle: each call is recorded as "caller cycle", with what it got; it answers answer, or err.
type fakeUpdate struct {
	calls  []string
	got    []domain.CyclePatch
	answer domain.ReadCycle
	err    error
}

func (f *fakeUpdate) Execute(ctx context.Context, id uuid.UUID, p domain.CyclePatch) (domain.ReadCycle, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.ReadCycle{}, err
	}
	f.calls = append(f.calls, actor.UserID.String()+" "+id.String())
	f.got = append(f.got, p)
	return f.answer, f.err
}

func text(s string) *string { return &s }

// PATCH goes to the use case for the caller and the path's cycle with the fields given as given (the use case cleans
// them), a date left out not set, one null set to none; the answer is 200 with the cycle the use case answers.
func TestUpdateCycle(t *testing.T) {
	sprint, _ := cycles(t)
	order := -2.5
	for _, tt := range []struct {
		body string
		want domain.CyclePatch
	}{
		{`{}`, domain.CyclePatch{}},
		{`{"name":" Two ","description":"","sort_order":-2.5}`, domain.CyclePatch{Name: text(" Two "), Description: text(""), SortOrder: &order}},
		{`{"start_date":"2026-11-02","end_date":null}`, domain.CyclePatch{SetStart: true, Start: day(t, "2026-11-02"), SetEnd: true}},
		{`{"end_date":"2026-11-20"}`, domain.CyclePatch{SetEnd: true, End: day(t, "2026-11-20")}},
	} {
		update := &fakeUpdate{answer: sprint}
		res, body := do(t, newServer(t, httpadapter.UseCases{UpdateCycle: update}), request(http.MethodPatch, "/api/v0/cycles/"+sprint.ID.String(), "alice", tt.body))
		if res.StatusCode != http.StatusOK || body != sprintJSON+"\n" {
			t.Errorf("PATCH %s = %d %s, want 200 %s", tt.body, res.StatusCode, body, sprintJSON)
		}
		if want := []string{aliceID.String() + " " + sprint.ID.String()}; !slices.Equal(update.calls, want) || !reflect.DeepEqual(update.got, []domain.CyclePatch{tt.want}) {
			t.Errorf("PATCH %s: the use case got %q %+v, want %q %+v", tt.body, update.calls, update.got, want, tt.want)
		}
	}
}

// A body with a field it may not have (the status, owner, archive and creation are the server's), null for a field
// that is not nullable, a field of another type, or a date that is no day: refused as bad_request before the use case.
func TestUpdateCycleHoldsTheBodyToItsStructure(t *testing.T) {
	update := &fakeUpdate{}
	h := newServer(t, httpadapter.UseCases{UpdateCycle: update})
	for _, body := range []string{`{"status":"draft"}`, `{"owned_by_id":null}`, `{"archived_at":null}`, `{"project_id":"0199a2b4-0000-7000-8000-0000000000a1"}`,
		`{"name":null}`, `{"description":null}`, `{"sort_order":null}`, `{"sort_order":"1"}`, `{"name":1}`, `{"start_date":"2026-02-30"}`,
		`{"end_date":"2026-11-20T00:00:00Z"}`} {
		res, got := do(t, h, request(http.MethodPatch, "/api/v0/cycles/"+uuid.NewV7().String(), "alice", body))
		var problem struct{ Code string }
		if err := json.Unmarshal([]byte(got), &problem); err != nil || res.StatusCode != http.StatusBadRequest || problem.Code != "bad_request" {
			t.Errorf("PATCH %s = %d %s, want 400 bad_request", body, res.StatusCode, got)
		}
	}
	if len(update.calls) != 0 {
		t.Errorf("the use case got %q, want nothing", update.calls)
	}
}

// The use case's refusals, as the contract declares them, and its failure.
func TestUpdateCycleRefusals(t *testing.T) {
	_, empty := domain.CheckCyclePatch(domain.CyclePatch{Name: text(" ")})
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"an empty name", empty, http.StatusUnprocessableEntity,
			`{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"name","code":"too_short","message":"must not be empty"}]}`},
		{"no cycle", domain.ErrCycleNotFound, http.StatusNotFound, cycleNotFoundJSON},
		{"a project guest", shared.Forbidden(), http.StatusForbidden, forbiddenJSON},
		{"an archived project", domain.ErrProjectArchived, http.StatusConflict,
			`{"status":409,"code":"project.archived","title":"Conflict","detail":"The project is archived; unarchive it to change it."}`},
		{"cycles turned off", domain.ErrCyclesDisabled, http.StatusConflict,
			`{"status":409,"code":"planning.cycles_disabled","title":"Conflict","detail":"Cycles are turned off in this project; turn them on to change them."}`},
		{"an archived cycle", domain.ErrArchived, http.StatusConflict,
			`{"status":409,"code":"planning.archived","title":"Conflict","detail":"The cycle is archived; unarchive it to change it."}`},
		{"a completed cycle", domain.ErrCycleCompleted, http.StatusConflict,
			`{"status":409,"code":"planning.cycle_completed","title":"Conflict","detail":"The cycle is completed; only its order can change."}`},
		{"overlapping dates", domain.ErrDatesOverlap, http.StatusConflict,
			`{"status":409,"code":"planning.cycle_dates_overlap","title":"Conflict","detail":"The dates overlap another cycle of this project."}`},
		{"a busy project", shared.ServerBusy(time.Second), http.StatusServiceUnavailable, busy},
		{"a failure", errGone, http.StatusInternalServerError, internalErrorJSON},
	} {
		h := newServer(t, httpadapter.UseCases{UpdateCycle: &fakeUpdate{err: tt.err}})
		res, body := do(t, h, request(http.MethodPatch, "/api/v0/cycles/"+uuid.NewV7().String(), "alice", `{"name":"A"}`))
		if res.StatusCode != tt.status || body != tt.want+"\n" || tt.want == busy && res.Header.Get("Retry-After") != "1" {
			t.Errorf("%s: PATCH = %d %s; want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}
