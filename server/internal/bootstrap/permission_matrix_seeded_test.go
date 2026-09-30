package bootstrap

import (
	"context"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	workspacedomain "github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The rows prepareMatrix seeds, and how a row's request names them.

// matrixMemberships are the memberships prepareMatrix seeds, workspace by
// workspace, each workspace's creator first: acme with its admin, member,
// guest and the member later removed; gone with its admin and the member;
// other, whose admin was never a member of acme and where the removed
// member is still active.
var matrixMemberships = []struct {
	slug string
	c    caller
	role shared.Role
}{
	{"acme", callerAdmin, shared.RoleAdmin}, {"acme", callerMember, shared.RoleMember}, {"acme", callerGuest, shared.RoleGuest},
	{"acme", callerRemoved, shared.RoleMember},
	{"gone", callerDeleted, shared.RoleAdmin}, {"gone", callerMember, shared.RoleMember},
	{"other", callerNever, shared.RoleAdmin}, {"other", callerRemoved, shared.RoleMember},
}

// seeded are the ids of the rows prepareMatrix seeds that a request can
// name: each membership, by the workspace's slug and the column. t is the
// test that asks for them (in).
type seeded struct {
	t           testing.TB
	memberships map[string]uuid.UUID
}

// newSeeded names an id for each of matrixMemberships before prepareMatrix
// writes them, so that matrixViolations, without a database, sees the keys
// and the workspaces the cells will.
func newSeeded() seeded {
	s := seeded{memberships: map[string]uuid.UUID{}}
	for _, m := range matrixMemberships {
		s.memberships[m.slug+"/"+string(m.c)] = uuid.NewV7()
	}
	return s
}

// in is s for the test t, which a key never seeded fails.
func (s seeded) in(t testing.TB) seeded {
	s.t = t
	return s
}

// membership is the id of c's membership of the workspace slug. A key that
// was never seeded fails the test at once: its id would name no row, and an
// outsider's cell would get its 404 whatever the rule.
func (s seeded) membership(slug string, c caller) uuid.UUID {
	id, ok := s.memberships[slug+"/"+string(c)]
	if !ok {
		s.t.Helper()
		s.t.Fatalf("no membership of %s by %s is seeded", slug, c)
	}
	return id
}

// workspaceOfRow is the slug of the workspace the seeded row id is under,
// false for an id no seeded row has.
func (s seeded) workspaceOfRow(id uuid.UUID) (string, bool) {
	for key, seededID := range s.memberships {
		if seededID == id {
			slug, _, _ := strings.Cut(key, "/")
			return slug, true
		}
	}
	return "", false
}

// matrixSeed writes the prepared workspaces, memberships and settings
// through the workspace store, and keeps the workspaces' ids by slug; exec
// runs the SQL that stands in for the store P5 adds.
type matrixSeed struct {
	t          *testing.T
	store      *workspacepg.Store
	ids        map[caller]uuid.UUID
	now        time.Time
	workspaces map[string]uuid.UUID // by slug
}

func (s matrixSeed) workspace(slug string, admin caller) {
	s.t.Helper()
	w, err := s.store.CreateWorkspace(context.Background(), workspaceapp.WorkspaceRow{
		ID: uuid.NewV7(), Name: slug, Slug: slug, Timezone: "UTC", CreatedBy: s.ids[admin], Now: s.now,
	})
	if err != nil {
		s.t.Fatal(err)
	}
	s.workspaces[slug] = w.ID
}

func (s matrixSeed) join(id uuid.UUID, slug string, c caller, role shared.Role) {
	s.t.Helper()
	if err := s.store.CreateMember(context.Background(), workspaceapp.MemberRow{
		ID: id, WorkspaceID: s.workspaces[slug], MemberID: s.ids[c], Role: role, CreatedBy: s.ids[c], Now: s.now,
	}); err != nil {
		s.t.Fatal(err)
	}
}

func (s matrixSeed) preferences(slug string, c caller, p workspacedomain.PreferencesPatch) {
	s.t.Helper()
	if _, err := s.store.UpsertPreferences(context.Background(), workspaceapp.PreferencesRow{
		ID: uuid.NewV7(), WorkspaceID: s.workspaces[slug], UserID: s.ids[c], Patch: p, Now: s.now,
	}); err != nil {
		s.t.Fatal(err)
	}
}

func (s matrixSeed) exec(pool *pgxpool.Pool, sql string, args ...any) {
	s.t.Helper()
	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		s.t.Fatal(err)
	}
}
