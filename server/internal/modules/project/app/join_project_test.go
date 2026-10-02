package app_test

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// newJoin is JoinProject over newGrowth's fakes, its clock logged, each
// caller deciding as his roles in acme: gina its admin, hank and dave its
// members, carol its guest and Web's.
func newJoin() (*app.JoinProject, *writeFixture) {
	f := newGrowth()
	for user, g := range map[uuid.UUID]shared.Grant{gina: {WorkspaceRole: shared.RoleAdmin}, hank: {WorkspaceRole: shared.RoleMember},
		dave: {WorkspaceRole: shared.RoleMember}, carol: {WorkspaceRole: shared.RoleGuest, ProjectRole: shared.RoleGuest}} {
		f.auth.grants[grantKey{user, acme.ID}] = g
	}
	return app.NewJoinProject(f.locks(), f.store, f.tx, clockAt{clockNow, f.log}), f
}

// beforeJoin are the calls of user's joining project before it writes:
// the transaction, the project's workspace, its lock, his workspace
// membership FOR SHARE, the project's lock, the decision, his membership
// of the project.
func beforeJoin(user, project uuid.UUID) []string {
	return append(lockedTo(project), fmt.Sprintf("ShareMembers %s [%s]", acme.ID, user), "LockProject "+project.String(),
		fmt.Sprintf("Authorize %s %s on %s/%s", user, domain.ActionJoin, acme.ID, project), fmt.Sprintf("Memberships %s [%s]", project, user))
}

// JoinProject, in one transaction and in the order of M3 design 3.6, reads
// the project's workspace, locks it, the caller's membership of it and the
// project, decides, reads his membership of the project and the clock,
// then makes his membership with his workspace role, or restores his ended
// one with the lesser of its role and his workspace role, by the order of
// the roles (9.1's table; its fourth row, a workspace guest, is refused
// before, TestJoinProjectRefuses), and makes his display settings at
// 65535 (3.18). The answer is the project as he sees it, his role in it.
func TestJoinProject(t *testing.T) {
	tests := []struct {
		name      string
		user      uuid.UUID
		ended     *shared.Role // his ended membership's role; nil: none
		workspace shared.Role  // his workspace role
		want      shared.Role
	}{
		{"a workspace member", hank, nil, shared.RoleMember, shared.RoleMember},
		{"a workspace admin", gina, nil, shared.RoleAdmin, shared.RoleAdmin},
		{"a guest before, a workspace member", dave, ptr(shared.RoleGuest), shared.RoleMember, shared.RoleGuest},
		{"an admin before, a workspace member", dave, ptr(shared.RoleAdmin), shared.RoleMember, shared.RoleMember},
		{"a member before, a workspace admin", dave, ptr(shared.RoleMember), shared.RoleAdmin, shared.RoleMember},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newJoin()
			f.auth.grants[grantKey{tt.user, acme.ID}] = shared.Grant{WorkspaceRole: tt.workspace}
			var ended *app.Membership
			if tt.ended != nil {
				m := f.store.projects[webID].members[tt.user]
				m.Role = *tt.ended
				f.store.projects[webID].members[tt.user] = m
				ended = &m
			}

			got, err := uc.Execute(as(tt.user), webID)

			want := slices.Concat(beforeJoin(tt.user, webID), []string{"Now"}, grown(tt.user, ended, tt.want, 65535, tt.user),
				[]string{fmt.Sprintf("GetProject %s for %s", webID, tt.user)})
			if err != nil || !slices.Equal(f.log.calls, want) {
				t.Fatalf("Execute() = %v, calls\n%q\nwant\n%q", err, f.log.calls, want)
			}
			if got.ID != webID || got.MemberRole == nil || *got.MemberRole != tt.want {
				t.Errorf("Execute() = %+v, want Web with his role %d", got, tt.want)
			}
		})
	}
}

// An active member of the project joining is left as he is: nothing is
// written, and the answer is the project as he sees it.
func TestJoinProjectLeavesAnActiveMemberAsHeIs(t *testing.T) {
	uc, f := newJoin()
	before := maps.Clone(f.store.projects[webID].members)
	got, err := uc.Execute(as(bob), webID)
	want := append(beforeJoin(bob, webID), fmt.Sprintf("GetProject %s for %s", webID, bob))
	if err != nil || !slices.Equal(f.log.calls, want) || got.MemberRole == nil || *got.MemberRole != shared.RoleAdmin {
		t.Errorf("Execute() = %+v, %v, calls %q; want Web, bob its admin, calls %q", got, err, f.log.calls, want)
	}
	if !maps.Equal(f.store.projects[webID].members, before) {
		t.Errorf("the memberships: %v, want them as they were", f.store.projects[webID].members)
	}
}

// Refusals, each in its place, and nothing written: no caller before the
// transaction; a project not there at its workspace; one not visible at
// the decision; a workspace guest, carol, who is the project's member,
// forbidden after the decision and before his membership is read (M3
// design 3.5, 9.2's PG cell), as is a workspace role outside the three,
// above them: the roles that may join are a set.
func TestJoinProjectRefuses(t *testing.T) {
	tests := []struct {
		name  string
		ctx   context.Context
		id    uuid.UUID
		setup func(f *writeFixture)
		want  error
		calls []string
	}{
		{"no caller", context.Background(), webID, nil, shared.Unauthenticated(), nil},
		{"no project", as(hank), uuid.Nil(), nil, domain.ErrNotFound, noProject},
		{"not seen", as(erin), webID, nil, domain.ErrNotFound, beforeJoin(erin, webID)[:6]},
		{"a workspace guest, the project's member", as(carol), webID, nil, shared.Forbidden(), beforeJoin(carol, webID)[:6]},
		{"a workspace role above the three", as(erin), webID, func(f *writeFixture) {
			f.auth.grants[grantKey{erin, acme.ID}] = shared.Grant{WorkspaceRole: 25}
		}, shared.Forbidden(), beforeJoin(erin, webID)[:6]},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newJoin()
			if tt.setup != nil {
				tt.setup(f)
			}
			before := maps.Clone(f.store.projects[webID].members)
			got, err := uc.Execute(tt.ctx, tt.id)
			if !sameError(err, tt.want) || got.ID != uuid.Nil() || !slices.Equal(f.log.calls, tt.calls) {
				t.Errorf("Execute() = %+v, %v, calls %q; want %v, calls %q", got, err, f.log.calls, tt.want, tt.calls)
			}
			if !maps.Equal(f.store.projects[webID].members, before) {
				t.Errorf("the memberships after the refusal: %v, want them as they were", f.store.projects[webID].members)
			}
		})
	}
}

// An archived project is joined as any other (M3 design 3.19).
func TestJoinProjectJoinsAnArchivedProject(t *testing.T) {
	uc, f := newJoin()
	got, err := uc.Execute(as(hank), opsID)
	if err != nil || got.ID != opsID || got.MemberRole == nil || *got.MemberRole != shared.RoleMember || !f.store.projects[opsID].members[hank].Active {
		t.Errorf("Execute() on ops = %+v, %v; want hank its member", got, err)
	}
}

// Every port's failure comes back as itself, the commit's too, after the
// calls before it and none after; a project the store cannot read after
// the write is an internal error. hank, a workspace member, joins anew;
// dave, one too, restores his ended membership as a member's.
func TestJoinProjectReturnsEachFailure(t *testing.T) {
	all := func(f *writeFixture, user uuid.UUID) []string {
		var ended *app.Membership
		if m, ok := f.store.projects[webID].members[user]; ok {
			ended = &m
		}
		return slices.Concat(beforeJoin(user, webID), []string{"Now"}, grown(user, ended, shared.RoleMember, 65535, user),
			[]string{fmt.Sprintf("GetProject %s for %s", webID, user)})
	}
	fail := func(method string) func(f *writeFixture) {
		return func(f *writeFixture) { f.store.errs = map[string]error{method: errDisk} }
	}
	tests := []struct {
		name  string
		user  uuid.UUID
		fail  func(f *writeFixture)
		calls int // how many of the calls ran
	}{
		{"the project's workspace", hank, fail("ProjectWorkspace"), 2},
		{"the workspace's lock", hank, func(f *writeFixture) { f.workspaces.err = errDisk }, 3},
		{"his membership's lock", hank, func(f *writeFixture) { f.members.err = errDisk }, 4},
		{"the project's lock", hank, fail("LockProject"), 5},
		{"the decision", hank, func(f *writeFixture) { f.auth.errs[grantKey{hank, acme.ID}] = errDisk }, 6},
		{"his membership", hank, fail("Memberships"), 7},
		{"a new membership", hank, fail("CreateMember"), 9},
		{"the display settings", hank, fail("EnsurePreferences"), 10},
		{"a restored membership", dave, fail("RestoreMember"), 9},
		{"the answer", hank, fail("GetProject"), 11},
		{"the commit", hank, func(f *writeFixture) { f.tx.commitErr = errDisk }, 11},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newJoin()
			tt.fail(f)
			want := all(f, tt.user)[:tt.calls]
			got, err := uc.Execute(as(tt.user), webID)
			if !errors.Is(err, errDisk) || got.ID != uuid.Nil() || !slices.Equal(f.log.calls, want) {
				t.Errorf("Execute() = %+v, %v, calls %q; want %v after %q", got, err, f.log.calls, errDisk, want)
			}
		})
	}
	uc, f := newJoin()
	f.store.missing = true
	var e *shared.Error
	if got, err := uc.Execute(as(hank), webID); err == nil || errors.As(err, &e) || got.ID != uuid.Nil() {
		t.Errorf("Execute() with the project not read back = %+v, %v; want an internal error", got, err)
	}
}
