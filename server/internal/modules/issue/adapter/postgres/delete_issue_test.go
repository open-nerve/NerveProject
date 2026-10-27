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

// A work item's subtree (M4 design 3.6, 3.9) is it and each undeleted work
// item under it, however deep, archived ones too, in id order. In web, the
// work item root has under it a, archived, and b; a has c under it; b has
// d under it, deleted, which has e under it, which is not: a deleted work
// item's are not reached through it. other is another top work item, with
// f under it, and in ops, x is under y and y under x, a loop no write
// makes: the read ends, each once. The work items are drawn so that the
// subtree's id order is none of the order they were made or linked in. A
// deleted or missing work item has none.
func TestTheSubtreeOfAWorkItem(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, ops := newProject(t, pool, acme, "WEB"), newProject(t, pool, acme, "OPS")
	c, b, root, f, e, a, d, other, x, y := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(),
		uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	for n, id := range []uuid.UUID{root, a, b, c, d, e, other, f} {
		newIssue(t, pool, web, id, n+1, alice)
	}
	newIssue(t, pool, ops, x, 1, alice)
	newIssue(t, pool, ops, y, 2, alice)
	for _, link := range [][2]uuid.UUID{{a, root}, {b, root}, {c, a}, {d, b}, {e, d}, {f, other}, {x, y}, {y, x}} {
		exec(t, pool, "UPDATE issues SET parent_id = $2 WHERE id = $1", link[0], link[1])
	}
	exec(t, pool, "UPDATE issues SET archived_at = $2 WHERE id = $1", a, now)
	exec(t, pool, "UPDATE issues SET deleted_at = $2 WHERE id = $1", d, now)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, tt := range []struct {
		name string
		id   uuid.UUID
		want []uuid.UUID
	}{
		{"root", root, slices.SortedFunc(slices.Values([]uuid.UUID{root, a, b, c}), uuid.UUID.Compare)},
		{"a, archived", a, slices.SortedFunc(slices.Values([]uuid.UUID{a, c}), uuid.UUID.Compare)},
		{"c, with none under it", c, []uuid.UUID{c}},
		{"other", other, slices.SortedFunc(slices.Values([]uuid.UUID{other, f}), uuid.UUID.Compare)},
		{"x, in a loop", x, slices.SortedFunc(slices.Values([]uuid.UUID{x, y}), uuid.UUID.Compare)},
		{"d, deleted", d, nil},
		{"no work item", uuid.NewV7(), nil},
	} {
		got, err := s.Subtree(ctx, tt.id)
		if err != nil || !slices.Equal(got, tt.want) {
			t.Errorf("Subtree(%s) = %v, %v; want %v", tt.name, got, err, tt.want)
		}
	}
	cancelled, stop := context.WithCancel(context.Background())
	stop()
	if _, err := s.Subtree(cancelled, root); !errors.Is(err, context.Canceled) {
		t.Errorf("Subtree() on a cancelled context = %v; want context.Canceled", err)
	}
}

// Each statement of a deleted subtree (M4 design 3.6, delete_issue.sql)
// soft-deletes the undeleted rows of its table of each work item it names,
// at the moment and by the account given, and nothing else. In web, the
// work item item has the rows a and b of the table (of the work items'
// table, item itself) and c, which alice deleted an hour before; the work
// item sub has its row d (sub itself); the work item gone has its row g,
// deleted then too (gone itself); the work item other has its row o,
// other's id after gone's and before item's and sub's, so that a statement
// that took its work items by the ids' order, not by equality, would reach
// o. One statement names sub, item and gone, in descending id order, by
// bob at later: a, b and d are deleted at later by bob, those of the first
// work item it names and of the others; c and g keep alice's deletion, and
// o its row as it was; every other column of every row is kept. Each
// statement answers a failed write as its failure.
func TestTheStatementsOfADeletedSubtreeDeleteItsRows(t *testing.T) {
	for _, st := range []struct {
		table  string
		method func(*postgresadapter.Store, context.Context, app.IssueDeletion) error
	}{
		{"issues", (*postgresadapter.Store).DeleteIssueRows},
		{"issue_assignees", (*postgresadapter.Store).DeleteIssueAssigneeRows},
		{"issue_labels", (*postgresadapter.Store).DeleteIssueLabelRows},
		{"issue_relations", (*postgresadapter.Store).DeleteIssueRelationRows},
		{"issue_links", (*postgresadapter.Store).DeleteIssueLinkRows},
		{"issue_subscribers", (*postgresadapter.Store).DeleteIssueSubscriberRows},
		{"issue_mentions", (*postgresadapter.Store).DeleteIssueMentionRows},
		{"issue_description_versions", (*postgresadapter.Store).DeleteIssueDescriptionVersionRows},
		{"issue_comments", (*postgresadapter.Store).DeleteIssueCommentRows},
		{"issue_reactions", (*postgresadapter.Store).DeleteIssueReactionRows},
		{"comment_reactions", (*postgresadapter.Store).DeleteIssueCommentReactionRows},
		{"issue_activities", (*postgresadapter.Store).DeleteIssueActivityRows},
	} {
		t.Run(st.table, func(t *testing.T) {
			s, pool := newStore(t)
			alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
			web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
			gone, other, item, sub := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
			for n, id := range []uuid.UUID{item, gone, other, sub} {
				newIssue(t, pool, web, id, n+1, alice)
			}
			deleted, earlier := []uuid.UUID{item, sub}, []uuid.UUID{gone}
			if st.table != "issues" {
				a, b, c, d, g, o := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
				insert := insertOf(t, st.table)
				for n, row := range []struct{ issue, id uuid.UUID }{{item, a}, {item, b}, {item, c}, {sub, d}, {gone, g}, {other, o}} {
					insert(t, pool, web, row.issue, row.id, alice, n)
				}
				deleted, earlier = []uuid.UUID{a, b, d}, []uuid.UUID{c, g}
			}
			before, later := now.Add(-time.Hour), now.Add(time.Hour)
			for _, id := range earlier {
				exec(t, pool, "UPDATE "+st.table+" SET deleted_at = $2, updated_at = $2, updated_by_id = $3 WHERE id = $1", id, before, alice)
			}
			rows, stamps := rowsAsText(t, pool, st.table), audits(t, pool, st.table)

			if err := st.method(s, context.Background(), app.IssueDeletion{IssueIDs: []uuid.UUID{sub, item, gone}, By: bob, At: later}); err != nil {
				t.Fatal(err)
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
			if err := st.method(s, cancelled, app.IssueDeletion{IssueIDs: []uuid.UUID{other}, By: bob, At: later}); !errors.Is(err, context.Canceled) {
				t.Errorf("the statement on a cancelled context = %v; want context.Canceled", err)
			}
		})
	}
}
