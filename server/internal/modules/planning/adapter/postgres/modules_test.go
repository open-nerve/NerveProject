package postgresadapter_test

import (
	"context"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// account inserts an account named name.
func account(t *testing.T, pool *pgxpool.Pool, name string) uuid.UUID {
	t.Helper()
	id := uuid.NewV7()
	exec(t, pool, "INSERT INTO users (id, email, password, display_name) VALUES ($1, $2, 'x', $3)", id, name+"@corp.com", name)
	return id
}

// The undeleted module, archived or not, reads back with every field: its dates as days, each alone too, and the
// accounts of its undeleted members' rows in id order — not a member removed, nor another module's. A module without
// members has none; a deleted one, and one that is not there, is not found.
func TestGetModule(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newWorld(t, pool)
	at := now.Truncate(time.Microsecond)
	members := sortedIDs(3)
	for _, id := range members {
		exec(t, pool, "INSERT INTO users (id, email, password, display_name) VALUES ($1, $2, 'x', 'm')", id, id.String()+"@corp.com")
	}
	lead := account(t, pool, "lea")
	archived := at.Add(-time.Minute)
	want := domain.Module{ID: uuid.NewV7(), ProjectID: w.web, WorkspaceID: w.workspace, Name: "Auth", Description: "Sign in",
		StartDate: dayOf(t, "2026-11-02"), TargetDate: dayOf(t, "2026-11-13"), Status: domain.ModuleInProgress, LeadID: &lead,
		MemberIDs: []uuid.UUID{members[0], members[2]}, SortOrder: -9900.5, ArchivedAt: &archived,
		CreatedAt: at.Add(-time.Hour), UpdatedAt: at, CreatedByID: &w.by, UpdatedByID: &lead}
	exec(t, pool, `INSERT INTO modules (id, workspace_id, project_id, name, description, start_date, target_date, status, lead_id, sort_order,
		archived_at, created_at, updated_at, created_by_id, updated_by_id) VALUES ($1, $2, $3, 'Auth', 'Sign in', '2026-11-02', '2026-11-13',
		'in-progress', $4, -9900.5, $5, $6, $7, $8, $4)`, want.ID, w.workspace, w.web, lead, archived, want.CreatedAt, at, w.by)
	other := uuid.NewV7()
	exec(t, pool, `INSERT INTO modules (id, workspace_id, project_id, name, target_date) VALUES ($1, $2, $3, 'Other', '2026-12-01')`, other, w.workspace, w.web)
	member := func(module, account uuid.UUID, deleted *time.Time) {
		exec(t, pool, `INSERT INTO module_members (id, workspace_id, project_id, module_id, member_id, deleted_at) VALUES ($1, $2, $3, $4, $5, $6)`,
			uuid.NewV7(), w.workspace, w.web, module, account, deleted)
	}
	// Written out of id order; the removed member's row deleted, and another module's member.
	member(want.ID, members[2], nil)
	member(want.ID, members[1], &at)
	member(want.ID, members[0], nil)
	member(other, members[1], nil)
	got, found, err := s.GetModule(ctx, want.ID)
	if err != nil || !found || jsonOf(t, got) != jsonOf(t, want) {
		t.Fatalf("GetModule = %s, %t, %v\nwant %s", jsonOf(t, got), found, err, jsonOf(t, want))
	}
	got, found, err = s.GetModule(ctx, other)
	if err != nil || !found || got.Status != domain.ModulePlanned || got.StartDate != nil || got.TargetDate.String() != "2026-12-01" ||
		got.LeadID != nil || got.ArchivedAt != nil || len(got.MemberIDs) != 1 || got.MemberIDs[0] != members[1] {
		t.Fatalf("the other module = %+v, %t, %v", got, found, err)
	}
	alone := uuid.NewV7()
	exec(t, pool, `INSERT INTO modules (id, workspace_id, project_id, name) VALUES ($1, $2, $3, 'Alone')`, alone, w.workspace, w.ops)
	if got, found, err := s.GetModule(ctx, alone); err != nil || !found || len(got.MemberIDs) != 0 || got.ProjectID != w.ops {
		t.Fatalf("a module without members = %+v, %t, %v", got, found, err)
	}
	exec(t, pool, "UPDATE modules SET deleted_at = now() WHERE id = $1", other)
	if _, found, err := s.GetModule(ctx, other); err != nil || found {
		t.Fatalf("a deleted module read: %t, %v", found, err)
	}
	if _, found, err := s.GetModule(ctx, uuid.NewV7()); err != nil || found {
		t.Fatalf("a missing module read: %t, %v", found, err)
	}
}

// The undeleted module link reads back with its module, address, title — none or empty too — metadata and creation; a
// deleted one, and one that is not there, is not found.
func TestGetModuleLink(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newWorld(t, pool)
	at := now.Truncate(time.Microsecond)
	module := uuid.NewV7()
	exec(t, pool, `INSERT INTO modules (id, workspace_id, project_id, name) VALUES ($1, $2, $3, 'Auth')`, module, w.workspace, w.web)
	title, empty, favicon := "Spec", "", "https://nerve.dev/favicon.ico"
	links := []domain.ModuleLink{
		{ID: uuid.NewV7(), WorkspaceID: w.workspace, ProjectID: w.web, ModuleID: module, URL: "https://nerve.dev/spec", Title: &title,
			Metadata: shared.LinkMetadata{"title": &title, "favicon": &favicon, "error": nil}, CreatedByID: &w.by, CreatedAt: at.Add(-time.Hour), UpdatedAt: at},
		{ID: uuid.NewV7(), WorkspaceID: w.workspace, ProjectID: w.web, ModuleID: module, URL: "http://nerve.dev", Title: &empty,
			Metadata: shared.LinkMetadata{}, CreatedAt: at, UpdatedAt: at},
		{ID: uuid.NewV7(), WorkspaceID: w.workspace, ProjectID: w.web, ModuleID: module, URL: "http://nerve.dev/none",
			Metadata: shared.LinkMetadata{}, CreatedAt: at, UpdatedAt: at},
	}
	for _, l := range links {
		exec(t, pool, `INSERT INTO module_links (id, workspace_id, project_id, module_id, url, title, metadata, created_by_id, updated_by_id,
			created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8, $9, $10)`, l.ID, l.WorkspaceID, l.ProjectID, l.ModuleID, l.URL, l.Title,
			jsonOf(t, l.Metadata), l.CreatedByID, l.CreatedAt, l.UpdatedAt)
		got, found, err := s.GetModuleLink(ctx, l.ID)
		if err != nil || !found || jsonOf(t, got) != jsonOf(t, l) {
			t.Fatalf("GetModuleLink = %s, %t, %v\nwant %s", jsonOf(t, got), found, err, jsonOf(t, l))
		}
	}
	exec(t, pool, "UPDATE module_links SET deleted_at = now() WHERE id = $1", links[0].ID)
	if _, found, err := s.GetModuleLink(ctx, links[0].ID); err != nil || found {
		t.Fatalf("a deleted link read: %t, %v", found, err)
	}
	if _, found, err := s.GetModuleLink(ctx, uuid.NewV7()); err != nil || found {
		t.Fatalf("a missing link read: %t, %v", found, err)
	}
}

// The project's undeleted modules: the unarchived ones by sort order, then creation, then id; the archived ones alone,
// the latest archived first, then by id. Another project's and deleted ones are in neither. Each reads as GetModule
// reads it, its undeleted members in id order.
func TestListModules(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newWorld(t, pool)
	at := now.Truncate(time.Microsecond)
	ids := sortedIDs(7)
	add := func(id, project uuid.UUID, sortOrder float64, created time.Time) {
		t.Helper()
		if err := s.CreateModule(ctx, domain.Module{ID: id, ProjectID: project, WorkspaceID: w.workspace, Name: id.String(),
			Status: domain.ModulePlanned, SortOrder: sortOrder, CreatedAt: created, UpdatedAt: created}); err != nil {
			t.Fatal(err)
		}
	}
	// ids[2] and ids[1] tie on sort order and creation: by id. ids[0] ties on sort order, created later.
	add(ids[2], w.web, 100, at)
	add(ids[1], w.web, 100, at)
	add(ids[0], w.web, 100, at.Add(time.Second))
	add(ids[3], w.web, -5, at.Add(time.Hour))
	add(ids[4], w.web, 1, at)
	add(ids[5], w.web, 2, at)
	add(ids[6], w.ops, 0, at)
	deleted := uuid.NewV7()
	add(deleted, w.web, 0, at)
	exec(t, pool, "UPDATE modules SET deleted_at = now() WHERE id = $1", deleted)
	// ids[4] and ids[5] archived at one moment, ids[1] later.
	exec(t, pool, "UPDATE modules SET archived_at = $2 WHERE id = ANY($1)", []uuid.UUID{ids[4], ids[5]}, at)
	exec(t, pool, "UPDATE modules SET archived_at = $2 WHERE id = $1", ids[1], at.Add(time.Minute))
	// ids[3]'s members, written out of id order, one removed; ids[1]'s one.
	members := sortedIDs(3)
	for _, id := range members {
		exec(t, pool, "INSERT INTO users (id, email, password, display_name) VALUES ($1, $2, 'x', 'm')", id, id.String()+"@corp.com")
	}
	member := func(module, account uuid.UUID, deleted *time.Time) {
		exec(t, pool, `INSERT INTO module_members (id, workspace_id, project_id, module_id, member_id, deleted_at) VALUES ($1, $2, $3, $4, $5, $6)`,
			uuid.NewV7(), w.workspace, w.web, module, account, deleted)
	}
	member(ids[3], members[2], nil)
	member(ids[3], members[1], &at)
	member(ids[3], members[0], nil)
	member(ids[1], members[1], nil)
	for _, tt := range []struct {
		archived bool
		want     []uuid.UUID
	}{{false, []uuid.UUID{ids[3], ids[2], ids[0]}}, {true, []uuid.UUID{ids[1], ids[4], ids[5]}}} {
		got, err := s.ListModules(ctx, w.web, tt.archived)
		if err != nil {
			t.Fatal(err)
		}
		var gotIDs []uuid.UUID
		for _, m := range got {
			gotIDs = append(gotIDs, m.ID)
			one, found, err := s.GetModule(ctx, m.ID)
			if err != nil || !found || jsonOf(t, one) != jsonOf(t, m) {
				t.Errorf("listed %s, read %s, %t, %v", jsonOf(t, m), jsonOf(t, one), found, err)
			}
		}
		if !slices.Equal(gotIDs, tt.want) {
			t.Errorf("archived=%t: %v, want %v", tt.archived, gotIDs, tt.want)
		}
		if want := []uuid.UUID{members[0], members[2]}; !tt.archived && (len(got) == 0 || !slices.Equal(got[0].MemberIDs, want)) {
			t.Errorf("the first listed = %s, want %s with the members %v", jsonOf(t, got), ids[3], want)
		}
	}
	if got, err := s.ListModules(ctx, uuid.NewV7(), false); err != nil || len(got) != 0 {
		t.Fatalf("a project without modules: %v, %v", got, err)
	}
}
