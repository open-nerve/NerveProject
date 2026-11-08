package app_test

import (
	"errors"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// web's modules, drawn in this order, ascending: moduleGone, which is not
// one of web's undeleted modules; Auth and Billing, open; Old, archived;
// and moduleLost, not web's either, drawn last.
var (
	moduleGone    = uuid.NewV7()
	moduleAuth    = app.PlanningTarget{ID: uuid.NewV7(), Name: "Auth"}
	moduleBilling = app.PlanningTarget{ID: uuid.NewV7(), Name: "Billing"}
	moduleOld     = app.PlanningTarget{ID: uuid.NewV7(), Name: "Old", Archived: true}
	moduleLost    = uuid.NewV7()
)

// modulePlanning is planningOf(w) with web's three modules.
func modulePlanning(w *world) *fakePlanning {
	p := planningOf(w)
	p.modules = []app.PlanningTarget{moduleAuth, moduleBilling, moduleOld}
	return p
}

// namedModules are the modules ts as an activity names them.
func namedModules(ts ...app.PlanningTarget) []domain.Named {
	var out []domain.Named
	for _, t := range ts {
		out = append(out, domain.Named{ID: t.ID, Name: t.Name})
	}
	return out
}

// idsOf are the ids of ts, in their order.
func idsOf(ts ...app.PlanningTarget) []uuid.UUID {
	var out []uuid.UUID
	for _, t := range ts {
		out = append(out, t.ID)
	}
	return out
}

// A creation that joins modules, given in any order (M6 P3a spec 2.3, R1,
// R4, R5): planning read once under the lock, first, with the cycle it
// joins too when it joins one, the modules in id order; the work item, its
// cycle's row, its modules' rows, in id order; its activity, the
// creation's row, the cycle's, then each module's, in id order; the answer
// in them; issue.created's changes end with the cycle, then the modules.
func TestCreateIssueJoinsModules(t *testing.T) {
	asked := idsOf(moduleAuth, moduleBilling)
	for _, cycle := range []*app.PlanningTarget{nil, &cycleNow} {
		_, w := newCreate()
		events := &recordedEvents{t: t}
		in := domain.NewIssue{Name: "Fix", ModuleIDs: idsOf(moduleBilling, moduleAuth)}
		targets, rows := line("Targets", web, []uuid.UUID(nil), asked), 3
		var ended, inCycle []string
		var joined *domain.Named
		if cycle != nil {
			in.CycleID, joined = &cycle.ID, namedCycle(*cycle)
			targets, rows = line("Targets", web, []uuid.UUID{cycle.ID}, asked), 4
			ended, inCycle = []string{completedAt(cycle.ID)}, []string{line("InsertCycle", []uuid.UUID{cycle.ID})}
		}
		got, err := joinCreate(w, modulePlanning(w), events).Execute(as(alice), web, in)
		if err != nil || !slices.Equal(got.ModuleIDs, asked) || !reflect.DeepEqual(got.CycleID, in.CycleID) {
			t.Fatalf("Execute() = %+v, %v; want the work item in Auth and Billing, in the cycle %v", got, err, in.CycleID)
		}
		id := w.issues.issue.ID
		calls := slices.Concat(creating(), []string{targets, line("States", web), line("Next", web), line("GreatestSortOrder", web, backlog.ID), "Now"},
			ended, []string{line("InsertIssue", id)}, inCycle, []string{line("InsertModules", asked), line("InsertSubscribers", []uuid.UUID{alice}),
				line("InsertActivities", rows), line("GetIssue", id)})
		if !slices.Equal(w.log.calls, calls) {
			t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
		}
		wantActivity(t, w, id, append(domain.CreationActivity(nil), domain.CreationJoins(joined, namedModules(moduleAuth, moduleBilling)...)...))
		if len(events.rows) != 1 {
			t.Fatalf("events %+v, want issue.created", events.rows)
		}
		changes := events.rows[0].Changes
		want := []shared.Change{{Field: "modules", New: evptr("Auth"), NewID: &moduleAuth.ID}, {Field: "modules", New: evptr("Billing"), NewID: &moduleBilling.ID}}
		if cycle != nil {
			want = append([]shared.Change{{Field: "cycles", New: evptr(cycle.Name), NewID: &cycle.ID}}, want...)
		}
		if last := changes[len(changes)-len(want):]; !reflect.DeepEqual(last, want) {
			t.Errorf("issue.created's last changes %+v, want %+v", last, want)
		}
	}
}

// A creation's refusals for its modules, in spec R2's order: an archived
// project first; the cycles turned off, then the modules, before an
// archived target, planning read once; an archived module before the 422;
// a module that is not one of the project's undeleted modules is the
// 422's, at its place in module_ids as given, after the label's and the
// cycle's problems. Nothing is written.
func TestCreateIssueRefusesTheModules(t *testing.T) {
	ghost := uuid.NewV7()
	targets := func(cycles []uuid.UUID, modules ...uuid.UUID) string { return line("Targets", web, cycles, modules) }
	for _, tt := range []struct {
		name  string
		set   func(w *world, p *fakePlanning)
		in    domain.NewIssue
		want  error
		calls []string
	}{
		{"an archived project, modules off", func(w *world, p *fakePlanning) { w.project.locked.Archived, p.modulesOff = true, true },
			domain.NewIssue{Name: "Fix", ModuleIDs: idsOf(moduleAuth)}, domain.ErrProjectArchived, nil},
		{"modules off, a module archived", func(_ *world, p *fakePlanning) { p.modulesOff = true }, domain.NewIssue{Name: "Fix", ModuleIDs: idsOf(moduleOld)},
			domain.ErrModulesDisabled, []string{targets(nil, moduleOld.ID)}},
		{"cycles and modules off", func(_ *world, p *fakePlanning) { p.off, p.modulesOff = true, true },
			domain.NewIssue{Name: "Fix", CycleID: &cycleNow.ID, ModuleIDs: idsOf(moduleAuth)}, domain.ErrCyclesDisabled,
			[]string{targets([]uuid.UUID{cycleNow.ID}, moduleAuth.ID)}},
		{"modules off, the cycle archived", func(_ *world, p *fakePlanning) { p.modulesOff = true },
			domain.NewIssue{Name: "Fix", CycleID: &cycleOld.ID, ModuleIDs: idsOf(moduleAuth)}, domain.ErrModulesDisabled,
			[]string{targets([]uuid.UUID{cycleOld.ID}, moduleAuth.ID)}},
		{"an archived module, a cycle, a module and a label not the project's", func(*world, *fakePlanning) {},
			domain.NewIssue{Name: "Fix", CycleID: &cycleGone, ModuleIDs: []uuid.UUID{moduleAuth.ID, moduleGone, moduleOld.ID}, LabelIDs: []uuid.UUID{ghost}},
			domain.ErrPlanningArchived, []string{targets([]uuid.UUID{cycleGone}, moduleGone, moduleAuth.ID, moduleOld.ID)}},
		{"modules not the project's, at their places, a cycle and a label not either", func(*world, *fakePlanning) {},
			domain.NewIssue{Name: "Fix", CycleID: &cycleGone, ModuleIDs: []uuid.UUID{moduleAuth.ID, moduleGone, moduleLost}, LabelIDs: []uuid.UUID{ghost}},
			shared.Invalid(domain.UnknownLabel(0), domain.CycleNotAllowed(), domain.ModuleNotAllowed(1), domain.ModuleNotAllowed(2)),
			[]string{targets([]uuid.UUID{cycleGone}, moduleGone, moduleAuth.ID, moduleLost), line("States", web), line("Labels", web, []uuid.UUID{ghost})}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, w := newCreate()
			p := modulePlanning(w)
			tt.set(w, p)
			_, err := joinCreate(w, p, discardEvents{}).Execute(as(alice), web, tt.in)
			wantRefusal(t, w, err, tt.want, append(creating(), tt.calls...))
			if w.issues.issue != nil || w.issues.modules != nil {
				t.Errorf("wrote the work item %+v in %v", w.issues.issue, w.issues.modules)
			}
		})
	}
}

// inModules is updateIssue over newUpdate's world with its work item in the
// cycle c, nil for none, and the modules ms, planning's, and the events it
// publishes.
func inModules(t *testing.T, c *app.PlanningTarget, ms ...app.PlanningTarget) (*app.UpdateIssue, *world, uuid.UUID, *fakePlanning, *recordedEvents) {
	_, w, id := newUpdate()
	if c != nil {
		w.issues.cycle = &c.ID
	}
	w.issues.modules = idsOf(ms...)
	p, events := modulePlanning(w), &recordedEvents{t: t}
	return joinUpdate(w, p, events), w, id, p, events
}

// A change of the work item's modules replaces the whole set (M6 P3a spec
// 2.3, R1–R4, 2.4): planning read once under the lock about the modules it
// joins and those it leaves, in id order, not those it keeps; the clock;
// the work item's fields; the rows of the modules left soft-deleted, then
// those joined inserted, all at the moment; an activity row for each
// module joined, then each left, in id order; issue.updated's changes are
// those rows. A module kept, archived too, is neither asked about nor
// checked; one left, archived too, is named.
func TestUpdateIssueChangesTheModules(t *testing.T) {
	for _, tt := range []struct {
		name           string
		from, to       []app.PlanningTarget
		joined, leaves []app.PlanningTarget
	}{
		{"joins", nil, []app.PlanningTarget{moduleBilling, moduleAuth}, []app.PlanningTarget{moduleAuth, moduleBilling}, nil},
		{"replaces, keeping an archived one", []app.PlanningTarget{moduleAuth, moduleOld}, []app.PlanningTarget{moduleOld, moduleBilling},
			[]app.PlanningTarget{moduleBilling}, []app.PlanningTarget{moduleAuth}},
		{"leaves every one, an archived one too", []app.PlanningTarget{moduleAuth, moduleOld}, nil, nil, []app.PlanningTarget{moduleAuth, moduleOld}},
		{"leaves an archived one, keeping another", []app.PlanningTarget{moduleAuth, moduleOld}, []app.PlanningTarget{moduleAuth}, nil,
			[]app.PlanningTarget{moduleOld}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, id, _, events := inModules(t, nil, tt.from...)
			asked := ascendingIDs(slices.Concat(idsOf(tt.joined...), idsOf(tt.leaves...)))
			calls := append(locked(id), line("Targets", web, []uuid.UUID(nil), asked), "Now", line("UpdateIssue", id))
			if len(tt.leaves) > 0 {
				calls = append(calls, line("RemoveModules", idsOf(tt.leaves...)))
			}
			if len(tt.joined) > 0 {
				calls = append(calls, line("InsertModules", idsOf(tt.joined...)))
			}
			calls = append(calls, line("InsertActivities", len(tt.joined)+len(tt.leaves)), line("GetIssue", id), counted(id))
			got, err := uc.Execute(as(alice), id, domain.IssuePatch{ModuleIDs: set(idsOf(tt.to...)...)})
			if want := ascendingIDs(idsOf(tt.to...)); err != nil || !slices.Equal(got.ModuleIDs, want) || got.UpdatedByID == nil || *got.UpdatedByID != alice ||
				!got.UpdatedAt.Equal(clockNow) {
				t.Fatalf("Execute() = %+v, %v; want it in %v, changed by alice at the moment", got, err, want)
			}
			if !slices.Equal(w.log.calls, calls) {
				t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
			}
			wantActivity(t, w, id, domain.UpdateActivity(domain.Update{ModulesAdded: namedModules(tt.joined...), ModulesRemoved: namedModules(tt.leaves...)}))
			var changes []shared.Change
			for _, r := range w.issues.activities {
				a := r.Activity
				changes = append(changes, shared.Change{Field: "modules", Old: a.OldValue, New: a.NewValue, OldID: a.OldIdentifier, NewID: a.NewIdentifier})
			}
			wantIssueEvent(t, events, domain.EventIssueUpdated, id, changes)
		})
	}
}

// ascendingIDs are ids in id order, nil for none.
func ascendingIDs(ids []uuid.UUID) []uuid.UUID {
	if len(ids) == 0 {
		return nil
	}
	return slices.SortedFunc(slices.Values(ids), uuid.UUID.Compare)
}

// A patch giving the modules the work item is in, in any order, or none
// for none, changes nothing: planning is not asked, with modules turned
// off too, and nothing is written.
func TestUpdateIssueKeepingTheModulesWritesNothing(t *testing.T) {
	for what, tt := range map[string]struct{ from, to []app.PlanningTarget }{
		"Auth and Old, in another order": {[]app.PlanningTarget{moduleAuth, moduleOld}, []app.PlanningTarget{moduleOld, moduleAuth}},
		"none":                           {nil, nil},
	} {
		t.Run(what, func(t *testing.T) {
			uc, w, id, p, _ := inModules(t, nil, tt.from...)
			p.modulesOff = true
			if _, err := uc.Execute(as(alice), id, domain.IssuePatch{ModuleIDs: set(idsOf(tt.to...)...)}); err != nil ||
				!slices.Equal(w.log.calls, append(locked(id), counted(id))) {
				t.Errorf("Execute() = %v; calls %q, want none after the decision", err, w.log.calls)
			}
		})
	}
}

// An update's refusals for its modules, in spec R2's order: an archived
// work item before the modules turned off, which refuse leaving one too;
// an archived module joined next, one kept not checked; a module joined
// that is not the project's undeleted module is the 422's, at its place in
// module_ids as given, after the label's and the cycle's problems. Nothing
// is written.
func TestUpdateIssueRefusesTheModules(t *testing.T) {
	ghost := uuid.NewV7()
	targets := func(cycles []uuid.UUID, modules ...uuid.UUID) string { return line("Targets", web, cycles, modules) }
	for _, tt := range []struct {
		name  string
		set   func(w *world, p *fakePlanning)
		from  []app.PlanningTarget
		p     domain.IssuePatch
		want  error
		calls []string
	}{
		{"an archived work item, modules off", func(w *world, p *fakePlanning) { w.issues.archivedAt, p.modulesOff = &earlier, true }, nil,
			domain.IssuePatch{ModuleIDs: set(moduleAuth.ID)}, domain.ErrIssueArchived, nil},
		{"modules off, leaving", func(_ *world, p *fakePlanning) { p.modulesOff = true }, []app.PlanningTarget{moduleAuth},
			domain.IssuePatch{ModuleIDs: set()}, domain.ErrModulesDisabled, []string{targets(nil, moduleAuth.ID)}},
		{"modules off, joining an archived one", func(_ *world, p *fakePlanning) { p.modulesOff = true }, nil,
			domain.IssuePatch{ModuleIDs: set(moduleOld.ID)}, domain.ErrModulesDisabled, []string{targets(nil, moduleOld.ID)}},
		{"joining an archived one, keeping one, a label not the project's", func(*world, *fakePlanning) {}, []app.PlanningTarget{moduleAuth},
			domain.IssuePatch{ModuleIDs: set(moduleAuth.ID, moduleOld.ID), LabelIDs: set(ghost)}, domain.ErrPlanningArchived,
			[]string{targets(nil, moduleOld.ID)}},
		{"modules not the project's, at their places, keeping an archived one, a cycle and a label not either", func(*world, *fakePlanning) {},
			[]app.PlanningTarget{moduleOld},
			domain.IssuePatch{SetCycleID: true, CycleID: &cycleGone, ModuleIDs: set(moduleOld.ID, moduleAuth.ID, moduleGone, moduleLost), LabelIDs: set(ghost)},
			shared.Invalid(domain.UnknownLabel(0), domain.CycleNotAllowed(), domain.ModuleNotAllowed(2), domain.ModuleNotAllowed(3)),
			[]string{targets([]uuid.UUID{cycleGone}, moduleGone, moduleAuth.ID, moduleLost), line("Labels", web, []uuid.UUID{bug, ghost})}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, id, p, _ := inModules(t, nil, tt.from...)
			tt.set(w, p)
			_, err := uc.Execute(as(alice), id, tt.p)
			wantRefusal(t, w, err, tt.want, append(locked(id), tt.calls...))
			if w.issues.changed != nil || !slices.Equal(w.issues.modules, idsOf(tt.from...)) {
				t.Errorf("wrote %+v, the modules %v; want nothing, the modules %v", w.issues.changed, w.issues.modules, idsOf(tt.from...))
			}
		})
	}
}

// A change of both the cycle and the modules reads planning once, about
// the cycles and the modules together (M6 P3a spec R1); then the clock;
// whether the cycle joined has ended; the work item's fields; the cycle's
// rows, then the modules', those left then those joined; the cycle's
// activity row, then each module joined, then each left.
func TestUpdateIssueChangesTheCycleAndTheModulesAtOnce(t *testing.T) {
	uc, w, id, _, events := inModules(t, &cycleNow, moduleAuth)
	calls := append(locked(id), line("Targets", web, []uuid.UUID{cycleNow.ID, cycleNext.ID}, idsOf(moduleAuth, moduleBilling)), "Now",
		completedAt(cycleNext.ID), line("UpdateIssue", id), line("RemoveCycle", []uuid.UUID{cycleNow.ID}), line("InsertCycle", []uuid.UUID{cycleNext.ID}),
		line("RemoveModules", idsOf(moduleAuth)), line("InsertModules", idsOf(moduleBilling)), line("InsertActivities", 3), line("GetIssue", id), counted(id))
	got, err := uc.Execute(as(alice), id, domain.IssuePatch{SetCycleID: true, CycleID: &cycleNext.ID, ModuleIDs: set(moduleBilling.ID)})
	if err != nil || got.CycleID == nil || *got.CycleID != cycleNext.ID || !slices.Equal(got.ModuleIDs, idsOf(moduleBilling)) {
		t.Fatalf("Execute() = %+v, %v; want it in Sprint 2 and Billing", got, err)
	}
	if !slices.Equal(w.log.calls, calls) {
		t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
	}
	rows := domain.UpdateActivity(domain.Update{Cycle: &domain.Changed[*domain.Named]{Old: namedCycle(cycleNow), New: namedCycle(cycleNext)},
		ModulesAdded: namedModules(moduleBilling), ModulesRemoved: namedModules(moduleAuth)})
	wantActivity(t, w, id, rows)
	if len(events.rows) != 1 || len(events.rows[0].Changes) != 3 || events.rows[0].Changes[0].Field != "cycles" || events.rows[0].Changes[2].Field != "modules" {
		t.Errorf("events %+v, want issue.updated with the cycle's change, then the modules'", events.rows)
	}
}

// A module the work item leaves that planning does not answer as the
// project's is an error, not a refusal: deleting a module takes its work
// items out of it, so it cannot be (M6 P3a spec R6).
func TestUpdateIssueLeavingAModuleThatIsNotThere(t *testing.T) {
	_, w, id := newUpdate()
	w.issues.modules = []uuid.UUID{moduleGone}
	_, err := joinUpdate(w, modulePlanning(w), discardEvents{}).Execute(as(alice), id, domain.IssuePatch{ModuleIDs: set()})
	var se *shared.Error
	if err == nil || errors.As(err, &se) || w.issues.changed != nil {
		t.Errorf("Execute() = %v, wrote %+v; want an error that is no refusal, nothing written", err, w.issues.changed)
	}
}
