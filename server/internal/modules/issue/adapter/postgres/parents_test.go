package postgresadapter_test

import (
	"context"
	"reflect"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// A work item's parent through the store (M4 design 3.9): InsertIssue
// writes the parent it is given, up, and GetIssue reads it. Every other
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
	if got, want := parents(), map[uuid.UUID]*uuid.UUID{up: nil, top: nil, other: &up, id: &up}; !reflect.DeepEqual(got, want) {
		t.Errorf("the parents %v, want %v", got, want)
	}
}
