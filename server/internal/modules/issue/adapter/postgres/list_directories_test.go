package postgresadapter_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"uuid"
)

func TestListDirectories(t *testing.T) {
	s, pool := newStore(t)
	owner := newAccount(t, pool, "owner@corp.com")
	left := newAccount(t, pool, "left@corp.com")
	guest := newAccount(t, pool, "guest@corp.com")
	deleted := newAccount(t, pool, "deleted@corp.com")
	outside := newAccount(t, pool, "outside@corp.com")
	wid := newWorkspace(t, pool, "acme")
	web, other := newProject(t, pool, wid, "WEB"), newProject(t, pool, wid, "OTH")
	for _, actor := range []uuid.UUID{left, guest} {
		exec(t, pool, "INSERT INTO project_members (id,workspace_id,project_id,member_id,role) VALUES ($1,$2,$3,$4,15)", uuid.NewV7(), wid, web.id, actor)
	}
	first, second, foreign := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	newIssue(t, pool, web, first, 1, owner)
	newIssue(t, pool, web, second, 2, owner)
	newIssue(t, pool, other, foreign, 1, owner)
	for _, row := range []struct {
		p            project
		issue, actor uuid.UUID
		gone         bool
	}{
		{web, second, guest, false}, {web, first, left, false}, {web, second, left, false}, {web, first, deleted, true}, {other, foreign, outside, false},
	} {
		exec(t, pool, "INSERT INTO issue_assignees (id,workspace_id,project_id,issue_id,assignee_id,deleted_at) VALUES ($1,$2,$3,$4,$5,CASE WHEN $6 THEN now() END)", uuid.NewV7(), wid, row.p.id, row.issue, row.actor, row.gone)
	}
	// These changes happen after a valid assignment; neither removes the
	// existing assignee relation or its place in the name directory.
	exec(t, pool, "UPDATE project_members SET is_active=false WHERE project_id=$1 AND member_id=$2", web.id, left)
	exec(t, pool, "UPDATE project_members SET role=5 WHERE project_id=$1 AND member_id=$2", web.id, guest)
	exec(t, pool, "UPDATE issues SET archived_at=now() WHERE id=$1", second)
	got, err := s.AssigneeIDs(context.Background(), []uuid.UUID{web.id, web.id})
	if want := []uuid.UUID{left, guest}; err != nil || !slices.Equal(got, want) {
		t.Errorf("historical assignee IDs: got %v, %v; want %v", got, err, want)
	}
	for _, empty := range [][]uuid.UUID{nil, {}, {uuid.NewV7()}} {
		got, err := s.AssigneeIDs(context.Background(), empty)
		if err != nil || got == nil || len(got) != 0 {
			t.Errorf("empty assignee scope: %v, %v", got, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got, err := s.AssigneeIDs(ctx, []uuid.UUID{web.id}); !errors.Is(err, context.Canceled) || got != nil {
		t.Errorf("cancelled assignees: %v, %v", got, err)
	}
}
