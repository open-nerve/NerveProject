package postgresadapter_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

func purgeID(n int) uuid.UUID { return uuid.MustParse(fmt.Sprintf("0199a2b4-0000-7000-8000-%012x", n)) }

func purgeMethods(s *postgresadapter.Store) map[string]func(context.Context, app.PurgeBatch) ([]uuid.UUID, error) {
	return map[string]func(context.Context, app.PurgeBatch) ([]uuid.UUID, error){
		"comment_reactions":          s.PurgeCommentReactions,
		"issue_reactions":            s.PurgeIssueReactions,
		"issue_mentions":             s.PurgeIssueMentions,
		"issue_description_versions": s.PurgeIssueDescriptionVersions,
		"issue_activities":           s.PurgeIssueActivities,
		"issue_comments":             s.PurgeIssueComments,
		"issue_links":                s.PurgeIssueLinks,
		"issue_relations":            s.PurgeIssueRelations,
		"issue_assignees":            s.PurgeIssueAssignees,
		"issue_labels":               s.PurgeIssueLabels,
		"issue_subscribers":          s.PurgeIssueSubscribers,
		"draft_issues":               s.PurgeDraftIssues,
		"issues":                     s.PurgeIssues,
	}
}

// Missing a table binding, cutoff, cursor or limit must lose a direct deletion
// or a retained row. Controls have live parent chains so CASCADE cannot hide it.
func TestIssuePurgeAllTables(t *testing.T) {
	for _, st := range batchStatements {
		t.Run(st.table, func(t *testing.T) {
			s, pool := newStore(t)
			ctx := context.Background()
			by := newAccount(t, pool, "purge@corp.com")
			p := newProject(t, pool, newWorkspace(t, pool, "purge"), "PURGE")
			parent := uuid.NewV7()
			newIssue(t, pool, p, parent, 1, by)
			cutoff := now.Add(-60 * 24 * time.Hour)
			old, older, recent := now.Add(-61*24*time.Hour), now.Add(-62*24*time.Hour), now.Add(-59*24*time.Hour)
			for n, row := range []struct {
				n  int
				at *time.Time
			}{{30, &older}, {10, &old}, {20, &cutoff}, {40, &recent}, {50, nil}} {
				st.insert(t, pool, p, parent, purgeID(row.n), by, n)
				if row.at != nil {
					exec(t, pool, "UPDATE "+st.table+" SET deleted_at=$2 WHERE id=$1", purgeID(row.n), *row.at)
				}
			}
			method := purgeMethods(s)[st.table]
			// Smaller eligible id remains: removing id > After deletes it instead of 30.
			ids, err := method(ctx, app.PurgeBatch{Cutoff: cutoff, After: purgeID(15), Limit: 1})
			if err != nil || !slices.Equal(ids, []uuid.UUID{purgeID(30)}) {
				t.Fatalf("nonzero cursor batch=%v,%v; want [30] while eligible 10 remains", ids, err)
			}
			rollback := errors.New("rollback caller")
			err = postgres.NewTxManager(pool, time.Second).WithinTx(ctx, func(txctx context.Context) error {
				ids, e := method(txctx, app.PurgeBatch{Cutoff: cutoff, Limit: 1})
				if e != nil || !slices.Equal(ids, []uuid.UUID{purgeID(10)}) {
					t.Fatalf("independent batch=%v,%v; want [10]", ids, e)
				}
				return rollback
			})
			if !errors.Is(err, rollback) {
				t.Fatal(err)
			}
			if count, err := issue.Provide(pool).Purger.Purge(ctx, cutoff, 1); err != nil || count != 0 {
				t.Fatalf("remaining purge=%d,%v; want 0", count, err)
			}
			for _, n := range []int{10, 20, 30, 40, 50} {
				var exists bool
				if err := pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM "+st.table+" WHERE id=$1)", purgeID(n)).Scan(&exists); err != nil || exists != (n != 10 && n != 30) {
					t.Fatalf("%s id %d exists=%v err=%v", st.table, n, exists, err)
				}
			}
			// Make every table nonempty for the real Provide binding, not only its adapter.
			st.insert(t, pool, p, parent, purgeID(60), by, 60)
			exec(t, pool, "UPDATE "+st.table+" SET deleted_at=$2 WHERE id=$1", purgeID(60), old)
			if count, err := issue.Provide(pool).Purger.Purge(ctx, cutoff, 1); err != nil || count != 1 {
				t.Fatalf("bound %s direct deletion=%d,%v; want 1", st.table, count, err)
			}
			// A later statement's failure cannot roll back the prior pool batch.
			for _, n := range []int{61, 70} {
				st.insert(t, pool, p, parent, purgeID(n), by, n)
				exec(t, pool, "UPDATE "+st.table+" SET deleted_at=$2 WHERE id=$1", purgeID(n), old)
			}
			exec(t, pool, `CREATE FUNCTION fail_purge() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF OLD.id='0199a2b4-0000-7000-8000-000000000046'::uuid THEN RAISE EXCEPTION 'purge fixture failure'; END IF; RETURN OLD; END $$`)
			exec(t, pool, "CREATE TRIGGER fail_purge BEFORE DELETE ON "+st.table+" FOR EACH ROW EXECUTE FUNCTION fail_purge()")
			count, purgeErr := issue.Provide(pool).Purger.Purge(ctx, cutoff, 1)
			var pe *pgconn.PgError
			if count != 1 || !errors.As(purgeErr, &pe) || pe.Message != "purge fixture failure" {
				t.Fatalf("%s committed count=%d,%v; want 1 and original PG error", st.table, count, purgeErr)
			}
			var committed, failed bool
			if err := pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM "+st.table+" WHERE id=$1),EXISTS(SELECT 1 FROM "+st.table+" WHERE id=$2)", purgeID(61), purgeID(70)).Scan(&committed, &failed); err != nil || committed || !failed {
				t.Fatalf("%s persisted progress=%v,%v err=%v; want false,true", st.table, committed, failed, err)
			}
		})
	}
	t.Run("child order and cascades", testIssuePurgeGraph)
	t.Run("workspace drafts cutoff", func(t *testing.T) {
		_, pool := newStore(t)
		ctx := context.Background()
		by := newAccount(t, pool, "loose@corp.com")
		workspace := newWorkspace(t, pool, "loose")
		cutoff, old, recent := now.Add(-60*24*time.Hour), now.Add(-61*24*time.Hour), now.Add(-59*24*time.Hour)
		for _, row := range []struct {
			n  int
			at *time.Time
		}{{30, &old}, {10, &old}, {20, &cutoff}, {40, &recent}, {50, nil}} {
			exec(t, pool, "INSERT INTO draft_issues(id,workspace_id,created_by_id,deleted_at) VALUES($1,$2,$3,$4)", purgeID(row.n), workspace, by, row.at)
		}
		if count, err := issue.Provide(pool).Purger.Purge(ctx, cutoff, 1); err != nil || count != 2 {
			t.Fatalf("workspace-only drafts=%d,%v; want 2", count, err)
		}
		var ids []uuid.UUID
		if err := pool.QueryRow(ctx, "SELECT array_agg(id ORDER BY id) FROM draft_issues").Scan(&ids); err != nil || !slices.Equal(ids, []uuid.UUID{purgeID(20), purgeID(40), purgeID(50)}) {
			t.Fatalf("workspace draft controls=%v,%v; want 20,40,50", ids, err)
		}
	})
}

func testIssuePurgeGraph(t *testing.T) {
	_, pool := newStore(t)
	ctx := context.Background()
	by := newAccount(t, pool, "graph@corp.com")
	p := newProject(t, pool, newWorkspace(t, pool, "graph"), "GRAPH")
	cutoff, old, recent := now.Add(-60*24*time.Hour), now.Add(-61*24*time.Hour), now.Add(-59*24*time.Hour)
	root, mid, leaf, live := purgeID(90), purgeID(91), purgeID(92), purgeID(93)
	for n, id := range []uuid.UUID{root, mid, leaf, live} {
		newIssue(t, pool, p, id, n+1, by)
	}
	exec(t, pool, "UPDATE issues SET parent_id=$2 WHERE id=$1", mid, root)
	exec(t, pool, "UPDATE issues SET parent_id=$2 WHERE id=$1", leaf, mid)
	// Old root has a recent descendant: only root's direct DELETE is counted.
	exec(t, pool, "UPDATE issues SET deleted_at=$2 WHERE id=$1", root, old)
	exec(t, pool, "UPDATE issues SET deleted_at=$2 WHERE id=ANY($1)", []uuid.UUID{mid, leaf}, recent)
	for n, st := range batchStatements {
		if st.table == "issues" {
			continue
		}
		id := purgeID(100 + n)
		st.insert(t, pool, p, root, id, by, n)
		exec(t, pool, "UPDATE "+st.table+" SET deleted_at=$2 WHERE id=$1", id, old)
	}
	// An explicit old comment and its old reaction must both count directly.
	exec(t, pool, "UPDATE issue_comments SET deleted_at=$2 WHERE issue_id=$1", root, old)
	// comment_reactions' fixture creates an additional comment.
	// Opposite direction and all stored relation kinds are independently eligible.
	for n, kind := range []string{"blocked_by", "duplicate", "relates_to"} {
		id := purgeID(150 + n)
		newRelation(t, pool, p, live, root, id, kind, by)
		exec(t, pool, "UPDATE issue_relations SET deleted_at=$2 WHERE id=$1", id, old)
	}
	// Workspace-only draft is not reachable through an issue or project purge.
	exec(t, pool, "INSERT INTO draft_issues(id,workspace_id,created_by_id,deleted_at) VALUES($1,$2,$3,$4)", purgeID(160), p.workspace, by, old)
	// 12 owned child rows + the reaction fixture's extra comment + 3 reverse
	// relations + 1 workspace-only draft + 1 root; recent descendants cascade.
	const wantDirect = 18
	if count, err := issue.Provide(pool).Purger.Purge(ctx, cutoff, 1); err != nil || count != wantDirect {
		t.Fatalf("graph direct count=%d,%v; want %d", count, err, wantDirect)
	}
	var remaining []uuid.UUID
	rows, err := pool.Query(ctx, "SELECT id FROM issues ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		remaining = append(remaining, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	// aRelation adds a live peer outside the purged parent chain.
	if !slices.Contains(remaining, live) || slices.Contains(remaining, mid) || slices.Contains(remaining, leaf) || len(remaining) != 2 {
		t.Fatalf("cascade survivors=%v; want live and relation peer", remaining)
	}
}

// Invalid batches fail before attempting to use a nil pool.
func TestIssuePurgeRejectsInvalidBatch(t *testing.T) {
	for table, method := range purgeMethods(postgresadapter.New(nil)) {
		for _, limit := range []int{-1, 0, 2147483648} {
			if _, err := method(context.Background(), app.PurgeBatch{Limit: limit}); err == nil {
				t.Fatalf("%s accepted %d", table, limit)
			}
		}
	}
}
