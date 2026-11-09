package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakePlanningProjects is project's PlanningProjects answering what it holds,
// recording what it was asked.
type fakePlanningProjects struct {
	facts project.PlanningFacts
	found bool
	roles map[uuid.UUID]shared.Role
	err   error
	asked []string
}

func (f *fakePlanningProjects) PlanningFacts(_ context.Context, id uuid.UUID) (project.PlanningFacts, bool, error) {
	f.asked = append(f.asked, "facts "+id.String())
	return f.facts, f.found, f.err
}

func (f *fakePlanningProjects) LockPlanning(_ context.Context, id uuid.UUID) (project.PlanningFacts, bool, error) {
	f.asked = append(f.asked, "lock "+id.String())
	return f.facts, f.found, f.err
}

func (f *fakePlanningProjects) SharePlanning(_ context.Context, id uuid.UUID) (project.PlanningFacts, bool, error) {
	f.asked = append(f.asked, "share "+id.String())
	return f.facts, f.found, f.err
}

func (f *fakePlanningProjects) ActiveRoles(_ context.Context, id uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]shared.Role, error) {
	f.asked = append(f.asked, fmt.Sprint("roles ", id, " ", ids))
	return f.roles, f.err
}

// planningProjects hands planning project's answers to the same questions:
// the facts, read, locked or shared (M6 P3b spec R6), converted, every
// field; the active roles of the
// accounts asked for in the project asked for (M6 P2 spec R5), as they came;
// found and the error as they came.
func TestPlanningProjectsConvertsProjectsAnswers(t *testing.T) {
	web, acme, ann, bob := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	fake := &fakePlanningProjects{facts: project.PlanningFacts{WorkspaceID: acme, Archived: true, Timezone: "Asia/Shanghai", CycleView: true, ModuleView: true},
		found: true, roles: map[uuid.UUID]shared.Role{ann: shared.RoleAdmin, bob: shared.RoleGuest}}
	p := planningProjects{projects: fake}
	ctx := context.Background()
	want := planning.ProjectFacts{WorkspaceID: acme, Archived: true, Timezone: "Asia/Shanghai", CycleView: true, ModuleView: true}

	if f, found, err := p.Facts(ctx, web); err != nil || !found || f != want {
		t.Errorf("Facts() = %+v, %v, %v; want %+v, found", f, found, err, want)
	}
	if f, found, err := p.Lock(ctx, web); err != nil || !found || f != want {
		t.Errorf("Lock() = %+v, %v, %v; want %+v, found", f, found, err, want)
	}
	if f, found, err := p.Share(ctx, web); err != nil || !found || f != want {
		t.Errorf("Share() = %+v, %v, %v; want %+v, found", f, found, err, want)
	}
	if roles, err := p.Roles(ctx, web, []uuid.UUID{ann, bob}); err != nil || !maps.Equal(roles, fake.roles) {
		t.Errorf("Roles() = %v, %v; want %v", roles, err, fake.roles)
	}
	if want := []string{"facts " + web.String(), "lock " + web.String(), "share " + web.String(), fmt.Sprint("roles ", web, " ", []uuid.UUID{ann, bob})}; !slices.Equal(fake.asked, want) {
		t.Errorf("asked %q, want %q", fake.asked, want)
	}

	boom := errors.New("boom")
	fake.found, fake.err = false, boom
	if _, found, err := p.Lock(ctx, web); !errors.Is(err, boom) || found {
		t.Errorf("Lock() = %v, %v; want the error, not found", found, err)
	}
	if _, found, err := p.Share(ctx, web); !errors.Is(err, boom) || found {
		t.Errorf("Share() = %v, %v; want the error, not found", found, err)
	}
	if _, err := p.Roles(ctx, web, []uuid.UUID{ann}); !errors.Is(err, boom) {
		t.Errorf("Roles() = %v; want the error", err)
	}
}

// fakeIssueTargets is planning's IssueTargets answering what it holds,
// recording what it was asked.
type fakeIssueTargets struct {
	facts         planning.IssueTargetFacts
	completed     bool
	open, openMod []uuid.UUID
	err           error
	asked         []string
}

func (f *fakeIssueTargets) Targets(_ context.Context, project uuid.UUID, cycles, modules []uuid.UUID) (planning.IssueTargetFacts, error) {
	f.asked = append(f.asked, fmt.Sprint("targets ", project, " ", cycles, " ", modules))
	return f.facts, f.err
}

func (f *fakeIssueTargets) CycleCompleted(_ context.Context, project, cycle uuid.UUID, now time.Time) (bool, error) {
	f.asked = append(f.asked, fmt.Sprint("completed ", project, " ", cycle, " ", now.Format(time.RFC3339Nano)))
	return f.completed, f.err
}

func (f *fakeIssueTargets) OpenTargets(_ context.Context, project uuid.UUID, now time.Time) ([]uuid.UUID, []uuid.UUID, error) {
	f.asked = append(f.asked, fmt.Sprint("open ", project, " ", now.Format(time.RFC3339Nano)))
	return f.open, f.openMod, f.err
}

// issuePlanning hands issue planning's answers to the same questions, asked
// of the project, cycles, modules and moment asked about: the facts
// converted, each toggle its own and every target's every field in their
// order; whether a cycle has ended and the open ids as they came; the
// error as it came.
func TestIssuePlanningConvertsPlanningsAnswers(t *testing.T) {
	web, c1, c2, m1, m2 := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	now := time.Date(2026, 11, 7, 17, 0, 0, 1, time.UTC)
	fake := &fakeIssueTargets{
		facts: planning.IssueTargetFacts{CyclesEnabled: true, Cycles: []planning.IssueTarget{{ID: c1, Name: "Sprint 1"}, {ID: c2, Name: "Sprint 2", Archived: true}},
			Modules: []planning.IssueTarget{{ID: m1, Name: "Core", Archived: true}}},
		completed: true, open: []uuid.UUID{c1, c2}, openMod: []uuid.UUID{m2},
	}
	var p issue.PlanningPorts = issuePlanning{targets: fake}
	ctx := context.Background()

	got, err := p.Targets(ctx, web, []uuid.UUID{c1, c2}, []uuid.UUID{m1})
	if err != nil || !got.CyclesEnabled || got.ModulesEnabled ||
		!slices.Equal(got.Cycles, []issue.PlanningTarget{{ID: c1, Name: "Sprint 1"}, {ID: c2, Name: "Sprint 2", Archived: true}}) ||
		!slices.Equal(got.Modules, []issue.PlanningTarget{{ID: m1, Name: "Core", Archived: true}}) {
		t.Errorf("Targets() = %+v, %v; want planning's facts converted", got, err)
	}
	fake.facts = planning.IssueTargetFacts{ModulesEnabled: true}
	if got, err := p.Targets(ctx, web, nil, nil); err != nil || got.CyclesEnabled || !got.ModulesEnabled || len(got.Cycles) != 0 || len(got.Modules) != 0 {
		t.Errorf("Targets() = %+v, %v; want the modules' toggle alone", got, err)
	}
	if completed, err := p.CycleCompleted(ctx, web, c2, now); err != nil || !completed {
		t.Errorf("CycleCompleted() = %v, %v; want completed", completed, err)
	}
	if cycles, modules, err := p.OpenTargets(ctx, web, now); err != nil || !slices.Equal(cycles, fake.open) || !slices.Equal(modules, fake.openMod) {
		t.Errorf("OpenTargets() = %v, %v, %v; want %v, %v", cycles, modules, err, fake.open, fake.openMod)
	}
	at := now.Format(time.RFC3339Nano)
	if want := []string{fmt.Sprint("targets ", web, " ", []uuid.UUID{c1, c2}, " ", []uuid.UUID{m1}), fmt.Sprint("targets ", web, " [] []"),
		fmt.Sprint("completed ", web, " ", c2, " ", at), fmt.Sprint("open ", web, " ", at)}; !slices.Equal(fake.asked, want) {
		t.Errorf("asked %q, want %q", fake.asked, want)
	}

	boom := errors.New("boom")
	fake.err = boom
	if _, err := p.Targets(ctx, web, []uuid.UUID{c1}, nil); !errors.Is(err, boom) {
		t.Errorf("Targets() = %v; want the error", err)
	}
	if _, err := p.CycleCompleted(ctx, web, c1, now); !errors.Is(err, boom) {
		t.Errorf("CycleCompleted() = %v; want the error", err)
	}
	if _, _, err := p.OpenTargets(ctx, web, now); !errors.Is(err, boom) {
		t.Errorf("OpenTargets() = %v; want the error", err)
	}
}

// fakeIssueDirectory is planning's IssueDirectory answering what it holds,
// recording what it was asked.
type fakeIssueDirectory struct {
	cycles, modules []planning.IssueDirectoryName
	err             error
	asked           []string
}

func (f *fakeIssueDirectory) CycleNames(_ context.Context, projects []uuid.UUID) ([]planning.IssueDirectoryName, error) {
	f.asked = append(f.asked, fmt.Sprint("cycles ", projects))
	return f.cycles, f.err
}

func (f *fakeIssueDirectory) ModuleNames(_ context.Context, projects []uuid.UUID) ([]planning.IssueDirectoryName, error) {
	f.asked = append(f.asked, fmt.Sprint("modules ", projects))
	return f.modules, f.err
}

// issuePlanning hands issue planning's names of the cycles and of the
// modules of the projects asked about (M6 P3b spec R2): every name's every
// field, in their order; none as none; the error as it came.
func TestIssuePlanningConvertsPlanningsNames(t *testing.T) {
	web, ops, c1, c2, m1 := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	fake := &fakeIssueDirectory{cycles: []planning.IssueDirectoryName{{ID: c2, Name: "Sprint 2"}, {ID: c1, Name: "Sprint 1"}},
		modules: []planning.IssueDirectoryName{{ID: m1, Name: "Core"}}}
	var p issue.PlanningPorts = issuePlanning{directory: fake}
	ctx := context.Background()

	if got, err := p.CycleNames(ctx, []uuid.UUID{web, ops}); err != nil || !slices.Equal(got, []issue.PlanningName{{ID: c2, Name: "Sprint 2"}, {ID: c1, Name: "Sprint 1"}}) {
		t.Errorf("CycleNames() = %+v, %v; want planning's names converted", got, err)
	}
	if got, err := p.ModuleNames(ctx, []uuid.UUID{ops}); err != nil || !slices.Equal(got, []issue.PlanningName{{ID: m1, Name: "Core"}}) {
		t.Errorf("ModuleNames() = %+v, %v; want planning's names converted", got, err)
	}
	fake.cycles = nil
	if got, err := p.CycleNames(ctx, []uuid.UUID{web}); err != nil || len(got) != 0 {
		t.Errorf("CycleNames() = %+v, %v; want none", got, err)
	}
	if want := []string{fmt.Sprint("cycles ", []uuid.UUID{web, ops}), fmt.Sprint("modules ", []uuid.UUID{ops}), fmt.Sprint("cycles ", []uuid.UUID{web})}; !slices.Equal(fake.asked, want) {
		t.Errorf("asked %q, want %q", fake.asked, want)
	}

	boom := errors.New("boom")
	fake.err = boom
	if _, err := p.CycleNames(ctx, []uuid.UUID{web}); !errors.Is(err, boom) {
		t.Errorf("CycleNames() = %v; want the error", err)
	}
	if _, err := p.ModuleNames(ctx, []uuid.UUID{web}); !errors.Is(err, boom) {
		t.Errorf("ModuleNames() = %v; want the error", err)
	}
}
