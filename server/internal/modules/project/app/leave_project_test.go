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

// newLeave is LeaveProject over newMemberWrites' fakes, its clock logged,
// each of web's members deciding as his roles in acme and web: alice its
// member, carol and ivy its guests, frank its second admin; gina, acme's
// admin, its member, as newMemberWrites has her; hank, acme's member, sees
// web and is not its member: the Authorizer's 403.
func newLeave() (*app.LeaveProject, *writeFixture) {
	f := newMemberWrites()
	delete(f.auth.errs, grantKey{alice, acme.ID})
	for user, g := range map[uuid.UUID]shared.Grant{alice: {WorkspaceRole: shared.RoleMember, ProjectRole: shared.RoleMember},
		carol: {WorkspaceRole: shared.RoleGuest, ProjectRole: shared.RoleGuest}, ivy: {WorkspaceRole: shared.RoleGuest, ProjectRole: shared.RoleGuest},
		frank: {WorkspaceRole: shared.RoleMember, ProjectRole: shared.RoleAdmin, ProjectAdmin: true}} {
		f.auth.grants[grantKey{user, acme.ID}] = g
	}
	f.auth.errs[grantKey{hank, acme.ID}] = shared.Forbidden()
	return app.NewLeaveProject(f.locks(), f.store, f.tx, clockAt{clockNow, f.log}), f
}

// left are the calls of user's leaving project: its locks and decision, the
// other admins read when admin is set, the clock, the ending at that time,
// by himself.
func left(user, project uuid.UUID, admin bool) []string {
	calls := lockedDecision(user, project, domain.ActionLeave)
	if admin {
		calls = append(calls, fmt.Sprintf("HasOtherAdmin %s but %s", project, user))
	}
	return append(calls, "Now", fmt.Sprintf("EndMember %s of %s by %s at %s", user, project, user, clockNow.Format(timeFormat)))
}

// LeaveProject, in one transaction and in the order of M3 design 3.6, locks
// the project, decides, reads the other admins when the caller is an admin,
// reads the clock, then ends the caller's membership, by himself at that
// time; it keeps its role, and no other membership changes. alice, a
// member, carol and ivy, guests, gina, a member who is acme's admin, leave
// web; bob and frank, its admins, each leave while the other stays.
func TestLeaveProject(t *testing.T) {
	for _, tt := range []struct {
		user  uuid.UUID
		admin bool
	}{{alice, false}, {carol, false}, {ivy, false}, {gina, false}, {bob, true}, {frank, true}} {
		uc, f := newLeave()
		before := maps.Clone(f.store.projects[webID].members)
		if err := uc.Execute(as(tt.user), webID); err != nil || !slices.Equal(f.log.calls, left(tt.user, webID, tt.admin)) {
			t.Errorf("Execute() by %s = %v, calls\n%q\nwant\n%q", tt.user, err, f.log.calls, left(tt.user, webID, tt.admin))
		}
		ended := before[tt.user]
		ended.Active = false
		before[tt.user] = ended
		if got := f.store.projects[webID].members; !maps.Equal(got, before) {
			t.Errorf("the memberships after %s left: %v; want %v, his ended alone", tt.user, got, before)
		}
	}
}

// Refusals, each in its place, and no membership changed: no caller, before
// the transaction; a project that is not there, a workspace or project
// deleted while its lock waited, a project moved meanwhile, and a caller
// who does not see the project, each project.not_found; one who sees it and
// is not its member, the Authorizer's 403. Then 3.7's rule 1: an admin
// whose project has no other active admin, the other's membership ended or
// the other a member now, and one who is the project's only member, is
// project.sole_admin.
func TestLeaveProjectRefuses(t *testing.T) {
	upTo := func(n int) []string { return lockedDecision(bob, webID, domain.ActionLeave)[:n] }
	soleAdmin := append(lockedDecision(bob, webID, domain.ActionLeave), fmt.Sprintf("HasOtherAdmin %s but %s", webID, bob))
	frankAs := func(role shared.Role, active bool) func(f *writeFixture) {
		return func(f *writeFixture) {
			f.store.projects[webID].members[frank] = app.Membership{ID: frankInWeb, Role: role, Active: active}
		}
	}
	for _, tt := range []struct {
		name    string
		ctx     context.Context
		project uuid.UUID
		set     func(f *writeFixture)
		want    error
		calls   []string
	}{
		{"no caller", context.Background(), webID, nil, shared.Unauthenticated(), nil},
		{"no project", as(bob), uuid.Nil(), nil, domain.ErrNotFound, noProject},
		{"acme deleted while its lock waited", as(bob), webID, func(f *writeFixture) { f.workspaces.gone = true }, domain.ErrNotFound, upTo(3)},
		{"web deleted while its lock waited", as(bob), webID, func(f *writeFixture) { f.store.deleted = true }, domain.ErrNotFound, upTo(4)},
		{"web moved to another workspace", as(bob), webID, func(f *writeFixture) { f.store.moved = uuid.NewV7() }, domain.ErrNotFound, upTo(4)},
		{"a caller who does not see web", as(erin), webID, nil, domain.ErrNotFound, lockedDecision(erin, webID, domain.ActionLeave)},
		{"a caller who sees web, not its member", as(hank), webID, nil, shared.Forbidden(), lockedDecision(hank, webID, domain.ActionLeave)},
		{"web's only admin, the other's membership ended", as(bob), webID, frankAs(shared.RoleAdmin, false), domain.ErrSoleAdmin, soleAdmin},
		{"web's only admin, the other a member now", as(bob), webID, frankAs(shared.RoleMember, true), domain.ErrSoleAdmin, soleAdmin},
		{"ops' only admin and member", as(bob), opsID, nil, domain.ErrSoleAdmin,
			append(lockedDecision(bob, opsID, domain.ActionLeave), fmt.Sprintf("HasOtherAdmin %s but %s", opsID, bob))},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newLeave()
			if tt.set != nil {
				tt.set(f)
			}
			web, ops := maps.Clone(f.store.projects[webID].members), maps.Clone(f.store.projects[opsID].members)
			outcome{tt.name, tt.want, tt.calls}.check(t, uc.Execute(tt.ctx, tt.project), f)
			if !maps.Equal(f.store.projects[webID].members, web) || !maps.Equal(f.store.projects[opsID].members, ops) {
				t.Errorf("the memberships after the refusal: %v, %v; want them as they were, %v, %v", f.store.projects[webID].members,
					f.store.projects[opsID].members, web, ops)
			}
		})
	}
}

// Every port's failure comes back as itself, the commit's too, after the
// calls before it and none after.
func TestLeaveProjectReturnsEachFailure(t *testing.T) {
	all := left(bob, webID, true)
	fail := func(method string) func(f *writeFixture) {
		return func(f *writeFixture) { f.store.errs = map[string]error{method: errDisk} }
	}
	for _, tt := range []struct {
		name  string
		fail  func(f *writeFixture)
		calls int // how many of the leaving's calls ran
	}{
		{"the project's workspace", fail("ProjectWorkspace"), 2},
		{"the workspace's lock", func(f *writeFixture) { f.workspaces.err = errDisk }, 3},
		{"the project's lock", fail("LockProject"), 4},
		{"the decision", func(f *writeFixture) { f.auth.errs[grantKey{bob, acme.ID}] = errDisk }, 5},
		{"the other admins", fail("HasOtherAdmin"), 6},
		{"the ending", fail("EndMember"), 8},
		{"the commit", func(f *writeFixture) { f.tx.commitErr = errDisk }, 8},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newLeave()
			tt.fail(f)
			outcome{tt.name, errDisk, all[:tt.calls]}.check(t, uc.Execute(as(bob), webID), f)
		})
	}
}
