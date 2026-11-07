package postgresadapter_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
)

// A module is written with every value and reads back as written, its members' rows each with its id, at the module's
// moment and by its writer; a module without anything but its name, with its defaults. A name another undeleted module
// of the project has is domain.ErrModuleNameTaken in any case, an archived one's too, not a deleted one's nor one of
// another project; the refused module is not written.
func TestCreateModule(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newWorld(t, pool)
	at := now.Truncate(time.Microsecond)
	lead, member := account(t, pool, "lea"), account(t, pool, "mem")
	m := domain.Module{ID: uuid.NewV7(), ProjectID: w.web, WorkspaceID: w.workspace, Name: "Auth", Description: "Sign in",
		StartDate: dayOf(t, "2026-11-02"), TargetDate: dayOf(t, "2026-11-13"), Status: domain.ModuleInProgress, LeadID: &lead,
		MemberIDs: []uuid.UUID{lead, member}, SortOrder: 55535.5, CreatedAt: at, UpdatedAt: at, CreatedByID: &w.by, UpdatedByID: &w.by}
	if err := s.CreateModule(ctx, m); err != nil {
		t.Fatal(err)
	}
	rows := app.MemberRows{WorkspaceID: w.workspace, ProjectID: w.web, ModuleID: m.ID, IDs: sortedIDs(2), Accounts: m.MemberIDs, By: w.by, At: at}
	if err := s.InsertModuleMembers(ctx, rows); err != nil {
		t.Fatal(err)
	}
	if got, found, err := s.GetModule(ctx, m.ID); err != nil || !found || jsonOf(t, got) != jsonOf(t, m) {
		t.Fatalf("GetModule = %s, %t, %v\nwant %s", jsonOf(t, got), found, err, jsonOf(t, m))
	}
	type memberRow struct {
		ID, WorkspaceID, ProjectID, MemberID uuid.UUID
		CreatedByID, UpdatedByID             *uuid.UUID
		Stamped, Deleted                     bool
	}
	written, err := pool.Query(ctx, `SELECT id, workspace_id, project_id, member_id, created_by_id, updated_by_id, created_at = $2 AND updated_at = $2,
		deleted_at IS NOT NULL FROM module_members WHERE module_id = $1 ORDER BY id`, m.ID, at)
	if err != nil {
		t.Fatal(err)
	}
	var got []memberRow
	for written.Next() {
		var r memberRow
		if err := written.Scan(&r.ID, &r.WorkspaceID, &r.ProjectID, &r.MemberID, &r.CreatedByID, &r.UpdatedByID, &r.Stamped, &r.Deleted); err != nil {
			t.Fatal(err)
		}
		got = append(got, r)
	}
	if written.Err() != nil {
		t.Fatal(written.Err())
	}
	want := []memberRow{{rows.IDs[0], w.workspace, w.web, lead, &w.by, &w.by, true, false}, {rows.IDs[1], w.workspace, w.web, member, &w.by, &w.by, true, false}}
	if jsonOf(t, got) != jsonOf(t, want) {
		t.Fatalf("member rows %s\nwant %s", jsonOf(t, got), jsonOf(t, want))
	}

	// No member: the read's empty array.
	bare := domain.Module{ID: uuid.NewV7(), ProjectID: w.web, WorkspaceID: w.workspace, Name: "Bare", Status: domain.ModulePlanned,
		MemberIDs: []uuid.UUID{}, SortOrder: 65535, CreatedAt: at, UpdatedAt: at}
	if err := s.CreateModule(ctx, bare); err != nil {
		t.Fatal(err)
	}
	if got, found, err := s.GetModule(ctx, bare.ID); err != nil || !found || jsonOf(t, got) != jsonOf(t, bare) {
		t.Fatalf("GetModule = %s, %t, %v\nwant %s", jsonOf(t, got), found, err, jsonOf(t, bare))
	}

	named := func(name string, project uuid.UUID) error {
		return s.CreateModule(ctx, domain.Module{ID: uuid.NewV7(), ProjectID: project, WorkspaceID: w.workspace, Name: name, Status: domain.ModulePlanned,
			CreatedAt: at, UpdatedAt: at})
	}
	for _, name := range []string{"Auth", "AUTH", "auth", "bARE"} {
		if err := named(name, w.web); !errors.Is(err, domain.ErrModuleNameTaken) {
			t.Fatalf("%s in Web: %v; want planning.module_name_taken", name, err)
		}
	}
	exec(t, pool, "UPDATE modules SET archived_at = now() WHERE id = $1", m.ID)
	if err := named("AUTH", w.web); !errors.Is(err, domain.ErrModuleNameTaken) {
		t.Fatalf("an archived module's name: %v; want planning.module_name_taken", err)
	}
	if err := named("auth", w.ops); err != nil {
		t.Fatalf("another project's module's name: %v", err)
	}
	exec(t, pool, "UPDATE modules SET deleted_at = now() WHERE id = $1", m.ID)
	if err := named("auth", w.web); err != nil {
		t.Fatalf("a deleted module's name: %v", err)
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM modules").Scan(&count); err != nil || count != 4 {
		t.Fatalf("modules = %d, %v; want Auth, Bare, Ops' auth and the new auth", count, err)
	}
}

// The lowest sort order of the project's undeleted modules, archived ones too, not a deleted one's nor another
// project's; none for a project without one.
func TestLowestModuleSortOrder(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newWorld(t, pool)
	if lowest, err := s.LowestModuleSortOrder(ctx, w.web); err != nil || lowest != nil {
		t.Fatalf("no module: %v, %v; want none", lowest, err)
	}
	module := func(project uuid.UUID, name string, sortOrder float64, set string) {
		id := uuid.NewV7()
		exec(t, pool, "INSERT INTO modules (id, workspace_id, project_id, name, sort_order) VALUES ($1, $2, $3, $4, $5)", id, w.workspace, project, name,
			sortOrder)
		if set != "" {
			exec(t, pool, "UPDATE modules SET "+set+" = now() WHERE id = $1", id)
		}
	}
	module(w.web, "A", 100, "")
	module(w.web, "B", 50, "archived_at")
	module(w.web, "C", 10, "deleted_at")
	module(w.ops, "D", 1, "")
	if lowest, err := s.LowestModuleSortOrder(ctx, w.web); err != nil || lowest == nil || *lowest != 50 {
		t.Fatalf("lowest = %v, %v; want the archived 50", lowest, err)
	}
}

// UpdateModule writes a module's name, description, dates, status, lead, sort order and last change; what it does not
// change — its project, archive, creation and members' rows — stays as it was. A name another undeleted module of the
// project has is domain.ErrModuleNameTaken in any case, an archived one's too, not a deleted one's nor one of another
// project; its own name in another case is its to take. A refused change leaves the module as it was.
func TestUpdateModule(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newWorld(t, pool)
	at := now.Truncate(time.Microsecond)
	lead, editor := account(t, pool, "lea"), account(t, pool, "eve")
	m := domain.Module{ID: uuid.NewV7(), ProjectID: w.web, WorkspaceID: w.workspace, Name: "Auth", Description: "Sign in",
		StartDate: dayOf(t, "2026-11-02"), TargetDate: dayOf(t, "2026-11-13"), Status: domain.ModulePlanned, LeadID: &lead,
		MemberIDs: []uuid.UUID{lead}, SortOrder: 5, CreatedAt: at, UpdatedAt: at, CreatedByID: &w.by, UpdatedByID: &w.by}
	if err := s.CreateModule(ctx, m); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertModuleMembers(ctx, app.MemberRows{WorkspaceID: w.workspace, ProjectID: w.web, ModuleID: m.ID, IDs: sortedIDs(1),
		Accounts: m.MemberIDs, By: w.by, At: at}); err != nil {
		t.Fatal(err)
	}
	changed := m
	changed.Name, changed.Description, changed.StartDate, changed.TargetDate = "Auth 2", "", nil, dayOf(t, "2026-11-30")
	changed.Status, changed.LeadID, changed.SortOrder = domain.ModuleInProgress, nil, -0.5
	changed.UpdatedAt, changed.UpdatedByID = at.Add(time.Minute), &editor
	// What UpdateModule does not write, changed here, must not reach the rows.
	moved := changed
	archived := at.Add(time.Hour)
	moved.ProjectID, moved.WorkspaceID, moved.ArchivedAt, moved.CreatedAt, moved.CreatedByID = w.ops, uuid.NewV7(), &archived, archived, &editor
	moved.MemberIDs = []uuid.UUID{editor}
	if err := s.UpdateModule(ctx, moved); err != nil {
		t.Fatal(err)
	}
	if got, found, err := s.GetModule(ctx, m.ID); err != nil || !found || jsonOf(t, got) != jsonOf(t, changed) {
		t.Fatalf("after UpdateModule: %s, %t, %v\nwant %s", jsonOf(t, got), found, err, jsonOf(t, changed))
	}

	other := func(name, set string, project uuid.UUID) {
		t.Helper()
		id := uuid.NewV7()
		exec(t, pool, "INSERT INTO modules (id, workspace_id, project_id, name) VALUES ($1, $2, $3, $4)", id, w.workspace, project, name)
		if set != "" {
			exec(t, pool, "UPDATE modules SET "+set+" = now() WHERE id = $1", id)
		}
	}
	other("Billing", "", w.web)
	other("Old", "archived_at", w.web)
	other("Gone", "deleted_at", w.web)
	other("Ops", "", w.ops)
	renamed := changed
	for _, name := range []string{"billing", "BILLING", "old"} {
		renamed.Name = name
		if err := s.UpdateModule(ctx, renamed); !errors.Is(err, domain.ErrModuleNameTaken) {
			t.Fatalf("renamed %s: %v; want planning.module_name_taken", name, err)
		}
	}
	if got, _, err := s.GetModule(ctx, m.ID); err != nil || jsonOf(t, got) != jsonOf(t, changed) {
		t.Fatalf("after the refused renames: %s, %v", jsonOf(t, got), err)
	}
	for _, name := range []string{"AUTH 2", "gone", "ops"} {
		renamed.Name = name
		if err := s.UpdateModule(ctx, renamed); err != nil {
			t.Fatalf("renamed %s: %v", name, err)
		}
		if got, _, err := s.GetModule(ctx, m.ID); err != nil || got.Name != name {
			t.Fatalf("renamed %s: read %q, %v", name, got.Name, err)
		}
	}
}

// RemoveModuleMembers soft-deletes the module's undeleted rows of the accounts given, deleted and last changed by one
// account at one moment; a row of an account not given, a row deleted before, which keeps its deletion, and another
// module's row of the same account are untouched. The account may be a member again, in a row of its own.
func TestRemoveModuleMembers(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newWorld(t, pool)
	at := now.Truncate(time.Microsecond)
	people := sortedIDs(4)
	for _, id := range people {
		exec(t, pool, "INSERT INTO users (id, email, password, display_name) VALUES ($1, $2, 'x', 'm')", id, id.String()+"@corp.com")
	}
	a, b, c, d := people[0], people[1], people[2], people[3]
	auth, billing := uuid.NewV7(), uuid.NewV7()
	for _, id := range []uuid.UUID{auth, billing} {
		exec(t, pool, "INSERT INTO modules (id, workspace_id, project_id, name) VALUES ($1, $2, $3, $4)", id, w.workspace, w.web, id.String())
	}
	member := func(module, account uuid.UUID, deleted *time.Time) uuid.UUID {
		id := uuid.NewV7()
		exec(t, pool, `INSERT INTO module_members (id, workspace_id, project_id, module_id, member_id, created_by_id, updated_by_id, created_at,
			updated_at, deleted_at) VALUES ($1, $2, $3, $4, $5, $6, $6, $7, $7, $8)`, id, w.workspace, w.web, module, account, w.by, at, deleted)
		return id
	}
	earlier := at.Add(-time.Hour)
	rows := map[string]uuid.UUID{"a": member(auth, a, nil), "b": member(auth, b, nil), "c": member(auth, c, nil), "d": member(auth, d, &earlier),
		"billing a": member(billing, a, nil)}
	editor := account(t, pool, "eve")
	later := at.Add(time.Minute)
	if err := s.RemoveModuleMembers(ctx, auth, []uuid.UUID{a, c, d}, later, editor); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{"a": "removed", "c": "removed", "b": "kept", "billing a": "kept", "d": "earlier"} {
		var deleted, updated *time.Time
		var by uuid.UUID
		if err := pool.QueryRow(ctx, "SELECT deleted_at, updated_at, updated_by_id FROM module_members WHERE id = $1", rows[name]).Scan(&deleted,
			&updated, &by); err != nil {
			t.Fatal(err)
		}
		var ok bool
		switch want {
		case "removed":
			ok = deleted != nil && deleted.Equal(later) && updated.Equal(later) && by == editor
		case "kept":
			ok = deleted == nil && updated.Equal(at) && by == w.by
		case "earlier":
			ok = deleted != nil && deleted.Equal(earlier) && updated.Equal(at) && by == w.by
		}
		if !ok {
			t.Errorf("%s's row: deleted %v, changed %v by %s; want %s", name, deleted, updated, by, want)
		}
	}
	if err := s.InsertModuleMembers(ctx, app.MemberRows{WorkspaceID: w.workspace, ProjectID: w.web, ModuleID: auth, IDs: sortedIDs(1),
		Accounts: []uuid.UUID{a}, By: editor, At: later}); err != nil {
		t.Fatal(err)
	}
	if got, _, err := s.GetModule(ctx, auth); err != nil || !slices.Equal(got.MemberIDs, []uuid.UUID{a, b}) {
		t.Fatalf("Auth's members = %v, %v; want a again, and b", got.MemberIDs, err)
	}
}
