package postgresadapter_test

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// Reading a live, missing, or unrequested project must not invent a deletion
// time; reading a deleted project's time must not acquire its row lock.
func TestDeletedIssueProjects(t *testing.T) {
	s, pool := newStore(t)
	by := newAccount(t, pool, "deleted-facts@corp.com")
	workspace := newWorkspace(t, pool, "deleted-facts")
	live, gone, other := newProject(t, s, workspace, "Live", "LIVE", by), newProject(t, s, workspace, "Gone", "GONE", by), newProject(t, s, workspace, "Other", "OTHER", by)
	exec(t, pool, "UPDATE projects SET deleted_at=$2 WHERE id=ANY($1::uuid[])", []uuid.UUID{gone, other}, now)
	err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(pgtest.Soon(t), func(ctx context.Context) error {
		got, err := s.DeletedIssueProjects(ctx, []uuid.UUID{gone, live, uuid.NewV7(), gone})
		if err != nil {
			return err
		}
		if len(got) != 1 || !got[gone].Equal(now) {
			t.Fatalf("deletion facts %+v", got)
		}
		if pgtest.RowWaits(t, pool, "projects", gone, "FOR UPDATE") {
			t.Fatal("fact read locked project")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, err := s.DeletedIssueProjects(context.Background(), nil); err != nil || len(got) != 0 {
		t.Fatal(got, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.DeletedIssueProjects(ctx, []uuid.UUID{gone}); !errors.Is(err, context.Canceled) {
		t.Fatal("read error swallowed", err)
	}
}
