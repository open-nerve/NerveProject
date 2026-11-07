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

// fakeListModules is listModules: each call is recorded as "caller project archived"; it answers list, or err.
type fakeListModules struct {
	calls []string
	list  []domain.Module
	err   error
}

func (f *fakeListModules) Execute(ctx context.Context, projectID uuid.UUID, archived bool) ([]domain.Module, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	f.calls = append(f.calls, fmt.Sprintf("%s %s %t", actor.UserID, projectID, archived))
	return f.list, f.err
}

// fakeGetModule is getModule: each call is recorded as "caller module"; it answers answer, or err.
type fakeGetModule struct {
	calls  []string
	answer domain.Module
	err    error
}

func (f *fakeGetModule) Execute(ctx context.Context, id uuid.UUID) (domain.Module, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Module{}, err
	}
	f.calls = append(f.calls, actor.UserID.String()+" "+id.String())
	return f.answer, f.err
}

const moduleNotFoundJSON = `{"status":404,"code":"planning.module_not_found","title":"Not Found","detail":"The module does not exist, or you cannot see it."}`

// GET goes to the use case for the caller, the path's project and archived as given, false when left out; the
// answer is 200 with its modules in its order, each as POST answers one, and [] for none.
func TestListModules(t *testing.T) {
	billing, auth := modules(t)
	path := "/api/v0/projects/" + webID.String() + "/modules"
	for _, tt := range []struct {
		query    string
		archived bool
		list     []domain.Module
		want     string
	}{
		{"", false, []domain.Module{auth, billing}, `{"data":[` + authJSON + `,` + billingJSON + `]}`},
		{"?archived=false", false, nil, `{"data":[]}`},
		{"?archived=true", true, []domain.Module{billing}, `{"data":[` + billingJSON + `]}`},
	} {
		list := &fakeListModules{list: tt.list}
		res, body := do(t, newServer(t, httpadapter.UseCases{ListModules: list}), request(http.MethodGet, path+tt.query, "alice", ""))
		if res.StatusCode != http.StatusOK || body != tt.want+"\n" {
			t.Errorf("GET %s = %d %s, want 200 %s", tt.query, res.StatusCode, body, tt.want)
		}
		if want := []string{fmt.Sprintf("%s %s %t", aliceID, webID, tt.archived)}; !slices.Equal(list.calls, want) {
			t.Errorf("GET %s: calls %q, want %q", tt.query, list.calls, want)
		}
	}
}

// archived that is no boolean is refused as bad_request before the use case (M6 P2 spec R9, as P1's).
func TestListModulesHoldsArchivedToABoolean(t *testing.T) {
	list := &fakeListModules{}
	h := newServer(t, httpadapter.UseCases{ListModules: list})
	for _, query := range []string{"?archived=yes", "?archived=1x", "?archived="} {
		res, got := do(t, h, request(http.MethodGet, "/api/v0/projects/"+webID.String()+"/modules"+query, "alice", ""))
		var problem struct{ Code string }
		if err := json.Unmarshal([]byte(got), &problem); err != nil || res.StatusCode != http.StatusBadRequest || problem.Code != "bad_request" {
			t.Errorf("GET %s = %d %s, want 400 bad_request", query, res.StatusCode, got)
		}
	}
	if len(list.calls) != 0 {
		t.Errorf("the use case got %q, want nothing", list.calls)
	}
}

// GET of a module goes to the use case for the caller and the path's module; the answer is 200 with the module as
// POST answers one, an archived one too. An id that is no id is refused before it.
func TestGetModule(t *testing.T) {
	billing, _ := modules(t)
	get := &fakeGetModule{answer: billing}
	h := newServer(t, httpadapter.UseCases{GetModule: get})
	res, body := do(t, h, request(http.MethodGet, "/api/v0/modules/"+billing.ID.String(), "alice", ""))
	if res.StatusCode != http.StatusOK || body != billingJSON+"\n" {
		t.Errorf("GET = %d %s, want 200 %s", res.StatusCode, body, billingJSON)
	}
	if want := []string{aliceID.String() + " " + billing.ID.String()}; !slices.Equal(get.calls, want) {
		t.Errorf("calls %q, want %q", get.calls, want)
	}
	res, body = do(t, h, request(http.MethodGet, "/api/v0/modules/Billing", "alice", ""))
	if res.StatusCode != http.StatusBadRequest || len(get.calls) != 1 {
		t.Errorf("GET Billing = %d %s, the use case %q; want 400 before it", res.StatusCode, body, get.calls)
	}
}

// The module reads' refusals, as the contract declares them, and their failure.
func TestModuleReadRefusals(t *testing.T) {
	for _, tt := range []struct {
		name   string
		err    error
		status int
		list   string // listModules's answer, "" when it is not one of its
		get    string // getModule's
	}{
		{"no project", domain.ErrProjectNotFound, http.StatusNotFound, projectNotFoundJSON, ""},
		{"no module", domain.ErrModuleNotFound, http.StatusNotFound, "", moduleNotFoundJSON},
		{"a role refused", shared.Forbidden(), http.StatusForbidden, forbiddenJSON, forbiddenJSON},
		{"a failure", errGone, http.StatusInternalServerError, internalErrorJSON, internalErrorJSON},
	} {
		h := newServer(t, httpadapter.UseCases{ListModules: &fakeListModules{err: tt.err}, GetModule: &fakeGetModule{err: tt.err}})
		if tt.list != "" {
			res, body := do(t, h, request(http.MethodGet, "/api/v0/projects/"+webID.String()+"/modules", "alice", ""))
			if res.StatusCode != tt.status || body != tt.list+"\n" {
				t.Errorf("%s: list = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.list)
			}
		}
		if tt.get != "" {
			res, body := do(t, h, request(http.MethodGet, "/api/v0/modules/"+uuid.NewV7().String(), "alice", ""))
			if res.StatusCode != tt.status || body != tt.get+"\n" {
				t.Errorf("%s: get = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.get)
			}
		}
	}
}
