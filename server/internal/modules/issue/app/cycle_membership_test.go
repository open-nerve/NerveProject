package app_test

import (
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// web's cycles, drawn in this order, ascending: Sprint 1 and Sprint 2,
// open; Old, archived; Ended, completed on any day; and cycleGone, which
// is not one of web's undeleted cycles.
var (
	cycleNow   = app.PlanningTarget{ID: uuid.NewV7(), Name: "Sprint 1"}
	cycleNext  = app.PlanningTarget{ID: uuid.NewV7(), Name: "Sprint 2"}
	cycleOld   = app.PlanningTarget{ID: uuid.NewV7(), Name: "Old", Archived: true}
	cycleEnded = app.PlanningTarget{ID: uuid.NewV7(), Name: "Ended"}
	cycleGone  = uuid.NewV7()
)

// planningOf is planning over w's log: web's cycles on, the four of them.
func planningOf(w *world) *fakePlanning {
	return &fakePlanning{log: w.log, cycles: []app.PlanningTarget{cycleNow, cycleNext, cycleOld, cycleEnded}, completed: []uuid.UUID{cycleEnded.ID}}
}

// joinCreate and joinUpdate are createIssue and updateIssue over w's
// ports, planning p's, their events published to events.
func joinCreate(w *world, p *fakePlanning, events shared.EventPublisher) *app.CreateIssue {
	return app.NewCreateIssue(app.CreateIssueDeps{Locks: w.locks, Creation: app.NewIssueCreation(app.IssueCreationDeps{Attachments: noIssueAttachments{},
		Events: events, Sanitizer: descriptionSanitizer{}, Issues: w.issues, Sequences: w.project, States: w.project, Labels: w.project,
		Members: w.project, Planning: p, Profiles: w.profiles, Clock: clockAt{at: clockNow, log: w.log}})})
}

func joinUpdate(w *world, p *fakePlanning, events shared.EventPublisher) *app.UpdateIssue {
	return app.NewUpdateIssue(app.UpdateIssueDeps{Events: events, Sanitizer: descriptionSanitizer{}, Locks: w.locks, Issues: w.issues,
		States: w.project, Labels: w.project, Members: w.project, Planning: p, Profiles: w.profiles, Clock: clockAt{at: clockNow, log: w.log}})
}

// namedCycle is the cycle t as an activity names it.
func namedCycle(t app.PlanningTarget) *domain.Named {
	return &domain.Named{ID: t.ID, Name: t.Name}
}

// completedAt is the call asking whether the cycle has ended at the clock's
// moment, the write's (fakePlanning).
func completedAt(cycle uuid.UUID) string {
	return line("CycleCompleted", web, cycle, clockNow.Format(time.RFC3339Nano))
}

// creating are the calls of alice's creation in web up to the decision
// under the lock.
func creating() []string {
	return []string{line("Find", web, "outside tx"), line("Authorize", alice, domain.ActionCreate, acme, web, "outside tx"), "Begin", line("Lock", web),
		line("Authorize", alice, domain.ActionCreate, acme, web)}
}

// A creation that joins a cycle (M6 P3a spec R1, R2, R5): planning's facts
// read once under the lock, first; the clock; whether the cycle has ended
// at that moment, the write's; the work item, its cycle's row, its
// subscribers; its activity, the creation's row, then the cycle's; the
// answer in the cycle; issue.created's changes end with the cycle, as the
// labels'.
func TestCreateIssueJoinsACycle(t *testing.T) {
	_, w := newCreate()
	events := &recordedEvents{t: t}
	got, err := joinCreate(w, planningOf(w), events).Execute(as(alice), web, domain.NewIssue{Name: "Fix", CycleID: &cycleNow.ID})
	if err != nil || got.CycleID == nil || *got.CycleID != cycleNow.ID {
		t.Fatalf("Execute() = %+v, %v; want the work item in Sprint 1", got, err)
	}
	id := w.issues.issue.ID
	calls := append(creating(), line("Targets", web, []uuid.UUID{cycleNow.ID}, []uuid.UUID(nil)), line("States", web), line("Next", web),
		line("GreatestSortOrder", web, backlog.ID), "Now", completedAt(cycleNow.ID), line("InsertIssue", id), line("InsertCycle", []uuid.UUID{cycleNow.ID}),
		line("InsertSubscribers", []uuid.UUID{alice}), line("InsertActivities", 2), line("GetIssue", id))
	if !slices.Equal(w.log.calls, calls) {
		t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
	}
	wantActivity(t, w, id, append(domain.CreationActivity(nil), domain.CreationJoins(namedCycle(cycleNow))...))
	if len(events.rows) != 1 {
		t.Fatalf("events %+v, want issue.created", events.rows)
	}
	changes := events.rows[0].Changes
	if last := changes[len(changes)-1]; !reflect.DeepEqual(last, shared.Change{Field: "cycles", New: evptr("Sprint 1"), NewID: &cycleNow.ID}) {
		t.Errorf("issue.created's last change %+v, want the cycle", last)
	}
}

// A creation's refusals for its cycle, in spec R2's order: an archived
// project before the cycles turned off, which come before any other check,
// planning read once; an archived cycle next; a cycle that is not one of
// the project's undeleted cycles is the 422's, with every other problem;
// one that has ended at the write's moment last, after the clock. Nothing
// is written.
func TestCreateIssueRefusesTheCycle(t *testing.T) {
	ghost := uuid.NewV7()
	targets := func(id uuid.UUID) string { return line("Targets", web, []uuid.UUID{id}, []uuid.UUID(nil)) }
	for _, tt := range []struct {
		name  string
		set   func(w *world, p *fakePlanning)
		in    domain.NewIssue
		want  error
		calls []string
	}{
		{"an archived project, cycles off", func(w *world, p *fakePlanning) { w.project.locked.Archived, p.off = true, true },
			domain.NewIssue{Name: "Fix", CycleID: &cycleNow.ID}, domain.ErrProjectArchived, nil},
		{"cycles off, the cycle archived", func(_ *world, p *fakePlanning) { p.off = true }, domain.NewIssue{Name: "Fix", CycleID: &cycleOld.ID},
			domain.ErrCyclesDisabled, []string{targets(cycleOld.ID)}},
		{"an archived cycle, a label not the project's", func(*world, *fakePlanning) {},
			domain.NewIssue{Name: "Fix", CycleID: &cycleOld.ID, LabelIDs: []uuid.UUID{ghost}}, domain.ErrPlanningArchived, []string{targets(cycleOld.ID)}},
		{"no such cycle, a label not the project's", func(*world, *fakePlanning) {},
			domain.NewIssue{Name: "Fix", CycleID: &cycleGone, LabelIDs: []uuid.UUID{ghost}},
			shared.Invalid(domain.UnknownLabel(0), domain.CycleNotAllowed()),
			[]string{targets(cycleGone), line("States", web), line("Labels", web, []uuid.UUID{ghost})}},
		{"an ended cycle", func(*world, *fakePlanning) {}, domain.NewIssue{Name: "Fix", CycleID: &cycleEnded.ID}, domain.ErrCycleCompleted,
			[]string{targets(cycleEnded.ID), line("States", web), line("Next", web), line("GreatestSortOrder", web, backlog.ID), "Now",
				completedAt(cycleEnded.ID)}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, w := newCreate()
			p := planningOf(w)
			tt.set(w, p)
			_, err := joinCreate(w, p, discardEvents{}).Execute(as(alice), web, tt.in)
			wantRefusal(t, w, err, tt.want, append(creating(), tt.calls...))
			if w.issues.issue != nil || w.issues.cycle != nil {
				t.Errorf("wrote the work item %+v in %v", w.issues.issue, w.issues.cycle)
			}
		})
	}
}

// in is updateIssue over newUpdate's world with its work item in the
// cycle c, nil for none, planning's, and the events it publishes.
func in(t *testing.T, c *app.PlanningTarget) (*app.UpdateIssue, *world, uuid.UUID, *fakePlanning, *recordedEvents) {
	_, w, id := newUpdate()
	if c != nil {
		w.issues.cycle = &c.ID
	}
	p, events := planningOf(w), &recordedEvents{t: t}
	return joinUpdate(w, p, events), w, id, p, events
}

// A change of the work item's cycle (M6 P3a spec R1–R4, 2.4): planning's
// facts of the cycle it joins and the one it leaves read once under the
// lock, in id order; the clock; whether the cycle joined, alone, has ended
// at the write's moment; the work item's fields, the row of the cycle left
// soft-deleted, then the one joined inserted, all at the moment; one
// activity row; issue.updated's changes are that row's. Leaving a cycle,
// archived or ended too, asks nothing more; nor does moving out of one.
func TestUpdateIssueChangesTheCycle(t *testing.T) {
	for _, tt := range []struct {
		name     string
		from, to *app.PlanningTarget
		asked    []uuid.UUID
	}{
		{"joins", nil, &cycleNow, []uuid.UUID{cycleNow.ID}},
		{"moves", &cycleNow, &cycleNext, []uuid.UUID{cycleNow.ID, cycleNext.ID}},
		{"moves out of an archived one", &cycleOld, &cycleNext, []uuid.UUID{cycleNext.ID, cycleOld.ID}},
		{"moves out of an ended one", &cycleEnded, &cycleNow, []uuid.UUID{cycleNow.ID, cycleEnded.ID}},
		{"leaves", &cycleNow, nil, []uuid.UUID{cycleNow.ID}},
		{"leaves an archived one", &cycleOld, nil, []uuid.UUID{cycleOld.ID}},
		{"leaves an ended one", &cycleEnded, nil, []uuid.UUID{cycleEnded.ID}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, id, _, events := in(t, tt.from)
			p := domain.IssuePatch{SetCycleID: true}
			change := domain.Changed[*domain.Named]{}
			calls := append(locked(id), line("Targets", web, tt.asked, []uuid.UUID(nil)), "Now")
			if tt.to != nil {
				p.CycleID, change.New = &tt.to.ID, namedCycle(*tt.to)
				calls = append(calls, completedAt(tt.to.ID))
			}
			calls = append(calls, line("UpdateIssue", id))
			if tt.from != nil {
				change.Old = namedCycle(*tt.from)
				calls = append(calls, line("RemoveCycle", []uuid.UUID{tt.from.ID}))
			}
			if tt.to != nil {
				calls = append(calls, line("InsertCycle", []uuid.UUID{tt.to.ID}))
			}
			calls = append(calls, line("InsertActivities", 1), line("GetIssue", id), counted(id))
			got, err := uc.Execute(as(alice), id, p)
			if err != nil || !reflect.DeepEqual(got.CycleID, p.CycleID) || got.UpdatedByID == nil || *got.UpdatedByID != alice || !got.UpdatedAt.Equal(clockNow) {
				t.Fatalf("Execute() = %+v, %v; want it in %v, changed by alice at the moment", got, err, p.CycleID)
			}
			if !slices.Equal(w.log.calls, calls) {
				t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
			}
			wantActivity(t, w, id, domain.UpdateActivity(domain.Update{Cycle: &change}))
			row := w.issues.activities[0].Activity
			wantIssueEvent(t, events, domain.EventIssueUpdated, id, []shared.Change{{Field: "cycles", Old: row.OldValue, New: row.NewValue,
				OldID: row.OldIdentifier, NewID: row.NewIdentifier}})
		})
	}
}

// A patch giving the cycle the work item is in, or none for none, changes
// nothing: planning is not asked, with cycles turned off too, and nothing
// is written.
func TestUpdateIssueKeepingTheCycleWritesNothing(t *testing.T) {
	for what, from := range map[string]*app.PlanningTarget{"in Sprint 1": &cycleNow, "in none": nil} {
		t.Run(what, func(t *testing.T) {
			uc, w, id, p, _ := in(t, from)
			p.off = true
			patch := domain.IssuePatch{SetCycleID: true}
			if from != nil {
				patch.CycleID = &from.ID
			}
			if _, err := uc.Execute(as(alice), id, patch); err != nil || !slices.Equal(w.log.calls, append(locked(id), counted(id))) {
				t.Errorf("Execute() = %v; calls %q, want none after the decision", err, w.log.calls)
			}
		})
	}
}

// An update's refusals for its cycle, in spec R2's order: an archived work
// item before the cycles turned off, which refuse leaving a cycle too; an
// archived cycle joined next; one not the project's undeleted cycle is the
// 422's, with every other problem; one that has ended at the write's
// moment last, after the clock. Nothing is written.
func TestUpdateIssueRefusesTheCycle(t *testing.T) {
	ghost := uuid.NewV7()
	targets := func(ids ...uuid.UUID) string { return line("Targets", web, ids, []uuid.UUID(nil)) }
	for _, tt := range []struct {
		name  string
		set   func(w *world, p *fakePlanning)
		from  *app.PlanningTarget
		p     domain.IssuePatch
		want  error
		calls []string
	}{
		{"an archived work item, cycles off", func(w *world, p *fakePlanning) { w.issues.archivedAt, p.off = &earlier, true }, nil,
			domain.IssuePatch{SetCycleID: true, CycleID: &cycleNow.ID}, domain.ErrIssueArchived, nil},
		{"cycles off, leaving", func(_ *world, p *fakePlanning) { p.off = true }, &cycleNow, domain.IssuePatch{SetCycleID: true},
			domain.ErrCyclesDisabled, []string{targets(cycleNow.ID)}},
		{"cycles off, joining an archived one", func(_ *world, p *fakePlanning) { p.off = true }, nil,
			domain.IssuePatch{SetCycleID: true, CycleID: &cycleOld.ID}, domain.ErrCyclesDisabled, []string{targets(cycleOld.ID)}},
		{"moving into an archived one, a label not the project's", func(*world, *fakePlanning) {}, &cycleNow,
			domain.IssuePatch{SetCycleID: true, CycleID: &cycleOld.ID, LabelIDs: set(ghost)}, domain.ErrPlanningArchived,
			[]string{targets(cycleNow.ID, cycleOld.ID)}},
		{"no such cycle, a label not the project's", func(*world, *fakePlanning) {}, nil,
			domain.IssuePatch{SetCycleID: true, CycleID: &cycleGone, LabelIDs: set(ghost)},
			shared.Invalid(domain.UnknownLabel(0), domain.CycleNotAllowed()),
			[]string{targets(cycleGone), line("Labels", web, []uuid.UUID{bug, ghost})}},
		{"an ended one", func(*world, *fakePlanning) {}, &cycleNow, domain.IssuePatch{SetCycleID: true, CycleID: &cycleEnded.ID},
			domain.ErrCycleCompleted, []string{targets(cycleNow.ID, cycleEnded.ID), "Now", completedAt(cycleEnded.ID)}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, id, p, _ := in(t, tt.from)
			tt.set(w, p)
			_, err := uc.Execute(as(alice), id, tt.p)
			wantRefusal(t, w, err, tt.want, append(locked(id), tt.calls...))
			var kept *uuid.UUID
			if tt.from != nil {
				kept = &tt.from.ID
			}
			if w.issues.changed != nil || !reflect.DeepEqual(w.issues.cycle, kept) {
				t.Errorf("wrote %+v, the cycle %v; want nothing, the cycle %v", w.issues.changed, w.issues.cycle, kept)
			}
		})
	}
}
