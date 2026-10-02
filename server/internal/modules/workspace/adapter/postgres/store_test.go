package postgresadapter_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// now is the fixed clock's time: whole microseconds, as timestamptz stores
// them, so the audit columns read back equal to it (M2 design 3.13).
var now = clocktest.At(time.Date(2026, 9, 29, 10, 0, 0, 123456789, time.UTC)).Now()

func newStore(t *testing.T) (*postgresadapter.Store, *pgxpool.Pool) {
	t.Helper()
	pool, err := postgres.NewPool(context.Background(), config.DatabaseConfig{URL: pgtest.NewDatabase(t), MaxConns: 4})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return postgresadapter.New(pool), pool
}

func exec(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatal(err)
	}
}

// newAccount inserts the users row that the workspace tables reference.
// identity owns the table: the test writes it directly, as a fixture.
func newAccount(t *testing.T, pool *pgxpool.Pool, email string) uuid.UUID {
	t.Helper()
	id := uuid.NewV7()
	exec(t, pool, "INSERT INTO users (id, email, password, display_name) VALUES ($1, $2, 'x', 'x')", id, email)
	return id
}

// newWorkspace creates a workspace named name with slug, and admin as its
// admin, and returns it as stored.
func newWorkspace(t *testing.T, s *postgresadapter.Store, name, slug string, admin uuid.UUID) domain.Workspace {
	t.Helper()
	return newWorkspaceWithID(t, s, uuid.NewV7(), name, slug, admin)
}

// newWorkspaceWithID is newWorkspace with the workspace's id given.
func newWorkspaceWithID(t *testing.T, s *postgresadapter.Store, id uuid.UUID, name, slug string, admin uuid.UUID) domain.Workspace {
	t.Helper()
	w, err := s.CreateWorkspace(context.Background(), app.WorkspaceRow{
		ID: id, Name: name, Slug: slug, Timezone: "UTC", CreatedBy: admin, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	join(t, s, w.ID, admin, shared.RoleAdmin)
	return w
}

// join makes user a member of workspace with role.
func join(t *testing.T, s *postgresadapter.Store, workspace, user uuid.UUID, role shared.Role) {
	t.Helper()
	m := app.MemberRow{ID: uuid.NewV7(), WorkspaceID: workspace, MemberID: user, Role: role, CreatedBy: user, Now: now}
	if err := s.CreateMember(context.Background(), m); err != nil {
		t.Fatal(err)
	}
}

// CreateWorkspace stores the row with the use case's values and answers it
// as stored: the time is the clock's, in UTC, to the microsecond.
func TestCreateWorkspaceStoresTheRow(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	size := "2-10"
	row := app.WorkspaceRow{ID: uuid.NewV7(), Name: "研发部", Slug: "rd", OrganizationSize: &size, Timezone: "Asia/Shanghai", CreatedBy: alice, Now: now}

	got, err := s.CreateWorkspace(context.Background(), row)

	want := domain.Workspace{ID: row.ID, Name: "研发部", Slug: "rd", OrganizationSize: &size, Timezone: "Asia/Shanghai", CreatedAt: now, UpdatedAt: now}
	if err != nil || !sameWorkspace(got, want) || got.CreatedAt.Location() != time.UTC {
		t.Fatalf("CreateWorkspace() = %+v, %v; want %+v", got, err, want)
	}
	var createdBy, updatedBy uuid.UUID
	var created, updated time.Time
	var deleted *time.Time
	if err := pool.QueryRow(context.Background(),
		"SELECT created_by_id, updated_by_id, created_at, updated_at, deleted_at FROM workspaces WHERE id = $1", row.ID).
		Scan(&createdBy, &updatedBy, &created, &updated, &deleted); err != nil {
		t.Fatal(err)
	}
	if createdBy != alice || updatedBy != alice || !created.Equal(now) || !updated.Equal(now) || deleted != nil {
		t.Errorf("row: created_by %s updated_by %s at %v, %v, deleted %v; want alice at %v", createdBy, updatedBy, created, updated, deleted, now)
	}
}

// sameWorkspace compares two workspaces, the organization size by value.
func sameWorkspace(a, b domain.Workspace) bool {
	sizeA, sizeB := a.OrganizationSize, b.OrganizationSize
	a.OrganizationSize, b.OrganizationSize = nil, nil
	return a == b && (sizeA == nil) == (sizeB == nil) && (sizeA == nil || *sizeA == *sizeB)
}

// A slug is unique among undeleted workspaces only: a deleted workspace's
// slug can be used again (M3 design 3.10).
func TestCreateWorkspaceSlugTaken(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	first := newWorkspace(t, s, "Acme", "acme", alice)

	_, err := s.CreateWorkspace(context.Background(), app.WorkspaceRow{ID: uuid.NewV7(), Name: "Other", Slug: "acme", Timezone: "UTC", CreatedBy: alice, Now: now})
	if !errors.Is(err, domain.ErrSlugTaken) {
		t.Fatalf("a taken slug: %v, want workspace.slug_taken", err)
	}
	exec(t, pool, "UPDATE workspaces SET deleted_at = $2 WHERE id = $1", first.ID, now)
	if _, err := s.CreateWorkspace(context.Background(), app.WorkspaceRow{ID: uuid.NewV7(), Name: "Acme 2", Slug: "acme", Timezone: "UTC", CreatedBy: alice, Now: now}); err != nil {
		t.Errorf("the slug of a deleted workspace: %v, want it free", err)
	}
}

func TestCreateMemberStoresTheRow(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	w := newWorkspace(t, s, "Acme", "acme", alice)
	id := uuid.NewV7()

	if err := s.CreateMember(context.Background(), app.MemberRow{ID: id, WorkspaceID: w.ID, MemberID: bob, Role: shared.RoleGuest, CreatedBy: alice, Now: now}); err != nil {
		t.Fatal(err)
	}

	var workspace, member, createdBy, updatedBy uuid.UUID
	var role int
	var active bool
	var created, updated time.Time
	if err := pool.QueryRow(context.Background(),
		"SELECT workspace_id, member_id, role, is_active, created_by_id, updated_by_id, created_at, updated_at FROM workspace_members WHERE id = $1", id).
		Scan(&workspace, &member, &role, &active, &createdBy, &updatedBy, &created, &updated); err != nil {
		t.Fatal(err)
	}
	if workspace != w.ID || member != bob || role != 5 || !active || createdBy != alice || updatedBy != alice || !created.Equal(now) || !updated.Equal(now) {
		t.Errorf("row = %s %s role %d active %v by %s/%s at %v/%v", workspace, member, role, active, createdBy, updatedBy, created, updated)
	}
}

// fixture is three accounts and seven workspaces, in every state for alice:
//   - beta (alice admin, bob member, carol removed), acme (bob admin, alice
//     guest), beta2 (bob removed, alice member; the same name as beta, and a
//     smaller id though created after it, so only the id breaks the tie);
//   - gone (alice admin, deleted), left (alice removed), dropped (alice's
//     row deleted), bobs (bob alone).
type fixture struct {
	s                                            *postgresadapter.Store
	pool                                         *pgxpool.Pool
	alice, bob, carol                            uuid.UUID
	beta, acme, beta2, gone, left, dropped, bobs domain.Workspace
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	s, pool := newStore(t)
	f := fixture{s: s, pool: pool}
	f.alice, f.bob, f.carol = newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	beta2ID := uuid.NewV7() // drawn before beta's: v7 ids increase
	f.beta = newWorkspace(t, s, "Beta", "beta", f.alice)
	join(t, s, f.beta.ID, f.bob, shared.RoleMember)
	join(t, s, f.beta.ID, f.carol, shared.RoleMember)
	exec(t, pool, "UPDATE workspace_members SET is_active = false WHERE workspace_id = $1 AND member_id = $2", f.beta.ID, f.carol)
	f.acme = newWorkspace(t, s, "Acme", "acme", f.bob)
	join(t, s, f.acme.ID, f.alice, shared.RoleGuest)
	f.beta2 = newWorkspaceWithID(t, s, beta2ID, "Beta", "beta-2", f.bob)
	join(t, s, f.beta2.ID, f.alice, shared.RoleMember)
	exec(t, pool, "UPDATE workspace_members SET is_active = false WHERE workspace_id = $1 AND member_id = $2", f.beta2.ID, f.bob)
	f.gone = newWorkspace(t, s, "Gone", "gone", f.alice)
	exec(t, pool, "UPDATE workspaces SET deleted_at = $2 WHERE id = $1", f.gone.ID, now)
	f.left = newWorkspace(t, s, "Left", "left", f.bob)
	join(t, s, f.left.ID, f.alice, shared.RoleAdmin)
	exec(t, pool, "UPDATE workspace_members SET is_active = false WHERE workspace_id = $1 AND member_id = $2", f.left.ID, f.alice)
	f.dropped = newWorkspace(t, s, "Dropped", "dropped", f.bob)
	join(t, s, f.dropped.ID, f.alice, shared.RoleMember)
	exec(t, pool, "UPDATE workspace_members SET deleted_at = $3 WHERE workspace_id = $1 AND member_id = $2", f.dropped.ID, f.alice, now)
	f.bobs = newWorkspace(t, s, "Bob's", "bobs", f.bob)
	return f
}

// ListWorkspaces lists a user's active memberships of undeleted workspaces,
// by name then id, with his role and the active members counted: not another
// account's workspace, not a deleted one, not one he was removed from, not a
// deleted membership (M3 design 3.12).
func TestListWorkspaces(t *testing.T) {
	f := newFixture(t)
	type item struct { // exported fields: fmt prints the ids as UUIDs
		ID      uuid.UUID
		Role    shared.Role
		Members int
	}
	// beta and beta2 have the same name: the smaller id first, beta2's, though
	// beta was created first.
	if f.beta2.ID.Compare(f.beta.ID) >= 0 {
		t.Fatal("the fixture's beta2 must have the smaller id")
	}
	betas := []item{{f.beta2.ID, shared.RoleMember, 1}, {f.beta.ID, shared.RoleAdmin, 2}}
	tests := []struct {
		name string
		user uuid.UUID
		want []item
	}{
		{"alice", f.alice, append([]item{{f.acme.ID, shared.RoleGuest, 2}}, betas...)},
		{"bob", f.bob, []item{{f.acme.ID, shared.RoleAdmin, 2}, {f.beta.ID, shared.RoleMember, 2}, {f.bobs.ID, shared.RoleAdmin, 1},
			{f.dropped.ID, shared.RoleAdmin, 1}, {f.left.ID, shared.RoleAdmin, 1}}},
		{"carol, removed", f.carol, nil},
		{"no account", uuid.NewV7(), nil},
	}
	for _, tt := range tests {
		got, err := f.s.ListWorkspaces(context.Background(), tt.user)
		if err != nil {
			t.Fatal(err)
		}
		var items []item
		for _, w := range got {
			items = append(items, item{w.ID, w.Role, w.TotalMembers})
		}
		if !slices.Equal(items, tt.want) {
			t.Errorf("%s: ListWorkspaces() = %v, want %v", tt.name, items, tt.want)
		}
	}
	// The columns are the stored ones: acme, changed through the store an
	// hour after it was made, has a creation time and a last change's that
	// are two.
	later := now.Add(time.Hour)
	if _, err := f.s.UpdateWorkspace(context.Background(), f.acme.ID, domain.WorkspacePatch{}, f.bob, later); err != nil {
		t.Fatal(err)
	}
	got, err := f.s.ListWorkspaces(context.Background(), f.alice)
	if err != nil || len(got) == 0 {
		t.Fatalf("alice's ListWorkspaces() = %+v, %v; want acme first", got, err)
	}
	if w := got[0]; w.Name != "Acme" || w.Slug != "acme" || w.Timezone != "UTC" || w.OrganizationSize != nil || !w.CreatedAt.Equal(now) ||
		!w.UpdatedAt.Equal(later) {
		t.Errorf("ListWorkspaces()[0] = %+v, want acme as stored, made at %v, changed at %v", w, now, later)
	}
}

// WorkspaceBySlug finds an undeleted workspace as stored, with its active
// members counted: not beta's removed carol, not dropped's deleted row of
// alice. beta, changed through the store an hour after it was made, has a
// creation time and a last change's that are two.
func TestWorkspaceBySlug(t *testing.T) {
	f := newFixture(t)
	later := now.Add(time.Hour)
	if _, err := f.s.UpdateWorkspace(context.Background(), f.beta.ID, domain.WorkspacePatch{}, f.alice, later); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		w       domain.Workspace
		members int
		changed time.Time
	}{{f.beta, 2, later}, {f.dropped, 1, now}} {
		got, err := f.s.WorkspaceBySlug(context.Background(), tt.w.Slug)
		want := tt.w
		want.TotalMembers, want.UpdatedAt = tt.members, tt.changed
		if err != nil || !sameWorkspace(got, want) {
			t.Errorf("WorkspaceBySlug(%s) = %+v, %v; want %+v", tt.w.Slug, got, err, want)
		}
	}
	for _, slug := range []string{"gone", "BETA", "bet", "nothing"} {
		if _, err := f.s.WorkspaceBySlug(context.Background(), slug); !errors.Is(err, app.ErrNotFound) {
			t.Errorf("WorkspaceBySlug(%q) = %v, want app.ErrNotFound", slug, err)
		}
	}
}

func TestSlugTaken(t *testing.T) {
	f := newFixture(t)
	for slug, want := range map[string]bool{"beta": true, "beta-2": true, "gone": false, "BETA": false, "nothing": false} {
		if got, err := f.s.SlugTaken(context.Background(), slug); err != nil || got != want {
			t.Errorf("SlugTaken(%q) = %v, %v; want %v", slug, got, err, want)
		}
	}
}

// ActiveRole is a caller's role in a workspace only while the membership
// counts: every pair of the fixture, each in its own state (M3 design 6.5).
func TestActiveRole(t *testing.T) {
	f := newFixture(t)
	tests := []struct {
		name            string
		workspace, user uuid.UUID
		want            shared.Role // 0: not active
	}{
		{"alice admin of beta", f.beta.ID, f.alice, shared.RoleAdmin},
		{"bob member of beta", f.beta.ID, f.bob, shared.RoleMember},
		{"carol removed from beta", f.beta.ID, f.carol, 0},
		{"alice guest of acme", f.acme.ID, f.alice, shared.RoleGuest},
		{"alice member of beta2", f.beta2.ID, f.alice, shared.RoleMember},
		{"bob removed from beta2", f.beta2.ID, f.bob, 0},
		{"alice admin of the deleted gone", f.gone.ID, f.alice, 0},
		{"alice removed from left", f.left.ID, f.alice, 0},
		{"alice's membership of dropped deleted", f.dropped.ID, f.alice, 0},
		{"alice never in bobs", f.bobs.ID, f.alice, 0},
		{"carol never in acme", f.acme.ID, f.carol, 0},
		{"no workspace", uuid.NewV7(), f.alice, 0},
	}
	for _, tt := range tests {
		role, ok, err := f.s.ActiveRole(context.Background(), tt.workspace, tt.user)
		if err != nil || ok != (tt.want != 0) || role != tt.want {
			t.Errorf("%s: ActiveRole() = %d, %v, %v; want %d", tt.name, role, ok, err, tt.want)
		}
	}
}

// ActiveRole reads in the transaction ctx carries: it sees the membership
// the transaction wrote before committing.
func TestActiveRoleReadsInTheTransaction(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	w := newWorkspace(t, s, "Acme", "acme", alice)
	tx := postgres.NewTxManager(pool, 2*time.Second)
	var inside shared.Role
	err := tx.WithinTx(context.Background(), func(ctx context.Context) error {
		if err := s.CreateMember(ctx, app.MemberRow{ID: uuid.NewV7(), WorkspaceID: w.ID, MemberID: bob, Role: shared.RoleMember, CreatedBy: alice, Now: now}); err != nil {
			return err
		}
		var err error
		inside, _, err = s.ActiveRole(ctx, w.ID, bob)
		return errors.Join(err, errors.New("roll back"))
	})
	if err == nil || inside != shared.RoleMember {
		t.Errorf("in the transaction: role %d, %v; want 15", inside, err)
	}
	if _, ok, err := s.ActiveRole(context.Background(), w.ID, bob); ok || err != nil {
		t.Errorf("after the rollback: %v, %v; want no membership", ok, err)
	}
}
