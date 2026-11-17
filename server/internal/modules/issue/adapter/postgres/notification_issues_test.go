package postgresadapter_test

import (
	"context"
	"errors"
	"testing"
	"uuid"
)

// A project's work items remain while one of them is undeleted, archived or not, and a sub-item too; deleted ones,
// another project's and a project without any do not count (M7 P1b spec 2.6, 2.8). It reads what is committed.
func TestProjectIssuesRemain(t *testing.T) {
	s, pool := newStore(t)
	by := newAccount(t, pool, "remain@corp.com")
	acme := newWorkspace(t, pool, "remain")
	web, ops, empty := newProject(t, pool, acme, "WEB"), newProject(t, pool, acme, "OPS"), newProject(t, pool, acme, "EMP")
	parent, child, shelved, far := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	newIssue(t, pool, web, parent, 1, by)
	newIssue(t, pool, web, child, 2, by)
	newIssue(t, pool, web, shelved, 3, by)
	newIssue(t, pool, ops, far, 1, by)
	exec(t, pool, "UPDATE issues SET parent_id = $1 WHERE id = $2", parent, child)
	exec(t, pool, "UPDATE issues SET archived_at = now() WHERE id = $1", shelved)
	remain := func(p project, want bool) {
		t.Helper()
		if got, err := s.ProjectIssuesRemain(context.Background(), p.id); err != nil || got != want {
			t.Fatalf("project %s remain=%t,%v; want %t", p.id, got, err, want)
		}
	}
	remain(web, true)
	remain(empty, false)
	for _, id := range []uuid.UUID{parent, child} {
		exec(t, pool, "UPDATE issues SET deleted_at = now() WHERE id = $1", id)
		remain(web, true)
	}
	exec(t, pool, "UPDATE issues SET deleted_at = now() WHERE id = $1", shelved)
	remain(web, false)
	remain(ops, true)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.ProjectIssuesRemain(cancelled, ops.id); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled read=%v", err)
	}
}
