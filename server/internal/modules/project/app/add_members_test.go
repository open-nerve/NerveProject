package app_test

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// newAdd is AddProjectMembers over newGrowth's fakes, its clock logged.
func newAdd() (*app.AddProjectMembers, *writeFixture) {
	f := newGrowth()
	return app.NewAddProjectMembers(app.AddMembersDeps{Locks: f.locks(), Projects: f.store, Tx: f.tx, Clock: clockAt{clockNow, f.log}}), f
}

// beforeTargets are the calls of user's addition of the accounts of in to
// project, before the targets' check: the transaction, the project's
// workspace, its lock, the targets' workspace memberships FOR SHARE, the
// project's lock, the decision, the targets' memberships of the project.
func beforeTargets(user, project uuid.UUID, in []domain.NewMember) []string {
	ids := make([]uuid.UUID, len(in))
	for i, m := range in {
		ids[i] = m.MemberID
	}
	return append(lockedTo(project), fmt.Sprintf("ShareMembers %s %v", acme.ID, ids), "LockProject "+project.String(),
		fmt.Sprintf("Authorize %s %s on %s/%s", user, domain.ActionMemberAdd, acme.ID, project), fmt.Sprintf("Memberships %s %v", project, ids))
}

// AddProjectMembers, in one transaction and in the order of M3 design 3.6,
// reads the project's workspace, locks it, the targets' memberships of it
// and the project, decides, reads the targets' memberships of the project,
// checks them, reads the clock, then for each target in the request's
// order reads his least place, restores his ended membership with the role
// asked for (dave, once a member, now an admin) or makes one, and makes
// his display settings at 10000 before his least place, or at the default
// (3.18); the answer is the targets as stored, in the request's order: a
// restored membership keeps its id.
func TestAddProjectMembers(t *testing.T) {
	uc, f := newAdd()
	in := []domain.NewMember{{MemberID: hank, Role: shared.RoleMember}, {MemberID: dave, Role: shared.RoleAdmin},
		{MemberID: gina, Role: shared.RoleAdmin}, {MemberID: ivy, Role: shared.RoleGuest}}
	daves := f.store.projects[webID].members[dave]

	got, err := uc.Execute(as(bob), webID, in)

	want := append(beforeTargets(bob, webID, in), "Now", "LowestSortOrder "+acme.ID.String()+" "+hank.String())
	want = append(want, grown(hank, nil, shared.RoleMember, -10005, bob)...)
	want = append(want, "LowestSortOrder "+acme.ID.String()+" "+dave.String())
	want = append(want, grown(dave, &daves, shared.RoleAdmin, 65535, bob)...)
	want = append(want, "LowestSortOrder "+acme.ID.String()+" "+gina.String())
	want = append(want, grown(gina, nil, shared.RoleAdmin, 65535, bob)...)
	want = append(want, "LowestSortOrder "+acme.ID.String()+" "+ivy.String())
	want = append(want, grown(ivy, nil, shared.RoleGuest, 65535, bob)...)
	want = append(want, "ListMembers "+webID.String())
	if err != nil || !slices.Equal(f.log.calls, want) {
		t.Fatalf("Execute() = %v, calls\n%q\nwant\n%q", err, f.log.calls, want)
	}
	members := f.store.projects[webID].members
	var answer []domain.Member
	for _, m := range in {
		answer = append(answer, domain.Member{ID: members[m.MemberID].ID, ProjectID: webID, MemberID: m.MemberID, Role: m.Role, CreatedAt: now})
	}
	if !reflect.DeepEqual(got, answer) || got[1].ID != daves.ID {
		t.Errorf("Execute() = %+v\nwant %+v, dave's membership %s restored", got, answer, daves.ID)
	}
}

// Refusals, each in its place, and nothing written: the request's values,
// and no caller, before the transaction; a project not there: 404 at its
// workspace; one not visible, or a member's 403, at the decision, whatever
// the targets (the matrix's PM and X cells, M3 design 9.2); each target
// refused after the decision, all of them in one 422 by their places: erin
// is no member of the workspace, alice is a member of the project already,
// gina, a workspace admin, is asked for as a member, ivy, a workspace
// guest, as a member; hank as an admin passes.
func TestAddProjectMembersRefuses(t *testing.T) {
	erins := []domain.NewMember{{MemberID: erin, Role: shared.RoleMember}}
	mixed := []domain.NewMember{{MemberID: erin, Role: shared.RoleMember}, {MemberID: alice, Role: shared.RoleMember},
		{MemberID: gina, Role: shared.RoleMember}, {MemberID: ivy, Role: shared.RoleMember}, {MemberID: hank, Role: shared.RoleAdmin}}
	tests := []struct {
		name  string
		ctx   context.Context
		id    uuid.UUID
		in    []domain.NewMember
		want  error
		calls []string
	}{
		{"none", as(bob), webID, nil, shared.Invalid(shared.FieldError{Field: "members", Code: "too_short"}), nil},
		{"no caller", context.Background(), webID, erins, shared.Unauthenticated(), nil},
		{"no project", as(bob), uuid.Nil(), erins, domain.ErrNotFound, noProject},
		{"not seen, an invalid target", as(erin), webID, erins, domain.ErrNotFound, beforeTargets(erin, webID, erins)[:6]},
		{"a project member, an invalid target", as(alice), webID, erins, shared.Forbidden(), beforeTargets(alice, webID, erins)[:6]},
		{"the targets", as(bob), webID, mixed, shared.Invalid(
			shared.FieldError{Field: "members[0].member_id", Code: "not_allowed"}, shared.FieldError{Field: "members[1].member_id", Code: "duplicate"},
			shared.FieldError{Field: "members[2].role", Code: "not_allowed"}, shared.FieldError{Field: "members[3].role", Code: "not_allowed"}),
			beforeTargets(bob, webID, mixed)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newAdd()
			before := maps.Clone(f.store.projects[webID].members)
			got, err := uc.Execute(tt.ctx, tt.id, tt.in)
			if !sameError(err, tt.want) || got != nil || !slices.Equal(f.log.calls, tt.calls) {
				t.Errorf("Execute() = %+v, %v, calls %q; want %v, calls %q", got, err, f.log.calls, tt.want, tt.calls)
			}
			if !maps.Equal(f.store.projects[webID].members, before) {
				t.Errorf("the memberships after the refusal: %v, want them as they were", f.store.projects[webID].members)
			}
		})
	}
}

// An archived project's members are added as any other's (M3 design 3.19).
// hank's ended membership of ops, an admin's, is restored with the role
// asked for, a guest's, keeping its id: the grantor decides (convention 6),
// down as well as up (TestAddProjectMembers restores dave up).
func TestAddProjectMembersToAnArchivedProject(t *testing.T) {
	uc, f := newAdd()
	hanks := app.Membership{ID: uuid.NewV7(), Role: shared.RoleAdmin}
	f.store.projects[opsID].members[hank] = hanks
	got, err := uc.Execute(as(bob), opsID, []domain.NewMember{{MemberID: hank, Role: shared.RoleGuest}})
	if err != nil || len(got) != 1 || got[0].ID != hanks.ID || got[0].MemberID != hank || got[0].ProjectID != opsID || got[0].Role != shared.RoleGuest ||
		f.store.projects[opsID].members[hank] != (app.Membership{ID: hanks.ID, Role: shared.RoleGuest, Active: true}) {
		t.Errorf("Execute() on ops = %+v, %v; want hank's ended membership %s restored as its guest", got, err, hanks.ID)
	}
}

// Every port's failure comes back as itself, the commit's too, after the
// calls before it and none after; a membership the store cannot list after
// its write is an internal error.
func TestAddProjectMembersReturnsEachFailure(t *testing.T) {
	in := []domain.NewMember{{MemberID: hank, Role: shared.RoleMember}, {MemberID: dave, Role: shared.RoleMember}}
	fail := func(method string) func(f *writeFixture) {
		return func(f *writeFixture) { f.store.errs = map[string]error{method: errDisk} }
	}
	tests := []struct {
		name  string
		fail  func(f *writeFixture)
		calls int // how many of the calls ran
	}{
		{"the project's workspace", fail("ProjectWorkspace"), 2},
		{"the workspace's lock", func(f *writeFixture) { f.workspaces.err = errDisk }, 3},
		{"the targets' lock", func(f *writeFixture) { f.members.err = errDisk }, 4},
		{"the project's lock", fail("LockProject"), 5},
		{"the decision", func(f *writeFixture) { f.auth.errs[grantKey{bob, acme.ID}] = errDisk }, 6},
		{"the memberships", fail("Memberships"), 7},
		{"the least place", fail("LowestSortOrder"), 9},
		{"a new membership", fail("CreateMember"), 10},
		{"the display settings", fail("EnsurePreferences"), 11},
		{"a restored membership", fail("RestoreMember"), 13},
		{"the answer", fail("ListMembers"), 15},
		{"the commit", func(f *writeFixture) { f.tx.commitErr = errDisk }, 15},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newAdd()
			tt.fail(f)
			daves := f.store.projects[webID].members[dave]
			all := slices.Concat(beforeTargets(bob, webID, in), []string{"Now", "LowestSortOrder " + acme.ID.String() + " " + hank.String()},
				grown(hank, nil, shared.RoleMember, -10005, bob), []string{"LowestSortOrder " + acme.ID.String() + " " + dave.String()},
				grown(dave, &daves, shared.RoleMember, 65535, bob), []string{"ListMembers " + webID.String()})
			got, err := uc.Execute(as(bob), webID, in)
			if !errors.Is(err, errDisk) || got != nil || !slices.Equal(f.log.calls, all[:tt.calls]) {
				t.Errorf("Execute() = %+v, %v, calls %q; want %v after %q", got, err, f.log.calls, errDisk, all[:tt.calls])
			}
		})
	}
	uc, f := newAdd()
	f.store.missing = true
	var e *shared.Error
	if got, err := uc.Execute(as(bob), webID, in); err == nil || errors.As(err, &e) || got != nil {
		t.Errorf("Execute() with the memberships not listed = %+v, %v; want an internal error", got, err)
	}
}
