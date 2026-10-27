package postgresadapter_test

import (
	"context"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// Replacing SHARE with NO KEY UPDATE blocks a simultaneous SHARE, while
// removing ORDER BY fails the held-smaller-row assertion with reversed input.
func TestShareIssueProjects(t *testing.T) {
	s, pool := newStore(t)
	acme := newWorkspace(t, pool, "acme")
	beta := newWorkspace(t, pool, "beta")
	low, high, gone, other := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	for _, p := range []struct {
		id, w uuid.UUID
		name  string
	}{{high, acme, "Alpha"}, {low, acme, "Web"}, {gone, beta, "Gone"}, {other, beta, "Other"}} {
		exec(t, pool, "INSERT INTO projects (id,workspace_id,name,identifier) VALUES ($1,$2,$3::text,upper($3::text))", p.id, p.w, p.name)
	}
	exec(t, pool, "UPDATE projects SET archived_at=now() WHERE id=$1", low)
	exec(t, pool, "UPDATE projects SET deleted_at=now() WHERE id=$1", gone)
	ids, err := s.UndeletedIssueProjectIDs(context.Background(), acme)
	if err != nil || !slices.Equal(ids, []uuid.UUID{low, high}) {
		t.Fatalf("ids=%v err=%v", ids, err)
	}
	t.Run("share mode duplicate and absent rows", func(t *testing.T) {
		err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
			got, err := s.ShareIssueProjects(ctx, []uuid.UUID{high, low, low, gone, uuid.NewV7()})
			if err != nil {
				return err
			}
			if !maps.Equal(got, map[uuid.UUID]uuid.UUID{low: acme, high: acme}) {
				t.Fatalf("shared=%v", got)
			}
			for _, id := range []uuid.UUID{low, high} {
				if !waits(t, pool, id, "FOR NO KEY UPDATE") || waits(t, pool, id, "FOR SHARE") || waits(t, pool, id, "FOR KEY SHARE") {
					t.Errorf("project %s must hold SHARE, compatible with SHARE and KEY SHARE", id)
				}
			}
			if waits(t, pool, gone, "FOR UPDATE") || waits(t, pool, other, "FOR UPDATE") || pgtest.RowWaits(t, pool, "workspaces", acme, "FOR UPDATE") {
				t.Error("unrequested resource locked")
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("nil and empty take no row locks", func(t *testing.T) {
		for _, ids := range [][]uuid.UUID{nil, {}} {
			err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
				got, err := s.ShareIssueProjects(ctx, ids)
				if err != nil {
					return err
				}
				if len(got) != 0 || waits(t, pool, low, "FOR UPDATE") || waits(t, pool, high, "FOR UPDATE") {
					t.Fatalf("empty shared=%v", got)
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		}
	})
	t.Run("reverse input acquires smaller first in one statement", func(t *testing.T) {
		held, err := pool.Begin(pgtest.Soon(t))
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = held.Rollback(context.Background()) }()
		if _, err = held.Exec(context.Background(), "SELECT 1 FROM projects WHERE id=$1 FOR NO KEY UPDATE", high); err != nil {
			t.Fatal(err)
		}
		done := make(chan error, 1)
		go func() {
			done <- postgres.NewTxManager(pool, 10*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
				_, err := s.ShareIssueProjects(ctx, []uuid.UUID{high, low})
				return err
			})
		}()
		pgtest.WaitForLockWaitOn(t, pool, "projects", 10*time.Second)
		if !waits(t, pool, low, "FOR NO KEY UPDATE") {
			t.Error("smaller project not held while waiting for larger")
		}
		// One SQL must acquire both rows; separate calls would expose an idle
		// transaction between them rather than one active blocked statement.
		var query string
		if err := pool.QueryRow(context.Background(), "SELECT query FROM pg_stat_activity WHERE wait_event_type='Lock' AND query LIKE '%projects%' AND pid <> pg_backend_pid() LIMIT 1").Scan(&query); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(query, "id = ANY") {
			t.Errorf("expected one set-based project query; got %q", query)
		}
		if err := held.Commit(context.Background()); err != nil {
			t.Fatal(err)
		}
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(10 * time.Second):
			t.Fatal("SHARE did not finish")
		}
	})
}
