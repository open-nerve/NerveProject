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
// workspace without projects. Each read runs on a cancelled context
// against a project alice is a member of and has display settings in, so
// that the right answer is none of the zero values.
func TestAFailedReadIsAnErrorNotAnAnswer(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web := newProject(t, s, acme, "Web", "WEB", alice)
	ctx := context.Background()
	if err := s.CreateMember(ctx, app.MemberRow{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web, MemberID: alice,
		Role: shared.RoleAdmin, CreatedBy: alice, Now: now}); err != nil {
		t.Fatal(err)
	}
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
