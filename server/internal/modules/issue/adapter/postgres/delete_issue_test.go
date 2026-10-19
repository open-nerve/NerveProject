package postgresadapter_test

import (
	"context"
	"errors"
	"maps"
	"slices"
	"testing"
	"time"
	"uuid"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// insertOf is how a row of table is inserted under a work item, as the
// batch statements' test inserts it (batchStatements).
func insertOf(t *testing.T, table string) inserter {
	t.Helper()
	i := slices.IndexFunc(batchStatements, func(st batchStatement) bool { return st.table == table })
	if i < 0 {
		t.Fatalf("no batch statement writes %s", table)
	}
	return batchStatements[i].insert
}

// Each statement of a deleted work item (M4 design 3.6, delete_issue.sql)
// soft-deletes the work item's undeleted rows of its table at the moment
// and by the account given, and nothing else. In web, the work item item
// has the rows a and b of the table (of the work items' table, item
// itself) and c, which alice deleted an hour before; the work item gone
// has its row g, deleted then too (of the work items' table, gone itself);
// the work item other has its row o, other's id after gone's and before
// item's, so that a statement that took its work item by the ids' order,
// not by equality, would reach o on item or on gone. The statement on
// item, then on gone, by bob at later: a and b (item) are deleted at later
// by bob; c and g (gone) keep alice's deletion, and o its row as it was;
// every other column of every row is kept. Each statement answers a failed
// write as its failure.
func TestTheStatementsOfADeletedWorkItemDeleteItsRows(t *testing.T) {
	for _, st := range []struct {
		table  string
		method func(*postgresadapter.Store, context.Context, app.IssueDeletion) error
	}{
		{"issues", (*postgresadapter.Store).DeleteIssueRow},
		{"issue_assignees", (*postgresadapter.Store).DeleteIssueAssigneeRows},
		{"issue_labels", (*postgresadapter.Store).DeleteIssueLabelRows},
		{"issue_subscribers", (*postgresadapter.Store).DeleteIssueSubscriberRows},
		{"issue_activities", (*postgresadapter.Store).DeleteIssueActivityRows},
	} {
		t.Run(st.table, func(t *testing.T) {
			s, pool := newStore(t)
			alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
			web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
			gone, other, item := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
			for n, id := range []uuid.UUID{item, gone, other} {
				newIssue(t, pool, web, id, n+1, alice)
			}
			deleted, earlier := []uuid.UUID{item}, []uuid.UUID{gone}
			if st.table != "issues" {
				a, b, c, g, o := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
				insert := insertOf(t, st.table)
				for n, row := range []struct{ issue, id uuid.UUID }{{item, a}, {item, b}, {item, c}, {gone, g}, {other, o}} {
					insert(t, pool, web, row.issue, row.id, alice, n)
				}
				deleted, earlier = []uuid.UUID{a, b}, []uuid.UUID{c, g}
			}
			before, later := now.Add(-time.Hour), now.Add(time.Hour)
			for _, id := range earlier {
				exec(t, pool, "UPDATE "+st.table+" SET deleted_at = $2, updated_at = $2, updated_by_id = $3 WHERE id = $1", id, before, alice)
			}
			rows, stamps := rowsAsText(t, pool, st.table), audits(t, pool, st.table)

			for _, id := range []uuid.UUID{item, gone} {
				if err := st.method(s, context.Background(), app.IssueDeletion{IssueID: id, By: bob, At: later}); err != nil {
					t.Fatal(err)
				}
			}

			if after := rowsAsText(t, pool, st.table); !maps.Equal(after, rows) {
				t.Errorf("the rows but their audit columns:\n%v\nwant them as they were:\n%v", after, rows)
			}
			want := maps.Clone(stamps)
			for _, id := range deleted {
				want[id] = deletion(later, bob)
			}
			if after := audits(t, pool, st.table); !maps.Equal(after, want) {
				t.Errorf("the audit columns:\n%v\nwant\n%v", after, want)
			}

			cancelled, cancel := context.WithCancel(context.Background())
			cancel()
			if err := st.method(s, cancelled, app.IssueDeletion{IssueID: other, By: bob, At: later}); !errors.Is(err, context.Canceled) {
				t.Errorf("the statement on a cancelled context = %v; want context.Canceled", err)
			}
		})
	}
}
