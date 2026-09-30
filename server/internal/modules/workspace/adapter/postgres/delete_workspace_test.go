package postgresadapter_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// deletion is a row's audit columns after a deletion, as the tests read
// them.
type deletion struct {
	deletedAt *time.Time
	updatedAt time.Time
	updatedBy *uuid.UUID
}

// deletions reads the audit columns of the rows sql selects (id first),
// keyed by id.
func deletions(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) map[uuid.UUID]deletion {
	t.Helper()
	rows, err := pool.Query(context.Background(), sql, args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[uuid.UUID]deletion{}
	for rows.Next() {
		var id uuid.UUID
		var d deletion
		if err := rows.Scan(&id, &d.deletedAt, &d.updatedAt, &d.updatedBy); err != nil {
			t.Fatal(err)
		}
		out[id] = d
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// deletedAtBy reports whether d was deleted at when by by.
func (d deletion) deletedAtBy(when time.Time, by uuid.UUID) bool {
	return d.deletedAt != nil && d.deletedAt.Equal(when) && d.updatedAt.Equal(when) && d.updatedBy != nil && *d.updatedBy == by
}

// The three steps, in one transaction, soft-delete the workspace, every
// membership of it, active or not, and every member's settings in it, at
// the same moment and by the same account; a membership and a settings row
// deleted before keep their time, and another workspace keeps everything.
// Running the steps again changes nothing, nobody's last_workspace_id is
// cleared, and the slug is free again.
func TestDeletingAWorkspaceSoftDeletesItsRows(t *testing.T) {
	s, pool := newStore(t)
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	dave := newAccount(t, pool, "dave@corp.com")
	acme, beta := newWorkspace(t, s, "Acme", "acme", alice), newWorkspace(t, s, "Beta", "beta", alice)
	join(t, s, acme.ID, bob, shared.RoleMember)
	join(t, s, acme.ID, carol, shared.RoleGuest)
	join(t, s, acme.ID, dave, shared.RoleMember)
	exec(t, pool, "UPDATE workspace_members SET is_active = false WHERE member_id = $1", carol)
	join(t, s, beta.ID, bob, shared.RoleMember)
	for _, r := range []app.PreferencesRow{
		{ID: uuid.NewV7(), WorkspaceID: acme.ID, UserID: alice, Now: now}, {ID: uuid.NewV7(), WorkspaceID: acme.ID, UserID: bob, Now: now},
		{ID: uuid.NewV7(), WorkspaceID: acme.ID, UserID: carol, Now: now}, {ID: uuid.NewV7(), WorkspaceID: beta.ID, UserID: alice, Now: now},
	} {
		upsert(t, s, r)
	}
	earlier := now.Add(-time.Hour)
	exec(t, pool, "UPDATE workspace_user_properties SET deleted_at = $1 WHERE user_id = $2", earlier, carol)
	exec(t, pool, "UPDATE workspace_members SET deleted_at = $1 WHERE member_id = $2", earlier, dave)
	later := now.Add(time.Hour)
	// alice and bob last landed in acme: the deletion leaves it (M3 design
	// 3.14), the landing rule skips a workspace the caller cannot list.
	exec(t, pool, "INSERT INTO profiles (id, user_id, last_workspace_id) VALUES ($1, $2, $3), ($4, $5, $3)",
		uuid.NewV7(), alice, acme.ID, uuid.NewV7(), bob)

	steps := []func(ctx context.Context, id, by uuid.UUID, now time.Time) error{
		s.DeleteWorkspace, s.DeleteWorkspaceMembers, s.DeleteWorkspacePreferences,
	}
	err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
		for _, step := range steps {
			if err := step(ctx, acme.ID, bob, later); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// Once more, by alice an hour later: nothing is left undeleted, so every
	// row keeps bob's deletion.
	for _, step := range steps {
		if err := step(context.Background(), acme.ID, alice, later.Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
	}

	for what, rows := range map[string]map[uuid.UUID]deletion{
		"acme": deletions(t, pool, "SELECT id, deleted_at, updated_at, updated_by_id FROM workspaces WHERE id = $1", acme.ID),
		"members": deletions(t, pool, `SELECT id, deleted_at, updated_at, updated_by_id FROM workspace_members
			WHERE workspace_id = $1 AND member_id <> $2`, acme.ID, dave),
		"settings": deletions(t, pool, `SELECT id, deleted_at, updated_at, updated_by_id FROM workspace_user_properties
			WHERE workspace_id = $1 AND user_id <> $2`, acme.ID, carol),
	} {
		want := map[string]int{"acme": 1, "members": 3, "settings": 2}[what]
		if len(rows) != want {
			t.Errorf("%s: %d rows, want %d", what, len(rows), want)
		}
		for id, d := range rows {
			if !d.deletedAtBy(later, bob) {
				t.Errorf("%s %s: %+v, want deleted at %v by bob", what, id, d, later)
			}
		}
	}
	var active []bool
	if err := pool.QueryRow(context.Background(), `SELECT array_agg(is_active ORDER BY is_active) FROM workspace_members
		WHERE workspace_id = $1 AND member_id <> $2`, acme.ID, dave).Scan(&active); err != nil || len(active) != 3 || active[0] || !active[1] || !active[2] {
		t.Errorf("is_active of acme's members: %v, %v; want false, true, true as they were", active, err)
	}
	var carolDeleted, daveDeleted time.Time
	if err := pool.QueryRow(context.Background(), `SELECT (SELECT deleted_at FROM workspace_user_properties WHERE user_id = $1),
		(SELECT deleted_at FROM workspace_members WHERE member_id = $2)`, carol, dave).
		Scan(&carolDeleted, &daveDeleted); err != nil || !carolDeleted.Equal(earlier) || !daveDeleted.Equal(earlier) {
		t.Errorf("carol's settings deleted at %v, dave's membership at %v, %v; want %v, as before", carolDeleted, daveDeleted, err, earlier)
	}
	betaRows := deletions(t, pool, `
		SELECT id, deleted_at, updated_at, updated_by_id FROM workspaces WHERE id = $1
		UNION ALL SELECT id, deleted_at, updated_at, updated_by_id FROM workspace_members WHERE workspace_id = $1
		UNION ALL SELECT id, deleted_at, updated_at, updated_by_id FROM workspace_user_properties WHERE workspace_id = $1`, beta.ID)
	if len(betaRows) != 4 {
		t.Errorf("beta's rows: %d, want 4: the workspace, alice's and bob's memberships, alice's settings", len(betaRows))
	}
	for id, d := range betaRows {
		if d.deletedAt != nil || !d.updatedAt.Equal(now) {
			t.Errorf("beta's row %s: %+v, want it untouched", id, d)
		}
	}
	var landing int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM profiles WHERE last_workspace_id = $1", acme.ID).
		Scan(&landing); err != nil || landing != 2 {
		t.Errorf("profiles still landing in acme: %d, %v; want alice's and bob's, as before", landing, err)
	}
	if _, err := s.WorkspaceBySlug(context.Background(), "acme"); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("WorkspaceBySlug(acme) after the deletion: %v, want app.ErrNotFound", err)
	}
	newWorkspace(t, s, "Acme again", "acme", bob)
}

// failingSettings is the store with its last deletion step failing.
type failingSettings struct {
	*postgresadapter.Store
}

var errDiskFull = errors.New("disk full")

func (failingSettings) DeleteWorkspacePreferences(context.Context, uuid.UUID, uuid.UUID, time.Time) error {
	return errDiskFull
}

// allowAll allows every action, as the workspace's admin.
type allowAll struct{}

func (allowAll) Authorize(context.Context, shared.Actor, shared.Action, shared.Target) (shared.Grant, error) {
	return shared.Grant{WorkspaceRole: shared.RoleAdmin}, nil
}

// The use case's cascade is one transaction on the database: when its last
// step fails, the workspace and its members are not deleted either.
func TestAFailedDeletionLeavesTheWorkspace(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme := newWorkspace(t, s, "Acme", "acme", alice)
	join(t, s, acme.ID, bob, shared.RoleMember)
	uc := app.NewDeleteWorkspace(failingSettings{s}, allowAll{}, postgres.NewTxManager(pool, 2*time.Second), clocktest.At(now),
		slog.New(slog.DiscardHandler))

	err := uc.Execute(shared.WithActor(context.Background(), shared.Actor{UserID: alice}), "acme")

	if !errors.Is(err, errDiskFull) {
		t.Fatalf("Execute() = %v, want %v", err, errDiskFull)
	}
	if w, err := s.WorkspaceBySlug(context.Background(), "acme"); err != nil || w.TotalMembers != 2 {
		t.Errorf("acme after the failure: %+v, %v; want it there with its 2 members", w, err)
	}
	if got := deletions(t, pool, "SELECT id, deleted_at, updated_at, updated_by_id FROM workspace_members WHERE deleted_at IS NOT NULL"); len(got) != 0 {
		t.Errorf("deleted memberships after the failure: %+v, want none", got)
	}
}
