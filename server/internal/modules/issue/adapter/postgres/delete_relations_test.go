package postgresadapter_test

import (
	"context"
	"maps"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// The relations' deletion statements (delete_issue.sql, delete_rows.sql;
// M4 design 3.5, 3.6): the rows of a deleted work item or project both
// ways, whichever project holds them, locked in id order.

// relationRow is a relation as the tests insert it: its id, the project
// that holds it, its issue_id end, its related end and its stored type.
type relationRow struct {
	id             uuid.UUID
	p              project
	issue, related uuid.UUID
	kind           string
}

// The relations of a deleted subtree are deleted both ways, of each stored
// type, whichever project holds the row. Web's item and sub are deleted; x
// is another work item of web, y and z work items of ops. Of each type one
// row has item or sub at its issue_id end, and one at its related end:
// item relates_to x (a row of web) and x relates_to sub (web's); sub is
// blocked_by ops' z (web's, the other end in ops) and ops' y is blocked_by
// item (a row of ops); item is a duplicate of z (web's) and y of sub
// (ops'). x is a duplicate of y, and z is blocked_by x, naming neither;
// x's relation to item, which alice deleted an hour before, shares item
// and x's pair. One statement names sub and item, by bob at later: the six
// rows that name them are deleted at later by bob; the two naming neither
// are as they were, and the one deleted before keeps alice's deletion;
// every other column of every row is kept.
func TestTheRelationsOfADeletedSubtreeAreDeletedBothWays(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, ops := newProject(t, pool, acme, "WEB"), newProject(t, pool, acme, "OPS")
	item, sub, x, y, z := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	for n, id := range []uuid.UUID{item, sub, x} {
		newIssue(t, pool, web, id, n+1, alice)
	}
	newIssue(t, pool, ops, y, 1, alice)
	newIssue(t, pool, ops, z, 2, alice)
	before, later := now.Add(-time.Hour), now.Add(time.Hour)
	xItem := uuid.NewV7()
	newRelation(t, pool, web, x, item, xItem, "relates_to", alice)
	exec(t, pool, "UPDATE issue_relations SET deleted_at = $2, updated_at = $2, updated_by_id = $3 WHERE id = $1", xItem, before, alice)
	named := []relationRow{{uuid.NewV7(), web, item, x, "relates_to"}, {uuid.NewV7(), web, x, sub, "relates_to"},
		{uuid.NewV7(), web, sub, z, "blocked_by"}, {uuid.NewV7(), ops, y, item, "blocked_by"}, {uuid.NewV7(), web, item, z, "duplicate"},
		{uuid.NewV7(), ops, y, sub, "duplicate"}}
	neither := []relationRow{{uuid.NewV7(), web, x, y, "duplicate"}, {uuid.NewV7(), ops, z, x, "blocked_by"}}
	for _, r := range slices.Concat(named, neither) {
		newRelation(t, pool, r.p, r.issue, r.related, r.id, r.kind, alice)
	}
	rows, stamps := rowsAsText(t, pool, "issue_relations"), audits(t, pool, "issue_relations")

	if err := postgres.NewTxManager(pool, 5*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
		return app.NewRelationDeletions(s, relationDeletionProjects{}).ForIssues(ctx, web.id, app.IssueDeletion{IssueIDs: []uuid.UUID{sub, item}, By: bob, At: later})
	}); err != nil {
		t.Fatal(err)
	}

	if after := rowsAsText(t, pool, "issue_relations"); !maps.Equal(after, rows) {
		t.Errorf("the rows but their audit columns:\n%v\nwant them as they were:\n%v", after, rows)
	}
	want := maps.Clone(stamps)
	for _, r := range named {
		want[r.id] = deletion(later, bob)
	}
	if after := audits(t, pool, "issue_relations"); !maps.Equal(after, want) {
		t.Errorf("the audit columns:\n%v\nwant\n%v", after, want)
	}
}

// The statement of a deleted subtree's relations locks them FOR NO KEY
// UPDATE in id order before it writes any (convention 5's exception), and
// passes over a row deleted while it waits for it. Web's item has the
// relations a, b and c, drawn in that order, so of ascending ids, item at
// the issue_id end of a and c and the related end of b; they are inserted
// in the opposite order, so that a scan of the table, or of the index on
// either end, meets c first. Another transaction soft-deletes b, as
// another project's deletion does, by alice an hour earlier, and holds it.
// The statement, naming item, waits for b holding a, which a FOR SHARE
// then waits for, and not c, which it has not reached. Once the other
// commits, the statement deletes a and c at the moment and by the account
// given, and b keeps alice's deletion.
func TestTheRelationsOfADeletedSubtreeAreLockedInIDOrder(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	item, p, q, r := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	for n, id := range []uuid.UUID{item, p, q, r} {
		newIssue(t, pool, web, id, n+1, alice)
	}
	a, b, c := uuid.NewV7(), uuid.NewV7(), uuid.NewV7() // drawn in this order: ascending ids
	newRelation(t, pool, web, item, r, c, "relates_to", alice)
	newRelation(t, pool, web, q, item, b, "relates_to", alice)
	newRelation(t, pool, web, item, p, a, "relates_to", alice)
	earlier, later := now.Add(-time.Hour), now.Add(time.Hour)

	soon := pgtest.Soon(t) // ends the test's statements and the statement's wait for b
	held, err := pool.Begin(soon)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = held.Rollback(context.Background()) }()
	if _, err := held.Exec(soon, "UPDATE issue_relations SET deleted_at = $2, updated_at = $2, updated_by_id = $3 WHERE id = $1", b, earlier,
		alice); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		done <- postgres.NewTxManager(pool, 5*time.Second).WithinTx(soon, func(ctx context.Context) error {
			return app.NewRelationDeletions(s, relationDeletionProjects{}).ForIssues(ctx, web.id, app.IssueDeletion{IssueIDs: []uuid.UUID{item}, By: bob, At: later})
		})
	}()
	pgtest.WaitForLockWaitOn(t, pool, "issue_relations", 5*time.Second)
	if !pgtest.RowWaits(t, pool, "issue_relations", a, "FOR SHARE") || pgtest.RowWaits(t, pool, "issue_relations", c, "FOR SHARE") {
		t.Error("while the statement waits for b, a FOR SHARE of a does not wait or one of c does; want a locked first, c not yet")
	}
	if err := held.Commit(soon); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("RelationDeletions.ForIssues() = %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("RelationDeletions.ForIssues() did not end within 10s")
	}
	got := audits(t, pool, "issue_relations")
	for name, row := range map[string]struct {
		id   uuid.UUID
		want string
	}{"a": {a, deletion(later, bob)}, "b": {b, deletion(earlier, alice)}, "c": {c, deletion(later, bob)}} {
		if got[row.id] != row.want {
			t.Errorf("%s: %s, want %s", name, got[row.id], row.want)
		}
	}
}

// A deleted project's batch of relations takes the rows that name one of
// its work items at either end, of each stored type, whichever project
// holds them, the work items deleted before it among them, as the work
// items' batches delete them first. Of each type one row is web's, a work
// item of web at its issue_id end, and one is ops', a work item of web at
// its related end: w1 relates_to w2 (a row of web) and ops' o1 relates_to
// w1 (a row of ops); w2 is blocked_by ops' o2 (web's, the other end in
// ops) and ops' o3 is blocked_by w2 (ops'); w1 is a duplicate of o3
// (web's) and o1 of w2 (ops'). w1 is deleted. ops' o1 is blocked_by o2,
// naming no work item of web; o2's relation to w1, which alice deleted an
// hour before, keeps her deletion. One batch of web's, by bob at later,
// takes and deletes the six rows naming web's work items, and only them.
func TestARelationsBatchTakesTheRowsOfTheProjectsWorkItemsBothWays(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, ops := newProject(t, pool, acme, "WEB"), newProject(t, pool, acme, "OPS")
	w1, w2, o1, o2, o3 := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	newIssue(t, pool, web, w1, 1, alice)
	newIssue(t, pool, web, w2, 2, alice)
	for n, id := range []uuid.UUID{o1, o2, o3} {
		newIssue(t, pool, ops, id, n+1, alice)
	}
	earlier, later := now.Add(-time.Hour), now.Add(time.Hour)
	exec(t, pool, "UPDATE issues SET deleted_at = $2 WHERE id = $1", w1, later)
	named := []relationRow{{uuid.NewV7(), web, w1, w2, "relates_to"}, {uuid.NewV7(), ops, o1, w1, "relates_to"},
		{uuid.NewV7(), web, w2, o2, "blocked_by"}, {uuid.NewV7(), ops, o3, w2, "blocked_by"}, {uuid.NewV7(), web, w1, o3, "duplicate"},
		{uuid.NewV7(), ops, o1, w2, "duplicate"}}
	gone := uuid.NewV7()
	for _, r := range slices.Concat(named, []relationRow{{uuid.NewV7(), ops, o1, o2, "blocked_by"}, {gone, ops, o2, w1, "relates_to"}}) {
		newRelation(t, pool, r.p, r.issue, r.related, r.id, r.kind, alice)
	}
	exec(t, pool, "UPDATE issue_relations SET deleted_at = $2, updated_at = $2, updated_by_id = $3 WHERE id = $1", gone, earlier, alice)
	stamps := audits(t, pool, "issue_relations")

	var ids []uuid.UUID
	err := postgres.NewTxManager(pool, 5*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
		var err error
		ids, err = relationBatch(s, ctx, app.RowBatch{ProjectID: web.id, Limit: 10, By: bob, At: later})
		return err
	})
	var want []uuid.UUID
	for _, r := range named {
		want = append(want, r.id)
	}
	if got := slices.SortedFunc(slices.Values(ids), uuid.UUID.Compare); err != nil || !slices.Equal(got, want) {
		t.Errorf("web's batch = %v, %v; want %v", got, err, want)
	}
	for _, r := range named {
		stamps[r.id] = deletion(later, bob)
	}
	if after := audits(t, pool, "issue_relations"); !maps.Equal(after, stamps) {
		t.Errorf("the audit columns:\n%v\nwant\n%v", after, stamps)
	}
}
