package app_test

import (
	"context"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// intakeScope is project's side of the counts as a fake: the projects the
// caller sees, as Visible answers them, and of those asked, those whose
// intake is on, in id order, as IntakeOn answers them; each read logged.
type intakeScope struct {
	app.VisibleProjects
	log     *callLog
	visible []app.VisibleProject
	on      []uuid.UUID
}

func (f *intakeScope) Visible(ctx context.Context, workspaceID, actor uuid.UUID) ([]app.VisibleProject, error) {
	return f.visible, f.log.add(ctx, "Visible", workspaceID, actor)
}

func (f *intakeScope) IntakeOn(ctx context.Context, ids []uuid.UUID) ([]uuid.UUID, error) {
	if err := f.log.add(ctx, "IntakeOn", ids); err != nil {
		return nil, err
	}
	var on []uuid.UUID
	for _, id := range ascendingIDs(f.on) {
		if slices.Contains(ids, id) {
			on = append(on, id)
		}
	}
	return on, nil
}

// intakeCounter is the counts' store as a fake: each count logged with
// its caller, its projects' ids and its moment, its projects kept; it
// answers each project given its number in pending, none for one it has
// not, in the order given.
type intakeCounter struct {
	log     *callLog
	pending map[uuid.UUID]int64
	asked   [][]app.VisibleProject
}

func (f *intakeCounter) CountPendingIntake(ctx context.Context, actor uuid.UUID, projects []app.VisibleProject, now time.Time) ([]app.IntakeCount, error) {
	f.asked = append(f.asked, projects)
	if err := f.log.add(ctx, "CountPendingIntake", actor, len(projects), now.Equal(clockNow)); err != nil {
		return nil, err
	}
	out := make([]app.IntakeCount, len(projects))
	for n, p := range projects {
		out[n] = app.IntakeCount{ProjectID: p.ID, Pending: f.pending[p.ID]}
	}
	return out, nil
}

// intakeCountWorld is acme's pending counts over newGet's world: alice
// sees web, where she is a guest who sees only her own work items, ops, a
// member, and off and archived, whose intakes the project's port does not
// answer as on (one turned off, one archived since); web and ops have
// their intake on.
type intakeCountWorld struct {
	*world
	uc      *app.CountIntakeIssues
	scope   *intakeScope
	counter *intakeCounter
}

func newIntakeCount() (*intakeCountWorld, uuid.UUID, uuid.UUID) {
	_, w, _ := newGet()
	off, archived := uuid.NewV7(), uuid.NewV7()
	scope := &intakeScope{log: w.log, on: []uuid.UUID{web, ops}, visible: []app.VisibleProject{{ID: off, Role: shared.RoleAdmin},
		{ID: web, Role: shared.RoleGuest, OwnOnly: true}, {ID: archived, Role: shared.RoleMember}, {ID: ops, Role: shared.RoleMember}}}
	counter := &intakeCounter{log: w.log, pending: map[uuid.UUID]int64{web: 2}}
	uc := app.NewCountIntakeIssues(app.CountIntakeIssuesDeps{ReadTx: fakeTx{log: w.log}, Locks: w.locks, Workspaces: &fakeWorkspaces{log: w.log, slug: "acme", id: acme},
		Projects: scope, Intakes: scope, Counts: counter, Clock: loggedNow(w.log)})
	return &intakeCountWorld{world: w, uc: uc, scope: scope, counter: counter}, off, archived
}

// The pending counts of acme (M7 P2b spec 2.6, DP-20), in one read
// snapshot: the workspace and the decision on intake.count in it; the
// projects alice sees, as their memberships give them; which of them have
// their intake on, no other project counted, one turned off or archived
// not; the clock, read once; and the count of those, at that moment, each
// with whether she sees only her own work items there, as Visible gave
// it. Every one is answered, with none pending too, in id order.
func TestCountIntakeIssuesCountsTheProjectsSeenWithTheirIntakeOn(t *testing.T) {
	w, off, archived := newIntakeCount()
	got, err := w.uc.Execute(as(alice), "acme")
	scope := []app.VisibleProject{{ID: web, Role: shared.RoleGuest, OwnOnly: true}, {ID: ops, Role: shared.RoleMember}}
	slices.SortFunc(scope, func(a, b app.VisibleProject) int { return a.ID.Compare(b.ID) })
	want := []app.IntakeCount{{ProjectID: scope[0].ID, Pending: w.counter.pending[scope[0].ID]}, {ProjectID: scope[1].ID, Pending: w.counter.pending[scope[1].ID]}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("Execute() = %+v, %v; want %+v", got, err, want)
	}
	if !reflect.DeepEqual(w.counter.asked, [][]app.VisibleProject{scope}) {
		t.Errorf("counted %+v, want %+v", w.counter.asked, scope)
	}
	calls := []string{"Begin", line("WorkspaceBySlug", "acme"), line("Authorize", alice, domain.ActionCountIntake, acme, uuid.Nil()),
		line("Visible", acme, alice), line("IntakeOn", []uuid.UUID{off, web, archived, ops}), "Now", line("CountPendingIntake", alice, 2, true)}
	if !slices.Equal(w.log.calls, calls) {
		t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
	}
}

// A caller who sees no project, or none with its intake on, is answered
// none, no project read or counted after it, and the clock not read.
func TestCountIntakeIssuesWithoutAProjectToCount(t *testing.T) {
	for what, set := range map[string]func(s *intakeScope){
		"no project seen":            func(s *intakeScope) { s.visible = nil },
		"no project with its intake": func(s *intakeScope) { s.on = nil },
	} {
		t.Run(what, func(t *testing.T) {
			w, _, _ := newIntakeCount()
			set(w.scope)
			got, err := w.uc.Execute(as(alice), "acme")
			if err != nil || got == nil || len(got) != 0 || slices.Contains(w.log.calls, "Now") || len(w.counter.asked) != 0 {
				t.Errorf("Execute() = %+v, %v after the calls %q; want none, nothing counted", got, err, w.log.calls)
			}
		})
	}
}

// A workspace not there or not seen is workspace.not_found, a role the
// rule does not allow forbidden, before any project is read; a request
// without a caller is unauthenticated. A failing port ends the count with
// its failure, as itself. None answers a count.
func TestCountIntakeIssuesRefusals(t *testing.T) {
	decided := []string{"Begin", line("WorkspaceBySlug", "acme"), line("Authorize", alice, domain.ActionCountIntake, acme, uuid.Nil())}
	for _, tt := range []struct {
		name  string
		ctx   context.Context
		slug  string
		set   func(w *intakeCountWorld)
		want  error
		calls []string
	}{
		{"no caller", context.Background(), "acme", func(*intakeCountWorld) {}, shared.Unauthenticated(), nil},
		{"no workspace", as(alice), "nowhere", func(*intakeCountWorld) {}, domain.ErrWorkspaceNotFound, []string{"Begin", line("WorkspaceBySlug", "nowhere")}},
		{"a workspace not seen", as(alice), "acme", func(w *intakeCountWorld) { w.auth.answers = []error{shared.ErrNotVisible} }, domain.ErrWorkspaceNotFound, decided},
		{"a role not allowed", as(alice), "acme", func(w *intakeCountWorld) { w.auth.answers = []error{shared.Forbidden()} }, shared.Forbidden(), decided},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w, _, _ := newIntakeCount()
			tt.set(w)
			got, err := w.uc.Execute(tt.ctx, tt.slug)
			if !reflect.DeepEqual(err, tt.want) || got != nil || !slices.Equal(w.log.calls, tt.calls) {
				t.Errorf("Execute() = %+v, %v, calls %q; want %v, calls %q", got, err, w.log.calls, tt.want, tt.calls)
			}
		})
	}
	for _, port := range []string{"WorkspaceBySlug", "Authorize", "Visible", "IntakeOn", "CountPendingIntake"} {
		t.Run(port, func(t *testing.T) {
			w, _, _ := newIntakeCount()
			w.log.fail = port
			got, err := w.uc.Execute(as(alice), "acme")
			if got != nil {
				t.Errorf("Execute() = %+v, want none", got)
			}
			wantStoppedAt(t, w.world, err, port)
		})
	}
}
