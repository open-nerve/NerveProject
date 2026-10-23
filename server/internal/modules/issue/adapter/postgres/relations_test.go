package postgresadapter_test

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// Which work items relate to a work item (M4 design 3.10): those with an
// undeleted relation to it, of any type, either way, of any project. Web's
// item is a duplicate of b, a is blocked_by it, and ops' o relates_to it;
// its relation to c was deleted; d relates to b alone. Of a, b, c, d and
// o, asked for in descending id order, a, b and o relate to it, in id
// order; none of a work item without relations; a cancelled read is its
// failure.
func TestRelated(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, ops := newProject(t, pool, acme, "WEB"), newProject(t, pool, acme, "OPS")
	item, a, b, c, d, o := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	for n, id := range []uuid.UUID{item, a, b, c, d} {
		newIssue(t, pool, web, id, n+1, alice)
	}
	newIssue(t, pool, ops, o, 1, alice)
	gone := uuid.NewV7()
	for _, r := range []relationRow{{uuid.NewV7(), web, item, b, "duplicate"}, {uuid.NewV7(), web, a, item, "blocked_by"},
		{uuid.NewV7(), ops, o, item, "relates_to"}, {gone, web, item, c, "relates_to"}, {uuid.NewV7(), web, d, b, "relates_to"}} {
		newRelation(t, pool, r.p, r.issue, r.related, r.id, r.kind, alice)
	}
	exec(t, pool, "UPDATE issue_relations SET deleted_at = $2 WHERE id = $1", gone, now)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	got, err := s.Related(ctx, item, []uuid.UUID{o, d, c, b, a})
	if want := slices.SortedFunc(slices.Values([]uuid.UUID{a, b, o}), uuid.UUID.Compare); err != nil || !slices.Equal(got, want) {
		t.Errorf("Related(item) = %v, %v; want %v", got, err, want)
	}
	if got, err := s.Related(ctx, c, []uuid.UUID{item, a}); err != nil || len(got) != 0 {
		t.Errorf("Related(c) = %v, %v; want none", got, err)
	}
	cancelled, stop := context.WithCancel(context.Background())
	stop()
	if _, err := s.Related(cancelled, item, []uuid.UUID{a}); !errors.Is(err, context.Canceled) {
		t.Errorf("Related() on a cancelled context = %v; want context.Canceled", err)
	}
}

// storedRelations are the relations' rows, by id, each as text: its
// workspace, project, ends and type; who made it and when; who wrote it
// last and when; and whether it is deleted.
func storedRelations(t *testing.T, pool *pgxpool.Pool) map[uuid.UUID]string {
	t.Helper()
	rows, err := pool.Query(context.Background(), `SELECT id, workspace_id, project_id, issue_id, related_issue_id, relation_type, created_by_id,
		created_at, updated_by_id, updated_at, deleted_at IS NOT NULL FROM issue_relations`)
	if err != nil {
		t.Fatal(err)
	}
	out := map[uuid.UUID]string{}
	for rows.Next() {
		var id, workspace, project, issue, related uuid.UUID
		var kind string
		var made, written *uuid.UUID
		var madeAt, writtenAt time.Time
		var deleted bool
		if err := rows.Scan(&id, &workspace, &project, &issue, &related, &kind, &made, &madeAt, &written, &writtenAt, &deleted); err != nil {
			t.Fatal(err)
		}
		out[id] = relationText(workspace, project, issue, related, kind, made, madeAt, written, writtenAt, deleted)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// relationText is a relation's row as storedRelations writes it.
func relationText(workspace, project, issue, related uuid.UUID, kind string, made *uuid.UUID, madeAt time.Time, written *uuid.UUID,
	writtenAt time.Time, deleted bool) string {
	return fmt.Sprintf("%s %s %s %s %s, made by %v at %s, written by %v at %s, deleted %v", workspace, project, issue, related, kind, *made,
		madeAt.UTC().Format(time.RFC3339Nano), *written, writtenAt.UTC().Format(time.RFC3339Nano), deleted)
}

// Relations are inserted as given (M4 design 3.10, 4.6): ops' o blocked by
// web's item, a row of ops, and item relating to web's b, a row of web, by
// alice at the moment, in their order; every column of each row is as
// given. Each insert's relations hold their accounts by pointers of their
// own, as domain.NewRelation makes them for the use case: the guard
// compares the accounts, not the pointers. Relations of more than one
// workspace, moment or account are refused and none is written, each
// beside alone, c's relation to item, which is written by itself after
// them; so is one of a pair that relates already, of any type, either way
// (issue_relations_pair_key), and a cancelled insert.
func TestInsertRelations(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	web, ops := newProject(t, pool, acme, "WEB"), newProject(t, pool, acme, "OPS")
	item, b, c, o := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	newIssue(t, pool, web, item, 1, alice)
	newIssue(t, pool, web, b, 2, alice)
	newIssue(t, pool, web, c, 3, alice)
	newIssue(t, pool, ops, o, 1, alice)
	workItem := func(id, p uuid.UUID) domain.Issue { return domain.Issue{ID: id, WorkspaceID: acme, ProjectID: p} }
	at := now.Add(time.Hour)
	rs := []domain.Relation{domain.NewRelation(uuid.NewV7(), workItem(o, ops.id), workItem(item, web.id), domain.BlockedBy, alice, at),
		domain.NewRelation(uuid.NewV7(), workItem(item, web.id), workItem(b, web.id), domain.RelatesTo, alice, at)}
	if err := s.InsertRelations(context.Background(), rs); err != nil {
		t.Fatal(err)
	}
	want := map[uuid.UUID]string{}
	for _, r := range rs {
		want[r.ID] = relationText(r.WorkspaceID, r.ProjectID, r.IssueID, r.RelatedID, string(r.Type), &alice, at, &alice, at, false)
	}
	if got := storedRelations(t, pool); !maps.Equal(got, want) {
		t.Errorf("the relations:\n%v\nwant\n%v", got, want)
	}

	later := rs[1]
	later.ID, later.IssueID, later.RelatedID, later.CreatedAt = uuid.NewV7(), b, o, at.Add(time.Second)
	byBob := later
	byBob.CreatedAt, byBob.CreatedByID = at, &bob
	elsewhere := byBob
	elsewhere.CreatedByID, elsewhere.WorkspaceID = &alice, beta
	again := domain.NewRelation(uuid.NewV7(), workItem(item, web.id), workItem(o, ops.id), domain.Duplicate, alice, at)
	alone := domain.NewRelation(uuid.NewV7(), workItem(c, web.id), workItem(item, web.id), domain.RelatesTo, alice, at)
	for name, rs := range map[string][]domain.Relation{"two workspaces": {alone, elsewhere}, "two moments": {alone, later},
		"two accounts": {alone, byBob}, "a pair related": {again}} {
		before := storedRelations(t, pool)
		if err := s.InsertRelations(context.Background(), rs); err == nil {
			t.Errorf("%s: InsertRelations() = nil, want an error", name)
		}
		if after := storedRelations(t, pool); !maps.Equal(after, before) {
			t.Errorf("%s: the relations\n%v\nwant them as they were\n%v", name, after, before)
		}
	}
	if err := s.InsertRelations(context.Background(), []domain.Relation{alone}); err != nil {
		t.Errorf("InsertRelations() of alone by itself = %v, want nil", err)
	}
	cancelled, stop := context.WithCancel(context.Background())
	stop()
	if err := s.InsertRelations(cancelled, []domain.Relation{again}); !errors.Is(err, context.Canceled) {
		t.Errorf("InsertRelations() on a cancelled context = %v; want context.Canceled", err)
	}
}

// Work items are marked changed at the moment and by the account given
// (M4 design 3.7), every other column of theirs as it was: web's a and
// ops' o, by bob at later; b, not named, is as it was. A deleted work item
// among those named is not written, and is an error; so is a cancelled
// write.
func TestMarkChanged(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, ops := newProject(t, pool, acme, "WEB"), newProject(t, pool, acme, "OPS")
	a, b, o, gone := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	for n, id := range []uuid.UUID{a, b, gone} {
		newIssue(t, pool, web, id, n+1, alice)
	}
	newIssue(t, pool, ops, o, 1, alice)
	exec(t, pool, "UPDATE issues SET deleted_at = $2 WHERE id = $1", gone, now)
	rows, stamps := rowsAsText(t, pool, "issues"), audits(t, pool, "issues")
	later := now.Add(time.Hour)

	if err := s.MarkChanged(context.Background(), app.ChangedRows{IDs: []uuid.UUID{a, o}, By: bob, At: later}); err != nil {
		t.Fatal(err)
	}

	if after := rowsAsText(t, pool, "issues"); !maps.Equal(after, rows) {
		t.Errorf("the work items but their audit columns:\n%v\nwant them as they were:\n%v", after, rows)
	}
	for _, id := range []uuid.UUID{a, o} {
		stamps[id] = fmt.Sprintf("undeleted, written at %s by %s", later.UTC().Format(time.RFC3339Nano), bob)
	}
	if after := audits(t, pool, "issues"); !maps.Equal(after, stamps) {
		t.Errorf("the audit columns:\n%v\nwant\n%v", after, stamps)
	}
	if err := s.MarkChanged(context.Background(), app.ChangedRows{IDs: []uuid.UUID{b, gone}, By: bob, At: later}); err == nil {
		t.Error("MarkChanged() of a deleted work item = nil, want an error")
	}
	cancelled, stop := context.WithCancel(context.Background())
	stop()
	if err := s.MarkChanged(cancelled, app.ChangedRows{IDs: []uuid.UUID{b}, By: bob, At: later}); !errors.Is(err, context.Canceled) {
		t.Errorf("MarkChanged() on a cancelled context = %v; want context.Canceled", err)
	}
}

// A work item's relations are read either way, each as stored, in the
// order they were made, (created_at, id) (M4 design 3.10): of web's item,
// a's blocked_by it, made first by bob; ops' o's duplicate of it and
// item's relates_to b, made a second later by alice, o's id drawn before,
// so first. Its relation to c was deleted; d's to b is not its. A work
// item without relations has none; a cancelled read is its failure.
func TestRelations(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, ops := newProject(t, pool, acme, "WEB"), newProject(t, pool, acme, "OPS")
	item, a, b, c, d, o := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	for n, id := range []uuid.UUID{item, a, b, c, d} {
		newIssue(t, pool, web, id, n+1, alice)
	}
	newIssue(t, pool, ops, o, 1, alice)
	ofO, ofItem, ofA, gone := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	later := now.Add(time.Second)
	want := []domain.Relation{
		{ID: ofA, WorkspaceID: acme, ProjectID: web.id, IssueID: a, RelatedID: item, Type: domain.BlockedBy, CreatedByID: &bob, CreatedAt: now},
		{ID: ofO, WorkspaceID: acme, ProjectID: ops.id, IssueID: o, RelatedID: item, Type: domain.Duplicate, CreatedByID: &alice, CreatedAt: later},
		{ID: ofItem, WorkspaceID: acme, ProjectID: web.id, IssueID: item, RelatedID: b, Type: domain.RelatesTo, CreatedByID: &alice,
			CreatedAt: later},
	}
	for _, r := range []domain.Relation{want[2], want[0], want[1]} {
		p := web
		if r.ProjectID == ops.id {
			p = ops
		}
		newRelation(t, pool, p, r.IssueID, r.RelatedID, r.ID, string(r.Type), *r.CreatedByID)
		exec(t, pool, "UPDATE issue_relations SET created_at = $2 WHERE id = $1", r.ID, r.CreatedAt)
	}
	newRelation(t, pool, web, item, c, gone, "relates_to", alice)
	exec(t, pool, "UPDATE issue_relations SET deleted_at = $2 WHERE id = $1", gone, now)
	newRelation(t, pool, web, d, b, uuid.NewV7(), "relates_to", alice)

	if got, err := s.Relations(context.Background(), item); err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("Relations(item) = %+v, %v; want %+v", got, err, want)
	}
	if got, err := s.Relations(context.Background(), c); err != nil || len(got) != 0 {
		t.Errorf("Relations(c) = %+v, %v; want none", got, err)
	}
	cancelled, stop := context.WithCancel(context.Background())
	stop()
	if _, err := s.Relations(cancelled, item); !errors.Is(err, context.Canceled) {
		t.Errorf("Relations() on a cancelled context = %v; want context.Canceled", err)
	}
}
