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

// fakeCycleProgress is getCycleProgress: each call is recorded as "caller cycle"; it answers answer, or err.
type fakeCycleProgress struct {
	calls  []string
	answer domain.CycleProgress
	err    error
}

func (f *fakeCycleProgress) Execute(ctx context.Context, id uuid.UUID) (domain.CycleProgress, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.CycleProgress{}, err
	}
	f.calls = append(f.calls, actor.UserID.String()+" "+id.String())
	return f.answer, f.err
}

// fakeModuleProgress is getModuleProgress, as fakeCycleProgress.
type fakeModuleProgress struct {
	calls  []string
	answer domain.Progress
	err    error
}

func (f *fakeModuleProgress) Execute(ctx context.Context, id uuid.UUID) (domain.Progress, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Progress{}, err
	}
	f.calls = append(f.calls, actor.UserID.String()+" "+id.String())
	return f.answer, f.err
}

var bugID = uuid.MustParse("0199a2b4-0000-7000-8000-0000000000b1")

// someProgress is a progress with every kind of value: an assignee's row and the row of none, a label's and the row
// of none, a burndown of three days, the last after today.
func someProgress(t *testing.T) domain.Progress {
	five, three := int64(5), int64(3)
	return domain.Progress{Counts: domain.Counts{Total: 6, Backlog: 1, Unstarted: 1, Started: 1, Completed: 2, Cancelled: 1},
		Assignees: []domain.AssigneeDistribution{{AssigneeID: &aliceID, Tally: domain.Tally{Total: 3, Completed: 1, Pending: 2}},
			{Tally: domain.Tally{Total: 2, Completed: 1, Pending: 1}}},
		Labels: []domain.LabelDistribution{{LabelID: &bugID, Tally: domain.Tally{Total: 4, Completed: 2, Pending: 2}},
			{Tally: domain.Tally{Total: 1, Pending: 1}}},
		Burndown: &domain.Burndown{StartDate: *day(t, "2026-11-09"), EndDate: *day(t, "2026-11-11"), Scope: 5,
			Points: []domain.BurndownPoint{{Date: *day(t, "2026-11-09"), Remaining: &five}, {Date: *day(t, "2026-11-10"), Remaining: &three},
				{Date: *day(t, "2026-11-11")}}}}
}

// someProgressJSON is someProgress as the API answers it.
const someProgressJSON = `{"assignees":[{"assignee_id":"0199a2b4-0000-7000-8000-000000000001","completed_issues":1,"pending_issues":2,` +
	`"total_issues":3},{"assignee_id":null,"completed_issues":1,"pending_issues":1,"total_issues":2}],"backlog_issues":1,` +
	`"burndown":{"end_date":"2026-11-11","points":[{"date":"2026-11-09","remaining":5},{"date":"2026-11-10","remaining":3},` +
	`{"date":"2026-11-11","remaining":null}],"scope":5,"start_date":"2026-11-09"},"cancelled_issues":1,"completed_issues":2,` +
	`"labels":[{"completed_issues":2,"label_id":"0199a2b4-0000-7000-8000-0000000000b1","pending_issues":2,"total_issues":4},` +
	`{"completed_issues":0,"label_id":null,"pending_issues":1,"total_issues":1}],"started_issues":1,"total_issues":6,"unstarted_issues":1}`

// sameJSON reports whether a and b are the same JSON value, whatever the order of their keys.
func sameJSON(t *testing.T, a, b []byte) bool {
	t.Helper()
	var x, y any
	if err := json.Unmarshal(a, &x); err != nil {
		t.Fatalf("%s: %v", a, err)
	}
	if err := json.Unmarshal(b, &y); err != nil {
		t.Fatalf("%s: %v", b, err)
	}
	return reflect.DeepEqual(x, y)
}

// GET of a module's progress goes to the use case for the caller and the path's module; the answer is 200 with every
// number of the progress the use case answers, by the API's names, null for an id or a point it has not. An id that
// is no id is refused before it.
func TestGetModuleProgress(t *testing.T) {
	get := &fakeModuleProgress{answer: someProgress(t)}
	h := newServer(t, httpadapter.UseCases{GetModuleProgress: get})
	module := uuid.NewV7()
	res, body := do(t, h, request(http.MethodGet, "/api/v0/modules/"+module.String()+"/progress", "alice", ""))
	if res.StatusCode != http.StatusOK || body != someProgressJSON+"\n" {
		t.Errorf("GET = %d %s, want 200 %s", res.StatusCode, body, someProgressJSON)
	}
	if want := []string{aliceID.String() + " " + module.String()}; !slices.Equal(get.calls, want) {
		t.Errorf("calls %q, want %q", get.calls, want)
	}
	// Without a burndown, nor any row: null, and empty lists.
	get.answer = domain.Progress{Counts: domain.Counts{Total: 1, Cancelled: 1}, Assignees: []domain.AssigneeDistribution{}, Labels: []domain.LabelDistribution{}}
	want := `{"assignees":[],"backlog_issues":0,"burndown":null,"cancelled_issues":1,"completed_issues":0,"labels":[],"started_issues":0,` +
		`"total_issues":1,"unstarted_issues":0}`
	if res, body := do(t, h, request(http.MethodGet, "/api/v0/modules/"+module.String()+"/progress", "alice", "")); res.StatusCode != http.StatusOK ||
		body != want+"\n" {
		t.Errorf("GET without a burndown = %d %s, want 200 %s", res.StatusCode, body, want)
	}
	if res, body := do(t, h, request(http.MethodGet, "/api/v0/modules/Release/progress", "alice", "")); res.StatusCode != http.StatusBadRequest ||
		len(get.calls) != 2 {
		t.Errorf("GET Release = %d %s, the use case %q; want 400 before it", res.StatusCode, body, get.calls)
	}
}

// GET of a cycle's progress goes to the use case for the caller and the path's cycle; the answer is 200 with its
// status, its live progress and its frozen one, each by the names the domain encodes it with, which a cycle's
// progress_snapshot stores (M6 P4 spec 2.4); null for none frozen. An id that is no id is refused before it.
func TestGetCycleProgress(t *testing.T) {
	frozen := domain.Frozen{Progress: domain.Progress{Counts: domain.Counts{Total: 2, Started: 2},
		Assignees: []domain.AssigneeDistribution{{AssigneeID: &aliceID, Tally: domain.Tally{Total: 2, Pending: 2}}}, Labels: []domain.LabelDistribution{}},
		FrozenAt: time.Date(2026, 11, 12, 1, 2, 3, 456789000, time.UTC)}
	get := &fakeCycleProgress{answer: domain.CycleProgress{Status: domain.StatusCompleted, Live: someProgress(t), Snapshot: &frozen}}
	h := newServer(t, httpadapter.UseCases{GetCycleProgress: get})
	cycle := uuid.NewV7()
	path := "/api/v0/cycles/" + cycle.String() + "/progress"
	res, body := do(t, h, request(http.MethodGet, path, "alice", ""))
	stored, err := domain.EncodeFrozen(frozen)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"status":"completed","live":` + someProgressJSON + `,"progress_snapshot":` + string(stored) + `}`
	if res.StatusCode != http.StatusOK || !sameJSON(t, []byte(body), []byte(want)) {
		t.Errorf("GET = %d %s, want 200 %s", res.StatusCode, body, want)
	}
	if want := []string{aliceID.String() + " " + cycle.String()}; !slices.Equal(get.calls, want) {
		t.Errorf("calls %q, want %q", get.calls, want)
	}
	// Not frozen, or not shown to the caller.
	get.answer.Status, get.answer.Snapshot = domain.StatusCurrent, nil
	want = `{"status":"current","live":` + someProgressJSON + `,"progress_snapshot":null}`
	if res, body := do(t, h, request(http.MethodGet, path, "alice", "")); res.StatusCode != http.StatusOK || !sameJSON(t, []byte(body), []byte(want)) {
		t.Errorf("GET not frozen = %d %s, want 200 %s", res.StatusCode, body, want)
	}
	if res, body := do(t, h, request(http.MethodGet, "/api/v0/cycles/Sprint/progress", "alice", "")); res.StatusCode != http.StatusBadRequest ||
		len(get.calls) != 2 {
		t.Errorf("GET Sprint = %d %s, the use case %q; want 400 before it", res.StatusCode, body, get.calls)
	}
}

// The progress use cases' refusals, as the contract declares them, and their failure.
func TestProgressRefusals(t *testing.T) {
	for _, tt := range []struct {
		name          string
		err           error
		status        int
		cycle, module string
	}{
		{"no cycle", domain.ErrCycleNotFound, http.StatusNotFound, cycleNotFoundJSON, ""},
		{"no module", domain.ErrModuleNotFound, http.StatusNotFound, "", moduleNotFoundJSON},
		{"a role refused", shared.Forbidden(), http.StatusForbidden, forbiddenJSON, forbiddenJSON},
		{"a failure", errGone, http.StatusInternalServerError, internalErrorJSON, internalErrorJSON},
	} {
		h := newServer(t, httpadapter.UseCases{GetCycleProgress: &fakeCycleProgress{err: tt.err}, GetModuleProgress: &fakeModuleProgress{err: tt.err}})
		for _, c := range []struct{ path, want string }{
			{"/api/v0/cycles/" + uuid.NewV7().String() + "/progress", tt.cycle},
			{"/api/v0/modules/" + uuid.NewV7().String() + "/progress", tt.module},
		} {
			if c.want == "" {
				continue
			}
			if res, body := do(t, h, request(http.MethodGet, c.path, "alice", "")); res.StatusCode != tt.status || body != c.want+"\n" {
				t.Errorf("%s: GET %s = %d %s, want %d %s", tt.name, c.path, res.StatusCode, body, tt.status, c.want)
			}
		}
	}
}
