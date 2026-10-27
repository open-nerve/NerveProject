package postgresadapter_test

import (
	"context"
	"maps"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// Missing cascade steps, retaining the creator as writer and revisiting
// previously deleted rows each change these independent audit expectations.
func TestLinkDeletionRows(t *testing.T) {
	for _, route := range []string{"subtree", "project", "workspace"} {
		t.Run(route, func(t *testing.T) {
			s, pool := newStore(t)
			creator, deleter := newAccount(t, pool, "creator@corp.com"), newAccount(t, pool, "deleter@corp.com")
			workspace := newWorkspace(t, pool, "acme")
			p, other := newProject(t, pool, workspace, "WEB"), newProject(t, pool, workspace, "OPS")
			root, child, grandchild, outside, far := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
			for n, id := range []uuid.UUID{root, child, grandchild, outside} {
				newIssue(t, pool, p, id, n+1, creator)
			}
			newIssue(t, pool, other, far, 1, creator)
			exec(t, pool, "UPDATE issues SET parent_id=$2 WHERE id=$1", child, root)
			exec(t, pool, "UPDATE issues SET parent_id=$2,archived_at=$3 WHERE id=$1", grandchild, child, now)
			ids := map[uuid.UUID]uuid.UUID{}
			for _, issue := range []uuid.UUID{grandchild, far, outside, root, child} {
				project := p
				if issue == far {
					project = other
				}
				id := uuid.NewV7()
				insertLink(t, pool, project, issue, id, creator, "http://169.254.169.254/link")
				ids[issue] = id
			}
			early := uuid.NewV7()
			insertLink(t, pool, p, child, early, creator, "http://169.254.169.254/early")
			before, at := now.Add(-time.Hour), now.Add(time.Hour)
			exec(t, pool, "UPDATE issue_links SET deleted_at=$2,updated_at=$2 WHERE id=$1", early, before)
			rows, stamps := rowsAsText(t, pool, "issue_links"), audits(t, pool, "issue_links")
			ctx := context.Background()
			var err error
			switch route {
			case "subtree":
				var subtree []uuid.UUID
				subtree, err = s.Subtree(ctx, root)
				if err == nil {
					err = s.DeleteIssueLinkRows(ctx, app.IssueDeletion{IssueIDs: subtree, By: deleter, At: at})
				}
			case "project":
				_, err = adapterDeletionRows(s, pool, 2).Execute(ctx, "project", p.id, at, deleter)
			case "workspace":
				for _, id := range []uuid.UUID{other.id, p.id} {
					if _, err = adapterDeletionRows(s, pool, 2).Execute(ctx, "project", id, at, deleter); err != nil {
						break
					}
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			want := maps.Clone(stamps)
			for issue, id := range ids {
				if route == "workspace" || issue != far && (route == "project" || issue != outside) {
					want[id] = deletion(at, deleter)
				}
			}
			if got := audits(t, pool, "issue_links"); !maps.Equal(got, want) {
				t.Errorf("link audits = %v; want %v", got, want)
			}
			if got := rowsAsText(t, pool, "issue_links"); !maps.Equal(got, rows) {
				t.Errorf("non-audit columns changed: %v; want %v", got, rows)
			}
		})
	}
}

// Reverse insertion order makes an unordered scan reach c first. While
// waiting for b, the statement must hold a and must not have reached c.
func TestLinkDeletionRowsLockOrder(t *testing.T) {
	for _, route := range []string{"subtree", "project"} {
		t.Run(route, func(t *testing.T) {
			s, pool := newStore(t)
			creator, deleter := newAccount(t, pool, "creator@corp.com"), newAccount(t, pool, "deleter@corp.com")
			p := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
			issue := uuid.NewV7()
			newIssue(t, pool, p, issue, 1, creator)
			a, b, c := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
			for n, id := range []uuid.UUID{c, b, a} {
				insertLink(t, pool, p, issue, id, creator, "http://169.254.169.254/"+string(rune('a'+n)))
			}
			before, at := now.Add(-time.Hour), now.Add(time.Hour)
			soon := pgtest.Soon(t)
			held, err := pool.Begin(soon)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = held.Rollback(context.Background()) }()
			if _, err := held.Exec(soon, "UPDATE issue_links SET deleted_at=$2,updated_at=$2,updated_by_id=$3 WHERE id=$1", b, before, creator); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() {
				done <- postgres.NewTxManager(pool, 5*time.Second).WithinTx(soon, func(ctx context.Context) error {
					if route == "subtree" {
						return s.DeleteIssueLinkRows(ctx, app.IssueDeletion{IssueIDs: []uuid.UUID{issue}, By: deleter, At: at})
					}
					_, err := s.DeleteProjectLinkRows(ctx, app.RowBatch{ProjectID: p.id, Limit: 2, By: deleter, At: at})
					return err
				})
			}()
			pgtest.WaitForLockWaitOn(t, pool, "issue_links", 5*time.Second)
			if !pgtest.RowWaits(t, pool, "issue_links", a, "FOR SHARE") || pgtest.RowWaits(t, pool, "issue_links", c, "FOR SHARE") {
				t.Error("want a locked before b and c not reached")
			}
			if err := held.Commit(soon); err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-done:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(10 * time.Second):
				t.Fatal("link deletion did not finish")
			}
			got := audits(t, pool, "issue_links")
			for id, want := range map[uuid.UUID]string{a: deletion(at, deleter), b: deletion(before, creator), c: deletion(at, deleter)} {
				if got[id] != want {
					t.Errorf("%s: %s; want %s", id, got[id], want)
				}
			}
		})
	}
}
