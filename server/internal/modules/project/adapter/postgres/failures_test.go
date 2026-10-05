package postgresadapter_test

import (
	"context"
	"errors"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// failed reports whether err is a cancelled context's failure as itself:
// no domain problem, which the API would answer in its place, wraps it.
func failed(err error) bool {
	var se *shared.Error
	return errors.Is(err, context.Canceled) && !errors.As(err, &se)
}

// A read that fails answers its error as itself (failed), never a
// plausible answer: not "no such project", which getProject would answer
// as project.not_found and the Authorizer would take for a project no one
// sees; not "no display
// settings", which createProject would take for an empty sidebar; not "no
// project has the identifier", which checkProjectIdentifier would answer
// as available; not an empty list, which listProjects would answer as a
// workspace without projects; not "no project of his", which an ending or
// a demotion would take for nothing to do, nor "not the only admin", which
// would let an ending end memberships rule 2 keeps, nor "no ended
// membership", which reactivate-member would report as none of the
// member's project memberships still ended, nor "no such membership", which
// a write on it would answer as project.member_not_found, nor "no other
// admin", which leaveProject would answer as project.sole_admin; not "no
// such state", which a write on it would answer as project.state_not_found,
// nor no states, which listStates and listWorkspaceStates would answer as
// none, nor "no sequence", which createState would take for the first
// state's, nor an empty group, which a deletion or a move would answer as
// project.state_last_in_group. Each read runs on a cancelled context
// against a project alice is the only admin of, beside bob, a member, and
// has display settings in, with its six default states, bob's membership
// of another project ended, so that the right answer is none of the zero
// values.
func TestAFailedReadIsAnErrorNotAnAnswer(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web := newProject(t, s, acme, "Web", "WEB", alice)
	ctx := context.Background()
	if err := s.CreateMember(ctx, app.MemberRow{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web, MemberID: alice,
		Role: shared.RoleAdmin, CreatedBy: alice, Now: now}); err != nil {
		t.Fatal(err)
	}
	bobs := seedMember(t, pool, acme, web, bob, 15, true)
	seedMember(t, pool, acme, newProject(t, s, acme, "Ops", "OPS", alice), bob, 15, false)
	if err := s.CreatePreferences(ctx, app.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web, UserID: alice,
		SortOrder: 10, CreatedBy: alice, Now: now}); err != nil {
		t.Fatal(err)
	}
	if n, err := s.CountInactive(ctx, acme, bob); err != nil || n != 1 {
		t.Fatalf("CountInactive() of bob = %d, %v; want his ended membership of Ops, 1", n, err)
	}
	todo := seedDefaultStates(t, s, acme, web, alice)["Todo"]
	cancelled, cancel := context.WithCancel(ctx)
	cancel()

	if p, found, err := s.GetProject(cancelled, web, alice); !failed(err) || found || p.ID != (uuid.UUID{}) {
		t.Errorf("GetProject() = %+v, %v, %v; want context.Canceled, not no project", p, found, err)
	}
	if f, found, err := s.ProjectFacts(cancelled, web, alice); !failed(err) || found || f != (app.AccessFacts{}) {
		t.Errorf("ProjectFacts() = %+v, %v, %v; want context.Canceled, not no project", f, found, err)
	}
	if lowest, err := s.LowestSortOrder(cancelled, acme, alice); !failed(err) || lowest != nil {
		t.Errorf("LowestSortOrder() = %s, %v; want context.Canceled, not none", jsonOf(t, lowest), err)
	}
	if taken, err := s.IdentifierTaken(cancelled, acme, "WEB"); !failed(err) || taken {
		t.Errorf("IdentifierTaken() = %v, %v; want context.Canceled, not available", taken, err)
	}
	if list, err := s.ListProjects(cancelled, acme, alice, domain.Visibility{}, false); !failed(err) || list != nil {
		t.Errorf("ListProjects() = %+v, %v; want context.Canceled, not an empty list", list, err)
	}
	if p, found, err := s.LockProject(cancelled, web); !failed(err) || found || p != (app.LockedProject{}) {
		t.Errorf("LockProject() = %+v, %v, %v; want context.Canceled, not no project", p, found, err)
	}
	if w, found, err := s.ProjectWorkspace(cancelled, web); !failed(err) || found || w != (uuid.UUID{}) {
		t.Errorf("ProjectWorkspace() = %s, %v, %v; want context.Canceled, not no project", w, found, err)
	}
	if m, err := s.Memberships(cancelled, web, []uuid.UUID{alice}); !failed(err) || m != nil {
		t.Errorf("Memberships() = %v, %v; want context.Canceled, not none", m, err)
	}
	if ids, err := s.LockActiveMemberProjects(cancelled, []uuid.UUID{acme}, alice); !failed(err) || ids != nil {
		t.Errorf("LockActiveMemberProjects() = %v, %v; want context.Canceled, not no project", ids, err)
	}
	if sole, err := s.SoleAdmin(cancelled, []uuid.UUID{web}, alice); !failed(err) || sole {
		t.Errorf("SoleAdmin() = %v, %v; want context.Canceled, not an answer", sole, err)
	}
	if ids, err := s.LockMemberProjects(cancelled, acme, alice); !failed(err) || ids != nil {
		t.Errorf("LockMemberProjects() = %v, %v; want context.Canceled, not no project", ids, err)
	}
	if n, err := s.CountInactive(cancelled, acme, bob); !failed(err) || n != 0 {
		t.Errorf("CountInactive() = %d, %v; want context.Canceled, not a count", n, err)
	}
	if m, found, err := s.MemberByID(cancelled, bobs); !failed(err) || found || m != (app.ProjectMembership{}) {
		t.Errorf("MemberByID() = %+v, %v, %v; want context.Canceled, not no membership", m, found, err)
	}
	if other, err := s.HasOtherAdmin(cancelled, web, bob); !failed(err) || other {
		t.Errorf("HasOtherAdmin() = %v, %v; want context.Canceled, not an answer", other, err)
	}
	if st, found, err := s.StateByID(cancelled, todo); !failed(err) || found || st != (domain.State{}) {
		t.Errorf("StateByID() = %+v, %v, %v; want context.Canceled, not no state", st, found, err)
	}
	if list, err := s.ListStates(cancelled, web); !failed(err) || list != nil {
		t.Errorf("ListStates() = %+v, %v; want context.Canceled, not none", list, err)
	}
	if greatest, err := s.GreatestSequence(cancelled, web); !failed(err) || greatest != nil {
		t.Errorf("GreatestSequence() = %s, %v; want context.Canceled, not none", jsonOf(t, greatest), err)
	}
	if n, err := s.CountGroupStates(cancelled, web, domain.GroupUnstarted); !failed(err) || n != 0 {
		t.Errorf("CountGroupStates() = %d, %v; want context.Canceled, not a count", n, err)
	}
	if list, err := s.ListWorkspaceStates(cancelled, acme, alice); !failed(err) || list != nil {
		t.Errorf("ListWorkspaceStates() = %+v, %v; want context.Canceled, not none", list, err)
	}
}

// A write that fails answers its error as itself (failed): never nil,
// which a use case would take for done; never a domain problem, such as a
// project's or a state's taken name or identifier; a deletion or a
// marking of a state, never "not written", which the use case would
// answer as project.state_default or project.state_not_found. Each write
// runs on a cancelled context, with values the database would take.
func TestAFailedWriteIsAnError(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web := newProject(t, s, acme, "Web", "WEB", alice)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	if err := s.CreateProject(cancelled, app.ProjectRow{ID: uuid.NewV7(), WorkspaceID: acme, Name: "Ops", Identifier: "OPS",
		Timezone: "UTC", CreatedBy: alice, Now: now}); !failed(err) {
		t.Errorf("CreateProject() = %v; want context.Canceled", err)
	}
	if err := s.CreateMember(cancelled, app.MemberRow{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web, MemberID: alice,
		Role: shared.RoleAdmin, CreatedBy: alice, Now: now}); !failed(err) {
		t.Errorf("CreateMember() = %v; want context.Canceled", err)
	}
	if err := s.CreatePreferences(cancelled, app.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web, UserID: alice,
		SortOrder: 10, CreatedBy: alice, Now: now}); !failed(err) {
		t.Errorf("CreatePreferences() = %v; want context.Canceled", err)
	}
	if err := s.CreateStates(cancelled, []app.StateRow{{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web, CreatedBy: alice, Now: now,
		State: domain.NewState{Name: "Backlog", Color: "#60646C", Group: "backlog", Default: true}}}); !failed(err) {
		t.Errorf("CreateStates() = %v; want context.Canceled", err)
	}
	if err := s.UpdateProject(cancelled, web, domain.ProjectPatch{Identifier: ptr("WEB")}, alice, now); !failed(err) {
		t.Errorf("UpdateProject() = %v; want context.Canceled", err)
	}
	if err := s.SetArchived(cancelled, web, true, alice, now); !failed(err) {
		t.Errorf("SetArchived() = %v; want context.Canceled", err)
	}
	if err := s.RestoreMember(cancelled, uuid.NewV7(), shared.RoleMember, alice, now); !failed(err) {
		t.Errorf("RestoreMember() = %v; want context.Canceled", err)
	}
	if err := s.EndMemberships(cancelled, []uuid.UUID{web}, alice, alice, now); !failed(err) {
		t.Errorf("EndMemberships() = %v; want context.Canceled", err)
	}
	if m, err := s.UpdateMemberRole(cancelled, uuid.NewV7(), shared.RoleMember, alice, now); !failed(err) || m != (domain.Member{}) {
		t.Errorf("UpdateMemberRole() = %+v, %v; want context.Canceled", m, err)
	}
	if err := s.EndMember(cancelled, web, alice, alice, now); !failed(err) {
		t.Errorf("EndMember() = %v; want context.Canceled", err)
	}
	if err := s.DemoteMemberships(cancelled, []uuid.UUID{web}, alice, alice, now); !failed(err) {
		t.Errorf("DemoteMemberships() = %v; want context.Canceled", err)
	}
	if err := s.EnsurePreferences(cancelled, app.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web, UserID: alice,
		SortOrder: 1, CreatedBy: alice, Now: now}); !failed(err) {
		t.Errorf("EnsurePreferences() = %v; want context.Canceled", err)
	}
	if m, err := s.ListMembers(cancelled, web); !failed(err) || m != nil {
		t.Errorf("ListMembers() = %v, %v; want context.Canceled, not none", m, err)
	}
	if p, found, err := s.ShareProject(cancelled, web); !failed(err) || found || p != (app.LockedProject{}) {
		t.Errorf("ShareProject() = %+v, %v, %v; want context.Canceled, not no project", p, found, err)
	}
	if p, found, err := s.Preferences(cancelled, web, alice); !failed(err) || found {
		t.Errorf("Preferences() = %+v, %v, %v; want context.Canceled, not none", p, found, err)
	}
	if _, err := s.UpsertPreferences(cancelled, app.PreferencesChange{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web, UserID: alice,
		Now: now}); !failed(err) {
		t.Errorf("UpsertPreferences() = %v; want context.Canceled", err)
	}
	for _, step := range deletionSteps(s) {
		if err := step.run(cancelled, app.Deletion{WorkspaceID: acme, ProjectID: &web, By: alice, Now: now}); !failed(err) {
			t.Errorf("%s() = %v; want context.Canceled", step.name, err)
		}
	}
	todo := seedDefaultStates(t, s, acme, web, alice)["Todo"]
	if st, err := s.CreateState(cancelled, app.StateRow{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web, CreatedBy: alice, Now: now,
		State: domain.NewState{Name: "Review", Color: "#000", Group: domain.GroupStarted}}); !failed(err) || st != (domain.State{}) {
		t.Errorf("CreateState() = %+v, %v; want context.Canceled", st, err)
	}
	if st, err := s.UpdateState(cancelled, todo, domain.StatePatch{Name: ptr("Next")}, alice, now); !failed(err) || st != (domain.State{}) {
		t.Errorf("UpdateState() = %+v, %v; want context.Canceled", st, err)
	}
	if deleted, err := s.DeleteState(cancelled, todo, alice, now); !failed(err) || deleted {
		t.Errorf("DeleteState() = %v, %v; want context.Canceled", deleted, err)
	}
	if marked, err := s.MarkDefaultState(cancelled, web, todo, alice, now); !failed(err) || marked {
		t.Errorf("MarkDefaultState() = %v, %v; want context.Canceled", marked, err)
	}
}

// Only the two unique keys of a name and an identifier are a 409: another
// unique key (a duplicate id) and a CHECK the domain should have kept (a
// lower-case identifier) are each an internal error, never a domain error.
func TestCreateProjectBreakingAnotherConstraintIsInternal(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web := newProject(t, s, acme, "Web", "WEB", alice)
	for _, tt := range []struct {
		name       string
		id         uuid.UUID
		identifier string
		constraint string
	}{
		{"a duplicate id", web, "OPS", "projects_pkey"},
		{"a lower-case identifier", uuid.NewV7(), "ops", "projects_identifier_check"},
	} {
		err := s.CreateProject(context.Background(), app.ProjectRow{ID: tt.id, WorkspaceID: acme, Name: "Ops", Identifier: tt.identifier,
			Timezone: "UTC", CreatedBy: alice, Now: now})
		var se *shared.Error
		var pgErr *pgconn.PgError
		if errors.As(err, &se) || !errors.As(err, &pgErr) || pgErr.ConstraintName != tt.constraint {
			t.Errorf("%s: CreateProject() = %v; want the violation of %s, not a domain error", tt.name, err, tt.constraint)
		}
	}
}
