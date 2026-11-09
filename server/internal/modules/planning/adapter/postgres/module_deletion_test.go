package postgresadapter_test

import (
	"context"
	"errors"
	"maps"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
)

// SetModuleArchived writes a module's archive and its last change alone: what else the module it is given has
// changed does not reach the row; unarchived, its archive is none again.
func TestSetModuleArchived(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newWorld(t, pool)
	at := now.Truncate(time.Microsecond)
	m := domain.Module{ID: uuid.NewV7(), ProjectID: w.web, WorkspaceID: w.workspace, Name: "Auth", Status: domain.ModuleCompleted, MemberIDs: []uuid.UUID{},
		SortOrder: 5, CreatedAt: at, UpdatedAt: at, CreatedByID: &w.by, UpdatedByID: &w.by}
	if err := s.CreateModule(ctx, m); err != nil {
		t.Fatal(err)
	}
	archiver := account(t, pool, "arc")
	later := at.Add(time.Minute)
	archived := m
	archived.ArchivedAt, archived.UpdatedAt, archived.UpdatedByID = &later, later, &archiver
	moved := archived
	moved.Name, moved.SortOrder, moved.Status = "Other", 1, domain.ModulePlanned
	unarchived := archived
	unarchived.ArchivedAt = nil
	for _, tt := range []struct{ write, want domain.Module }{{moved, archived}, {unarchived, unarchived}} {
		if err := s.SetModuleArchived(ctx, tt.write); err != nil {
			t.Fatal(err)
		}
		if got, _, err := s.GetModule(ctx, m.ID); err != nil || jsonOf(t, got) != jsonOf(t, tt.want) {
			t.Fatalf("after SetModuleArchived: %s, %v\nwant %s", jsonOf(t, got), err, jsonOf(t, tt.want))
		}
	}
}

// audited are the rows of table in project by id, each as its deletion and last change, and the rest of each row,
// as text.
func audited(t *testing.T, pool *pgxpool.Pool, table string, project uuid.UUID) (stamps map[uuid.UUID]string, rest string) {
	t.Helper()
	rows, err := pool.Query(context.Background(), `SELECT id, concat_ws(' ', deleted_at, updated_at, updated_by_id) FROM `+table+
		` WHERE project_id = $1`, project)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	stamps = map[uuid.UUID]string{}
	for rows.Next() {
		var id uuid.UUID
		var s string
		if err := rows.Scan(&id, &s); err != nil {
			t.Fatal(err)
		}
		stamps[id] = s
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT coalesce(jsonb_agg(to_jsonb(r) - 'deleted_at' - 'updated_at' - 'updated_by_id' ORDER BY id),
		'[]')::text FROM `+table+` r WHERE project_id = $1`, project).Scan(&rest); err != nil {
		t.Fatal(err)
	}
	return stamps, rest
}

// A module's deletion (M6 P2 spec 2.4, R15; P3b spec 2.5): DeleteModule soft-deletes the module,
// DeleteModuleMemberRows its undeleted members' rows, DeleteModuleLinkRows its undeleted links,
// DeleteModulePreferenceRows its page's undeleted display settings' rows, each at the moment and by the account given;
// a member removed, a link deleted and display settings deleted before keep their own deletion, the other module's
// rows theirs, and every other column of every row is kept. Again later, nothing changes. The module and its links are
// read no more. Each statement answers a failed write as its failure.
func TestDeleteModuleWithItsRows(t *testing.T) {
	w := newLinksWorld(t)
	ctx := context.Background()
	at := now.Truncate(time.Microsecond)
	kept, gone := account(t, w.pool, "kep"), account(t, w.pool, "gon")
	for _, module := range []uuid.UUID{w.auth, w.other} {
		if err := w.s.InsertModuleMembers(ctx, app.MemberRows{WorkspaceID: w.workspace, ProjectID: w.web, ModuleID: module, IDs: sortedIDs(3),
			Accounts: []uuid.UUID{w.by, kept, gone}, By: w.by, At: at.Add(-time.Hour)}); err != nil {
			t.Fatal(err)
		}
		for i, url := range []string{"https://nerve.dev/a", "https://nerve.dev/b", "https://nerve.dev/c"} {
			l := w.link(module, url, at.Add(-time.Hour))
			if err := w.s.InsertModuleLink(ctx, l); err != nil {
				t.Fatal(err)
			}
			if i == 2 {
				if err := w.s.DeleteModuleLink(ctx, l.ID, at.Add(-time.Minute), kept); err != nil {
					t.Fatal(err)
				}
			}
		}
		for _, user := range []uuid.UUID{w.by, kept, gone} {
			exec(t, w.pool, `INSERT INTO module_user_properties (id, workspace_id, project_id, module_id, user_id, filters, created_by_id, updated_by_id,
				created_at, updated_at) VALUES ($1, $2, $3, $4, $5, '{"cycle_id":["none"]}', $5, $5, $6, $6)`, uuid.NewV7(), w.workspace, w.web, module,
				user, at.Add(-time.Hour))
		}
	}
	exec(t, w.pool, `UPDATE module_user_properties SET deleted_at = $1, updated_at = $1, updated_by_id = $2 WHERE module_id = $3 AND user_id = $4`,
		at.Add(-time.Minute), kept, w.auth, gone)
	if err := w.s.RemoveModuleMembers(ctx, w.auth, []uuid.UUID{gone}, at.Add(-time.Minute), kept); err != nil {
		t.Fatal(err)
	}
	deleter := account(t, w.pool, "del")
	tables := []string{"modules", "module_members", "module_links", "module_user_properties"}
	before, rests, live := map[string]map[uuid.UUID]string{}, map[string]string{}, map[string][]uuid.UUID{"modules": {w.auth}}
	for _, table := range tables {
		before[table], rests[table] = audited(t, w.pool, table, w.web)
		if table != "modules" {
			live[table] = queryIDs(t, w.pool, `SELECT id FROM `+table+` WHERE module_id = $1 AND deleted_at IS NULL`, w.auth)
			if len(live[table]) != 2 {
				t.Fatalf("Auth's live rows of %s: %v; want two", table, live[table])
			}
		}
	}
	steps := []func(context.Context, uuid.UUID, time.Time, uuid.UUID) error{w.s.DeleteModule, w.s.DeleteModuleMemberRows, w.s.DeleteModuleLinkRows,
		w.s.DeleteModulePreferenceRows}
	for _, when := range []time.Time{at, at.Add(time.Hour)} {
		for _, step := range steps {
			if err := step(ctx, w.auth, when, deleter); err != nil {
				t.Fatal(err)
			}
		}
	}
	var stamp string
	if err := w.pool.QueryRow(ctx, `SELECT concat_ws(' ', $1::timestamptz, $1::timestamptz, $2::uuid)`, at, deleter).Scan(&stamp); err != nil {
		t.Fatal(err)
	}
	for _, table := range tables {
		after, rest := audited(t, w.pool, table, w.web)
		want := maps.Clone(before[table])
		for _, id := range live[table] {
			want[id] = stamp
		}
		if !maps.Equal(after, want) || rest != rests[table] {
			t.Errorf("%s: stamps %v, rest %s\nwant %v, %s", table, after, rest, want, rests[table])
		}
	}
	if _, found, err := w.s.GetModule(ctx, w.auth); err != nil || found {
		t.Fatalf("a deleted module read: %t, %v", found, err)
	}
	if links, err := w.s.ListModuleLinks(ctx, w.auth); err != nil || len(links) != 0 {
		t.Fatalf("a deleted module's links: %v, %v", links, err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	for i, step := range steps {
		if err := step(cancelled, w.other, at, deleter); !errors.Is(err, context.Canceled) {
			t.Errorf("statement %d on a cancelled context = %v; want context.Canceled", i, err)
		}
	}
}

// queryIDs are the ids sql reads with args.
func queryIDs(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) []uuid.UUID {
	t.Helper()
	rows, err := pool.Query(context.Background(), sql, args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}
