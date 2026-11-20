package postgresadapter_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/navigation/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

var (
	_ app.ViewReader = (*postgresadapter.Store)(nil)
	_ app.ViewLocker = (*postgresadapter.Store)(nil)
	_ app.ViewWriter = (*postgresadapter.Store)(nil)
)

func ptr[T any](v T) *T { return &v }

// views is newWorld with another account, another project of the workspace, and another workspace with a project.
type views struct {
	world
	other, ops, elsewhere, elsewhereProject uuid.UUID
}

func newViews(t *testing.T, pool *pgxpool.Pool) views {
	t.Helper()
	v := views{world: newWorld(t, pool), other: uuid.NewV7(), ops: uuid.NewV7(), elsewhere: uuid.NewV7(), elsewhereProject: uuid.NewV7()}
	exec(t, pool, "INSERT INTO users (id, email, password, display_name) VALUES ($1, 'ann@corp.com', 'x', 'ann')", v.other)
	exec(t, pool, "INSERT INTO projects (id, workspace_id, name, identifier) VALUES ($1, $2, 'Ops', 'OPS')", v.ops, v.workspace)
	exec(t, pool, "INSERT INTO workspaces (id, name, slug) VALUES ($1, 'Beta', 'beta')", v.elsewhere)
	exec(t, pool, "INSERT INTO projects (id, workspace_id, name, identifier) VALUES ($1, $2, 'Web', 'WEB')", v.elsewhereProject, v.elsewhere)
	return v
}

// view is a view numbered n of workspace and project (nil for the workspace's), owned by owner, made at at with the
// shared defaults: what a creation stores.
func view(n int, workspace uuid.UUID, project *uuid.UUID, owner uuid.UUID, at time.Time) domain.View {
	return domain.View{ID: numberedID(n), WorkspaceID: workspace, ProjectID: project, OwnedByID: owner, CreatedAt: at, UpdatedAt: at,
		Content: domain.Content{Name: "View", DisplayFilters: shared.DefaultListDisplayFilters(), DisplayProperties: shared.DefaultListDisplayProperties()}}
}

func insert(t *testing.T, s *postgresadapter.Store, vs ...domain.View) {
	t.Helper()
	for _, v := range vs {
		if err := s.InsertView(context.Background(), v, v.OwnedByID); err != nil {
			t.Fatal(err)
		}
	}
}

func ids(vs []domain.View) []uuid.UUID {
	out := make([]uuid.UUID, len(vs))
	for i, v := range vs {
		out[i] = v.ID
	}
	return out
}

// A view's content is stored and read back as written (spec 2.3): its icon, each filter, an explicitly empty one
// apart from an omitted one, its display filters with their grouping and its properties, with its kind, owner and
// moments; a creation's writer stamps both audit columns. A change replaces the content whole and stamps its moment
// and writer, keeping the creation.
func TestAViewRoundTrips(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newViews(t, pool)
	project := view(10, w.workspace, &w.project, w.owner, now.Add(-time.Hour))
	project.Content = domain.Content{Name: "Triage", Description: "plain\ntext",
		LogoProps: shared.LogoProps{InUse: ptr(shared.LogoInUseEmoji), Emoji: &shared.LogoEmoji{Value: ptr("128512")}, Icon: &shared.LogoIcon{Name: ptr("star")}},
		Filters: domain.Filters{ListFilters: shared.ListFilters{Priority: []string{"urgent", "high"}, LabelID: []string{}, TargetDate: []string{"2026-11-01;after"}},
			CycleID: []string{uuid.NewV7().String()}, ModuleID: []string{}},
		DisplayFilters: shared.ListDisplayFilters{Layout: "kanban", GroupBy: ptr("state"), SubGroupBy: ptr("priority"), OrderBy: "sort_order",
			ShowEmptyGroups: true, Calendar: shared.ListCalendar{Layout: "week", ShowWeekends: true}},
		DisplayProperties: shared.ListDisplayProperties{Key: true, Link: true}}
	workspace := view(20, w.workspace, nil, w.other, now.Add(-time.Minute))
	workspace.Filters = domain.Filters{ProjectID: []string{w.project.String()}, ListFilters: shared.ListFilters{StateGroup: []string{}}}
	workspace.DisplayFilters.Layout = "spreadsheet"
	insert(t, s, project, workspace)
	for _, want := range []domain.View{project, workspace} {
		got, found, err := s.View(ctx, want.ID)
		if err != nil || !found || !reflect.DeepEqual(got, want) {
			t.Fatalf("read %+v, %v, %v; want %+v", got, found, err, want)
		}
	}
	var filters, by string
	if err := pool.QueryRow(ctx, "SELECT filters::text, concat_ws(',', created_by_id, updated_by_id) FROM issue_views WHERE id = $1", project.ID).
		Scan(&filters, &by); err != nil {
		t.Fatal(err)
	}
	if want := `{"cycle_id": ["` + project.Filters.CycleID[0] + `"], "label_id": [], "priority": ["urgent", "high"], "module_id": [], "target_date": ["2026-11-01;after"]}`; filters != want {
		t.Fatalf("filters stored %s; want %s", filters, want)
	}
	if by != w.owner.String()+","+w.owner.String() {
		t.Fatalf("made by %s; want the creator twice", by)
	}
	changed := project
	changed.Content = domain.Content{Name: "Renamed", DisplayFilters: shared.DefaultListDisplayFilters(), DisplayProperties: shared.DefaultListDisplayProperties()}
	changed.UpdatedAt = now
	if err := s.UpdateView(ctx, changed, w.other); err != nil {
		t.Fatal(err)
	}
	if got, found, err := s.View(ctx, project.ID); err != nil || !found || !reflect.DeepEqual(got, changed) {
		t.Fatalf("changed %+v, %v, %v; want %+v", got, found, err, changed)
	}
	if err := pool.QueryRow(ctx, "SELECT concat_ws(',', created_by_id, updated_by_id) FROM issue_views WHERE id = $1", project.ID).Scan(&by); err != nil ||
		by != w.owner.String()+","+w.other.String() {
		t.Fatalf("changed by %s, %v; want made by the owner, changed by the writer", by, err)
	}
}

// The two lists (spec 2.4): the project's undeleted views, and the workspace's undeleted views without a project,
// each the newest first and, at one moment, by id descending; the scope of its partial index alone, so the project's
// list has no workspace view nor another project's, and the workspace's list no project view nor another workspace's.
// The order is the query's whatever plan reads it: each list is read as planned, which may walk its index in order, and
// again with index scans off, the rows reaching the sort as written. The views at one moment are written in neither id
// order, so an order by the moment alone fails here.
func TestTheListsNewestFirstThenByID(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newViews(t, pool)
	older, at := now.Add(-time.Hour), now.Add(-time.Minute)
	insert(t, s,
		view(20, w.workspace, &w.project, w.owner, at), view(10, w.workspace, &w.project, w.other, at), view(30, w.workspace, &w.project, w.owner, at),
		view(5, w.workspace, &w.project, w.owner, older), view(40, w.workspace, &w.project, w.owner, now.Add(-2*time.Hour)),
		view(50, w.workspace, &w.ops, w.owner, now), view(60, w.elsewhere, &w.elsewhereProject, w.owner, now),
		view(120, w.workspace, nil, w.owner, at), view(110, w.workspace, nil, w.other, at), view(130, w.workspace, nil, w.owner, at),
		view(105, w.workspace, nil, w.owner, older), view(140, w.workspace, nil, w.owner, now.Add(-2*time.Hour)),
		view(150, w.elsewhere, nil, w.owner, now))
	for _, id := range []uuid.UUID{numberedID(40), numberedID(140)} {
		if err := s.DeleteView(ctx, id, now, w.owner); err != nil {
			t.Fatal(err)
		}
	}
	for _, plan := range []string{"as planned", "without index scans"} {
		err := postgres.NewTxManager(pool, time.Second).WithinTx(ctx, func(txctx context.Context) error {
			if plan == "without index scans" {
				if _, err := postgres.DB(txctx, pool).Exec(txctx,
					"SELECT set_config('enable_indexscan', 'off', true), set_config('enable_bitmapscan', 'off', true)"); err != nil {
					return err
				}
			}
			projectViews, err := s.ProjectViews(txctx, w.project, nil)
			if want := []uuid.UUID{numberedID(30), numberedID(20), numberedID(10), numberedID(5)}; err != nil || !slices.Equal(ids(projectViews), want) {
				t.Errorf("%s: project's list %v, %v; want %v", plan, ids(projectViews), err, want)
			}
			workspaceViews, err := s.WorkspaceViews(txctx, w.workspace, nil)
			if want := []uuid.UUID{numberedID(130), numberedID(120), numberedID(110), numberedID(105)}; err != nil || !slices.Equal(ids(workspaceViews), want) {
				t.Errorf("%s: workspace's list %v, %v; want %v", plan, ids(workspaceViews), err, want)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if got, err := s.ProjectViews(ctx, w.elsewhereProject, nil); err != nil || !slices.Equal(ids(got), []uuid.UUID{numberedID(60)}) {
		t.Fatalf("another project's list %v, %v", ids(got), err)
	}
	if got, err := s.WorkspaceViews(ctx, uuid.NewV7(), nil); err != nil || got == nil || len(got) != 0 {
		t.Fatalf("no views %#v, %v; want an empty list", got, err)
	}
}

// An OwnOnly grant's list has only the views it owns (spec 2.2), the rest of the list as for anyone: in SQL, so the
// list is never read whole and filtered after.
func TestAListOfOnesOwnViews(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newViews(t, pool)
	insert(t, s, view(1, w.workspace, &w.project, w.owner, now), view(2, w.workspace, &w.project, w.other, now), view(3, w.workspace, &w.project, w.owner, now),
		view(11, w.workspace, nil, w.other, now), view(12, w.workspace, nil, w.owner, now))
	for _, tt := range []struct {
		name  string
		list  func(*uuid.UUID) ([]domain.View, error)
		owner uuid.UUID
		want  []uuid.UUID
	}{
		{"project, the owner", func(o *uuid.UUID) ([]domain.View, error) { return s.ProjectViews(ctx, w.project, o) }, w.owner, []uuid.UUID{numberedID(3), numberedID(1)}},
		{"project, the other", func(o *uuid.UUID) ([]domain.View, error) { return s.ProjectViews(ctx, w.project, o) }, w.other, []uuid.UUID{numberedID(2)}},
		{"workspace, the owner", func(o *uuid.UUID) ([]domain.View, error) { return s.WorkspaceViews(ctx, w.workspace, o) }, w.owner, []uuid.UUID{numberedID(12)}},
		{"workspace, the other", func(o *uuid.UUID) ([]domain.View, error) { return s.WorkspaceViews(ctx, w.workspace, o) }, w.other, []uuid.UUID{numberedID(11)}},
	} {
		if got, err := tt.list(&tt.owner); err != nil || !slices.Equal(ids(got), tt.want) {
			t.Errorf("%s: %v, %v; want %v", tt.name, ids(got), err, tt.want)
		}
	}
	if got, err := s.ProjectViews(ctx, w.project, nil); err != nil || len(got) != 3 {
		t.Fatalf("everyone's: %v, %v", ids(got), err)
	}
}

// A view is read by its id without a lock, of either kind, and is not found when it is deleted or not there. Under a
// write's locks it is read locked FOR UPDATE (spec 2.6): another transaction's FOR KEY SHARE, which FOR NO KEY UPDATE
// and FOR SHARE let through, waits for it, while the unlocked read does not. A deletion stamps the deletion's moment
// and deleter; one deleted already keeps its deletion, and a change of it changes nothing.
func TestReadLockAndDeleteAView(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newViews(t, pool)
	live, gone := view(1, w.workspace, &w.project, w.owner, now.Add(-time.Hour)), view(2, w.workspace, nil, w.owner, now.Add(-time.Hour))
	insert(t, s, live, gone)
	deletedAt := now.Add(-time.Minute)
	if err := s.DeleteView(ctx, gone.ID, deletedAt, w.other); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteView(ctx, gone.ID, now, w.owner); err != nil {
		t.Fatal(err)
	}
	renamed := gone
	renamed.Name, renamed.UpdatedAt = "Renamed", now
	if err := s.UpdateView(ctx, renamed, w.owner); err != nil {
		t.Fatal(err)
	}
	var stamp string
	if err := pool.QueryRow(ctx, "SELECT concat_ws(',', name, (deleted_at = $2 AND updated_at = $2)::text, updated_by_id) FROM issue_views WHERE id = $1", gone.ID, deletedAt).
		Scan(&stamp); err != nil || stamp != "View,true,"+w.other.String() {
		t.Fatalf("deleted %s, %v; want the first deletion's moment and deleter, unchanged", stamp, err)
	}
	for _, id := range []uuid.UUID{gone.ID, uuid.NewV7()} {
		if _, found, err := s.View(ctx, id); found || err != nil {
			t.Fatalf("read %s: found %v, %v", id, found, err)
		}
	}
	err := postgres.NewTxManager(pool, time.Second).WithinTx(ctx, func(txctx context.Context) error {
		for _, id := range []uuid.UUID{gone.ID, uuid.NewV7()} {
			if _, found, err := s.LockView(txctx, id); found || err != nil {
				t.Fatalf("locked %s: found %v, %v", id, found, err)
			}
		}
		got, found, err := s.LockView(txctx, live.ID)
		if err != nil || !found || !reflect.DeepEqual(got, live) {
			t.Fatalf("locked %+v, %v, %v; want %+v", got, found, err, live)
		}
		_, err = pool.Exec(ctx, "SELECT 1 FROM issue_views WHERE id = $1 FOR KEY SHARE NOWAIT", live.ID)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "55P03" {
			t.Fatalf("FOR KEY SHARE while locked: %v; want lock_not_available", err)
		}
		read, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		if got, found, err := s.View(read, live.ID); err != nil || !found || got.ID != live.ID {
			t.Fatalf("unlocked read while locked: %v, %v", found, err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
