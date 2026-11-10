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
	facts  project.PlanningFacts
	found  bool
	roles  map[uuid.UUID]shared.Role
	states []project.IssueState
	err    error
	asked  []string
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

func (f *fakePlanningProjects) IssueStates(_ context.Context, id uuid.UUID) ([]project.IssueState, error) {
	f.asked = append(f.asked, "states "+id.String())
	return f.states, f.err
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

// planningProjects hands planning the group of each of the states project reads of the project asked about, by state
// (M6 P4 spec R2): every state, the triage state too, whatever its name and whether it is the default; none as none;
// the error as it came.
func TestPlanningProjectsConvertsStateGroups(t *testing.T) {
	web, backlog, done, triage := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	fake := &fakePlanningProjects{states: []project.IssueState{{ID: backlog, Name: "Backlog", Group: "backlog", Default: true},
		{ID: done, Name: "Done", Group: "completed"}, {ID: triage, Name: "Triage", Group: "triage"}}}
	p := planningProjects{projects: fake}
	ctx := context.Background()

	want := map[uuid.UUID]planning.StateGroup{backlog: "backlog", done: "completed", triage: "triage"}
	if got, err := p.StateGroups(ctx, web); err != nil || !maps.Equal(got, want) {
		t.Errorf("StateGroups() = %v, %v; want %v", got, err, want)
	}
	fake.states = nil
	if got, err := p.StateGroups(ctx, web); err != nil || len(got) != 0 {
		t.Errorf("StateGroups() = %v, %v; want none", got, err)
	}
	if want := []string{"states " + web.String(), "states " + web.String()}; !slices.Equal(fake.asked, want) {
		t.Errorf("asked %q, want %q", fake.asked, want)
	}

	boom := errors.New("boom")
	fake.states, fake.err = []project.IssueState{{ID: backlog, Group: "backlog"}}, boom
	if got, err := p.StateGroups(ctx, web); !errors.Is(err, boom) || got != nil {
		t.Errorf("StateGroups() = %v, %v; want the error alone", got, err)
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

// fakePlanningStats is issue's PlanningStats answering what it holds,
// recording what it was asked.
type fakePlanningStats struct {
	items     []issue.PlanningItem
	counts    []issue.PlanningStateCount
	assignees []issue.PlanningAssignee
	err       error
	asked     []string
}

func (f *fakePlanningStats) ask(what string, project uuid.UUID, targets any, viewer *uuid.UUID) {
	by := "everyone"
	if viewer != nil {
		by = viewer.String()
	}
	f.asked = append(f.asked, fmt.Sprint(what, " ", project, " ", targets, " ", by))
}

func (f *fakePlanningStats) CycleItems(_ context.Context, project, cycle uuid.UUID, viewer *uuid.UUID) ([]issue.PlanningItem, error) {
	f.ask("cycle items", project, cycle, viewer)
	return f.items, f.err
}

func (f *fakePlanningStats) ModuleItems(_ context.Context, project, module uuid.UUID, viewer *uuid.UUID) ([]issue.PlanningItem, error) {
	f.ask("module items", project, module, viewer)
	return f.items, f.err
}

func (f *fakePlanningStats) CycleStateCounts(_ context.Context, project uuid.UUID, cycles []uuid.UUID, viewer *uuid.UUID) ([]issue.PlanningStateCount, error) {
	f.ask("cycle counts", project, cycles, viewer)
	return f.counts, f.err
}

func (f *fakePlanningStats) ModuleStateCounts(_ context.Context, project uuid.UUID, modules []uuid.UUID, viewer *uuid.UUID) ([]issue.PlanningStateCount, error) {
	f.ask("module counts", project, modules, viewer)
	return f.counts, f.err
}

func (f *fakePlanningStats) CycleAssignees(_ context.Context, project uuid.UUID, cycles []uuid.UUID, viewer *uuid.UUID) ([]issue.PlanningAssignee, error) {
	f.ask("cycle assignees", project, cycles, viewer)
	return f.assignees, f.err
}

// planningIssues hands planning issue's answers to the same questions (M6 P4 spec R2), asked of the project, the cycle
// or module, or the page of them, and the viewer asked about, an OwnOnly one or none: every work item's every field,
// every count's and every assignee's, in their order; none as none; the error as it came, and nothing with it.
func TestPlanningIssuesConvertsIssuesStats(t *testing.T) {
	web, c1, c2, m1, i1, i2 := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	todo, done, ann, bob, bug := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	at := time.Date(2026, 11, 9, 16, 30, 0, 123456000, time.UTC)
	fake := &fakePlanningStats{
		items: []issue.PlanningItem{{ID: i2, StateID: done, CompletedAt: &at, AssigneeIDs: []uuid.UUID{ann, bob}, LabelIDs: []uuid.UUID{bug}},
			{ID: i1, StateID: todo}},
		counts:    []issue.PlanningStateCount{{TargetID: c2, StateID: done, Issues: 3}, {TargetID: c1, StateID: todo, Issues: 1}},
		assignees: []issue.PlanningAssignee{{CycleID: c2, AssigneeID: bob}, {CycleID: c1, AssigneeID: ann}},
	}
	var p planning.IssueStats = planningIssues{stats: fake}
	ctx := context.Background()

	items := []planning.WorkItem{{ID: i2, StateID: done, CompletedAt: &at, AssigneeIDs: []uuid.UUID{ann, bob}, LabelIDs: []uuid.UUID{bug}},
		{ID: i1, StateID: todo}}
	if got, err := p.CycleItems(ctx, web, c1, &ann); err != nil || !slices.EqualFunc(got, items, sameWorkItem) {
		t.Errorf("CycleItems() = %+v, %v; want %+v", got, err, items)
	}
	if got, err := p.ModuleItems(ctx, web, m1, nil); err != nil || !slices.EqualFunc(got, items, sameWorkItem) {
		t.Errorf("ModuleItems() = %+v, %v; want %+v", got, err, items)
	}
	counts := []planning.StateCount{{TargetID: c2, StateID: done, Issues: 3}, {TargetID: c1, StateID: todo, Issues: 1}}
	if got, err := p.CycleCounts(ctx, web, []uuid.UUID{c1, c2}, nil); err != nil || !slices.Equal(got, counts) {
		t.Errorf("CycleCounts() = %+v, %v; want %+v", got, err, counts)
	}
	if got, err := p.ModuleCounts(ctx, web, []uuid.UUID{m1}, &bob); err != nil || !slices.Equal(got, counts) {
		t.Errorf("ModuleCounts() = %+v, %v; want %+v", got, err, counts)
	}
	assignees := []planning.CycleAssignee{{CycleID: c2, AssigneeID: bob}, {CycleID: c1, AssigneeID: ann}}
	if got, err := p.CycleAssignees(ctx, web, []uuid.UUID{c1, c2}, &ann); err != nil || !slices.Equal(got, assignees) {
		t.Errorf("CycleAssignees() = %+v, %v; want %+v", got, err, assignees)
	}
	if want := []string{fmt.Sprint("cycle items ", web, " ", c1, " ", ann), fmt.Sprint("module items ", web, " ", m1, " everyone"),
		fmt.Sprint("cycle counts ", web, " ", []uuid.UUID{c1, c2}, " everyone"), fmt.Sprint("module counts ", web, " ", []uuid.UUID{m1}, " ", bob),
		fmt.Sprint("cycle assignees ", web, " ", []uuid.UUID{c1, c2}, " ", ann)}; !slices.Equal(fake.asked, want) {
		t.Errorf("asked %q, want %q", fake.asked, want)
	}

	none := &fakePlanningStats{}
	p = planningIssues{stats: none}
	if got, err := p.CycleItems(ctx, web, c1, nil); err != nil || len(got) != 0 {
		t.Errorf("CycleItems() = %+v, %v; want none", got, err)
	}
	if got, err := p.ModuleItems(ctx, web, m1, nil); err != nil || len(got) != 0 {
		t.Errorf("ModuleItems() = %+v, %v; want none", got, err)
	}
	if got, err := p.CycleCounts(ctx, web, nil, nil); err != nil || len(got) != 0 {
		t.Errorf("CycleCounts() = %+v, %v; want none", got, err)
	}
	if got, err := p.ModuleCounts(ctx, web, nil, nil); err != nil || len(got) != 0 {
		t.Errorf("ModuleCounts() = %+v, %v; want none", got, err)
	}
	if got, err := p.CycleAssignees(ctx, web, nil, nil); err != nil || len(got) != 0 {
		t.Errorf("CycleAssignees() = %+v, %v; want none", got, err)
	}

	boom := errors.New("boom")
	fake.err = boom
	p = planningIssues{stats: fake}
	if got, err := p.CycleItems(ctx, web, c1, nil); !errors.Is(err, boom) || got != nil {
		t.Errorf("CycleItems() = %+v, %v; want the error alone", got, err)
	}
	if got, err := p.ModuleItems(ctx, web, m1, nil); !errors.Is(err, boom) || got != nil {
		t.Errorf("ModuleItems() = %+v, %v; want the error alone", got, err)
	}
	if got, err := p.CycleCounts(ctx, web, []uuid.UUID{c1}, nil); !errors.Is(err, boom) || got != nil {
		t.Errorf("CycleCounts() = %+v, %v; want the error alone", got, err)
	}
	if got, err := p.ModuleCounts(ctx, web, []uuid.UUID{m1}, nil); !errors.Is(err, boom) || got != nil {
		t.Errorf("ModuleCounts() = %+v, %v; want the error alone", got, err)
	}
	if got, err := p.CycleAssignees(ctx, web, []uuid.UUID{c1}, nil); !errors.Is(err, boom) || got != nil {
		t.Errorf("CycleAssignees() = %+v, %v; want the error alone", got, err)
	}
}

// sameWorkItem reports whether two work items are the same, every field.
func sameWorkItem(x, y planning.WorkItem) bool {
	return x.ID == y.ID && x.StateID == y.StateID && (x.CompletedAt == nil) == (y.CompletedAt == nil) &&
		(x.CompletedAt == nil || x.CompletedAt.Equal(*y.CompletedAt)) &&
		slices.Equal(x.AssigneeIDs, y.AssigneeIDs) && slices.Equal(x.LabelIDs, y.LabelIDs)
}
