package postgresadapter_test

import (
	"context"
	"maps"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// ShareProject reads the undeleted project's workspace and whether it is
// archived, and holds it FOR SHARE until the transaction ends: a FOR NO KEY
// UPDATE of it waits, another FOR SHARE does not, and no other project is
// held. A deleted project, and no project, are not found.
func TestShareProject(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	web, ops, old := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, beta, "Ops", "OPS", alice), newProject(t, s, acme, "Old", "OLD", alice)
	exec(t, pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", ops, now)
	exec(t, pool, "UPDATE projects SET deleted_at = $2 WHERE id = $1", old, now)
	for _, tt := range []struct {
		id    uuid.UUID
		want  app.LockedProject
		found bool
	}{{web, app.LockedProject{WorkspaceID: acme}, true}, {ops, app.LockedProject{WorkspaceID: beta, Archived: true}, true},
		{old, app.LockedProject{}, false}, {uuid.NewV7(), app.LockedProject{}, false}} {
		if got, found, err := s.ShareProject(context.Background(), tt.id); err != nil || found != tt.found || got != tt.want {
			t.Errorf("ShareProject(%s) = %+v, %v, %v; want %+v, %v", tt.id, got, found, err, tt.want, tt.found)
		}
	}
	err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
		if _, _, err := s.ShareProject(ctx, web); err != nil {
			return err
		}
		if !waits(t, pool, web, "FOR NO KEY UPDATE") || waits(t, pool, web, "FOR SHARE") || waits(t, pool, ops, "FOR NO KEY UPDATE") {
			t.Error("while the lock is held: want a FOR NO KEY UPDATE of web to wait, and neither its FOR SHARE nor ops' FOR NO KEY UPDATE")
		}
		return nil
	})
	if err != nil || waits(t, pool, web, "FOR NO KEY UPDATE") {
		t.Errorf("after the transaction: %v; want web free", err)
	}
}

// A project deleted while ShareProject waited for its lock is not found, as
// with LockProject: after the wait, Postgres evaluates deleted_at IS NULL
// again on the row's newest version (M3 design 3.6). Another transaction
// soft-deletes Web, as deleteProject does under its FOR NO KEY UPDATE;
// ShareProject waits for it, and once the deletion commits finds no
// project.
func TestTheProjectShareSeesADeletionItWaitedFor(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	web := newProject(t, s, newWorkspace(t, pool, "acme"), "Web", "WEB", alice)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "UPDATE projects SET deleted_at = $2 WHERE id = $1", web, now); err != nil {
		t.Fatal(err)
	}
	type answer struct {
		found bool
		err   error
	}
	done := make(chan answer, 1)
	go func() {
		var a answer
		a.err = postgres.NewTxManager(pool, 5*time.Second).WithinTx(ctx, func(ctx context.Context) error {
			var err error
			_, a.found, err = s.ShareProject(ctx, web)
			return err
		})
		done <- a
	}()
	pgtest.WaitForLockWaitOn(t, pool, "projects", 5*time.Second)
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case a := <-done:
		if a.err != nil || a.found {
			t.Errorf("ShareProject after the deletion committed = found %v, %v; want not found", a.found, a.err)
		}
	case <-ctx.Done():
		t.Fatal("ShareProject did not end within 10s")
	}
}

// preferencesRow inserts user's display settings in project with no value
// but the keys, so that every other column takes its default, and returns
// the row's id.
func preferencesRow(t *testing.T, pool *pgxpool.Pool, workspace, project, user uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.NewV7()
	exec(t, pool, `INSERT INTO project_user_properties (id, workspace_id, project_id, user_id) VALUES ($1, $2, $3, $4)`, id, workspace, project, user)
	return id
}

// A row with every column at its default reads as domain.DefaultPreferences:
// the defaults the use case answers while there is no row are the columns'.
// Preferences reads the account's undeleted row in the project alone: none
// for another account, another project or a deleted row.
func TestPreferences(t *testing.T) {
	s, pool := newStore(t)
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, ops := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Ops", "OPS", alice)
	preferencesRow(t, pool, acme, web, bob)
	views := preferencesRow(t, pool, acme, ops, bob)
	exec(t, pool, `UPDATE project_user_properties SET preferences = '{"navigation": {"default_tab": "views", "hide_in_more_menu": ["cycles"]}}',
		sort_order = 7 WHERE id = $1`, views)
	exec(t, pool, "UPDATE project_user_properties SET deleted_at = $2 WHERE id = $1", preferencesRow(t, pool, acme, web, carol), now)
	for _, tt := range []struct {
		project, user uuid.UUID
		want          domain.Preferences
		found         bool
	}{
		{web, bob, domain.DefaultPreferences(), true},
		{ops, bob, domain.Preferences{Navigation: domain.Navigation{DefaultTab: "views", HideInMoreMenu: []string{"cycles"}}, SortOrder: 7}, true},
		{web, carol, domain.Preferences{}, false},
		{web, alice, domain.Preferences{}, false},
		{uuid.NewV7(), bob, domain.Preferences{}, false},
	} {
		if got, found, err := s.Preferences(context.Background(), tt.project, tt.user); err != nil || found != tt.found || !reflect.DeepEqual(got, tt.want) {
			t.Errorf("Preferences(%s, %s) = %+v, %v, %v; want %+v, %v", tt.project, tt.user, got, found, err, tt.want, tt.found)
		}
	}
}

// UpsertPreferences inserts the account's row while he has no undeleted
// one, the defaults with the change applied, made and last changed by him
// at the moment given; with a row, it changes the fields the change gives,
// the navigation whole or the place alone, and the audit columns, to him
// and that moment even when another account made the row, and leaves the
// rest. The
// answer is the row as stored. A navigation that hides nothing, its list
// nil, is stored and answered as an empty list, never null (M3 design 4.8).
// Every other row keeps every column: another account's in the project,
// his in another project, his deleted one.
func TestUpsertPreferences(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, ops := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Ops", "OPS", alice)
	deleted := preferencesRow(t, pool, acme, web, bob)
	exec(t, pool, "UPDATE project_user_properties SET deleted_at = $2 WHERE id = $1", deleted, now)
	preferencesRow(t, pool, acme, ops, bob)
	preferencesRow(t, pool, acme, web, alice)
	others := tableRows(t, pool, "project_user_properties", uuid.Nil())
	upsert := func(p domain.PreferencesPatch, at time.Time) domain.Preferences {
		t.Helper()
		got, err := s.UpsertPreferences(context.Background(), app.PreferencesChange{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web, UserID: bob,
			Patch: p, Now: at})
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	var id uuid.UUID // bob's row in web, once inserted
	row := func() map[string]string {
		if err := pool.QueryRow(context.Background(), "SELECT id FROM project_user_properties WHERE project_id = $1 AND user_id = $2 AND deleted_at IS NULL",
			web, bob).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return columns(t, pool, "project_user_properties", id)
	}
	stamp := func(at time.Time) string { return `"` + at.Format("2006-01-02T15:04:05.999999") + `+00:00"` }
	views := domain.Navigation{DefaultTab: "views", HideInMoreMenu: []string{"intake", "cycles"}}

	if got := upsert(domain.PreferencesPatch{SortOrder: ptr(-5.5)}, now); !reflect.DeepEqual(got,
		domain.Preferences{Navigation: domain.DefaultPreferences().Navigation, SortOrder: -5.5}) {
		t.Errorf("the first change = %+v; want the defaults, at -5.5", got)
	}
	first := row()
	for k, want := range map[string]string{"sort_order": "-5.5", "created_by_id": `"` + bob.String() + `"`, "updated_by_id": `"` + bob.String() + `"`,
		"created_at": stamp(now), "updated_at": stamp(now), "deleted_at": "null", "workspace_id": `"` + acme.String() + `"`,
		"preferences": `{"navigation": {"default_tab": "work_items", "hide_in_more_menu": []}}`} {
		if first[k] != want {
			t.Errorf("the inserted row's %s = %s, want %s", k, first[k], want)
		}
	}

	// A row another account made and changed last, as alice's adding him makes it: the next change is his.
	exec(t, pool, "UPDATE project_user_properties SET created_by_id = $2, updated_by_id = $2 WHERE id = $1", id, alice)
	first = row()
	later := now.Add(time.Hour)
	if got := upsert(domain.PreferencesPatch{Navigation: &views}, later); !reflect.DeepEqual(got, domain.Preferences{Navigation: views, SortOrder: -5.5}) {
		t.Errorf("the second change = %+v; want views, still at -5.5", got)
	}
	want := maps.Clone(first)
	maps.Copy(want, map[string]string{"preferences": `{"navigation": {"default_tab": "views", "hide_in_more_menu": ["intake", "cycles"]}}`,
		"updated_at": stamp(later), "updated_by_id": `"` + bob.String() + `"`})
	if got := row(); !maps.Equal(got, want) {
		t.Errorf("after the second change: %v\nwant %v", got, want)
	}
	if got := upsert(domain.PreferencesPatch{}, later.Add(time.Hour)); !reflect.DeepEqual(got, domain.Preferences{Navigation: views, SortOrder: -5.5}) {
		t.Errorf("an empty change = %+v; want the settings as they were", got)
	}
	cycles := domain.Navigation{DefaultTab: "cycles"}
	if got := upsert(domain.PreferencesPatch{Navigation: &cycles}, later.Add(2*time.Hour)); !reflect.DeepEqual(got,
		domain.Preferences{Navigation: domain.Navigation{DefaultTab: "cycles", HideInMoreMenu: []string{}}, SortOrder: -5.5}) {
		t.Errorf("a change hiding nothing (nil) = %+v; want cycles, nothing hidden ([]), still at -5.5", got)
	}
	hidden := row()
	if got, want := hidden["preferences"], `{"navigation": {"default_tab": "cycles", "hide_in_more_menu": []}}`; got != want {
		t.Errorf("the row after a change hiding nothing (nil): preferences %s, want %s", got, want)
	}
	// The place alone, as a reorder in the sidebar changes it: the navigation stays.
	reordered := later.Add(3 * time.Hour)
	if got := upsert(domain.PreferencesPatch{SortOrder: ptr(2.25)}, reordered); !reflect.DeepEqual(got,
		domain.Preferences{Navigation: domain.Navigation{DefaultTab: "cycles", HideInMoreMenu: []string{}}, SortOrder: 2.25}) {
		t.Errorf("a change of the place alone = %+v; want cycles, nothing hidden, at 2.25", got)
	}
	if got, want := row(), changed(hidden, map[string]string{"sort_order": "2.25", "updated_at": stamp(reordered)}); !maps.Equal(got, want) {
		t.Errorf("after a change of the place alone: %v\nwant %v", got, want)
	}
	if after := tableRows(t, pool, "project_user_properties", id); after != others {
		t.Errorf("the other rows:\n%s\nwant\n%s", after, others)
	}
}

// EnsurePreferences inserts the account's display settings, the row with
// the id, workspace, project and place given, made by the account given at
// the moment given, the navigation the column's default, while he has no
// undeleted ones in the project, a deleted one not counting; undeleted ones
// stay as they are, and so does every other row: his in another workspace
// too, beta's, which a wrong workspace could name without breaking the
// foreign key. Only the partial unique key's conflict is let pass: an id
// another row has is an error, and writes nothing.
func TestEnsurePreferences(t *testing.T) {
	s, pool := newStore(t)
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	web, ops := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Ops", "OPS", alice)
	bobs := preferencesRow(t, pool, acme, web, bob)
	exec(t, pool, `UPDATE project_user_properties SET preferences = '{"navigation": {"default_tab": "views", "hide_in_more_menu": []}}',
		sort_order = 7 WHERE id = $1`, bobs)
	exec(t, pool, "UPDATE project_user_properties SET deleted_at = $2 WHERE id = $1", preferencesRow(t, pool, acme, web, carol), now)
	preferencesRow(t, pool, acme, ops, carol)
	preferencesRow(t, pool, beta, newProject(t, s, beta, "Web", "WEB", alice), carol)
	ensure := func(user uuid.UUID, at time.Time) uuid.UUID {
		t.Helper()
		id := uuid.NewV7()
		if err := s.EnsurePreferences(context.Background(), app.PreferencesRow{ID: id, WorkspaceID: acme, ProjectID: web, UserID: user, SortOrder: -3,
			CreatedBy: alice, Now: at}); err != nil {
			t.Fatal(err)
		}
		return id
	}
	before := tableRows(t, pool, "project_user_properties", uuid.Nil())

	ensure(bob, now.Add(time.Hour))
	if after := tableRows(t, pool, "project_user_properties", uuid.Nil()); after != before {
		t.Errorf("after bob's, who has his settings:\n%s\nwant\n%s", after, before)
	}
	carols := ensure(carol, now.Add(time.Hour))
	stamp := `"` + now.Add(time.Hour).Format("2006-01-02T15:04:05.999999") + `+00:00"`
	want := map[string]string{"id": `"` + carols.String() + `"`, "workspace_id": `"` + acme.String() + `"`, "project_id": `"` + web.String() + `"`,
		"user_id": `"` + carol.String() + `"`, "preferences": `{"navigation": {"default_tab": "work_items", "hide_in_more_menu": []}}`,
		"sort_order": "-3", "created_by_id": `"` + alice.String() + `"`, "updated_by_id": `"` + alice.String() + `"`, "created_at": stamp,
		"updated_at": stamp, "deleted_at": "null"}
	if got := columns(t, pool, "project_user_properties", carols); !maps.Equal(got, want) {
		t.Errorf("carol's new settings: %v\nwant %v", got, want)
	}
	if after := tableRows(t, pool, "project_user_properties", carols); after != before {
		t.Errorf("the other rows after carol's:\n%s\nwant\n%s", after, before)
	}
	// bob has no settings in Ops, so only the id conflicts.
	all := tableRows(t, pool, "project_user_properties", uuid.Nil())
	if err := s.EnsurePreferences(context.Background(), app.PreferencesRow{ID: bobs, WorkspaceID: acme, ProjectID: ops, UserID: bob, SortOrder: -3,
		CreatedBy: alice, Now: now}); err == nil {
		t.Error("EnsurePreferences() under another row's id = nil; want the primary key's violation")
	}
	if after := tableRows(t, pool, "project_user_properties", uuid.Nil()); after != all {
		t.Errorf("the rows after an insert under another row's id:\n%s\nwant\n%s", after, all)
	}
}
