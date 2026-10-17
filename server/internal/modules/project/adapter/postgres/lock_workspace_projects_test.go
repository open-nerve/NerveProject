package postgresadapter_test

import (
	"context"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// deleteWorkspace's lock of its projects (M4 design 3.5). It returns, in id
// order, acme's undeleted projects, the archived one too; until the
// transaction ends each of them is held FOR NO KEY UPDATE, which a FOR
// SHARE waits for and a foreign key's FOR KEY SHARE does not, and no other
// project is: not acme's deleted one, not beta's.
func TestLockWorkspaceProjects(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	web, ops := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Ops", "OPS", alice)
	arch, gone := newProject(t, s, acme, "Arch", "ARCH", alice), newProject(t, s, acme, "Gone", "GONE", alice)
	betas := newProject(t, s, beta, "Web", "WEB", alice)
	exec(t, pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", arch, now)
	exec(t, pool, "UPDATE projects SET deleted_at = $2 WHERE id = $1", gone, now)

	var locked []uuid.UUID
	err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
		var err error
		if locked, err = s.LockWorkspaceProjects(ctx, acme); err != nil {
			return err
		}
		for name, p := range map[string]struct {
			id   uuid.UUID
			held bool
		}{"Web": {web, true}, "Ops": {ops, true}, "Arch": {arch, true}, "Gone": {gone, false}, "beta's Web": {betas, false}} {
			if share, keyShare := waits(t, pool, p.id, "FOR SHARE"), waits(t, pool, p.id, "FOR KEY SHARE"); share != p.held || keyShare {
				t.Errorf("%s while locked: a FOR SHARE waits %v, a FOR KEY SHARE waits %v; want %v, false", name, share, keyShare, p.held)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := slices.SortedFunc(slices.Values([]uuid.UUID{web, ops, arch}), uuid.UUID.Compare); !slices.Equal(locked, want) {
		t.Errorf("LockWorkspaceProjects() = %v, want %v", locked, want)
	}
}

// LockWorkspaceProjects takes its locks in the projects' id order, whatever
// order the rows lie in: Web has the smaller id, but lies after Alpha in
// the table and in the indexes on the name and on the identifier. Alpha's
// row is held. LockWorkspaceProjects waits for it holding Web's, which a
// FOR SHARE then waits for; in any other order it would reach Alpha first
// and wait holding nothing.
func TestLockWorkspaceProjectsLocksInIDOrder(t *testing.T) {
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
			_, err := s.LockWorkspaceProjects(ctx, acme)
			return err
		})
	}()
	pgtest.WaitForLockWaitOn(t, pool, "projects", 10*time.Second)

	if !waits(t, pool, web, "FOR SHARE") {
		t.Error("a FOR SHARE of Web while LockWorkspaceProjects waits for Alpha does not wait; want Web locked first")
	}
	if err := held.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("LockWorkspaceProjects() = %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("LockWorkspaceProjects() did not end within 10s")
	}
}
