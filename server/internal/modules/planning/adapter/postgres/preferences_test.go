package postgresadapter_test

import (
	"context"
	"errors"
	"maps"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// A cycle's deletion (M6 P3b spec 2.5): DeleteCyclePreferenceRows soft-deletes the deleted cycle's undeleted display
// settings' rows, each account's of its page, at the moment and by the account given; a row deleted before keeps its
// own deletion, the other cycle's rows theirs, and every other column of every row is kept. Again later, nothing
// changes. The statement answers a failed write as its failure.
func TestDeleteCyclePreferenceRows(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newWorld(t, pool)
	at := now.Truncate(time.Microsecond)
	sprint, other := uuid.NewV7(), uuid.NewV7()
	kept, gone := account(t, pool, "kep"), account(t, pool, "gon")
	for _, c := range []uuid.UUID{sprint, other} {
		cycle(t, pool, w, w.web, c, nil)
		for _, user := range []uuid.UUID{w.by, kept, gone} {
			exec(t, pool, `INSERT INTO cycle_user_properties (id, workspace_id, project_id, cycle_id, user_id, filters, created_by_id, updated_by_id,
				created_at, updated_at) VALUES ($1, $2, $3, $4, $5, '{"priority":["urgent"]}', $5, $5, $6, $6)`, uuid.NewV7(), w.workspace, w.web, c, user,
				at.Add(-time.Hour))
		}
	}
	exec(t, pool, `UPDATE cycle_user_properties SET deleted_at = $1, updated_at = $1, updated_by_id = $2 WHERE cycle_id = $3 AND user_id = $4`,
		at.Add(-time.Minute), kept, sprint, gone)
	before, rest := audited(t, pool, "cycle_user_properties", w.web)
	live := queryIDs(t, pool, `SELECT id FROM cycle_user_properties WHERE cycle_id = $1 AND deleted_at IS NULL`, sprint)
	if len(before) != 6 || len(live) != 2 {
		t.Fatalf("rows %v, Sprint's live %v; want six, two of them Sprint's live", before, live)
	}
	deleter := account(t, pool, "del")
	for _, when := range []time.Time{at, at.Add(time.Hour)} {
		if err := s.DeleteCyclePreferenceRows(ctx, sprint, when, deleter); err != nil {
			t.Fatal(err)
		}
	}
	var stamp string
	if err := pool.QueryRow(ctx, `SELECT concat_ws(' ', $1::timestamptz, $1::timestamptz, $2::uuid)`, at, deleter).Scan(&stamp); err != nil {
		t.Fatal(err)
	}
	want := maps.Clone(before)
	for _, id := range live {
		want[id] = stamp
	}
	if after, afterRest := audited(t, pool, "cycle_user_properties", w.web); !maps.Equal(after, want) || afterRest != rest {
		t.Errorf("stamps %v, rest %s\nwant %v, %s", after, afterRest, want, rest)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if err := s.DeleteCyclePreferenceRows(cancelled, other, at, deleter); !errors.Is(err, context.Canceled) {
		t.Errorf("on a cancelled context = %v; want context.Canceled", err)
	}
}

// pageStore is a page's display settings as the store reads and writes them: the cycle's page's or the module's.
type pageStore[F domain.PageFilters] struct {
	table, column string
	read          func(s *postgresadapter.Store, ctx context.Context, page, user uuid.UUID) (domain.Preferences[F], bool, error)
	upsert        func(s *postgresadapter.Store, ctx context.Context, c app.PreferencesChange[F]) (domain.Preferences[F], error)
	filters       F // the page's filters, the other page's key among them and an empty array
	stored        string
}

func cyclePageStore() pageStore[domain.CycleFilters] {
	return pageStore[domain.CycleFilters]{table: "cycle_user_properties", column: "cycle_id",
		read: (*postgresadapter.Store).CyclePreferences, upsert: (*postgresadapter.Store).UpsertCyclePreferences,
		filters: domain.CycleFilters{ListFilters: shared.ListFilters{Priority: []string{"high"}, LabelID: []string{}}, ModuleID: []string{"none"}},
		stored:  `{"label_id": [], "priority": ["high"], "module_id": ["none"]}`}
}

func modulePageStore() pageStore[domain.ModuleFilters] {
	return pageStore[domain.ModuleFilters]{table: "module_user_properties", column: "module_id",
		read: (*postgresadapter.Store).ModulePreferences, upsert: (*postgresadapter.Store).UpsertModulePreferences,
		filters: domain.ModuleFilters{ListFilters: shared.ListFilters{Priority: []string{"high"}, LabelID: []string{}}, CycleID: []string{"none"}},
		stored:  `{"cycle_id": ["none"], "label_id": [], "priority": ["high"]}`}
}

// page inserts a live cycle, or module, of Web and answers its id.
func (p pageStore[F]) page(t *testing.T, pool *pgxpool.Pool, w world) uuid.UUID {
	t.Helper()
	id := uuid.NewV7()
	if p.table == "cycle_user_properties" {
		cycle(t, pool, w, w.web, id, nil)
	} else {
		exec(t, pool, `INSERT INTO modules (id, workspace_id, project_id, name, created_by_id, updated_by_id) VALUES ($1, $2, $3, $4, $5, $5)`,
			id, w.workspace, w.web, id.String(), w.by)
	}
	return id
}

// row inserts the account's row of the page with no value but the keys, every other column its default, deleted at
// deleted when it is not nil, and answers its id.
func (p pageStore[F]) row(t *testing.T, pool *pgxpool.Pool, w world, page, user uuid.UUID, deleted *time.Time) uuid.UUID {
	t.Helper()
	id := uuid.NewV7()
	exec(t, pool, `INSERT INTO `+p.table+` (id, workspace_id, project_id, `+p.column+`, user_id, deleted_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		id, w.workspace, w.web, page, user, deleted)
	return id
}

// columns is the row id of the page's table, each column as JSON text.
func (p pageStore[F]) columns(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) map[string]string {
	t.Helper()
	rows, err := pool.Query(context.Background(), "SELECT key, value::text FROM "+p.table+" r, jsonb_each(to_jsonb(r)) WHERE r.id = $1", id)
	if err != nil {
		t.Fatal(err)
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) ([2]string, error) {
		var kv [2]string
		return kv, r.Scan(&kv[0], &kv[1])
	})
	if err != nil || len(out) == 0 {
		t.Fatalf("the row %s of %s: %v", id, p.table, err)
	}
	m := map[string]string{}
	for _, kv := range out {
		m[kv[0]] = kv[1]
	}
	return m
}

// changed is before with the columns of change changed, as JSON text.
func changed(before, change map[string]string) map[string]string {
	out := maps.Clone(before)
	maps.Copy(out, change)
	return out
}

// A row with every column at its default reads as domain.DefaultPreferences: the defaults the use case answers while
// there is no row are the columns' (M6 P3b spec 2.3). The settings a row stores read back whole, the other page's key
// and an empty array among the filters. The account's undeleted row of the page alone is read: none for another
// account, a deleted row, or another page; reading writes nothing.
func TestReadPagePreferences(t *testing.T) {
	t.Run("cycle", func(t *testing.T) { readPagePreferences(t, cyclePageStore()) })
	t.Run("module", func(t *testing.T) { readPagePreferences(t, modulePageStore()) })
}

func readPagePreferences[F domain.PageFilters](t *testing.T, p pageStore[F]) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newWorld(t, pool)
	sprint, other := p.page(t, pool, w), p.page(t, pool, w)
	carol, alice := account(t, pool, "car"), account(t, pool, "ali")
	p.row(t, pool, w, sprint, w.by, nil)
	stored := p.row(t, pool, w, other, w.by, nil)
	exec(t, pool, `UPDATE `+p.table+` SET filters = $2, display_filters = jsonb_set(display_filters, '{group_by}', '"cycle_id"'),
		display_properties = jsonb_set(display_properties, '{modules}', 'false') WHERE id = $1`, stored, p.stored)
	p.row(t, pool, w, sprint, carol, &now)
	before := tableRows(t, pool, p.table, w.web)
	display, properties := shared.DefaultListDisplayFilters(), shared.DefaultListDisplayProperties()
	display.GroupBy, properties.Modules = new("cycle_id"), false
	for _, tt := range []struct {
		name       string
		page, user uuid.UUID
		want       domain.Preferences[F]
		found      bool
	}{
		{"defaults", sprint, w.by, domain.DefaultPreferences[F](), true},
		{"stored", other, w.by, domain.Preferences[F]{Filters: p.filters, DisplayFilters: display, DisplayProperties: properties}, true},
		{"deleted", sprint, carol, domain.Preferences[F]{}, false},
		{"no row", sprint, alice, domain.Preferences[F]{}, false},
		{"no page", uuid.NewV7(), w.by, domain.Preferences[F]{}, false},
	} {
		got, found, err := p.read(s, ctx, tt.page, tt.user)
		if err != nil || found != tt.found || !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: %+v, %v, %v; want %+v, %v", tt.name, got, found, err, tt.want, tt.found)
		}
	}
	if after := tableRows(t, pool, p.table, w.web); after != before {
		t.Errorf("rows after reading\n%s\nwant\n%s", after, before)
	}
}

// The first change of an account's settings on a page inserts his row: the defaults with the change applied, of the
// page's workspace, project and cycle or module, made and last changed by him at the change's moment. A later change
// replaces the fields it gives, each whole, and the audit columns, his and its moment even when another account made
// the row; an empty change only the audit columns; cleared filters are stored as {}, an empty array as []. Each field
// alone is honored on the insert too. A deleted row of his does not count, and every other row keeps every column:
// another account's on the page, his deleted one and his on another page. The answer is the settings as stored.
func TestUpsertPagePreferences(t *testing.T) {
	t.Run("cycle", func(t *testing.T) { upsertPagePreferences(t, cyclePageStore()) })
	t.Run("module", func(t *testing.T) { upsertPagePreferences(t, modulePageStore()) })
}

func upsertPagePreferences[F domain.PageFilters](t *testing.T, p pageStore[F]) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newWorld(t, pool)
	sprint, other := p.page(t, pool, w), p.page(t, pool, w)
	alice := account(t, pool, "ali")
	p.row(t, pool, w, sprint, w.by, &now)
	p.row(t, pool, w, other, w.by, nil)
	p.row(t, pool, w, sprint, alice, nil)
	others := tableRows(t, pool, p.table, w.web)
	at := now.Truncate(time.Microsecond)
	stamp := func(at time.Time) string { return `"` + at.UTC().Format("2006-01-02T15:04:05.999999") + `+00:00"` }
	upsert := func(patch domain.PreferencesPatch[F], at time.Time, user uuid.UUID) domain.Preferences[F] {
		t.Helper()
		got, err := p.upsert(s, ctx, app.PreferencesChange[F]{ID: uuid.NewV7(), WorkspaceID: w.workspace, ProjectID: w.web, Page: sprint, UserID: user,
			Patch: patch, Now: at})
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	var id uuid.UUID
	row := func() map[string]string {
		t.Helper()
		if err := pool.QueryRow(ctx, "SELECT id FROM "+p.table+" WHERE "+p.column+" = $1 AND user_id = $2 AND deleted_at IS NULL", sprint, w.by).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return p.columns(t, pool, id)
	}

	want := domain.DefaultPreferences[F]().Apply(domain.PreferencesPatch[F]{Filters: &p.filters})
	if got := upsert(domain.PreferencesPatch[F]{Filters: &p.filters}, at, w.by); !reflect.DeepEqual(got, want) {
		t.Errorf("the first change = %+v; want the defaults with its filters", got)
	}
	first := row()
	by := `"` + w.by.String() + `"`
	for k, v := range map[string]string{"workspace_id": `"` + w.workspace.String() + `"`, "project_id": `"` + w.web.String() + `"`,
		p.column: `"` + sprint.String() + `"`, "user_id": by, "created_by_id": by, "updated_by_id": by, "created_at": stamp(at), "updated_at": stamp(at),
		"deleted_at": "null", "filters": p.stored,
		"display_filters":    `{"layout": "list", "calendar": {"layout": "month", "show_weekends": false}, "group_by": null, "order_by": "-created_at", "sub_issue": true, "sub_group_by": null, "show_empty_groups": true}`,
		"display_properties": `{"key": true, "link": true, "cycle": true, "state": true, "labels": true, "modules": true, "assignee": true, "due_date": true, "priority": true, "created_on": true, "start_date": true, "updated_on": true, "sub_issue_count": true, "attachment_count": true}`,
	} {
		if first[k] != v {
			t.Errorf("the inserted row's %s = %s, want %s", k, first[k], v)
		}
	}

	// A row another account made and changed last: the next change is his.
	exec(t, pool, "UPDATE "+p.table+" SET created_by_id = $2, updated_by_id = $2 WHERE id = $1", id, alice)
	before := row()
	display := shared.ListDisplayFilters{Layout: "kanban", GroupBy: new("module_id"), OrderBy: "-cycle_name", Calendar: shared.ListCalendar{Layout: "week", ShowWeekends: true}}
	want.DisplayFilters = display
	if got := upsert(domain.PreferencesPatch[F]{DisplayFilters: &display}, at.Add(time.Hour), w.by); !reflect.DeepEqual(got, want) {
		t.Errorf("a change of the display filters = %+v; want %+v", got, want)
	}
	if got, wantRow := row(), changed(before, map[string]string{"updated_at": stamp(at.Add(time.Hour)), "updated_by_id": by,
		"display_filters": `{"layout": "kanban", "calendar": {"layout": "week", "show_weekends": true}, "group_by": "module_id", "order_by": "-cycle_name", "sub_issue": false, "sub_group_by": null, "show_empty_groups": false}`}); !maps.Equal(got, wantRow) {
		t.Errorf("after a change of the display filters: %v\nwant %v", got, wantRow)
	}
	properties := shared.ListDisplayProperties{Cycle: true}
	want.DisplayProperties = properties
	if got := upsert(domain.PreferencesPatch[F]{DisplayProperties: &properties}, at.Add(2*time.Hour), w.by); !reflect.DeepEqual(got, want) {
		t.Errorf("a change of the display properties = %+v; want %+v", got, want)
	}
	var none F
	want.Filters = none
	if got := upsert(domain.PreferencesPatch[F]{Filters: &none}, at.Add(3*time.Hour), w.by); !reflect.DeepEqual(got, want) || row()["filters"] != "{}" {
		t.Errorf("cleared filters = %+v, stored %s; want %+v, {}", got, row()["filters"], want)
	}
	before = row()
	if got := upsert(domain.PreferencesPatch[F]{}, at.Add(4*time.Hour), w.by); !reflect.DeepEqual(got, want) {
		t.Errorf("an empty change = %+v; want the settings as they were", got)
	}
	if got, wantRow := row(), changed(before, map[string]string{"updated_at": stamp(at.Add(4 * time.Hour))}); !maps.Equal(got, wantRow) {
		t.Errorf("after an empty change: %v\nwant %v", got, wantRow)
	}
	var after string
	if err := pool.QueryRow(ctx, "SELECT coalesce(jsonb_agg(to_jsonb(r) ORDER BY id), '[]')::text FROM "+p.table+" r WHERE project_id = $1 AND id <> $2",
		w.web, id).Scan(&after); err != nil || after != others {
		t.Errorf("the other rows: %v\n%s\nwant\n%s", err, after, others)
	}

	// Each field alone on the insert.
	for i, patch := range []domain.PreferencesPatch[F]{{DisplayFilters: &display}, {DisplayProperties: &properties}} {
		user := account(t, pool, []string{"fir", "sec"}[i])
		if got, want := upsert(patch, at, user), domain.DefaultPreferences[F]().Apply(patch); !reflect.DeepEqual(got, want) {
			t.Errorf("a first change of field %d alone = %+v; want %+v", i, got, want)
		}
	}
}
