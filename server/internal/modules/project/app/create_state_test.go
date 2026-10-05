package app_test

import (
	"context"
	"maps"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// newCreateState is CreateState over newStates' fakes, its clock logged.
func newCreateState() (*app.CreateState, *writeFixture, *fakeStates) {
	f, s := newStates()
	return app.NewCreateState(f.locks(), s, f.tx, clockAt{clockNow, f.log}), f, s
}

// review is the state the tests create: QA in the completed group, with a
// description, a name no state of web or ops has.
var review = domain.StateCreate{Name: "QA", Color: "#0EA5E9", Group: domain.GroupCompleted, Description: "Checked by QA"}

// stateCreated are the calls of user's creation of in in project at
// sequence: its locks and decision, the greatest sequence, the clock, the
// insert of the state at sequence, never the default, by user at that time.
func stateCreated(user, project uuid.UUID, in domain.StateCreate, sequence float64) []string {
	return append(lockedDecision(user, project, domain.ActionStateCreate), "GreatestSequence "+project.String(), "Now",
		"CreateState "+stateRow(app.StateRow{WorkspaceID: acme.ID, ProjectID: project, CreatedBy: user, Now: clockNow,
			State: domain.NewState{Name: in.Name, Color: in.Color, Group: in.Group, Description: in.Description, Sequence: sequence}}))
}

// CreateState, in one transaction and in the order of M3 design 3.6, locks
// the project, decides, reads the greatest sequence of its states and the
// clock, then inserts the state 15000 after it, by the caller at that time;
// it answers the state as stored, the time to the microsecond: web's after
// its Cancelled, 70000; archived ops's, as any other's (3.19), after its
// Backlog, 30000; and ops's, its states gone, at 65535. The use case makes
// each state's id: a second state's differs from the first's, and neither
// is the nil id, the project's, or a state's the fixture had.
func TestCreateState(t *testing.T) {
	for _, tt := range []struct {
		name     string
		project  uuid.UUID
		set      func(s *fakeStates)
		sequence float64
	}{
		{"web", webID, nil, 70000},
		{"archived ops", opsID, nil, 30000},
		{"ops without states", opsID, func(s *fakeStates) { delete(s.states, opsBacklog) }, 65535},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, s := newCreateState()
			if tt.set != nil {
				tt.set(s)
			}
			before := maps.Clone(s.states)
			got, err := uc.Execute(as(bob), tt.project, review)
			stored := s.states[got.ID]
			want := domain.State{ID: got.ID, WorkspaceID: acme.ID, ProjectID: tt.project, Name: "QA", Description: "Checked by QA", Color: "#0EA5E9",
				Group: domain.GroupCompleted, Sequence: tt.sequence, CreatedAt: now, UpdatedAt: now}
			if err != nil || got != want || stored != want {
				t.Errorf("Execute() = %+v, %v, stored %+v; want %+v", got, err, stored, want)
			}
			if want := stateCreated(bob, tt.project, review, tt.sequence); !slices.Equal(f.log.calls, want) {
				t.Errorf("calls\n%q\nwant\n%q", f.log.calls, want)
			}
			second := review
			second.Name = "QA again"
			again, err := uc.Execute(as(bob), tt.project, second)
			if err != nil || again.ID == got.ID {
				t.Errorf("a second state = %s, %v; want another id than the first's, %s", again.ID, err, got.ID)
			}
			for _, id := range []uuid.UUID{got.ID, again.ID} {
				if _, had := before[id]; had || id == uuid.Nil() || id == tt.project {
					t.Errorf("a state created as %s; want a new id: not the nil id, the project's or a state's the fixture had", id)
				}
			}
		})
	}
}

// Refusals, each in its place, and no state stored: no caller, and values
// the domain refuses, before the transaction; a project that is not there,
// a workspace or project deleted while its lock waited, a project moved
// meanwhile, and a caller who does not see the project, each
// project.not_found; a member, the Authorizer's 403. A name another state
// of the project has is the store's project.state_name_taken, after the
// insert. The state answered for another id is the write's own error.
func TestCreateStateRefuses(t *testing.T) {
	upTo := func(n int) []string { return lockedDecision(bob, webID, domain.ActionStateCreate)[:n] }
	taken := review
	taken.Name = "Review"
	for _, tt := range []struct {
		name    string
		ctx     context.Context
		project uuid.UUID
		in      domain.StateCreate
		set     func(f *writeFixture, s *fakeStates)
		want    error
		calls   []string
	}{
		{"no caller", context.Background(), webID, review, nil, shared.Unauthenticated(), nil},
		{"the triage group", as(bob), webID, domain.StateCreate{Name: "Intake", Color: "#000", Group: domain.GroupTriage}, nil,
			shared.Invalid(shared.FieldError{Field: "group", Code: shared.FieldNotAllowed}), nil},
		{"no project", as(bob), uuid.Nil(), review, nil, domain.ErrNotFound, noProject},
		{"acme deleted while its lock waited", as(bob), webID, review, func(f *writeFixture, _ *fakeStates) { f.workspaces.gone = true },
			domain.ErrNotFound, upTo(3)},
		{"web deleted while its lock waited", as(bob), webID, review, func(f *writeFixture, _ *fakeStates) { f.store.deleted = true },
			domain.ErrNotFound, upTo(4)},
		{"web moved to another workspace", as(bob), webID, review, func(f *writeFixture, _ *fakeStates) { f.store.moved = uuid.NewV7() },
			domain.ErrNotFound, upTo(4)},
		{"a caller who does not see web", as(erin), webID, review, nil, domain.ErrNotFound,
			lockedDecision(erin, webID, domain.ActionStateCreate)},
		{"a member", as(alice), webID, review, nil, shared.Forbidden(), lockedDecision(alice, webID, domain.ActionStateCreate)},
		{"a name web's Review has", as(bob), webID, taken, nil, domain.ErrStateNameTaken, stateCreated(bob, webID, taken, 70000)},
		{"the state answered for another id", as(bob), webID, review, func(_ *writeFixture, s *fakeStates) { s.changedAs = uuid.NewV7() },
			nil, stateCreated(bob, webID, review, 70000)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, s := newCreateState()
			if tt.set != nil {
				tt.set(f, s)
			}
			before := maps.Clone(s.states)
			_, err := uc.Execute(tt.ctx, tt.project, tt.in)
			outcome{tt.name, tt.want, tt.calls}.check(t, err, f)
			if tt.name != "the state answered for another id" && !maps.Equal(s.states, before) {
				t.Errorf("the states after the refusal: %v; want them as they were, %v", s.states, before)
			}
		})
	}
}

// Every port's failure comes back as itself, the commit's too, after the
// calls before it and none after.
func TestCreateStateReturnsEachFailure(t *testing.T) {
	all := stateCreated(bob, webID, review, 70000)
	fail := func(method string) func(f *writeFixture) {
		return func(f *writeFixture) { f.store.errs = map[string]error{method: errDisk} }
	}
	for _, tt := range []struct {
		name  string
		fail  func(f *writeFixture)
		calls int // how many of the creation's calls ran
	}{
		{"the project's workspace", fail("ProjectWorkspace"), 2},
		{"the workspace's lock", func(f *writeFixture) { f.workspaces.err = errDisk }, 3},
		{"the project's lock", fail("LockProject"), 4},
		{"the decision", func(f *writeFixture) { f.auth.errs[grantKey{bob, acme.ID}] = errDisk }, 5},
		{"the greatest sequence", fail("GreatestSequence"), 6},
		{"the insert", fail("CreateState"), 8},
		{"the commit", func(f *writeFixture) { f.tx.commitErr = errDisk }, 8},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, _ := newCreateState()
			tt.fail(f)
			_, err := uc.Execute(as(bob), webID, review)
			outcome{tt.name, errDisk, all[:tt.calls]}.check(t, err, f)
		})
	}
}
