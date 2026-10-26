package postgresadapter_test

import (
	"context"
	"fmt"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// Candidate discovery must not freeze the policy: both locks return the
// committed value after waiting for a concurrent policy update.
func TestArchivePoliciesLockedAfterWait(t *testing.T) {
	for _, many := range []bool{false, true} {
		t.Run(fmt.Sprintf("many=%v", many), func(t *testing.T) {
			s, pool := newStore(t)
			alice := newAccount(t, pool, "alice@corp.com")
			web := newProject(t, s, newWorkspace(t, pool, "acme"), "Web", "WEB", alice)
			exec(t, pool, "UPDATE projects SET archive_in=1 WHERE id=$1", web)
			ids, err := s.ArchiveProjects(context.Background())
			if err != nil || len(ids) != 1 || ids[0] != web {
				t.Fatalf("candidate IDs = %v, %v", ids, err)
			}
			held, err := pool.Begin(pgtest.Soon(t))
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = held.Rollback(context.Background()) }()
			if _, err := held.Exec(pgtest.Soon(t), "UPDATE projects SET archive_in=12 WHERE id=$1", web); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() {
				done <- postgres.NewTxManager(pool, 10*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
					if many {
						locked, err := s.LockIssueProjects(ctx, []uuid.UUID{web})
						if err != nil {
							return err
						}
						if p, ok := locked[web]; !ok || p.ArchiveIn != 12 {
							return fmt.Errorf("locked projects = %+v; want archive_in 12", locked)
						}
					} else {
						p, found, err := s.LockIssueProject(ctx, web)
						if err != nil {
							return err
						}
						if !found || p.ArchiveIn != 12 {
							return fmt.Errorf("locked project = %+v, %v; want archive_in 12", p, found)
						}
					}
					return nil
				})
			}()
			pgtest.WaitForLockWaitOn(t, pool, "projects", 10*time.Second)
			if err := held.Commit(pgtest.Soon(t)); err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-done:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(10 * time.Second):
				t.Fatal("project lock did not finish")
			}
		})
	}
}
