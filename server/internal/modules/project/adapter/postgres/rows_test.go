package postgresadapter_test

import (
	"context"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The rows under a project store the use case's values, their other
// columns at their defaults: an active membership, the default navigation,
// a state's empty description. Each insert leaves the table's other rows,
// of another project, another account and another workspace, as they were.
// No foreign key ties a row's workspace_id to its project's, and the
// workspace's deletion finds the rows by it: each row must carry its
// project's workspace. With beta in the database, a row written with
// another workspace's id fails the comparison.
func TestCreateTheRowsUnderAProject(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	ops, web := newProject(t, s, acme, "Ops", "OPS", alice), newProject(t, s, acme, "Web", "WEB", alice)
	site := newProject(t, s, beta, "Site", "SITE", alice)
	ctx := context.Background()
	// Rows of another project, of another account and of another
	// workspace's project, before each insert.
	for _, row := range []struct {
		workspace, project, user uuid.UUID
	}{{acme, ops, bob}, {acme, web, alice}, {beta, site, bob}} {
		if err := s.CreateMember(ctx, app.MemberRow{ID: uuid.NewV7(), WorkspaceID: row.workspace, ProjectID: row.project,
			MemberID: row.user, Role: shared.RoleAdmin, CreatedBy: alice, Now: now}); err != nil {
			t.Fatal(err)
		}
		if err := s.CreatePreferences(ctx, app.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: row.workspace, ProjectID: row.project,
			UserID: row.user, SortOrder: 1, CreatedBy: alice, Now: now}); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.CreateStates(ctx, []app.StateRow{
		{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: ops, CreatedBy: alice, Now: now,
			State: domain.NewState{Name: "Backlog", Color: "#000000", Sequence: 1, Group: "backlog", Default: true}},
		{ID: uuid.NewV7(), WorkspaceID: beta, ProjectID: site, CreatedBy: alice, Now: now,
			State: domain.NewState{Name: "Backlog", Color: "#000000", Sequence: 1, Group: "backlog", Default: true}},
	}); err != nil {
		t.Fatal(err)
	}
	member, prefs := uuid.NewV7(), uuid.NewV7()
	states := []app.StateRow{
		{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web, CreatedBy: bob, Now: now,
			State: domain.NewState{Name: "Backlog", Color: "#60646C", Sequence: 15000, Group: "backlog", Default: true}},
		{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web, CreatedBy: bob, Now: now,
			State: domain.NewState{Name: "Triage", Color: "#4E5355", Sequence: 65000, Group: "triage"}},
	}
	before := map[string]string{"project_members": tableRows(t, pool, "project_members", member),
		"project_user_properties": tableRows(t, pool, "project_user_properties", prefs), "states": tableRows(t, pool, "states", uuid.UUID{})}

	if err := s.CreateMember(ctx, app.MemberRow{
		ID: member, WorkspaceID: acme, ProjectID: web, MemberID: bob, Role: shared.RoleMember, CreatedBy: alice, Now: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreatePreferences(ctx, app.PreferencesRow{
		ID: prefs, WorkspaceID: acme, ProjectID: web, UserID: bob, SortOrder: -9999.5, CreatedBy: alice, Now: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateStates(ctx, states); err != nil {
		t.Fatal(err)
	}

	var got string
	if err := pool.QueryRow(ctx, `SELECT concat_ws(' | ',
		(SELECT concat_ws(' ', workspace_id, project_id, member_id, role, is_active, created_by_id, updated_by_id, created_at = $2,
			updated_at = $2, deleted_at IS NULL) FROM project_members WHERE id = $3),
		(SELECT concat_ws(' ', workspace_id, project_id, user_id, preferences, sort_order, created_by_id, updated_by_id, created_at = $2,
			updated_at = $2, deleted_at IS NULL) FROM project_user_properties WHERE id = $4),
		(SELECT string_agg(concat_ws(' ', workspace_id, project_id, name, description = '', color, sequence, "group", "default",
			created_by_id, updated_by_id, created_at = $2, updated_at = $2, deleted_at IS NULL), ' / ' ORDER BY sequence)
			FROM states WHERE project_id = $1))`, web, now, member, prefs).Scan(&got); err != nil {
		t.Fatal(err)
	}
	under := func(s string) string { return acme.String() + " " + web.String() + " " + s }
	a, b := alice.String(), bob.String()
	want := under(b+" 15 t "+a+" "+a+" t t t") + " | " +
		under(b+` {"navigation": {"default_tab": "work_items", "hide_in_more_menu": []}} -9999.5 `+a+" "+a+" t t t") + " | " +
		under("Backlog t #60646C 15000 backlog t "+b+" "+b+" t t t") + " / " +
		under("Triage t #4E5355 65000 triage f "+b+" "+b+" t t t")
	if got != want {
		t.Errorf("the rows =\n%s\nwant\n%s", got, want)
	}
	for table, rows := range map[string]string{"project_members": tableRows(t, pool, "project_members", member),
		"project_user_properties": tableRows(t, pool, "project_user_properties", prefs)} {
		if rows != before[table] {
			t.Errorf("the other rows of %s:\n%s\nwant\n%s", table, rows, before[table])
		}
	}
	var others string
	if err := pool.QueryRow(ctx, "SELECT string_agg(r::text, E'\\n' ORDER BY r::text) FROM states r WHERE project_id <> $1", web).Scan(&others); err != nil ||
		others != before["states"] {
		t.Errorf("the other states: %s, %v; want %s", others, err, before["states"])
	}
}

// CreateStates stops at the first state that fails, and names it: a
// second default state breaks states_project_id_default_key.
func TestCreateStatesStopsAtTheFirstFailure(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web := newProject(t, s, acme, "Web", "WEB", alice)
	row := func(name string) app.StateRow {
		return app.StateRow{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web, CreatedBy: alice, Now: now,
			State: domain.NewState{Name: name, Color: "#60646C", Group: "backlog", Default: true}}
	}

	err := s.CreateStates(context.Background(), []app.StateRow{row("One"), row("Two"), row("Three")})

	var names []string
	if err := pool.QueryRow(context.Background(), "SELECT coalesce(array_agg(name ORDER BY name), '{}') FROM states").Scan(&names); err != nil {
		t.Fatal(err)
	}
	if err == nil || !strings.Contains(err.Error(), `"Two"`) || !strings.Contains(err.Error(), "states_project_id_default_key") ||
		len(names) != 1 || names[0] != "One" {
		t.Errorf("CreateStates() = %v, the states %q; want Two's failure and only One stored", err, names)
	}
}

// LowestSortOrder is the least place of the account's in the workspace's
// projects, over his undeleted display settings, those of a project he
// left too; nil when he has none. Each row the query must pass over has a
// lower place than the answer: another workspace's, another account's, a
// deleted one. The answer is neither the first nor the last of alice's
// undeleted rows in acme, so that a read that does not sort misses it
// whichever way it reads the table.
func TestLowestSortOrder(t *testing.T) {
	s, pool := newStore(t)
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	web, ops, old := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Ops", "OPS", alice), newProject(t, s, acme, "Old", "OLD", alice)
	docs := newProject(t, s, acme, "Docs", "DOCS", alice)
	other := newProject(t, s, beta, "Web", "WEB", alice)
	for _, row := range []struct {
		project, user uuid.UUID
		sortOrder     float64
	}{{web, alice, 100}, {ops, alice, 50}, {old, alice, -5}, {docs, alice, 75}, {other, alice, -1000}, {web, bob, -2000}} {
		workspace := acme
		if row.project == other {
			workspace = beta
		}
		if err := s.CreatePreferences(context.Background(), app.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: workspace, ProjectID: row.project,
			UserID: row.user, SortOrder: row.sortOrder, CreatedBy: alice, Now: now}); err != nil {
			t.Fatal(err)
		}
	}
	// ops stands for a project alice left: its display settings stay
	// undeleted. old's were deleted.
	exec(t, pool, "UPDATE project_user_properties SET deleted_at = $2 WHERE project_id = $1", old, now)

	for _, tt := range []struct {
		name            string
		workspace, user uuid.UUID
		want            *float64
	}{
		{"alice in acme", acme, alice, ptr(50.0)},
		{"bob in acme", acme, bob, ptr(-2000.0)},
		{"alice in beta", beta, alice, ptr(-1000.0)},
		{"carol, none", acme, carol, nil},
		{"bob in beta, none", beta, bob, nil},
	} {
		got, err := s.LowestSortOrder(context.Background(), tt.workspace, tt.user)
		if err != nil || jsonOf(t, got) != jsonOf(t, tt.want) {
			t.Errorf("%s: LowestSortOrder() = %v, %v; want %v", tt.name, jsonOf(t, got), err, jsonOf(t, tt.want))
		}
	}
}
