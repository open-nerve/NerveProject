package app_test

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// newRemoveMember is RemoveProjectMember over newMemberWrites' fakes, its
// clock logged.
func newRemoveMember() (*app.RemoveProjectMember, *writeFixture) {
	f := newMemberWrites()
	return app.NewRemoveProjectMember(f.locks(), f.store, f.tx, clockAt{clockNow, f.log}), f
}

// removed are the calls of caller's removal of user's membership of web:
// its locks and decision, no workspace membership among them, the clock,
// the ending at that time.
func removed(caller, user uuid.UUID) []string {
	id := newMemberWrites().memberOf(webID, user)
	return append(memberLocked(id, user, webID, caller, domain.ActionMemberRemove, false), "Now",
		fmt.Sprintf("EndMember %s of %s by %s at %s", user, webID, caller, clockNow.Format(timeFormat)))
}

// RemoveProjectMember, in one transaction and in the order of M3 design
// 3.6, takes the membership's locks, decides, reads the clock, then ends
// the membership, by the caller at that time; it keeps its role. bob, a
// project admin, removes alice, a member, carol, a guest, and frank,
// another admin (3.5 refuses only a higher role); gina, the workspace's
// admin and a project member, removes alice, a member, and ivy, a guest.
func TestRemoveProjectMember(t *testing.T) {
	for _, tt := range []struct{ caller, user uuid.UUID }{{bob, alice}, {bob, carol}, {bob, frank}, {gina, alice}, {gina, ivy}} {
		uc, f := newRemoveMember()
		before := f.store.projects[webID].members[tt.user]
		if err := uc.Execute(as(tt.caller), f.memberOf(webID, tt.user)); err != nil || !slices.Equal(f.log.calls, removed(tt.caller, tt.user)) {
			t.Errorf("Execute() by %s of %s = %v, calls\n%q\nwant\n%q", tt.caller, tt.user, err, f.log.calls, removed(tt.caller, tt.user))
		}
		if m := f.store.projects[webID].members[tt.user]; m.Active || m.Role != before.Role || m.ID != before.ID {
			t.Errorf("the membership after the removal: %+v; want %+v ended", m, before)
		}
	}
}

// Refusals, each in its place, and nothing ended: no caller, before the
// transaction; a membership that is not there, a workspace, project or
// membership deleted while a lock waited, a project or membership moved
// meanwhile, and a caller who does not see the project, each
// project.member_not_found; a project member, the Authorizer's 403. Then,
// to a caller who may remove members, after the decision: an ended
// membership, 404, before any other check, and so when it ended while the
// locks waited, an admin's to the workspace's admin; his own, 409, the
// workspace's admin's too; a higher role, 403 project.role_too_high, from
// the workspace's admin too. A membership answered for another id is the
// write's own error. The row of his own membership ended while the locks
// waited pins only the order of the two checks, ended before own: on the
// wired app it cannot occur, as a caller who passes the decision has an
// active membership of the project, which the Authorizer reads under the
// same locks.
func TestRemoveProjectMemberRefuses(t *testing.T) {
	locked := func(f *writeFixture, caller, user uuid.UUID) []string {
		return memberLocked(f.memberOf(webID, user), user, webID, caller, domain.ActionMemberRemove, false)
	}
	upTo := func(n int) func(f *writeFixture, caller, user uuid.UUID) []string {
		return func(f *writeFixture, caller, user uuid.UUID) []string { return locked(f, caller, user)[:n] }
	}
	for _, tt := range []struct {
		name         string
		caller, user uuid.UUID // no caller when caller is zero; a membership not there when user is zero
		set          func(f *writeFixture)
		want         error
		calls        func(f *writeFixture, caller, user uuid.UUID) []string
	}{
		{"no caller", uuid.UUID{}, alice, nil, shared.Unauthenticated(), func(*writeFixture, uuid.UUID, uuid.UUID) []string { return nil }},
		{"no membership", bob, uuid.UUID{}, nil, domain.ErrMemberNotFound,
			func(*writeFixture, uuid.UUID, uuid.UUID) []string {
				return []string{"Begin", "MemberByID " + uuid.Nil().String()}
			}},
		{"acme deleted while its lock waited", bob, alice, func(f *writeFixture) { f.workspaces.gone = true }, domain.ErrMemberNotFound, upTo(3)},
		{"web deleted while its lock waited", bob, alice, func(f *writeFixture) { f.store.deleted = true }, domain.ErrMemberNotFound, upTo(4)},
		{"web moved to another workspace", bob, alice, func(f *writeFixture) { f.store.moved = uuid.NewV7() }, domain.ErrMemberNotFound, upTo(4)},
		{"the membership deleted while the locks waited", bob, alice, func(f *writeFixture) { f.store.reread.gone = true },
			domain.ErrMemberNotFound, upTo(5)},
		{"the membership moved to ops", bob, alice, func(f *writeFixture) { f.store.reread.project = opsID }, domain.ErrMemberNotFound, upTo(5)},
		{"a caller who does not see web", erin, alice, nil, domain.ErrMemberNotFound, locked},
		{"a project member", alice, carol, nil, shared.Forbidden(), locked},
		{"an ended membership", bob, dave, nil, domain.ErrMemberNotFound, locked},
		{"an admin's membership ended while the locks waited, by the workspace's admin", gina, frank,
			func(f *writeFixture) { f.store.reread.ended = true }, domain.ErrMemberNotFound, locked},
		{"his own membership ended while the locks waited", bob, bob, func(f *writeFixture) { f.store.reread.ended = true },
			domain.ErrMemberNotFound, locked},
		{"his own", bob, bob, nil, domain.ErrOwnMembership, locked},
		{"the workspace's admin's own", gina, gina, nil, domain.ErrOwnMembership, locked},
		{"an admin, by the workspace's admin who is a project member", gina, bob, nil, domain.ErrRoleTooHigh, locked},
		{"a member made an admin while the locks waited, by the workspace's admin", gina, alice,
			func(f *writeFixture) { f.store.reread.role = shared.RoleAdmin }, domain.ErrRoleTooHigh, locked},
		{"a membership answered for another id", bob, alice, func(f *writeFixture) { f.store.answersAs = uuid.NewV7() }, nil, upTo(2)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newRemoveMember()
			if tt.set != nil {
				tt.set(f)
			}
			ctx, id := context.Background(), uuid.Nil()
			if tt.caller != (uuid.UUID{}) {
				ctx = as(tt.caller)
			}
			if tt.user != (uuid.UUID{}) {
				id = f.memberOf(webID, tt.user)
			}
			before := f.store.projects[webID].members[tt.user]
			outcome{tt.name, tt.want, tt.calls(f, tt.caller, tt.user)}.check(t, uc.Execute(ctx, id), f)
			if after := f.store.projects[webID].members[tt.user]; after != before {
				t.Errorf("the membership after the refusal: %+v, want it as it was, %+v", after, before)
			}
		})
	}
}

// Every port's failure comes back as itself, the commit's too, after the
// calls before it and none after.
func TestRemoveProjectMemberReturnsEachFailure(t *testing.T) {
	all := removed(bob, alice)
	for _, tt := range []struct {
		name  string
		fail  func(f *writeFixture)
		calls int // how many of the removal's calls ran
	}{
		{"the read", func(f *writeFixture) { f.store.errs = map[string]error{"MemberByID": errDisk} }, 2},
		{"the workspace's lock", func(f *writeFixture) { f.workspaces.err = errDisk }, 3},
		{"the project's lock", func(f *writeFixture) { f.store.errs = map[string]error{"LockProject": errDisk} }, 4},
		{"the read under the locks", func(f *writeFixture) { f.store.reread.err = errDisk }, 5},
		{"the decision", func(f *writeFixture) { f.auth.errs = map[grantKey]error{{bob, acme.ID}: errDisk} }, 6},
		{"the ending", func(f *writeFixture) { f.store.errs = map[string]error{"EndMember": errDisk} }, 8},
		{"the commit", func(f *writeFixture) { f.tx.commitErr = errDisk }, 8},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newRemoveMember()
			tt.fail(f)
			outcome{tt.name, errDisk, all[:tt.calls]}.check(t, uc.Execute(as(bob), aliceInWeb), f)
		})
	}
}
