package postgresadapter_test

import (
	"context"
	"errors"
	"maps"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// A work item's sub-items as SubIssueCounts counts them (M4 design 3.9):
// its undeleted, unarchived direct sub-items, all of them and those the
// account given created, whoever changed them last. In web, p has a and
// b, alice's, c, bob's, changed last by alice, d, bob's, archived, and e,
// alice's, deleted; a has g under it, which is not p's. q has r, bob's; s
// has none. For alice, p has 3, 2 of them hers, and q 1, none hers; for
// bob, p has 3, 1 his, and q 1, his. s, and a work item not there, have
// none, and no count.
func TestSubIssueCounts(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	p, a, b, c, d, e, g, q, r, sub := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(),
		uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	for n, made := range []struct {
		id uuid.UUID
		by uuid.UUID
	}{{p, alice}, {a, alice}, {b, alice}, {c, bob}, {d, bob}, {e, alice}, {g, alice}, {q, alice}, {r, bob}, {sub, alice}} {
		newIssue(t, pool, web, made.id, n+1, made.by)
	}
	for _, link := range [][2]uuid.UUID{{a, p}, {b, p}, {c, p}, {d, p}, {e, p}, {g, a}, {r, q}} {
		exec(t, pool, "UPDATE issues SET parent_id = $2 WHERE id = $1", link[0], link[1])
	}
	exec(t, pool, "UPDATE issues SET archived_at = $2 WHERE id = $1", d, now)
	exec(t, pool, "UPDATE issues SET deleted_at = $2 WHERE id = $1", e, now)
	exec(t, pool, "UPDATE issues SET updated_by_id = $2 WHERE id = $1", c, alice)

	ctx := context.Background()
	asked := []uuid.UUID{p, q, sub, uuid.NewV7()}
	for _, tt := range []struct {
		name  string
		actor uuid.UUID
		want  map[uuid.UUID]app.SubIssueCount
	}{
		{"alice", alice, map[uuid.UUID]app.SubIssueCount{p: {All: 3, Own: 2}, q: {All: 1}}},
		{"bob", bob, map[uuid.UUID]app.SubIssueCount{p: {All: 3, Own: 1}, q: {All: 1, Own: 1}}},
	} {
		if got, err := s.SubIssueCounts(ctx, tt.actor, asked); err != nil || !maps.Equal(got, tt.want) {
			t.Errorf("SubIssueCounts() for %s = %v, %v; want %v", tt.name, got, err, tt.want)
		}
	}
	cancelled, stop := context.WithCancel(ctx)
	stop()
	if _, err := s.SubIssueCounts(cancelled, alice, asked); !errors.Is(err, context.Canceled) {
		t.Errorf("SubIssueCounts() on a cancelled context = %v; want context.Canceled", err)
	}
}

// family is a work item p of web and the work items under it, drawn in
// this order: a, alice's, assigned to her and labelled; b, bob's, made an
// hour before the others, though drawn after a; x, bob's; c, alice's,
// archived; d, alice's, deleted; g, under a, not under p. q has none
// under it.
type family struct {
	p, a, b, x, c, d, g, q uuid.UUID
}

// newFamily stores a family in web, of alice and bob.
func newFamily(t *testing.T, s *postgresadapter.Store, pool *pgxpool.Pool, web project, alice, bob uuid.UUID) family {
	t.Helper()
	f := family{p: uuid.NewV7(), a: uuid.NewV7(), b: uuid.NewV7(), x: uuid.NewV7(), c: uuid.NewV7(), d: uuid.NewV7(), g: uuid.NewV7(),
		q: uuid.NewV7()}
	for n, made := range []struct{ id, by uuid.UUID }{{f.p, alice}, {f.a, alice}, {f.b, bob}, {f.x, bob}, {f.c, alice}, {f.d, alice},
		{f.g, alice}, {f.q, alice}} {
		newIssue(t, pool, web, made.id, n+1, made.by)
	}
	for _, link := range [][2]uuid.UUID{{f.a, f.p}, {f.b, f.p}, {f.x, f.p}, {f.c, f.p}, {f.d, f.p}, {f.g, f.a}} {
		exec(t, pool, "UPDATE issues SET parent_id = $2 WHERE id = $1", link[0], link[1])
	}
	exec(t, pool, "UPDATE issues SET created_at = $2 WHERE id = $1", f.b, now.Add(-time.Hour))
	exec(t, pool, "UPDATE issues SET archived_at = $2 WHERE id = $1", f.c, now)
	exec(t, pool, "UPDATE issues SET deleted_at = $2 WHERE id = $1", f.d, now)
	ctx := context.Background()
	if err := s.InsertAssignees(ctx, linkRows(web, f.a, alice, now, alice)); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertLabels(ctx, linkRows(web, f.a, alice, now, newLabel(t, pool, web, "Bug"))); err != nil {
		t.Fatal(err)
	}
	return f
}

// read are the work items ids as GetIssue reads each, in their order.
func read(t *testing.T, s *postgresadapter.Store, ids ...uuid.UUID) []domain.Issue {
	t.Helper()
	out := []domain.Issue{}
	for _, id := range ids {
		i, found, err := s.GetIssue(context.Background(), id)
		if err != nil || !found {
			t.Fatalf("GetIssue(%s) = %v, %v", id, found, err)
		}
		out = append(out, i)
	}
	return out
}

// The work items GetIssues reads (M4 design 3.9): the undeleted ones among
// the ids given, archived ones too, each as GetIssue reads it, in id
// order, whatever the order given or stored; none for none. a, the first
// of them by id, moves under q after the others are stored, which gives
// its row a new place, the last of them both in the table and in the
// index on (project_id, state_id): a read without its ORDER BY answers it
// last whether the database scans the table or reads that index.
func TestGetIssues(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	f := newFamily(t, s, pool, web, alice, bob)
	exec(t, pool, "UPDATE issues SET parent_id = $2 WHERE id = $1", f.a, f.q)
	ctx := context.Background()
	for _, tt := range []struct {
		name string
		ids  []uuid.UUID
		want []domain.Issue
	}{
		{"some", []uuid.UUID{f.x, f.d, uuid.NewV7(), f.c, f.a}, read(t, s, f.a, f.x, f.c)},
		{"none", nil, read(t, s)},
	} {
		if got, err := s.GetIssues(ctx, tt.ids); err != nil || !reflect.DeepEqual(got, tt.want) {
			t.Errorf("GetIssues(%s) = %+v, %v; want %+v", tt.name, got, err, tt.want)
		}
	}
	cancelled, stop := context.WithCancel(ctx)
	stop()
	if got, err := s.GetIssues(cancelled, []uuid.UUID{f.a}); !errors.Is(err, context.Canceled) || got != nil {
		t.Errorf("GetIssues() on a cancelled context = %+v, %v; want context.Canceled", got, err)
	}
}
