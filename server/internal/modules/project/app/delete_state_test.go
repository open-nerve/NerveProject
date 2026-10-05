package app_test

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// newDeleteState is DeleteState over newStates' fakes, its clock logged.
func newDeleteState() (*app.DeleteState, *writeFixture, *fakeStates) {
	f, s := newStates()
	return app.NewDeleteState(f.locks(), s, f.tx, clockAt{clockNow, f.log}), f, s
}

// stateDeleted are the calls of user's deletion of the state id of project
// up to the deletion: its locks and decision, the clock, the deletion by
// user at that time.
func stateDeleted(user, id, project uuid.UUID) []string {
	return append(stateLocked(id, project, user, domain.ActionStateDelete), "Now",
		fmt.Sprintf("DeleteState %s by %s at %s", id, user, clockNow.Format(timeFormat)))
}

// DeleteState, in one transaction and in the order of M3 design 3.6 and
// 3.17, locks the state's project, decides, reads the clock, deletes the
// state by the caller at that time, then counts the states its group keeps:
// Review leaves In Progress in the started group. ops, archived, has its
// states deleted as any other's (3.19): Icebox leaves Backlog in the
// backlog group.
func TestDeleteState(t *testing.T) {
	icebox := uuid.NewV7()
	for _, tt := range []struct {
		name    string
		id      uuid.UUID
		project uuid.UUID
		group   domain.StateGroup
	}{
		{"Review", webReview, webID, domain.GroupStarted},
		{"archived ops's Icebox", icebox, opsID, domain.GroupBacklog},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, s := newDeleteState()
			s.states[icebox] = domain.State{ID: icebox, WorkspaceID: acme.ID, ProjectID: opsID, Name: "Icebox", Group: domain.GroupBacklog,
				Sequence: 20000}
			want := maps.Clone(s.states)
			delete(want, tt.id)
			err := uc.Execute(as(bob), tt.id)
			calls := append(stateDeleted(bob, tt.id, tt.project), fmt.Sprintf("CountGroupStates %s %s", tt.project, tt.group))
			if err != nil || !maps.Equal(s.states, want) {
				t.Errorf("Execute() = %v, the states after it %v; want nil, %v", err, s.states, want)
			}
			if !slices.Equal(f.log.calls, calls) {
				t.Errorf("calls\n%q\nwant\n%q", f.log.calls, calls)
			}
		})
	}
}

// Refusals, each in its place: no caller, before the transaction; a state
// that is not there, a workspace or project deleted while its lock waited,
// a state deleted or moved to ops meanwhile, and a caller who does not see
// web, each project.state_not_found; a member, the Authorizer's 403. Then,
// past the decision: Backlog, the default and its group's only state,
// which the deletion passes over and the state read again answers
// project.state_default, the guarded deletion coming before the group's
// count (P7a spec 3 item 13); a state the deletion passed over that is
// gone when read again,
// project.state_not_found; Todo, the only state of its group, deleted and
// project.state_last_in_group, which the transaction rolls back. A
// deletion that passed over a state neither gone nor the default is the
// write's own error. A refusal before the deletion leaves the states as
// they were.
func TestDeleteStateRefuses(t *testing.T) {
	upTo := func(n int) []string { return stateLocked(webReview, webID, bob, domain.ActionStateDelete)[:n] }
	readAgain := func(id uuid.UUID) []string { return append(stateDeleted(bob, id, webID), "StateByID "+id.String()) }
	for _, tt := range []struct {
		name  string
		ctx   context.Context
		id    uuid.UUID
		set   func(f *writeFixture, s *fakeStates)
		want  error
		calls []string
	}{
		{"no caller", context.Background(), webReview, nil, shared.Unauthenticated(), nil},
		{"no state", as(bob), uuid.Nil(), nil, domain.ErrStateNotFound, []string{"Begin", "StateByID " + uuid.Nil().String()}},
		{"acme deleted while its lock waited", as(bob), webReview, func(f *writeFixture, _ *fakeStates) { f.workspaces.gone = true },
			domain.ErrStateNotFound, upTo(3)},
		{"web deleted while its lock waited", as(bob), webReview, func(f *writeFixture, _ *fakeStates) { f.store.deleted = true },
			domain.ErrStateNotFound, upTo(4)},
		{"the state deleted while the locks waited", as(bob), webReview, func(f *writeFixture, _ *fakeStates) { f.store.reread.gone = true },
			domain.ErrStateNotFound, upTo(5)},
		{"the state moved to ops", as(bob), webReview, func(f *writeFixture, _ *fakeStates) { f.store.reread.project = opsID },
			domain.ErrStateNotFound, upTo(5)},
		{"a caller who does not see web", as(erin), webReview, nil, domain.ErrStateNotFound,
			stateLocked(webReview, webID, erin, domain.ActionStateDelete)},
		{"a member", as(alice), webReview, nil, shared.Forbidden(), stateLocked(webReview, webID, alice, domain.ActionStateDelete)},
		{"Backlog, the default and its group's only state", as(bob), webBacklog, nil, domain.ErrStateDefault, readAgain(webBacklog)},
		{"a state passed over and gone", as(bob), webReview, func(_ *writeFixture, s *fakeStates) { s.passOver = "gone" },
			domain.ErrStateNotFound, readAgain(webReview)},
		{"Todo, its group's only state", as(bob), webTodo, nil, domain.ErrStateLastInGroup,
			append(stateDeleted(bob, webTodo, webID), fmt.Sprintf("CountGroupStates %s %s", webID, domain.GroupUnstarted))},
		{"a state passed over and kept", as(bob), webReview, func(_ *writeFixture, s *fakeStates) { s.passOver = "kept" }, nil,
			readAgain(webReview)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, s := newDeleteState()
			if tt.set != nil {
				tt.set(f, s)
			}
			before := maps.Clone(s.states)
			err := uc.Execute(tt.ctx, tt.id)
			outcome{tt.name, tt.want, tt.calls}.check(t, err, f)
			beforeTheDeletion := len(tt.calls) <= len(stateLocked(tt.id, webID, bob, domain.ActionStateDelete))
			if beforeTheDeletion && !maps.Equal(s.states, before) {
				t.Errorf("the states after the refusal: %v; want them as they were, %v", s.states, before)
			}
		})
	}
}

// Every port's failure comes back as itself, the commit's too, after the
// calls before it and none after: Review deleted, so that the count runs.
func TestDeleteStateReturnsEachFailure(t *testing.T) {
	all := append(stateDeleted(bob, webReview, webID), fmt.Sprintf("CountGroupStates %s %s", webID, domain.GroupStarted))
	fail := func(method string) func(f *writeFixture) {
		return func(f *writeFixture) { f.store.errs = map[string]error{method: errDisk} }
	}
	for _, tt := range []struct {
		name  string
		fail  func(f *writeFixture)
		calls int // how many of the deletion's calls ran
	}{
		{"the state's read", fail("StateByID"), 2},
		{"the workspace's lock", func(f *writeFixture) { f.workspaces.err = errDisk }, 3},
		{"the project's lock", fail("LockProject"), 4},
		{"the read under the locks", func(f *writeFixture) { f.store.reread.err = errDisk }, 5},
		{"the decision", func(f *writeFixture) { f.auth.errs[grantKey{bob, acme.ID}] = errDisk }, 6},
		{"the deletion", fail("DeleteState"), 8},
		{"the count of the group", fail("CountGroupStates"), 9},
		{"the commit", func(f *writeFixture) { f.tx.commitErr = errDisk }, 9},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, _ := newDeleteState()
			tt.fail(f)
			err := uc.Execute(as(bob), webReview)
			outcome{tt.name, errDisk, all[:tt.calls]}.check(t, err, f)
		})
	}
}

// afterDeletion is s, which set changes once a deletion ran: how the read
// after the deletion answers.
type afterDeletion struct {
	*fakeStates
	set func(s *fakeStates)
}

func (s afterDeletion) DeleteState(ctx context.Context, id, by uuid.UUID, now time.Time) (bool, error) {
	deleted, err := s.fakeStates.DeleteState(ctx, id, by, now)
	s.set(s.fakeStates)
	return deleted, err
}

// The read that answers why the deletion passed over Backlog, the last
// call: its failure comes back as itself, and Backlog answered for another
// id is the write's own error, never project.state_default, as the lock
// path's read refuses it (rowWrite.read).
func TestDeleteStateChecksTheReadAfterIt(t *testing.T) {
	for _, tt := range []struct {
		name string
		set  func(s *fakeStates)
		want error
	}{
		{"the read failing", func(s *fakeStates) { s.errs = map[string]error{"StateByID": errDisk} }, errDisk},
		{"the read answering another state", func(s *fakeStates) { s.answersAs = uuid.NewV7() }, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f, s := newStates()
			err := app.NewDeleteState(f.locks(), afterDeletion{s, tt.set}, f.tx, clockAt{clockNow, f.log}).Execute(as(bob), webBacklog)
			outcome{tt.name, tt.want, append(stateDeleted(bob, webBacklog, webID), "StateByID "+webBacklog.String())}.check(t, err, f)
		})
	}
}
