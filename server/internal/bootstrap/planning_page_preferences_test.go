package bootstrap

import (
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// pagePreferences are the accounts' display settings of the page of a cycle
// or a module, target: table's rows, whose column names it, its parent a row
// of parent.
type pagePreferences struct {
	table, parent, column string
	target                uuid.UUID
}

func cyclePreferences(id uuid.UUID) pagePreferences {
	return pagePreferences{table: "cycle_user_properties", parent: "cycles", column: "cycle_id", target: id}
}

func modulePreferences(id uuid.UUID) pagePreferences {
	return pagePreferences{table: "module_user_properties", parent: "modules", column: "module_id", target: id}
}

// seed writes, directly, a row of the display settings of the page for each
// account named in live, and for each named in gone one deleted and last
// changed by alice an hour ago: fixtures, a row no change through
// /me/cycles|modules/{id}/preferences writes (M6 P3b spec 2.3), a deleted
// one or one last changed by another, among them.
func (p pagePreferences) seed(t *testing.T, w issueWorld, live, gone []string) {
	t.Helper()
	deleted := time.Now().Add(-time.Hour)
	for _, name := range live {
		p.insert(t, w, name, name, nil)
	}
	for _, name := range gone {
		p.insert(t, w, name, "alice", &deleted)
	}
}

// insert writes the account's row of the page, last changed by changer, and
// deleted then at deleted when it is not nil.
func (p pagePreferences) insert(t *testing.T, w issueWorld, account, changer string, deleted *time.Time) {
	t.Helper()
	tag, err := w.pool.Exec(pgtest.Soon(t), `INSERT INTO `+p.table+` (id, workspace_id, project_id, `+p.column+`, user_id, filters, created_by_id,
		updated_by_id, created_at, updated_at, deleted_at)
		SELECT $1::uuid, workspace_id, project_id, id, $2::uuid, '{"priority":["urgent"]}', $2::uuid, $3::uuid, coalesce($4::timestamptz, now()),
			coalesce($4::timestamptz, now()), $4::timestamptz FROM `+p.parent+` WHERE id = $5`,
		uuid.NewV7(), w.ids[account], w.ids[changer], deleted, p.target)
	if err != nil || tag.RowsAffected() != 1 {
		t.Fatalf("seeding %s's %s of %s: %d rows, %v; want 1", account, p.table, p.target, tag.RowsAffected(), err)
	}
}

// stamps are the page's rows in id order, each "live", or its deletion and
// last change.
func (p pagePreferences) stamps(t *testing.T, w issueWorld) []string {
	t.Helper()
	var out []string
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT coalesce(array_agg(CASE WHEN deleted_at IS NULL THEN 'live'
		ELSE concat_ws(' ', deleted_at, updated_at, updated_by_id) END ORDER BY id), '{}') FROM `+p.table+` WHERE `+p.column+` = $1`,
		p.target).Scan(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

// rows are the page's rows, each whole, as JSON in id order.
func (p pagePreferences) rows(t *testing.T, w issueWorld) string {
	t.Helper()
	var out string
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT coalesce(jsonb_agg(to_jsonb(r) ORDER BY id), '[]')::text FROM `+p.table+` r WHERE `+p.column+` = $1`,
		p.target).Scan(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

// wantDeletedWith checks that the page's rows live in before, its stamps
// before its target's deletion, were deleted with it, at at by by (M6 P3b
// spec 2.5), and that each deleted before kept its deletion.
func (p pagePreferences) wantDeletedWith(t *testing.T, w issueWorld, before []string, at time.Time, by uuid.UUID) {
	t.Helper()
	var stamp string
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT concat_ws(' ', $1::timestamptz, $1::timestamptz, $2::uuid)`, at, by).Scan(&stamp); err != nil {
		t.Fatal(err)
	}
	want := slices.Clone(before)
	for i, s := range want {
		if s == "live" {
			want[i] = stamp
		}
	}
	if got := p.stamps(t, w); !slices.Equal(got, want) {
		t.Errorf("%s of %s after its deletion: %q; want %q", p.table, p.target, got, want)
	}
}
