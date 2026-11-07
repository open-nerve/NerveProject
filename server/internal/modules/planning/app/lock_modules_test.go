package app

import (
	"context"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func (w *lockWorld) GetModule(ctx context.Context, id uuid.UUID) (domain.Module, bool, error) {
	w.log(ctx, "GetModule")
	m, ok := w.modules[id]
	return m, ok, nil
}

func (w *lockWorld) GetModuleLink(ctx context.Context, id uuid.UUID) (domain.ModuleLink, bool, error) {
	w.log(ctx, "GetModuleLink")
	l, ok := w.links[id]
	return l, ok, nil
}

func (w *lockWorld) Roles(ctx context.Context, _ uuid.UUID, _ []uuid.UUID) (map[uuid.UUID]shared.Role, error) {
	w.log(ctx, "Roles")
	return nil, nil
}

var (
	lockModule        = uuid.NewV7()
	lockLink          = uuid.NewV7()
	lockModuleCreator = uuid.NewV7()
	lockLinkCreator   = uuid.NewV7()
)

// newModuleWorld is the lock world with a module of the project, and a link of the module that another account
// created; the project's modules and cycles turned on.
func newModuleWorld() *lockWorld {
	w := newLockWorld()
	w.facts[lockProject] = ProjectFacts{WorkspaceID: lockWorkspace, Timezone: "UTC", CycleView: true, ModuleView: true}
	w.modules = map[uuid.UUID]domain.Module{lockModule: {ID: lockModule, ProjectID: lockProject, WorkspaceID: lockWorkspace, CreatedByID: &lockModuleCreator}}
	w.links = map[uuid.UUID]domain.ModuleLink{lockLink: {ID: lockLink, ModuleID: lockModule, ProjectID: lockProject, WorkspaceID: lockWorkspace,
		CreatedByID: &lockLinkCreator}}
	return w
}

func onModule(action shared.Action) write     { m := lockModule; return write{module: &m, action: action} }
func onModuleLink(action shared.Action) write { l := lockLink; return write{link: &l, action: action} }

// moveModule puts the module in another project of the workspace.
func (w *lockWorld) moveModule() {
	m := w.modules[lockModule]
	m.ProjectID = uuid.NewV7()
	w.facts[m.ProjectID] = w.facts[lockProject]
	w.modules[lockModule] = m
}

// A write on a module or a module link takes the cycles' lock path (M6 P2 spec R1): before any lock, what it names is
// read first — a link, then its module — then the project's facts, and the decision; then the gate; in its
// transaction the project's lock first, what it names read again under it, and the decision again. The decision's
// target is the project's, with the module's creator, whom module.delete allows (AllowCreator): a link's write is its
// module's (module.write), so the link's own creator is not the target's.
func TestTheLockPathOnModulesAndLinks(t *testing.T) {
	for _, tt := range []struct {
		name  string
		write write
		calls []string
		link  uuid.UUID
	}{
		{"a module", onModule(domain.ActionModuleWrite), []string{"GetModule outside tx", "Facts outside tx", "Authorize module.write outside tx",
			"Enter 1 outside tx", "Begin", "Lock", "GetModule", "Authorize module.write", "write", "End", "Leave"}, uuid.UUID{}},
		{"a module's deletion", onModule(domain.ActionModuleDelete), []string{"GetModule outside tx", "Facts outside tx", "Authorize module.delete outside tx",
			"Enter 1 outside tx", "Begin", "Lock", "GetModule", "Authorize module.delete", "write", "End", "Leave"}, uuid.UUID{}},
		{"a module link", onModuleLink(domain.ActionModuleWrite), []string{"GetModuleLink outside tx", "GetModule outside tx", "Facts outside tx",
			"Authorize module.write outside tx", "Enter 1 outside tx", "Begin", "Lock", "GetModuleLink", "GetModule", "Authorize module.write",
			"write", "End", "Leave"}, lockLink},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := newModuleWorld()
			h, err := w.run(tt.write)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(w.calls, tt.calls) {
				t.Fatalf("calls\n%q\nwant\n%q", w.calls, tt.calls)
			}
			if h.projectID != lockProject || h.module.ID != lockModule || h.link.ID != tt.link || h.project.WorkspaceID != lockWorkspace || h.grant != w.grant {
				t.Fatalf("held %+v", h)
			}
			want := shared.Target{WorkspaceID: lockWorkspace, ProjectID: lockProject, CreatorID: lockModuleCreator}
			if !slices.Equal(w.targets, []shared.Target{want, want}) {
				t.Fatalf("targets %+v, want the module's creator twice", w.targets)
			}
		})
	}
	w := newModuleWorld()
	if _, err := w.run(onProject(domain.ActionModuleWrite)); err != nil {
		t.Fatal(err)
	}
	want := []string{"Facts outside tx", "Authorize module.write outside tx", "Enter 1 outside tx", "Begin", "Lock", "Authorize module.write", "write", "End", "Leave"}
	if !slices.Equal(w.calls, want) {
		t.Fatalf("a project's module write: calls\n%q\nwant\n%q", w.calls, want)
	}
}

// The toggle a write is held to follows its action (M6 P2 spec R1, R7): a cycle action's the project's cycles, a
// module action's its modules, wherever it is on — the project, a cycle, a module, a module link. The other toggle
// does not refuse it.
func TestTheLockPathsToggleFollowsTheAction(t *testing.T) {
	for _, tt := range []struct {
		name                  string
		write                 write
		cycleView, moduleView bool
		want                  error
	}{
		{"a cycle write, modules off", onCycle(domain.ActionCycleWrite), true, false, nil},
		{"a cycle deletion, modules off", onCycle(domain.ActionCycleDelete), true, false, nil},
		{"a project's cycle write, cycles off", onProject(domain.ActionCycleWrite), false, true, domain.ErrCyclesDisabled},
		{"a cycle deletion, cycles off", onCycle(domain.ActionCycleDelete), false, true, domain.ErrCyclesDisabled},
		{"a project's module write, modules off", onProject(domain.ActionModuleWrite), true, false, domain.ErrModulesDisabled},
		{"a module write, modules off", onModule(domain.ActionModuleWrite), true, false, domain.ErrModulesDisabled},
		{"a module deletion, modules off", onModule(domain.ActionModuleDelete), true, false, domain.ErrModulesDisabled},
		{"a link write, modules off", onModuleLink(domain.ActionModuleWrite), true, false, domain.ErrModulesDisabled},
		{"a project's module write, cycles off", onProject(domain.ActionModuleWrite), false, true, nil},
		{"a module write, cycles off", onModule(domain.ActionModuleWrite), false, true, nil},
		{"a module deletion, cycles off", onModule(domain.ActionModuleDelete), false, true, nil},
		{"a link write, cycles off", onModuleLink(domain.ActionModuleWrite), false, true, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := newModuleWorld()
			w.facts[lockProject] = ProjectFacts{WorkspaceID: lockWorkspace, CycleView: tt.cycleView, ModuleView: tt.moduleView}
			if _, err := w.run(tt.write); !sameRefusal(err, tt.want) || (tt.want == nil) != (err == nil) {
				t.Fatalf("got %v; want %v", err, tt.want)
			}
		})
	}
}

// A module's and a link's refusals in the order of M6 P2 spec R2, as a cycle's: not found (or not visible) before
// forbidden (the role, or an OwnOnly grant on a write) before the archived project before the modules turned off;
// what a link names not found is the link's 404 — the link, or its module, not there or in a project gone. Each case
// also has every later condition, so a refusal answered out of order fails here. No gate nor transaction is taken.
func TestTheModuleLockPathsRefusalsInOrder(t *testing.T) {
	forbidden := shared.Forbidden()
	for _, tt := range []struct {
		name  string
		write write
		set   func(*lockWorld)
		want  error
	}{
		{"no module", onModule(domain.ActionModuleWrite), func(w *lockWorld) { delete(w.modules, lockModule) }, domain.ErrModuleNotFound},
		{"module's project not visible", onModule(domain.ActionModuleWrite), func(w *lockWorld) { w.refusal = shared.ErrNotVisible }, domain.ErrModuleNotFound},
		{"module's project gone", onModule(domain.ActionModuleWrite), func(w *lockWorld) { delete(w.facts, lockProject) }, domain.ErrModuleNotFound},
		{"no link", onModuleLink(domain.ActionModuleWrite), func(w *lockWorld) { delete(w.links, lockLink) }, domain.ErrModuleLinkNotFound},
		{"link's module gone", onModuleLink(domain.ActionModuleWrite), func(w *lockWorld) { delete(w.modules, lockModule) }, domain.ErrModuleLinkNotFound},
		{"link's project gone", onModuleLink(domain.ActionModuleWrite), func(w *lockWorld) { delete(w.facts, lockProject) }, domain.ErrModuleLinkNotFound},
		{"link's project not visible", onModuleLink(domain.ActionModuleWrite), func(w *lockWorld) { w.refusal = shared.ErrNotVisible }, domain.ErrModuleLinkNotFound},
		{"role refused", onModule(domain.ActionModuleDelete), func(w *lockWorld) { w.refusal = forbidden }, forbidden},
		{"own only on a module", onModule(domain.ActionModuleWrite), func(w *lockWorld) { w.grant.OwnOnly = true }, shared.Forbidden()},
		{"own only on a link", onModuleLink(domain.ActionModuleWrite), func(w *lockWorld) { w.grant.OwnOnly = true }, shared.Forbidden()},
		{"own only on a project", onProject(domain.ActionModuleWrite), func(w *lockWorld) { w.grant.OwnOnly = true }, shared.Forbidden()},
		{"project archived", onModuleLink(domain.ActionModuleWrite), func(w *lockWorld) {}, domain.ErrProjectArchived},
		{"modules off", onModule(domain.ActionModuleWrite), func(w *lockWorld) {
			w.facts[lockProject] = ProjectFacts{WorkspaceID: lockWorkspace, CycleView: true}
		}, domain.ErrModulesDisabled},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := newModuleWorld()
			// Every later condition holds too: the project archived and its modules off.
			w.facts[lockProject] = ProjectFacts{WorkspaceID: lockWorkspace, Archived: true, CycleView: true}
			tt.set(w)
			_, err := w.run(tt.write)
			if !sameRefusal(err, tt.want) {
				t.Fatalf("got %v; want %v", err, tt.want)
			}
			if slices.Contains(w.calls, "Begin") || slices.ContainsFunc(w.calls, func(c string) bool { return len(c) > 5 && c[:5] == "Enter" }) {
				t.Fatalf("a precheck refusal took the gate or a transaction: %q", w.calls)
			}
		})
	}
}

// What committed while the lock waited is decided again under the lock, the module and the link read again after the
// project's lock: deleted, moved to another project, a link's module deleted or moved, the project deleted or
// archived, the modules turned off, the caller removed or confined to his own work items. The body does not run.
func TestTheModuleLockPathDecidesAgainUnderTheLock(t *testing.T) {
	for _, tt := range []struct {
		name   string
		write  write
		change func(*lockWorld)
		want   error
	}{
		{"module deleted", onModule(domain.ActionModuleWrite), func(w *lockWorld) { delete(w.modules, lockModule) }, domain.ErrModuleNotFound},
		{"module moved", onModule(domain.ActionModuleDelete), (*lockWorld).moveModule, domain.ErrModuleNotFound},
		{"module's project deleted", onModule(domain.ActionModuleWrite), func(w *lockWorld) { delete(w.facts, lockProject) }, domain.ErrModuleNotFound},
		{"link deleted", onModuleLink(domain.ActionModuleWrite), func(w *lockWorld) { delete(w.links, lockLink) }, domain.ErrModuleLinkNotFound},
		{"link's module deleted", onModuleLink(domain.ActionModuleWrite), func(w *lockWorld) { delete(w.modules, lockModule) }, domain.ErrModuleLinkNotFound},
		{"link's module moved", onModuleLink(domain.ActionModuleWrite), (*lockWorld).moveModule, domain.ErrModuleLinkNotFound},
		{"link's project deleted", onModuleLink(domain.ActionModuleWrite), func(w *lockWorld) { delete(w.facts, lockProject) }, domain.ErrModuleLinkNotFound},
		{"caller removed from a module's project", onModule(domain.ActionModuleWrite), func(w *lockWorld) { w.refusal = shared.ErrNotVisible },
			domain.ErrModuleNotFound},
		{"caller removed from a link's project", onModuleLink(domain.ActionModuleWrite), func(w *lockWorld) { w.refusal = shared.ErrNotVisible },
			domain.ErrModuleLinkNotFound},
		{"guest rule", onModuleLink(domain.ActionModuleWrite), func(w *lockWorld) { w.grant.OwnOnly = true }, shared.Forbidden()},
		{"project archived", onModule(domain.ActionModuleWrite), func(w *lockWorld) {
			w.facts[lockProject] = ProjectFacts{WorkspaceID: lockWorkspace, Archived: true, CycleView: true, ModuleView: true}
		}, domain.ErrProjectArchived},
		{"modules turned off on a project", onProject(domain.ActionModuleWrite), func(w *lockWorld) {
			w.facts[lockProject] = ProjectFacts{WorkspaceID: lockWorkspace, CycleView: true}
		}, domain.ErrModulesDisabled},
		{"modules turned off on a link", onModuleLink(domain.ActionModuleWrite), func(w *lockWorld) {
			w.facts[lockProject] = ProjectFacts{WorkspaceID: lockWorkspace, CycleView: true}
		}, domain.ErrModulesDisabled},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := newModuleWorld()
			w.onLock = func() { tt.change(w) }
			_, err := w.run(tt.write)
			if !sameRefusal(err, tt.want) {
				t.Fatalf("got %v; want %v", err, tt.want)
			}
			if slices.Contains(w.calls, "write") || w.calls[len(w.calls)-1] != "Leave" {
				t.Fatalf("calls %q: the body ran or the gate was not left last", w.calls)
			}
		})
	}
}

// read runs a read of action on what n names, its body recording what it holds.
func (w *lockWorld) read(action shared.Action, n named) (held, error) {
	var got held
	err := w.reads().run(context.Background(), shared.Actor{UserID: uuid.NewV7()}, action, n, func(ctx context.Context, h held) error {
		w.log(ctx, "read")
		got = h
		return nil
	})
	return got, err
}

// A module's read decides module.read (M6 P2 spec R1) in its own snapshot, as a cycle's decides cycle.read: the
// module read first, then its project's facts, the module's creator in the target; a project's read, its facts alone.
// Its refusals are its 404 and the role's 403; an OwnOnly grant, an archived project and modules turned off refuse a
// read nothing (R6, R7).
func TestReadsOnModules(t *testing.T) {
	module := lockModule
	w := newModuleWorld()
	h, err := w.read(domain.ActionModuleRead, named{module: &module})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"Begin", "GetModule", "Facts", "Authorize module.read", "read", "End"}; !slices.Equal(w.calls, want) {
		t.Fatalf("calls\n%q\nwant\n%q", w.calls, want)
	}
	if want := (shared.Target{WorkspaceID: lockWorkspace, ProjectID: lockProject, CreatorID: lockModuleCreator}); h.module.ID != lockModule ||
		!slices.Equal(w.targets, []shared.Target{want}) {
		t.Fatalf("held %+v, targets %+v", h, w.targets)
	}
	w = newModuleWorld()
	if _, err := w.read(domain.ActionModuleRead, named{project: lockProject}); err != nil {
		t.Fatal(err)
	}
	if want := []string{"Begin", "Facts", "Authorize module.read", "read", "End"}; !slices.Equal(w.calls, want) {
		t.Fatalf("a project's calls\n%q\nwant\n%q", w.calls, want)
	}
	forbidden := shared.Forbidden()
	for _, tt := range []struct {
		name                string
		change              func(w *lockWorld)
		moduleWant, project error
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
		if _, err := w.read(domain.ActionModuleRead, named{module: &module}); !sameRefusal(err, tt.moduleWant) || (tt.moduleWant == nil) != (err == nil) {
			t.Errorf("%s: a module's read = %v, want %v", tt.name, err, tt.moduleWant)
		}
		w = newModuleWorld()
		tt.change(w)
		if _, err := w.read(domain.ActionModuleRead, named{project: lockProject}); !sameRefusal(err, tt.project) || (tt.project == nil) != (err == nil) {
			t.Errorf("%s: a project's read = %v, want %v", tt.name, err, tt.project)
		}
	}
}
