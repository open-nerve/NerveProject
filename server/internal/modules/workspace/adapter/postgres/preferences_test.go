package postgresadapter_test

import (
	"context"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// prefsRow is a stored row of workspace_user_properties, as the tests read it.
type prefsRow struct {
	id                   uuid.UUID
	mode                 string
	limit                int
	createdBy, updatedBy uuid.UUID
	createdAt, updatedAt time.Time
}

// undeletedPrefs reads the undeleted rows of user in workspace.
func undeletedPrefs(t *testing.T, pool *pgxpool.Pool, workspace, user uuid.UUID) []prefsRow {
	t.Helper()
	rows, err := pool.Query(context.Background(), `
		SELECT id, navigation_control_preference, navigation_project_limit, created_by_id, updated_by_id, created_at, updated_at
		FROM workspace_user_properties WHERE workspace_id = $1 AND user_id = $2 AND deleted_at IS NULL`, workspace, user)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []prefsRow
	for rows.Next() {
		var r prefsRow
		if err := rows.Scan(&r.id, &r.mode, &r.limit, &r.createdBy, &r.updatedBy, &r.createdAt, &r.updatedAt); err != nil {
			t.Fatal(err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func upsert(t *testing.T, s *postgresadapter.Store, r app.PreferencesRow) domain.Preferences {
	t.Helper()
	p, err := s.UpsertPreferences(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// domain.DefaultPreferences are the columns' defaults: a row inserted
// without the two columns reads as them.
func TestDefaultPreferencesAreTheColumnDefaults(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, s, "Acme", "acme", alice)
	exec(t, pool, "INSERT INTO workspace_user_properties (id, workspace_id, user_id) VALUES ($1, $2, $3)", uuid.NewV7(), acme.ID, alice)
	if p, found, err := s.Preferences(context.Background(), acme.ID, alice); err != nil || !found || p != domain.DefaultPreferences() {
		t.Errorf("Preferences() = %+v, %v, %v; want the defaults %+v", p, found, err, domain.DefaultPreferences())
	}
}

// The first change inserts the account's row: the defaults with the change
// applied, by the account, at the clock's time. Later ones change the fields
// they set on the same row, and only its updated_at and updated_by_id
// besides; an empty change leaves the values. Each account and workspace
// has its own row.
func TestUpsertPreferences(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme, beta := newWorkspace(t, s, "Acme", "acme", alice), newWorkspace(t, s, "Beta", "beta", alice)
	first := uuid.NewV7()
	hour := func(n time.Duration) time.Time { return now.Add(n * time.Hour) }

	steps := []struct {
		r    app.PreferencesRow
		want domain.Preferences
		at   time.Time
	}{
		{app.PreferencesRow{ID: first, WorkspaceID: acme.ID, UserID: alice, Patch: domain.PreferencesPatch{NavigationProjectLimit: ptr(3)}, Now: now},
			prefs("ACCORDION", 3), now},
		{app.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: acme.ID, UserID: alice, Patch: domain.PreferencesPatch{NavigationControl: ptr("TABBED")}, Now: hour(1)},
			prefs("TABBED", 3), hour(1)},
		{app.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: acme.ID, UserID: alice, Patch: domain.PreferencesPatch{NavigationProjectLimit: ptr(0)}, Now: hour(2)},
			prefs("TABBED", 0), hour(2)},
		{app.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: acme.ID, UserID: alice, Now: hour(3)}, prefs("TABBED", 0), hour(3)},
	}
	for i, step := range steps {
		if got := upsert(t, s, step.r); got != step.want {
			t.Errorf("step %d: UpsertPreferences() = %+v, want %+v", i, got, step.want)
		}
		rows := undeletedPrefs(t, pool, acme.ID, alice)
		want := prefsRow{first, step.want.NavigationControl, step.want.NavigationProjectLimit, alice, alice, now, step.at}
		if len(rows) != 1 || rows[0].id != want.id || rows[0].mode != want.mode || rows[0].limit != want.limit || rows[0].createdBy != alice ||
			rows[0].updatedBy != alice || !rows[0].createdAt.Equal(now) || !rows[0].updatedAt.Equal(step.at) {
			t.Errorf("step %d: rows %+v, want one: %+v", i, rows, want)
		}
	}

	if got := upsert(t, s, app.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: acme.ID, UserID: bob,
		Patch: domain.PreferencesPatch{NavigationProjectLimit: ptr(0)}, Now: now}); got != prefs("ACCORDION", 0) {
		t.Errorf("bob's first change = %+v, want ACCORDION, 0", got)
	}
	if got := upsert(t, s, app.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: beta.ID, UserID: alice, Now: now}); got != domain.DefaultPreferences() {
		t.Errorf("alice's first change in beta = %+v, want the defaults", got)
	}
	if p, _, err := s.Preferences(context.Background(), acme.ID, alice); err != nil || p != prefs("TABBED", 0) {
		t.Errorf("alice's settings in acme after bob's and beta's = %+v, %v; want TABBED, 0", p, err)
	}
}

// A deleted row is no conflict (the partial unique index): the next change
// inserts a new row from the defaults, and it is the one read.
func TestUpsertPreferencesAfterTheRowIsDeleted(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, s, "Acme", "acme", alice)
	upsert(t, s, app.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: acme.ID, UserID: alice,
		Patch: domain.PreferencesPatch{NavigationControl: ptr("TABBED"), NavigationProjectLimit: ptr(3)}, Now: now})
	exec(t, pool, "UPDATE workspace_user_properties SET deleted_at = $1", now)
	if _, found, err := s.Preferences(context.Background(), acme.ID, alice); err != nil || found {
		t.Fatalf("Preferences() of a deleted row: found %v, %v; want none", found, err)
	}

	second := uuid.NewV7()
	if got := upsert(t, s, app.PreferencesRow{ID: second, WorkspaceID: acme.ID, UserID: alice,
		Patch: domain.PreferencesPatch{NavigationProjectLimit: ptr(7)}, Now: now}); got != prefs("ACCORDION", 7) {
		t.Errorf("UpsertPreferences() after the deletion = %+v, want ACCORDION, 7", got)
	}
	if rows := undeletedPrefs(t, pool, acme.ID, alice); len(rows) != 1 || rows[0].id != second {
		t.Errorf("undeleted rows %+v, want the new one, %s", rows, second)
	}
	if p, found, err := s.Preferences(context.Background(), acme.ID, alice); err != nil || !found || p != prefs("ACCORDION", 7) {
		t.Errorf("Preferences() = %+v, %v, %v; want ACCORDION, 7", p, found, err)
	}
}

// Preferences reads the account's own undeleted row in that workspace:
// none before the first change, none for another account's or another
// workspace's row.
func TestPreferencesReadsTheAccountsOwnRow(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme, beta := newWorkspace(t, s, "Acme", "acme", alice), newWorkspace(t, s, "Beta", "beta", alice)
	if _, found, err := s.Preferences(context.Background(), acme.ID, alice); err != nil || found {
		t.Errorf("before any change: found %v, %v; want none", found, err)
	}
	upsert(t, s, app.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: acme.ID, UserID: bob, Patch: domain.PreferencesPatch{NavigationProjectLimit: ptr(1)}, Now: now})
	upsert(t, s, app.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: beta.ID, UserID: alice, Patch: domain.PreferencesPatch{NavigationProjectLimit: ptr(2)}, Now: now})
	if _, found, err := s.Preferences(context.Background(), acme.ID, alice); err != nil || found {
		t.Errorf("with bob's row in acme and alice's in beta: found %v, %v; want none for alice in acme", found, err)
	}
	for _, tt := range []struct {
		workspace, user uuid.UUID
		limit           int
	}{{acme.ID, bob, 1}, {beta.ID, alice, 2}} {
		if p, found, err := s.Preferences(context.Background(), tt.workspace, tt.user); err != nil || !found || p.NavigationProjectLimit != tt.limit {
			t.Errorf("Preferences(%s, %s) = %+v, %v, %v; want the limit %d", tt.workspace, tt.user, p, found, err, tt.limit)
		}
	}
}

// Two first changes at once leave one row (M3 design 3.18): the second
// insert waits on the first's index entry, and once that commits it changes
// the row instead, the fields it sets over the first's.
func TestConcurrentFirstChangesLeaveOneRow(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, s, "Acme", "acme", alice)
	tx := postgres.NewTxManager(pool, 5*time.Second)
	commit := hold(t, tx, func(ctx context.Context) error {
		_, err := s.UpsertPreferences(ctx, app.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: acme.ID, UserID: alice,
			Patch: domain.PreferencesPatch{NavigationProjectLimit: ptr(3)}, Now: now})
		return err
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	type answer struct {
		p   domain.Preferences
		err error
	}
	answered := make(chan answer, 1)
	go func() {
		p, err := s.UpsertPreferences(ctx, app.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: acme.ID, UserID: alice,
			Patch: domain.PreferencesPatch{NavigationControl: ptr("TABBED")}, Now: now})
		answered <- answer{p, err}
	}()
	pgtest.WaitForLockWait(t, pool, 5*time.Second)
	if err := commit(); err != nil {
		t.Fatal(err)
	}
	select {
	case a := <-answered:
		if a.err != nil || a.p != prefs("TABBED", 3) {
			t.Errorf("the second change = %+v, %v; want TABBED, 3", a.p, a.err)
		}
	case <-ctx.Done():
		t.Fatal("the second change did not end within 10s")
	}
	if rows := undeletedPrefs(t, pool, acme.ID, alice); len(rows) != 1 {
		t.Errorf("rows %+v, want one", rows)
	}
}

// prefs is the settings mode and limit.
func prefs(mode string, limit int) domain.Preferences {
	return domain.Preferences{NavigationControl: mode, NavigationProjectLimit: limit}
}
