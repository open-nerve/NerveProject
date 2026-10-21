package postgresadapter_test

import (
	"context"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// The issue module's lock of several projects (M4 design 3.5, 6.5,
// convention 7): those asked for that are undeleted, as LockIssueProject
// answers each, by id, an archived one too; not a deleted one, nor one
// that is not there. Until the transaction ends each is held FOR NO KEY
// UPDATE, which a FOR SHARE waits for and a foreign key's FOR KEY SHARE
// does not; Ops, not asked for, is not held, nor is the workspace's row.
func TestLockIssueProjects(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, arch, gone, ops := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Arch", "ARCH", alice), newProject(t, s, acme,
		"Gone", "GONE", alice), newProject(t, s, acme, "Ops", "OPS", alice)
	exec(t, pool, "UPDATE projects SET default_assignee_id = $2 WHERE id = $1", web, alice)
	exec(t, pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", arch, now)
	exec(t, pool, "UPDATE projects SET deleted_at = $2 WHERE id = $1", gone, now)

	err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
		locked, err := s.LockIssueProjects(ctx, []uuid.UUID{gone, uuid.NewV7(), arch, web})
		if err != nil {
			return err
		}
		want := map[uuid.UUID]app.IssueProject{web: {WorkspaceID: acme, DefaultAssigneeID: &alice, Identifier: "WEB"},
			arch: {WorkspaceID: acme, Archived: true, Identifier: "ARCH"}}
		if jsonOf(t, locked) != jsonOf(t, want) {
			t.Errorf("LockIssueProjects() = %+v, want %+v", locked, want)
		}
		for name, tt := range map[string]struct {
			id   uuid.UUID
			held bool
		}{"Web": {web, true}, "Arch": {arch, true}, "Ops": {ops, false}} {
			if share, keyShare := waits(t, pool, tt.id, "FOR SHARE"), waits(t, pool, tt.id, "FOR KEY SHARE"); share != tt.held || keyShare {
				t.Errorf("%s while locked: a FOR SHARE waits %v, a FOR KEY SHARE waits %v; want %v, false", name, share, keyShare, tt.held)
			}
		}
		if pgtest.RowWaits(t, pool, "workspaces", acme, "FOR UPDATE") {
			t.Error("a FOR UPDATE of acme waits while the projects are locked; want the workspace's row not locked")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// LockIssueProjects takes its locks in the projects' id order, whatever
// order they are asked in or their rows lie in: Web has the smaller id,
// but lies after Alpha in the table and in the indexes on the name and on
// the identifier, and is asked for after it. Alpha's row is held.
// LockIssueProjects waits for it holding Web's, which a FOR SHARE then
// waits for; in any other order it would reach Alpha first and wait
// holding nothing.
func TestLockIssueProjectsLocksInIDOrder(t *testing.T) {
	s, pool := newStore(t)
	acme := newWorkspace(t, pool, "acme")
	web, alpha := uuid.NewV7(), uuid.NewV7() // web drawn first: the smaller id
	for _, p := range []struct {
		id   uuid.UUID
		name string
	}{{alpha, "Alpha"}, {web, "Web"}} {
		exec(t, pool, "INSERT INTO projects (id, workspace_id, name, identifier) VALUES ($1, $2, $3::text, upper($3::text))", p.id, acme, p.name)
	}
	held, err := pool.Begin(pgtest.Soon(t))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = held.Rollback(context.Background()) }()
	if _, err := held.Exec(context.Background(), "SELECT 1 FROM projects WHERE id = $1 FOR NO KEY UPDATE", alpha); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		done <- postgres.NewTxManager(pool, 10*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
			_, err := s.LockIssueProjects(ctx, []uuid.UUID{alpha, web})
			return err
		})
	}()
	pgtest.WaitForLockWaitOn(t, pool, "projects", 10*time.Second)

	if !waits(t, pool, web, "FOR SHARE") {
		t.Error("a FOR SHARE of Web while LockIssueProjects waits for Alpha does not wait; want Web locked first")
	}
	if err := held.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("LockIssueProjects() = %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("LockIssueProjects() did not end within 10s")
	}
}

// A project deleted while LockIssueProjects waits for its row is left out
// (M4 design 3.5): another transaction holds Web FOR NO KEY UPDATE, as
// deleteProject does, and soft-deletes it; LockIssueProjects, which found
// Web undeleted, waits for it; once the deletion commits, it evaluates
// deleted_at again on the row's newest version and answers Ops alone.
func TestLockIssueProjectsLeavesOutAProjectDeletedWhileItWaited(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, ops := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Ops", "OPS", alice)
	deletion, err := pool.Begin(pgtest.Soon(t))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = deletion.Rollback(context.Background()) }()
	if _, err := deletion.Exec(context.Background(), "SELECT 1 FROM projects WHERE id = $1 FOR NO KEY UPDATE", web); err != nil {
		t.Fatal(err)
	}
	if _, err := deletion.Exec(context.Background(), "UPDATE projects SET deleted_at = now() WHERE id = $1", web); err != nil {
		t.Fatal(err)
	}
	type answer struct {
		locked map[uuid.UUID]app.IssueProject
		err    error
	}
	done := make(chan answer, 1)
	go func() {
		var a answer
		a.err = postgres.NewTxManager(pool, 10*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
			var err error
			a.locked, err = s.LockIssueProjects(ctx, []uuid.UUID{web, ops})
			return err
		})
		done <- a
	}()
	pgtest.WaitForLockWaitOn(t, pool, "projects", 10*time.Second)
	if err := deletion.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case a := <-done:
		if _, hasOps := a.locked[ops]; a.err != nil || len(a.locked) != 1 || !hasOps {
			t.Errorf("LockIssueProjects() = %+v, %v; want Ops (%s) alone, Web (%s) deleted while it waited", a.locked, a.err, ops, web)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("LockIssueProjects() did not end within 10s")
	}
}
