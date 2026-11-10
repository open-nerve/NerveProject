package app

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListModules is the lock world's list: the project's modules, archived or not as asked, in the map's id order (a
// store orders them; the use case keeps its order).
func (w *lockWorld) ListModules(ctx context.Context, project uuid.UUID, archived bool) ([]domain.Module, error) {
	if archived {
		w.log(ctx, "ListModules archived")
	} else {
		w.log(ctx, "ListModules")
	}
	var out []domain.Module
	for _, m := range w.modules {
		if m.ProjectID == project && (m.ArchivedAt != nil) == archived {
			out = append(out, m)
		}
	}
	slices.SortFunc(out, func(a, b domain.Module) int { return a.ID.Compare(b.ID) })
	return out, nil
}

func (w *lockWorld) getModule(id uuid.UUID) (domain.Module, error) {
	uc := NewGetModule(GetModuleDeps{Reads: w.reads()})
	return uc.Execute(shared.WithActor(context.Background(), shared.Actor{UserID: uuid.NewV7()}), id)
}

func (w *lockWorld) listModules(project uuid.UUID, archived bool) ([]domain.ModuleListItem, error) {
	return w.listModulesAs(shared.Actor{UserID: uuid.NewV7()}, project, archived)
}

func (w *lockWorld) listModulesAs(actor shared.Actor, project uuid.UUID, archived bool) ([]domain.ModuleListItem, error) {
	uc := NewListModules(ListModulesDeps{Reads: w.reads(), Modules: w, Groups: w, Stats: w})
	return uc.Execute(shared.WithActor(context.Background(), actor), project, archived)
}

// A module's read decides module.read in one transaction of its own, the read-only snapshot (Reads): the module read,
// then its project's facts, then the decision, the module's creator in the target; no gate, no lock, no clock. The
// answer is the module as read, an archived one too (M6 P2 spec 2.3), its members as the store reads them.
func TestGetModule(t *testing.T) {
	w := newModuleWorld()
	m := w.modules[lockModule]
	archived := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	m.Name, m.Status, m.ArchivedAt, m.MemberIDs = "Auth", domain.ModuleCompleted, &archived, ascendingIDs(2)
	w.modules[lockModule] = m
	got, err := w.getModule(lockModule)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"Begin", "GetModule", "Facts", "Authorize module.read", "End"}; !slices.Equal(w.calls, want) {
		t.Fatalf("calls\n%q\nwant\n%q", w.calls, want)
	}
	if !reflect.DeepEqual(got, m) {
		t.Fatalf("got %+v, want the module as read %+v", got, m)
	}
	if want := (shared.Target{WorkspaceID: lockWorkspace, ProjectID: lockProject, CreatorID: lockModuleCreator}); !slices.Equal(w.targets, []shared.Target{want}) {
		t.Fatalf("targets %+v", w.targets)
	}
}

// The project's modules, archived or not as asked, in the store's order, read in the snapshot after the decision on
// the project (module.read); a project without modules lists none.
func TestListModules(t *testing.T) {
	w := newModuleWorld()
	archived := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	second, old, elsewhere := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	w.modules[second] = domain.Module{ID: second, ProjectID: lockProject}
	w.modules[old] = domain.Module{ID: old, ProjectID: lockProject, ArchivedAt: &archived}
	w.modules[elsewhere] = domain.Module{ID: elsewhere, ProjectID: uuid.NewV7()}
	for _, tt := range []struct {
		archived bool
		list     string
		want     []uuid.UUID
	}{{false, "ListModules", []uuid.UUID{lockModule, second}}, {true, "ListModules archived", []uuid.UUID{old}}} {
		w.calls = nil
		got, err := w.listModules(lockProject, tt.archived)
		if err != nil {
			t.Fatal(err)
		}
		if want := []string{"Begin", "Facts", "Authorize module.read", tt.list, "StateGroups", "ModuleCounts", "End"}; !slices.Equal(w.calls, want) {
			t.Fatalf("archived=%t: calls\n%q\nwant\n%q", tt.archived, w.calls, want)
		}
		var ids []uuid.UUID
		for _, m := range got {
			ids = append(ids, m.ID)
		}
		if !slices.Equal(ids, tt.want) {
			t.Errorf("archived=%t: modules %v, want %v", tt.archived, ids, tt.want)
		}
	}
	empty := uuid.NewV7()
	w.facts[empty] = ProjectFacts{WorkspaceID: lockWorkspace}
	if got, err := w.listModules(empty, false); err != nil || len(got) != 0 {
		t.Fatalf("an empty project: %v, %v; want none", got, err)
	}
}

// What a module's read refuses: what it names not there, or not seen, is its 404 — planning.module_not_found for a
// module, whose project is gone too, project.not_found for a project; a role the rule refuses, the Authorizer's 403.
// What a write refuses and a read does not (M6 P2 spec R6, R7): an OwnOnly grant, an archived project, modules turned
// off.
func TestModuleReadsRefuse(t *testing.T) {
	forbidden := shared.Forbidden()
	for _, tt := range []struct {
		name              string
		change            func(w *lockWorld)
		getWant, listWant error
	}{
		{"no module", func(w *lockWorld) { delete(w.modules, lockModule) }, domain.ErrModuleNotFound, nil},
		{"no project", func(w *lockWorld) { delete(w.facts, lockProject) }, domain.ErrModuleNotFound, domain.ErrProjectNotFound},
		{"not seen", func(w *lockWorld) { w.refusal = shared.ErrNotVisible }, domain.ErrModuleNotFound, domain.ErrProjectNotFound},
		{"a role refused", func(w *lockWorld) { w.refusal = forbidden }, forbidden, forbidden},
		{"own only", func(w *lockWorld) { w.grant.OwnOnly = true }, nil, nil},
		{"archived project", func(w *lockWorld) { f := w.facts[lockProject]; f.Archived = true; w.facts[lockProject] = f }, nil, nil},
		{"modules off", func(w *lockWorld) { f := w.facts[lockProject]; f.ModuleView = false; w.facts[lockProject] = f }, nil, nil},
	} {
		w := newModuleWorld()
		tt.change(w)
		if _, err := w.getModule(lockModule); !errors.Is(err, tt.getWant) || (tt.getWant == nil) != (err == nil) {
			t.Errorf("%s: get = %v, want %v", tt.name, err, tt.getWant)
		}
		w = newModuleWorld()
		tt.change(w)
		if _, err := w.listModules(lockProject, false); !errors.Is(err, tt.listWant) || (tt.listWant == nil) != (err == nil) {
			t.Errorf("%s: list = %v, want %v", tt.name, err, tt.listWant)
		}
	}
}

// ascendingIDs are n new accounts in id order.
func ascendingIDs(n int) []uuid.UUID {
	out := make([]uuid.UUID, n)
	for i := range out {
		out[i] = uuid.NewV7()
	}
	slices.SortFunc(out, uuid.UUID.Compare)
	return out
}
