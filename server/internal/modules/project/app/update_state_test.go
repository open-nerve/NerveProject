package app_test

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// newUpdateState is UpdateState over newStates' fakes, its clock logged.
func newUpdateState() (*app.UpdateState, *writeFixture, *fakeStates) {
	f, s := newStates()
	return app.NewUpdateState(f.locks(), s, f.tx, clockAt{clockNow, f.log}), f, s
}

// stateLocked are the calls of a write by caller on the state id of
// project up to its decision on action: the transaction, the state read,
// acme's row FOR SHARE, the project FOR NO KEY UPDATE, the state read
// again, the decision.
func stateLocked(id, project, caller uuid.UUID, action shared.Action) []string {
	return rowLocked("StateByID", id, project, caller, action)
}

// stateUpdated are the calls of user's change p of the state id of web:
// its locks and decision, the count of its group when p moves it from
// group, the clock, the change by user at that time.
func stateUpdated(user, id uuid.UUID, p domain.StatePatch, from domain.StateGroup) []string {
	calls := stateLocked(id, webID, user, domain.ActionStateUpdate)
	if p.Group != nil && *p.Group != from {
		calls = append(calls, fmt.Sprintf("CountGroupStates %s %s", webID, from))
	}
	return append(calls, "Now", fmt.Sprintf("UpdateState %s %s by %s at %s", id, statePatch(p), user, clockNow.Format(timeFormat)))
}

// UpdateState, in one transaction and in the order of M3 design 3.6 and
// 6.7, locks the state's project, decides, counts the group a change of
// group takes the state from, reads the clock, then changes the fields
// given, by the caller at that time; it answers the state as stored, the
// time to the microsecond. Review leaves the started group, which keeps In
// Progress; In Progress stays in it, which counts nothing; Todo is renamed
// and moved down; and ops's Backlog, of an archived project, changes as any
// other (3.19).
func TestUpdateState(t *testing.T) {
	for _, tt := range []struct {
		name    string
		id      uuid.UUID
		p       domain.StatePatch
		changes func(s *domain.State) // what p changes of the state as it was
	}{
		{"Review to the completed group", webReview, domain.StatePatch{Group: ptr(domain.GroupCompleted), Color: ptr("#46A758")},
			func(s *domain.State) { s.Group, s.Color = domain.GroupCompleted, "#46A758" }},
		{"In Progress kept in its group", webStarted, domain.StatePatch{Group: ptr(domain.GroupStarted), Description: ptr("Being done")},
			func(s *domain.State) { s.Description = "Being done" }},
		{"Todo renamed and moved down", webTodo, domain.StatePatch{Name: ptr("Next"), Sequence: ptr(60000.0)},
			func(s *domain.State) { s.Name, s.Sequence = "Next", 60000 }},
		{"archived ops's Backlog", opsBacklog, domain.StatePatch{Name: ptr("Inbox")}, func(s *domain.State) { s.Name = "Inbox" }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, s := newUpdateState()
			before := s.states[tt.id]
			got, err := uc.Execute(as(bob), tt.id, tt.p)
			want := before
			tt.changes(&want)
			want.UpdatedAt = now
			if err != nil || got != want || s.states[tt.id] != want {
				t.Errorf("Execute() = %+v, %v, stored %+v; want %+v", got, err, s.states[tt.id], want)
			}
			calls := stateUpdated(bob, tt.id, tt.p, before.Group)
			if before.ProjectID == opsID {
				calls = append(stateLocked(tt.id, opsID, bob, domain.ActionStateUpdate), calls[6:]...)
			}
			if !slices.Equal(f.log.calls, calls) {
				t.Errorf("calls\n%q\nwant\n%q", f.log.calls, calls)
			}
		})
	}
}

// Refusals, each in its place, and no state changed: no caller, and values
// the domain refuses, before the transaction; a state that is not there, a
// workspace or project deleted while its lock waited, a state deleted or
// moved to ops meanwhile, and a caller who does not see web, each
// project.state_not_found; a member, the Authorizer's 403. Then Todo, the
// only state of its group, moved to another (project.state_last_in_group,
// before the change), and Done renamed to Review's name, the store's
// project.state_name_taken. A state changed as another is the write's own
// error.
func TestUpdateStateRefuses(t *testing.T) {
	upTo := func(n int) []string { return stateLocked(webTodo, webID, bob, domain.ActionStateUpdate)[:n] }
	rename := domain.StatePatch{Name: ptr("Later")}
	moveTodo := domain.StatePatch{Group: ptr(domain.GroupBacklog)}
	for _, tt := range []struct {
		name  string
		ctx   context.Context
		id    uuid.UUID
		p     domain.StatePatch
		set   func(f *writeFixture)
		want  error
		calls []string
	}{
		{"no caller", context.Background(), webTodo, rename, nil, shared.Unauthenticated(), nil},
		{"the triage group", as(bob), webTodo, domain.StatePatch{Group: ptr(domain.GroupTriage)}, nil,
			shared.Invalid(shared.FieldError{Field: "group", Code: shared.FieldNotAllowed}), nil},
		{"no state", as(bob), uuid.Nil(), rename, nil, domain.ErrStateNotFound, []string{"Begin", "StateByID " + uuid.Nil().String()}},
		{"acme deleted while its lock waited", as(bob), webTodo, rename, func(f *writeFixture) { f.workspaces.gone = true },
			domain.ErrStateNotFound, upTo(3)},
		{"web deleted while its lock waited", as(bob), webTodo, rename, func(f *writeFixture) { f.store.deleted = true },
			domain.ErrStateNotFound, upTo(4)},
		{"the state deleted while the locks waited", as(bob), webTodo, rename, func(f *writeFixture) { f.store.reread.gone = true },
			domain.ErrStateNotFound, upTo(5)},
		{"the state moved to ops", as(bob), webTodo, rename, func(f *writeFixture) { f.store.reread.project = opsID },
			domain.ErrStateNotFound, upTo(5)},
		{"a caller who does not see web", as(erin), webTodo, rename, nil, domain.ErrStateNotFound,
			stateLocked(webTodo, webID, erin, domain.ActionStateUpdate)},
		{"a member", as(alice), webTodo, rename, nil, shared.Forbidden(), stateLocked(webTodo, webID, alice, domain.ActionStateUpdate)},
		{"Todo, its group's only state, moved", as(bob), webTodo, moveTodo, nil, domain.ErrStateLastInGroup,
			append(upTo(6), fmt.Sprintf("CountGroupStates %s %s", webID, domain.GroupUnstarted))},
		{"Done renamed to Review's name", as(bob), webDone, domain.StatePatch{Name: ptr("Review")}, nil, domain.ErrStateNameTaken,
			stateUpdated(bob, webDone, domain.StatePatch{Name: ptr("Review")}, domain.GroupCompleted)},
		{"the change answered for another state", as(bob), webTodo, rename, func(f *writeFixture) { f.store.changedAs = uuid.NewV7() }, nil,
			stateUpdated(bob, webTodo, rename, domain.GroupUnstarted)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, s := newUpdateState()
			if tt.set != nil {
				tt.set(f)
			}
			before := maps.Clone(s.states)
			_, err := uc.Execute(tt.ctx, tt.id, tt.p)
			outcome{tt.name, tt.want, tt.calls}.check(t, err, f)
			if tt.name != "the change answered for another state" && !maps.Equal(s.states, before) {
				t.Errorf("the states after the refusal: %v; want them as they were, %v", s.states, before)
			}
		})
	}
}

// Every port's failure comes back as itself, the commit's too, after the
// calls before it and none after: Review moved to the backlog group, so
// that the count runs.
func TestUpdateStateReturnsEachFailure(t *testing.T) {
	move := domain.StatePatch{Group: ptr(domain.GroupBacklog)}
	all := stateUpdated(bob, webReview, move, domain.GroupStarted)
	fail := func(method string) func(f *writeFixture) {
		return func(f *writeFixture) { f.store.errs = map[string]error{method: errDisk} }
	}
	for _, tt := range []struct {
		name  string
		fail  func(f *writeFixture)
		calls int // how many of the change's calls ran
	}{
		{"the state's read", fail("StateByID"), 2},
		{"the workspace's lock", func(f *writeFixture) { f.workspaces.err = errDisk }, 3},
		{"the project's lock", fail("LockProject"), 4},
		{"the read under the locks", func(f *writeFixture) { f.store.reread.err = errDisk }, 5},
		{"the decision", func(f *writeFixture) { f.auth.errs[grantKey{bob, acme.ID}] = errDisk }, 6},
		{"the count of the group", fail("CountGroupStates"), 7},
		{"the change", fail("UpdateState"), 9},
		{"the commit", func(f *writeFixture) { f.tx.commitErr = errDisk }, 9},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, _ := newUpdateState()
			tt.fail(f)
			_, err := uc.Execute(as(bob), webReview, move)
			outcome{tt.name, errDisk, all[:tt.calls]}.check(t, err, f)
		})
	}
}
