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

// newUpdateMember is UpdateProjectMember over newMemberWrites' fakes, its
// clock logged.
func newUpdateMember() (*app.UpdateProjectMember, *writeFixture) {
	f := newMemberWrites()
	return app.NewUpdateProjectMember(f.locks(), f.store, f.tx, clockAt{clockNow, f.log}), f
}

// roleChanged are the calls of caller's change of user's membership of web to
// role: its locks and decision, the clock, the change at that time.
func roleChanged(f *writeFixture, caller, user uuid.UUID, role shared.Role) []string {
	id := f.memberOf(webID, user)
	return append(memberLocked(id, user, webID, caller, domain.ActionMemberUpdate, true), "Now",
		fmt.Sprintf("UpdateMemberRole %s as %d by %s at %s", id, role, caller, clockNow.Format(timeFormat)))
}

// UpdateProjectMember, in one transaction and in the order of M3 design
// 3.6, takes the membership's locks, decides, reads the clock, then changes
// the role, by the caller at that time, and answers the membership as
// stored. bob, a project admin who is a workspace member, makes alice, a
// member, a guest, and keeps carol, a workspace guest, a guest; gina, the
// workspace's admin and a project member, makes herself an admin, frank, an
// admin, a member, and alice an admin (3.5's exception).
func TestUpdateProjectMember(t *testing.T) {
	for _, tt := range []struct {
		name         string
		caller, user uuid.UUID
		role         shared.Role
	}{
		{"bob makes alice a guest", bob, alice, shared.RoleGuest},
		{"bob keeps carol a guest", bob, carol, shared.RoleGuest},
		{"gina makes herself an admin", gina, gina, shared.RoleAdmin},
		{"gina makes frank a member", gina, frank, shared.RoleMember},
		{"gina makes alice an admin", gina, alice, shared.RoleAdmin},
	} {
		uc, f := newUpdateMember()
		id, calls := f.memberOf(webID, tt.user), roleChanged(f, tt.caller, tt.user, tt.role)
		m, err := uc.Execute(as(tt.caller), id, tt.role)
		if want := (domain.Member{ID: id, ProjectID: webID, MemberID: tt.user, Role: tt.role, CreatedAt: memberSince}); err != nil || m != want {
			t.Errorf("%s: Execute() = %+v, %v; want %+v", tt.name, m, err, want)
		}
		if !slices.Equal(f.log.calls, calls) {
			t.Errorf("%s: calls\n%q\nwant\n%q", tt.name, f.log.calls, calls)
		}
	}
}

// Refusals, each in its place, and nothing changed: no caller, and a role
// outside the three, before the transaction; a membership that is not
// there, a workspace, project or membership deleted while a lock waited, a
// project or membership moved meanwhile, and a caller who does not see the
// project, each project.member_not_found, the last after the decision; a
// project member, the Authorizer's 403. Then, to a caller who may change
// roles, after the decision: an ended membership, 404, before any other
// check, and so when it ended while the locks waited; his own, 409; an
// admin's role and a role of admin, from a project admin who is no
// workspace admin, 403 project.role_too_high; a workspace guest made more
// than a guest, 422, from anyone. A member who is no active member of the
// workspace, or whose role is answered for another account, is the
// write's own error. An answer of the lock path for another key than it
// asked for is the path's to refuse (TestLocksCheckEachAnswerAgainstItsKey).
func TestUpdateProjectMemberRefuses(t *testing.T) {
	guestOnly := shared.Invalid(shared.FieldError{Field: "role", Code: shared.FieldNotAllowed, Message: "must be 5: the member is a guest of the workspace"})
	locked := func(f *writeFixture, caller, user uuid.UUID) []string {
		return memberLocked(f.memberOf(webID, user), user, webID, caller, domain.ActionMemberUpdate, true)
	}
	// upTo is the first n calls of caller's write on user's membership.
	upTo := func(n int) func(f *writeFixture, caller, user uuid.UUID) []string {
		return func(f *writeFixture, caller, user uuid.UUID) []string { return locked(f, caller, user)[:n] }
	}
	none := func(*writeFixture, uuid.UUID, uuid.UUID) []string { return nil }
	for _, tt := range []struct {
		name         string
		caller, user uuid.UUID // no caller when caller is zero; a membership not there when user is zero
		role         shared.Role
		set          func(f *writeFixture)
		want         error
		calls        func(f *writeFixture, caller, user uuid.UUID) []string
	}{
		{"no caller", uuid.UUID{}, alice, shared.RoleGuest, nil, shared.Unauthenticated(), none},
		{"a role outside the three", bob, alice, 10, nil,
			shared.Invalid(shared.FieldError{Field: "role", Code: shared.FieldInvalidFormat, Message: "is not 5, 15 or 20"}), none},
		{"no membership", bob, uuid.UUID{}, shared.RoleGuest, nil, domain.ErrMemberNotFound,
			func(*writeFixture, uuid.UUID, uuid.UUID) []string {
				return []string{"Begin", "MemberByID " + uuid.Nil().String()}
			}},
		{"acme deleted while its lock waited", bob, alice, shared.RoleGuest, func(f *writeFixture) { f.workspaces.gone = true },
			domain.ErrMemberNotFound, upTo(3)},
		{"web deleted while its lock waited", bob, alice, shared.RoleGuest, func(f *writeFixture) { f.store.deleted = true },
			domain.ErrMemberNotFound, upTo(5)},
		{"web moved to another workspace", bob, alice, shared.RoleGuest, func(f *writeFixture) { f.store.moved = uuid.NewV7() },
			domain.ErrMemberNotFound, upTo(5)},
		{"the membership deleted while the locks waited", bob, alice, shared.RoleGuest, func(f *writeFixture) { f.store.reread.gone = true },
			domain.ErrMemberNotFound, upTo(6)},
		{"the membership moved to ops", bob, alice, shared.RoleGuest, func(f *writeFixture) { f.store.reread.project = opsID },
			domain.ErrMemberNotFound, upTo(6)},
		{"a caller who does not see web", erin, alice, shared.RoleGuest, nil, domain.ErrMemberNotFound, locked},
		{"a project member", alice, carol, shared.RoleGuest, nil, shared.Forbidden(), locked},
		{"an ended membership", bob, dave, shared.RoleGuest, nil, domain.ErrMemberNotFound, locked},
		{"an ended membership made an admin", bob, dave, shared.RoleAdmin, nil, domain.ErrMemberNotFound, locked},
		{"an ended membership of one who left the workspace", bob, dave, shared.RoleGuest,
			func(f *writeFixture) { delete(f.members.roles[acme.ID], dave) }, domain.ErrMemberNotFound, locked},
		{"the membership ended while the locks waited", bob, alice, shared.RoleGuest, func(f *writeFixture) { f.store.reread.ended = true },
			domain.ErrMemberNotFound, locked},
		{"his own", bob, bob, shared.RoleMember, nil, domain.ErrOwnMembership, locked},
		{"another admin", bob, frank, shared.RoleMember, nil, domain.ErrRoleTooHigh, locked},
		{"a member made an admin while the locks waited", bob, alice, shared.RoleGuest,
			func(f *writeFixture) { f.store.reread.role = shared.RoleAdmin }, domain.ErrRoleTooHigh, locked},
		{"a member made an admin", bob, alice, shared.RoleAdmin, nil, domain.ErrRoleTooHigh, locked},
		{"a workspace guest made a member", bob, ivy, shared.RoleMember, nil, guestOnly, locked},
		{"a workspace guest made an admin by the workspace's admin", gina, ivy, shared.RoleAdmin, nil, guestOnly, locked},
		{"a member who is no active member of the workspace", bob, alice, shared.RoleGuest,
			func(f *writeFixture) { delete(f.members.roles[acme.ID], alice) }, nil, locked},
		{"his workspace role answered for another account", bob, alice, shared.RoleGuest,
			func(f *writeFixture) { f.members.answersFor = carol }, nil, locked},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newUpdateMember()
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
			_, err := uc.Execute(ctx, id, tt.role)
			outcome{tt.name, tt.want, tt.calls(f, tt.caller, tt.user)}.check(t, err, f)
			if after := f.store.projects[webID].members[tt.user]; after != before {
				t.Errorf("the membership after the refusal: %+v, want it as it was, %+v", after, before)
			}
		})
	}
}

// Every port's failure comes back as itself, the commit's too, after the
// calls before it and none after; a change answered for another membership
// is the write's own error.
func TestUpdateProjectMemberReturnsEachFailure(t *testing.T) {
	for _, tt := range []struct {
		name  string
		fail  func(f *writeFixture)
		want  error
		calls int // how many of the change's calls ran
	}{
		{"the read", func(f *writeFixture) { f.store.errs = map[string]error{"MemberByID": errDisk} }, errDisk, 2},
		{"the workspace's lock", func(f *writeFixture) { f.workspaces.err = errDisk }, errDisk, 3},
		{"the member's workspace membership", func(f *writeFixture) { f.members.err = errDisk }, errDisk, 4},
		{"the project's lock", func(f *writeFixture) { f.store.errs = map[string]error{"LockProject": errDisk} }, errDisk, 5},
		{"the read under the locks", func(f *writeFixture) { f.store.reread.err = errDisk }, errDisk, 6},
		{"the decision", func(f *writeFixture) { f.auth.errs = map[grantKey]error{{bob, acme.ID}: errDisk} }, errDisk, 7},
		{"the change", func(f *writeFixture) { f.store.errs = map[string]error{"UpdateMemberRole": errDisk} }, errDisk, 9},
		{"the change answered for another membership", func(f *writeFixture) { f.store.changedAs = uuid.NewV7() }, nil, 9},
		{"the commit", func(f *writeFixture) { f.tx.commitErr = errDisk }, errDisk, 9},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newUpdateMember()
			tt.fail(f)
			all := roleChanged(f, bob, alice, shared.RoleGuest)
			_, err := uc.Execute(as(bob), f.memberOf(webID, alice), shared.RoleGuest)
			outcome{tt.name, tt.want, all[:tt.calls]}.check(t, err, f)
		})
	}
}
