package postgresadapter_test

import (
	"context"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// navigationProjects are On (its views turned on), Off (turned off), Arch (archived, its views on) and Gone (deleted,
// its views on) of acme.
func navigationProjects(t *testing.T) (*postgresadapter.Store, *pgxpool.Pool, uuid.UUID, map[string]uuid.UUID) {
	t.Helper()
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, pool, "acme")
	ids := map[string]uuid.UUID{}
	for name, identifier := range map[string]string{"On": "ON", "Off": "OFF", "Arch": "ARCH", "Gone": "GONE"} {
		ids[name] = newProject(t, s, acme, name, identifier, alice)
	}
	for _, name := range []string{"On", "Arch", "Gone"} {
		exec(t, pool, "UPDATE projects SET issue_views_view = true WHERE id = $1", ids[name])
	}
	exec(t, pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", ids["Arch"], now)
	exec(t, pool, "UPDATE projects SET deleted_at = $2 WHERE id = $1", ids["Gone"], now)
	return s, pool, acme, ids
}

// navigationCases are what the navigation module reads of each project: its workspace and its views' toggle, an
// archived one's as ever (P3b spec R2), none of a deleted one or one that is not there.
func navigationCases(acme uuid.UUID, ids map[string]uuid.UUID) map[string]struct {
	id    uuid.UUID
	want  app.NavigationFacts
	found bool
} {
	return map[string]struct {
		id    uuid.UUID
		want  app.NavigationFacts
		found bool
	}{
		"On":      {ids["On"], app.NavigationFacts{WorkspaceID: acme, IssueViewsView: true}, true},
		"Off":     {ids["Off"], app.NavigationFacts{WorkspaceID: acme}, true},
		"Arch":    {ids["Arch"], app.NavigationFacts{WorkspaceID: acme, IssueViewsView: true}, true},
		"Gone":    {ids["Gone"], app.NavigationFacts{}, false},
		"Missing": {uuid.NewV7(), app.NavigationFacts{}, false},
	}
}

// The navigation module's facts of a project (P3b spec 2.1): NavigationFacts reads them without a lock; a FOR UPDATE of
// the project's row does not wait for it.
func TestNavigationFacts(t *testing.T) {
	s, pool, acme, ids := navigationProjects(t)
	for name, tt := range navigationCases(acme, ids) {
		err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
			f, found, err := s.NavigationFacts(ctx, tt.id)
			if err != nil {
				return err
			}
			if found != tt.found || f != tt.want {
				t.Errorf("NavigationFacts(%s) = %+v, %v; want %+v, %v", name, f, found, tt.want, tt.found)
			}
			if name != "Missing" && waits(t, pool, tt.id, "FOR UPDATE") {
				t.Errorf("%s after NavigationFacts: a FOR UPDATE waits; want no lock", name)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

// The same facts under the project's lock of a view's write (spec 2.6): ShareNavigation reads them as NavigationFacts
// does and holds the project FOR SHARE until the transaction ends: the FOR NO KEY UPDATE of a change of the toggle, of
// an archiving, of a membership's change or of a deletion waits for it, and so does a FOR UPDATE; another FOR SHARE does
// not, nor a foreign key's FOR KEY SHARE; the workspace's row is not locked (its FOR SHARE is the write's own, before).
func TestShareNavigation(t *testing.T) {
	s, pool, acme, ids := navigationProjects(t)
	for name, tt := range navigationCases(acme, ids) {
		err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
			f, found, err := s.ShareNavigation(ctx, tt.id)
			if err != nil {
				return err
			}
			if found != tt.found || f != tt.want {
				t.Errorf("ShareNavigation(%s) = %+v, %v; want %+v, %v", name, f, found, tt.want, tt.found)
			}
			if name == "Missing" {
				return nil // no row to probe
			}
			noKeyUpdate, update := waits(t, pool, tt.id, "FOR NO KEY UPDATE"), waits(t, pool, tt.id, "FOR UPDATE")
			share, keyShare := waits(t, pool, tt.id, "FOR SHARE"), waits(t, pool, tt.id, "FOR KEY SHARE")
			if noKeyUpdate != tt.found || update != tt.found || share || keyShare {
				t.Errorf("%s while shared: FOR NO KEY UPDATE waits %v, FOR UPDATE %v, FOR SHARE %v, FOR KEY SHARE %v; want %v, %v, false, false",
					name, noKeyUpdate, update, share, keyShare, tt.found, tt.found)
			}
			if pgtest.RowWaits(t, pool, "workspaces", acme, "FOR UPDATE") {
				t.Errorf("%s while shared: a FOR UPDATE of acme waits; want the workspace's row not locked", name)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if waits(t, pool, ids["On"], "FOR NO KEY UPDATE") {
		t.Error("after the transactions: a FOR NO KEY UPDATE of On waits; want it free")
	}
}

// A project deleted, or its views turned off, while ShareNavigation waits for its row under the writer's FOR NO KEY
// UPDATE, is read as the writer committed it: Postgres reads the row's newest version after the wait.
func TestShareNavigationAfterAConcurrentWrite(t *testing.T) {
	for name, tt := range map[string]struct {
		sql   string
		want  bool
		found bool
	}{
		"deletion":      {"UPDATE projects SET deleted_at = $2 WHERE id = $1", false, false},
		"views off":     {"UPDATE projects SET issue_views_view = false, updated_at = $2 WHERE id = $1", false, true},
		"views kept on": {"UPDATE projects SET updated_at = $2 WHERE id = $1", true, true},
	} {
		t.Run(name, func(t *testing.T) {
			s, pool, _, ids := navigationProjects(t)
			on := ids["On"]
			holder, err := pool.Begin(pgtest.Soon(t))
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = holder.Rollback(context.Background()) }()
			if _, err := holder.Exec(pgtest.Soon(t), tt.sql, on, now); err != nil {
				t.Fatal(err)
			}
			type result struct {
				f     app.NavigationFacts
				found bool
				err   error
			}
			done := make(chan result, 1)
			go func() {
				err := postgres.NewTxManager(pool, 5*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
					f, found, err := s.ShareNavigation(ctx, on)
					done <- result{f, found, err}
					return err
				})
				if err != nil {
					done <- result{err: err}
				}
			}()
			if !pgtest.RowWaits(t, pool, "projects", on, "FOR NO KEY UPDATE") {
				t.Fatal("the writer does not hold the project's row")
			}
			select {
			case r := <-done:
				t.Fatalf("ShareNavigation returned before the writer committed: %+v", r)
			case <-time.After(50 * time.Millisecond):
			}
			if err := holder.Commit(pgtest.Soon(t)); err != nil {
				t.Fatal(err)
			}
			if r := <-done; r.err != nil || r.found != tt.found || r.f.IssueViewsView != tt.want {
				t.Fatalf("ShareNavigation after the writer = %+v; want found %v, views on %v", r, tt.found, tt.want)
			}
		})
	}
}
