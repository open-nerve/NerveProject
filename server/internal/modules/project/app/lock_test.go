package app_test

import (
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The lock path every write on a project takes (lock.go) checks each answer
// it locks or reads by against what it asked for (M3 design 3.6 convention
// 2; P5b review §6): the workspace its lock answers, and the row under the
// project a write names, read before the locks and again under them. A
// mismatch is the write's own error, never a 404 or a 403, and nothing runs
// after it. Every write on a project reaches its locks through
// lockAndDecide or lockRowAndDecide, and both through Locks.lock, so the
// checks are the path's and are pinned here once, through one write that
// takes all three: bob's change of alice's role in web. A real store cannot
// answer another key: each check keeps the path consistent with its ports,
// and its mutant dies here, at the unit level, by nature.
func TestLocksCheckEachAnswerAgainstItsKey(t *testing.T) {
	for _, tt := range []struct {
		name  string
		set   func(f *writeFixture)
		calls int // how many of the write's calls ran
	}{
		{"acme's lock answered for another workspace", func(f *writeFixture) { f.workspaces.answersAs = uuid.NewV7() }, 3},
		{"the row answered for another id", func(f *writeFixture) { f.store.answersAs = uuid.NewV7() }, 2},
		{"the row read again for another id", func(f *writeFixture) { f.store.reread.id = uuid.NewV7() }, 6},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newUpdateMember()
			tt.set(f)
			id := f.memberOf(webID, alice)
			before := f.store.projects[webID].members[alice]
			_, err := uc.Execute(as(bob), id, shared.RoleGuest)
			outcome{tt.name, nil, memberLocked(id, alice, webID, bob, domain.ActionMemberUpdate, true)[:tt.calls]}.check(t, err, f)
			if after := f.store.projects[webID].members[alice]; after != before {
				t.Errorf("alice's membership after the refusal: %+v, want it as it was, %+v", after, before)
			}
		})
	}
}
