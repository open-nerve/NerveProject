package postgresadapter_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// A work item's ancestry (M4 design 3.9) is it and each work item above it,
// however high, each once, archived and deleted ones too: in web, c is
// under b, under a, archived, under root, deleted; d has none above it. In
// ops, x is under y and y under x, a loop no write makes: the read ends,
// each once. The work items are drawn so that no line's id order is its
// order. A work item not there has none.
func TestAWorkItemsAncestry(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, ops := newProject(t, pool, acme, "WEB"), newProject(t, pool, acme, "OPS")
	b, root, d, c, a, y, x := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	for n, id := range []uuid.UUID{root, a, b, c, d} {
		newIssue(t, pool, web, id, n+1, alice)
	}
	newIssue(t, pool, ops, x, 1, alice)
	newIssue(t, pool, ops, y, 2, alice)
	for _, link := range [][2]uuid.UUID{{a, root}, {b, a}, {c, b}, {x, y}, {y, x}} {
		exec(t, pool, "UPDATE issues SET parent_id = $2 WHERE id = $1", link[0], link[1])
	}
	exec(t, pool, "UPDATE issues SET archived_at = $2 WHERE id = $1", a, now)
	exec(t, pool, "UPDATE issues SET deleted_at = $2 WHERE id = $1", root, now)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, tt := range []struct {
		name string
		id   uuid.UUID
		want []uuid.UUID
	}{
		{"c", c, []uuid.UUID{c, b, a, root}},
		{"a, archived", a, []uuid.UUID{a, root}},
		{"root, deleted", root, []uuid.UUID{root}},
		{"d, with none above it", d, []uuid.UUID{d}},
		{"x, in a loop", x, []uuid.UUID{x, y}},
		{"no work item", uuid.NewV7(), nil},
	} {
		got, err := s.Ancestry(ctx, tt.id)
		if want := slices.SortedFunc(slices.Values(tt.want), uuid.UUID.Compare); err != nil ||
			!slices.Equal(slices.SortedFunc(slices.Values(got), uuid.UUID.Compare), want) {
			t.Errorf("Ancestry(%s) = %v, %v; want %v in any order", tt.name, got, err, want)
		}
	}
	cancelled, stop := context.WithCancel(context.Background())
	stop()
	if _, err := s.Ancestry(cancelled, c); !errors.Is(err, context.Canceled) {
		t.Errorf("Ancestry() on a cancelled context = %v; want context.Canceled", err)
	}
}

// A work item's parent through the store (M4 design 3.9): InsertIssue
// writes the parent it is given, up, and GetIssue reads it; UpdateIssue
// writes the one it is given, top, then none, NULL. Each time every other
// work item keeps its parent: other is under up, and up and top under
// none.
func TestAWorkItemsParentIsWritten(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	alice := newAccount(t, pool, "alice@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	up, top, other, id := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	for n, i := range []uuid.UUID{up, top, other} {
		newIssue(t, pool, web, i, n+1, alice)
	}
	exec(t, pool, "UPDATE issues SET parent_id = $2 WHERE id = $1", other, up)
	parents := func() map[uuid.UUID]*uuid.UUID {
		t.Helper()
		rows, err := pool.Query(ctx, "SELECT id, parent_id FROM issues")
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		out := map[uuid.UUID]*uuid.UUID{}
		for rows.Next() {
			var i uuid.UUID
			var parent *uuid.UUID
			if err := rows.Scan(&i, &parent); err != nil {
				t.Fatal(err)
			}
			out[i] = parent
		}
		return out
	}

	if err := s.InsertIssue(ctx, app.IssueRow{ID: id, WorkspaceID: web.workspace, ProjectID: web.id, SequenceID: 4, Name: "Part",
		StateID: web.state, Priority: domain.PriorityNone, ParentID: &up, SortOrder: 65535, By: alice, At: now}); err != nil {
		t.Fatal(err)
	}
	if got, found, err := s.GetIssue(ctx, id); err != nil || !found || !reflect.DeepEqual(got.ParentID, &up) {
		t.Errorf("GetIssue() = %+v, %v, %v; want it under %s", got, found, err, up)
	}
	for _, to := range []*uuid.UUID{&up, &top, nil} {
		if to != &up {
			if err := s.UpdateIssue(ctx, app.IssueFields{ID: id, Name: "Part", StateID: web.state, Priority: domain.PriorityNone, ParentID: to,
				By: alice, At: now}); err != nil {
				t.Fatal(err)
			}
		}
		if got, want := parents(), map[uuid.UUID]*uuid.UUID{up: nil, top: nil, other: &up, id: to}; !reflect.DeepEqual(got, want) {
			t.Errorf("the parents %v, want %v", got, want)
		}
	}
}
