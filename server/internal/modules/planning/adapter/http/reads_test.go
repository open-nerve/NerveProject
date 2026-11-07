package httpadapter_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"testing"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeList is listCycles: each call is recorded as "caller project archived"; it answers list, or err.
type fakeList struct {
	calls []string
	list  []domain.ReadCycle
	err   error
}

func (f *fakeList) Execute(ctx context.Context, projectID uuid.UUID, archived bool) ([]domain.ReadCycle, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	f.calls = append(f.calls, fmt.Sprintf("%s %s %t", actor.UserID, projectID, archived))
	return f.list, f.err
}

// fakeGet is getCycle: each call is recorded as "caller cycle"; it answers answer, or err.
type fakeGet struct {
	calls  []string
	answer domain.ReadCycle
	err    error
}

func (f *fakeGet) Execute(ctx context.Context, id uuid.UUID) (domain.ReadCycle, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.ReadCycle{}, err
	}
	f.calls = append(f.calls, actor.UserID.String()+" "+id.String())
	return f.answer, f.err
}

const (
	projectNotFoundJSON = `{"status":404,"code":"project.not_found","title":"Not Found","detail":"The project does not exist, or you cannot see it."}`
	cycleNotFoundJSON   = `{"status":404,"code":"planning.cycle_not_found","title":"Not Found","detail":"The cycle does not exist, or you cannot see it."}`
	forbiddenJSON       = `{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`
	internalErrorJSON   = `{"status":500,"code":"internal_error","title":"Internal Server Error"}`
)

// GET goes to the use case for the caller, the path's project and archived as given, false when left out; the
// answer is 200 with its cycles in its order, each as POST answers one, and [] for none.
func TestListCycles(t *testing.T) {
	sprint, backlog := cycles(t)
	path := "/api/v0/projects/" + webID.String() + "/cycles"
	for _, tt := range []struct {
		query    string
		archived bool
		list     []domain.ReadCycle
		want     string
	}{
		{"", false, []domain.ReadCycle{backlog, sprint}, `{"data":[` + backlogJSON + `,` + sprintJSON + `]}`},
		{"?archived=false", false, nil, `{"data":[]}`},
		{"?archived=true", true, []domain.ReadCycle{sprint}, `{"data":[` + sprintJSON + `]}`},
	} {
		list := &fakeList{list: tt.list}
		res, body := do(t, newServer(t, httpadapter.UseCases{ListCycles: list}), request(http.MethodGet, path+tt.query, "alice", ""))
		if res.StatusCode != http.StatusOK || body != tt.want+"\n" {
			t.Errorf("GET %s = %d %s, want 200 %s", tt.query, res.StatusCode, body, tt.want)
		}
		if want := []string{fmt.Sprintf("%s %s %t", aliceID, webID, tt.archived)}; !slices.Equal(list.calls, want) {
			t.Errorf("GET %s: calls %q, want %q", tt.query, list.calls, want)
		}
	}
}

// archived that is no boolean is refused as bad_request before the use case (M6 P1 spec R9).
func TestListCyclesHoldsArchivedToABoolean(t *testing.T) {
	list := &fakeList{}
	h := newServer(t, httpadapter.UseCases{ListCycles: list})
	for _, query := range []string{"?archived=yes", "?archived=1x", "?archived="} {
		res, got := do(t, h, request(http.MethodGet, "/api/v0/projects/"+webID.String()+"/cycles"+query, "alice", ""))
		var problem struct{ Code string }
		if err := json.Unmarshal([]byte(got), &problem); err != nil || res.StatusCode != http.StatusBadRequest || problem.Code != "bad_request" {
			t.Errorf("GET %s = %d %s, want 400 bad_request", query, res.StatusCode, got)
		}
	}
	if len(list.calls) != 0 {
		t.Errorf("the use case got %q, want nothing", list.calls)
	}
}

// GET of a cycle goes to the use case for the caller and the path's cycle; the answer is 200 with the cycle as POST
// answers one. An id that is no id is refused before it.
func TestGetCycle(t *testing.T) {
	sprint, _ := cycles(t)
	get := &fakeGet{answer: sprint}
	h := newServer(t, httpadapter.UseCases{GetCycle: get})
	res, body := do(t, h, request(http.MethodGet, "/api/v0/cycles/"+sprint.ID.String(), "alice", ""))
	if res.StatusCode != http.StatusOK || body != sprintJSON+"\n" {
		t.Errorf("GET = %d %s, want 200 %s", res.StatusCode, body, sprintJSON)
	}
	if want := []string{aliceID.String() + " " + sprint.ID.String()}; !slices.Equal(get.calls, want) {
		t.Errorf("calls %q, want %q", get.calls, want)
	}
	res, body = do(t, h, request(http.MethodGet, "/api/v0/cycles/Sprint", "alice", ""))
	if res.StatusCode != http.StatusBadRequest || len(get.calls) != 1 {
		t.Errorf("GET Sprint = %d %s, the use case %q; want 400 before it", res.StatusCode, body, get.calls)
	}
}

// The read use cases' refusals, as the contract declares them, and their failure.
func TestReadRefusals(t *testing.T) {
	for _, tt := range []struct {
		name   string
		err    error
		status int
		list   string // listCycles's answer, "" when it is not one of its
		get    string // getCycle's
	}{
		{"no project", domain.ErrProjectNotFound, http.StatusNotFound, projectNotFoundJSON, ""},
		{"no cycle", domain.ErrCycleNotFound, http.StatusNotFound, "", cycleNotFoundJSON},
		{"a role refused", shared.Forbidden(), http.StatusForbidden, forbiddenJSON, forbiddenJSON},
		{"a failure", errGone, http.StatusInternalServerError, internalErrorJSON, internalErrorJSON},
	} {
		h := newServer(t, httpadapter.UseCases{ListCycles: &fakeList{err: tt.err}, GetCycle: &fakeGet{err: tt.err}})
		if tt.list != "" {
			res, body := do(t, h, request(http.MethodGet, "/api/v0/projects/"+webID.String()+"/cycles", "alice", ""))
			if res.StatusCode != tt.status || body != tt.list+"\n" {
				t.Errorf("%s: list = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.list)
			}
		}
		if tt.get != "" {
			res, body := do(t, h, request(http.MethodGet, "/api/v0/cycles/"+uuid.NewV7().String(), "alice", ""))
			if res.StatusCode != tt.status || body != tt.get+"\n" {
				t.Errorf("%s: get = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.get)
			}
		}
	}
}
