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

// newMarkDefaultState is MarkDefaultState over newStates' fakes, its clock
// logged.
func newMarkDefaultState() (*app.MarkDefaultState, *writeFixture, *fakeStates) {
	f, s := newStates()
	return app.NewMarkDefaultState(f.locks(), s, f.tx, clockAt{clockNow, f.log}), f, s
}

// defaultMarked are the calls of user's making the state id of project its
// default: its locks and decision, the clock, the two statements by user at
// that time.
func defaultMarked(user, id, project uuid.UUID) []string {
	return append(stateLocked(id, project, user, domain.ActionStateMarkDefault), "Now",
		fmt.Sprintf("MarkDefaultState %s %s by %s at %s", project, id, user, clockNow.Format(timeFormat)))
}

// MarkDefaultState, in one transaction and in the order of M3 design 3.6
// and 3.17, locks the state's project, decides, reads the clock, then makes
// the state the default by the caller at that time, the project's default
// before it the default no longer: Todo becomes web's, and Backlog is no
// longer. Backlog made web's default again stays it; ops, archived, has
// its default made as any other's (3.19).
func TestMarkDefaultState(t *testing.T) {
	for _, tt := range []struct {
		name    string
		id      uuid.UUID
		project uuid.UUID
		was     uuid.UUID // the project's default before
	}{
		{"Todo", webTodo, webID, webBacklog},
		{"Backlog again", webBacklog, webID, webBacklog},
		{"archived ops's Backlog", opsBacklog, opsID, opsBacklog},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, s := newMarkDefaultState()
			want := maps.Clone(s.states)
			was, is := want[tt.was], want[tt.id]
			was.Default, was.UpdatedAt = false, now
			want[tt.was] = was
			is.Default, is.UpdatedAt = true, now
			want[tt.id] = is
			err := uc.Execute(as(bob), tt.id)
			if err != nil || !maps.Equal(s.states, want) {
				t.Errorf("Execute() = %v, the states after it %v; want nil, %v", err, s.states, want)
			}
			if calls := defaultMarked(bob, tt.id, tt.project); !slices.Equal(f.log.calls, calls) {
				t.Errorf("calls\n%q\nwant\n%q", f.log.calls, calls)
			}
		})
	}
}

// Refusals, each in its place: no caller, before the transaction; a state
// that is not there, a workspace or project deleted while its lock waited,
// a state deleted or moved to ops meanwhile, and a caller who does not see
// web, each project.state_not_found; a member, the Authorizer's 403. Then a
// state the second statement passed over, project.state_not_found, which
// the transaction rolls the first back from. A refusal before the
// statements leaves the states as they were.
func TestMarkDefaultStateRefuses(t *testing.T) {
	upTo := func(n int) []string { return stateLocked(webTodo, webID, bob, domain.ActionStateMarkDefault)[:n] }
	for _, tt := range []struct {
		name  string
		ctx   context.Context
		id    uuid.UUID
		set   func(f *writeFixture, s *fakeStates)
		want  error
		calls []string
	}{
		{"no caller", context.Background(), webTodo, nil, shared.Unauthenticated(), nil},
		{"no state", as(bob), uuid.Nil(), nil, domain.ErrStateNotFound, []string{"Begin", "StateByID " + uuid.Nil().String()}},
		{"acme deleted while its lock waited", as(bob), webTodo, func(f *writeFixture, _ *fakeStates) { f.workspaces.gone = true },
			domain.ErrStateNotFound, upTo(3)},
		{"web deleted while its lock waited", as(bob), webTodo, func(f *writeFixture, _ *fakeStates) { f.store.deleted = true },
			domain.ErrStateNotFound, upTo(4)},
		{"the state deleted while the locks waited", as(bob), webTodo, func(f *writeFixture, _ *fakeStates) { f.store.reread.gone = true },
			domain.ErrStateNotFound, upTo(5)},
		{"the state moved to ops", as(bob), webTodo, func(f *writeFixture, _ *fakeStates) { f.store.reread.project = opsID },
			domain.ErrStateNotFound, upTo(5)},
		{"a caller who does not see web", as(erin), webTodo, nil, domain.ErrStateNotFound,
			stateLocked(webTodo, webID, erin, domain.ActionStateMarkDefault)},
		{"a member", as(alice), webTodo, nil, shared.Forbidden(), stateLocked(webTodo, webID, alice, domain.ActionStateMarkDefault)},
		{"a state the second statement passed over", as(bob), webTodo, func(_ *writeFixture, s *fakeStates) { s.passOver = "gone" },
			domain.ErrStateNotFound, defaultMarked(bob, webTodo, webID)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, s := newMarkDefaultState()
			if tt.set != nil {
				tt.set(f, s)
			}
			before := maps.Clone(s.states)
			err := uc.Execute(tt.ctx, tt.id)
			outcome{tt.name, tt.want, tt.calls}.check(t, err, f)
			if beforeTheStatements := len(tt.calls) <= 6; beforeTheStatements && !maps.Equal(s.states, before) {
				t.Errorf("the states after the refusal: %v; want them as they were, %v", s.states, before)
			}
		})
	}
}

// Every port's failure comes back as itself, the commit's too, after the
// calls before it and none after.
func TestMarkDefaultStateReturnsEachFailure(t *testing.T) {
	all := defaultMarked(bob, webTodo, webID)
	fail := func(method string) func(f *writeFixture) {
		return func(f *writeFixture) { f.store.errs = map[string]error{method: errDisk} }
	}
	for _, tt := range []struct {
		name  string
		fail  func(f *writeFixture)
		calls int // how many of the write's calls ran
	}{
		{"the state's read", fail("StateByID"), 2},
		{"the workspace's lock", func(f *writeFixture) { f.workspaces.err = errDisk }, 3},
		{"the project's lock", fail("LockProject"), 4},
		{"the read under the locks", func(f *writeFixture) { f.store.reread.err = errDisk }, 5},
		{"the decision", func(f *writeFixture) { f.auth.errs[grantKey{bob, acme.ID}] = errDisk }, 6},
		{"the statements", fail("MarkDefaultState"), 8},
		{"the commit", func(f *writeFixture) { f.tx.commitErr = errDisk }, 8},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, _ := newMarkDefaultState()
			tt.fail(f)
			err := uc.Execute(as(bob), webTodo)
			outcome{tt.name, errDisk, all[:tt.calls]}.check(t, err, f)
		})
	}
}
