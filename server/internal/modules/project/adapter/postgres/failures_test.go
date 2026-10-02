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

// A read that fails answers its error, never a plausible answer: not "no
// such project", which getProject would answer as project.not_found and
// the Authorizer would take for a project no one sees; not "no display
// settings", which createProject would take for an empty sidebar; not "no
// project has the identifier", which checkProjectIdentifier would answer
// as available; not an empty list, which listProjects would answer as a
// workspace without projects; not "no project of his", which an ending or
// a demotion would take for nothing to do, nor "not the only admin", which
// would let an ending end memberships rule 2 keeps. Each read runs on a
// cancelled context against a project alice is the only admin of, beside
// bob, a member, and has display settings in, so that the right answer is
// none of the zero values.
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
	seedMember(t, pool, acme, web, bob, 15, true)
	if err := s.CreatePreferences(ctx, app.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web, UserID: alice,
		SortOrder: 10, CreatedBy: alice, Now: now}); err != nil {
		t.Fatal(err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	failed := func(err error) bool { return errors.Is(err, context.Canceled) }

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
}

// A write that fails answers its error, never nil, which a use case would
// take for done; a project's, never a taken identifier or name. Each write
// runs on a cancelled context, with values the database would take.
func TestAFailedWriteIsAnError(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web := newProject(t, s, acme, "Web", "WEB", alice)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	failed := func(err error) bool { return errors.Is(err, context.Canceled) }

	if err := s.CreateProject(cancelled, app.ProjectRow{ID: uuid.NewV7(), WorkspaceID: acme, Name: "Ops", Identifier: "OPS",
		Timezone: "UTC", CreatedBy: alice, Now: now}); !failed(err) || errors.Is(err, domain.ErrIdentifierTaken) || errors.Is(err, domain.ErrNameTaken) {
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
	if err := s.UpdateProject(cancelled, web, domain.ProjectPatch{Identifier: ptr("WEB")}, alice, now); !failed(err) ||
		errors.Is(err, domain.ErrIdentifierTaken) || errors.Is(err, domain.ErrNameTaken) {
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
