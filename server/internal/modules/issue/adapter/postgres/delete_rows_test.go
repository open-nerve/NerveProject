package postgresadapter_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// inserter inserts the row id of a table under the work item issue of p,
// by by, the n-th of the test, so each takes an account or a label of its
// own where the table's key asks for one.
type inserter func(t *testing.T, pool *pgxpool.Pool, p project, issue, id, by uuid.UUID, n int)

// batchStatement is one of a deleted project's batch statements: the table
// it writes, the store's method, and how a row of the table is inserted.
type batchStatement struct {
	table  string
	method func(*postgresadapter.Store, context.Context, app.RowBatch) ([]uuid.UUID, error)
	insert inserter
}

// child inserts a row of table under the work item, its column the value
// value takes.
func child(table, column string, value func(t *testing.T, pool *pgxpool.Pool, p project, n int) uuid.UUID) inserter {
	return func(t *testing.T, pool *pgxpool.Pool, p project, issue, id, by uuid.UUID, n int) {
		t.Helper()
		exec(t, pool, `INSERT INTO `+table+` (id, workspace_id, project_id, issue_id, `+column+`, created_by_id, updated_by_id, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $6, $7, $7)`, id, p.workspace, p.id, issue, value(t, pool, p, n), by, now)
	}
}

// anAccount is an account of its own for the n-th row.
func anAccount(t *testing.T, pool *pgxpool.Pool, _ project, n int) uuid.UUID {
	return newAccount(t, pool, fmt.Sprintf("u%d@corp.com", n))
}

// aLabel is a label of p of its own for the n-th row.
func aLabel(t *testing.T, pool *pgxpool.Pool, p project, n int) uuid.UUID {
	id := uuid.NewV7()
	exec(t, pool, "INSERT INTO labels (id, workspace_id, project_id, name) VALUES ($1, $2, $3, $4)", id, p.workspace, p.id, fmt.Sprint("L", n))
	return id
}

var batchStatements = []batchStatement{
	{"issues", (*postgresadapter.Store).DeleteProjectIssueRows, func(t *testing.T, pool *pgxpool.Pool, p project, _, id, by uuid.UUID, n int) {
		newIssue(t, pool, p, id, 100+n, by)
	}},
	{"issue_assignees", (*postgresadapter.Store).DeleteProjectAssigneeRows, child("issue_assignees", "assignee_id", anAccount)},
	{"issue_labels", (*postgresadapter.Store).DeleteProjectLabelRows, child("issue_labels", "label_id", aLabel)},
	{"issue_subscribers", (*postgresadapter.Store).DeleteProjectSubscriberRows, child("issue_subscribers", "subscriber_id", anAccount)},
	{"issue_activities", (*postgresadapter.Store).DeleteProjectActivityRows, func(t *testing.T, pool *pgxpool.Pool, p project, issue, id,
		by uuid.UUID, _ int) {
		exec(t, pool, `INSERT INTO issue_activities (id, workspace_id, project_id, issue_id, actor_id, created_by_id, updated_by_id, created_at,
			updated_at) VALUES ($1, $2, $3, $4, $5, $5, $5, $6, $6)`, id, p.workspace, p.id, issue, by, now)
	}},
}

// rowsAsText is every row of table but its audit columns of a deletion, by
// id, as text.
func rowsAsText(t *testing.T, pool *pgxpool.Pool, table string) map[uuid.UUID]string {
	t.Helper()
	rows, err := pool.Query(context.Background(), `SELECT id, (to_jsonb(t) - 'deleted_at' - 'updated_at' - 'updated_by_id')::text FROM `+table+` t`)
	if err != nil {
		t.Fatal(err)
	}
	out := map[uuid.UUID]string{}
	for rows.Next() {
		var id uuid.UUID
		var text string
		if err := rows.Scan(&id, &text); err != nil {
			t.Fatal(err)
		}
		out[id] = text
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// Each batch statement of a deleted project (M4 design 3.6) takes, in one
// statement, at most the limit of the project's undeleted rows of its
// table whose ids are after the bound, in id order, whatever order the
// rows lie in, and soft-deletes them at the moment and by the account
// given, and nothing else: a batch after c takes d and e only (rolled
// back); run again from the greatest id of each batch, its batches are the
// project's undeleted rows in id order, two at a time, then none. web's
// rows lie out of their id order in the table, a row of ops lies among
// them, and one of web's was deleted before, by alice, an hour earlier.
// Every other column of the rows deleted is kept, and every column of the
// earlier deleted row and of ops' rows. Each statement answers a failed
// read as its failure, never as no rows.
func TestTheBatchStatementsDeleteAProjectsRowsInIDOrder(t *testing.T) {
	for _, st := range batchStatements {
		t.Run(st.table, func(t *testing.T) {
			s, pool := newStore(t)
			alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
			acme := newWorkspace(t, pool, "acme")
			web, ops := newProject(t, pool, acme, "WEB"), newProject(t, pool, acme, "OPS")
			webIssue, opsIssue := uuid.NewV7(), uuid.NewV7()
			newIssue(t, pool, web, webIssue, 1, alice)
			newIssue(t, pool, ops, opsIssue, 1, alice)
			// drawn in this order: ascending ids
			a, gone, inOps, b, c, d, e := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
			for n, row := range []uuid.UUID{e, c, inOps, a, d, gone, b} {
				if row == inOps {
					st.insert(t, pool, ops, opsIssue, row, alice, n)
				} else {
					st.insert(t, pool, web, webIssue, row, alice, n)
				}
			}
			earlier, later := now.Add(-time.Hour), now.Add(time.Hour)
			exec(t, pool, "UPDATE "+st.table+" SET deleted_at = $2, updated_at = $2, updated_by_id = $3 WHERE id = $1", gone, earlier, alice)
			var webRows []uuid.UUID // web's undeleted rows of the table, in id order
			for _, id := range []uuid.UUID{webIssue, a, b, c, d, e} {
				if st.table == "issues" || id != webIssue {
					webRows = append(webRows, id)
				}
			}
			before := rowsAsText(t, pool, st.table)
			auditBefore := audits(t, pool, st.table)
			tx := postgres.NewTxManager(pool, 2*time.Second)

			rollBack := errors.New("roll back")
			err := tx.WithinTx(context.Background(), func(ctx context.Context) error {
				ids, err := st.method(s, ctx, app.RowBatch{ProjectID: web.id, After: c, Limit: 10, By: bob, At: later})
				if got := slices.SortedFunc(slices.Values(ids), uuid.UUID.Compare); err != nil || fmt.Sprint(got) != fmt.Sprint([]uuid.UUID{d, e}) {
					t.Errorf("a batch after c = %v, %v; want d and e only", got, err)
				}
				return rollBack
			})
			if !errors.Is(err, rollBack) {
				t.Fatal(err)
			}

			var got [][]uuid.UUID
			err = tx.WithinTx(context.Background(), func(ctx context.Context) error {
				for after := (uuid.UUID{}); ; {
					ids, err := st.method(s, ctx, app.RowBatch{ProjectID: web.id, After: after, Limit: 2, By: bob, At: later})
					if err != nil {
						return err
					}
					got = append(got, slices.SortedFunc(slices.Values(ids), uuid.UUID.Compare))
					if len(ids) == 0 || len(got) > 10 {
						return nil
					}
					after = slices.MaxFunc(ids, uuid.UUID.Compare)
				}
			})
			if err != nil {
				t.Fatal(err)
			}

			var want [][]uuid.UUID
			for chunk := range slices.Chunk(webRows, 2) {
				want = append(want, chunk)
			}
			want = append(want, nil)
			if fmt.Sprint(got) != fmt.Sprint(want) {
				t.Errorf("the batches %v, want %v", got, want)
			}
			if after := rowsAsText(t, pool, st.table); fmt.Sprint(after) != fmt.Sprint(before) {
				t.Errorf("the rows but their audit columns:\n%v\nwant them as they were:\n%v", after, before)
			}
			auditAfter := audits(t, pool, st.table)
			for id, was := range auditBefore {
				want := was
				if slices.Contains(webRows, id) {
					want = fmt.Sprintf("deleted at %s, written at %s by %s", later.Format(time.RFC3339Nano), later.Format(time.RFC3339Nano), bob)
				}
				if auditAfter[id] != want {
					t.Errorf("row %s: %s, want %s", id, auditAfter[id], want)
				}
			}

			cancelled, cancel := context.WithCancel(context.Background())
			cancel()
			if ids, err := st.method(s, cancelled, app.RowBatch{ProjectID: ops.id, Limit: 2, By: bob, At: later}); !errors.Is(err, context.Canceled) ||
				ids != nil {
				t.Errorf("a batch on a cancelled context = %v, %v; want context.Canceled, not no rows", ids, err)
			}
		})
	}
}

// audits are table's rows' audit columns of a deletion, by id, as text.
func audits(t *testing.T, pool *pgxpool.Pool, table string) map[uuid.UUID]string {
	t.Helper()
	rows, err := pool.Query(context.Background(), "SELECT id, deleted_at, updated_at, updated_by_id FROM "+table)
	if err != nil {
		t.Fatal(err)
	}
	out := map[uuid.UUID]string{}
	for rows.Next() {
		var id uuid.UUID
		var deleted *time.Time
		var written time.Time
		var by *uuid.UUID
		if err := rows.Scan(&id, &deleted, &written, &by); err != nil {
			t.Fatal(err)
		}
		d, writer := "undeleted", "nobody"
		if deleted != nil {
			d = "deleted at " + deleted.UTC().Format(time.RFC3339Nano)
		}
		if by != nil {
			writer = by.String()
		}
		out[id] = fmt.Sprintf("%s, written at %s by %s", d, written.UTC().Format(time.RFC3339Nano), writer)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}
