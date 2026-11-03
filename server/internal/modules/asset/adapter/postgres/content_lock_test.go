package postgresadapter

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

func TestContentBindingsOrderedDatabaseLocks(t *testing.T) {
	w := newAssetWorld(t)
	var ids []uuid.UUID
	for range 2 {
		a := w.putImage(t, w.preparedImage(t, domain.ScopeIssue, domain.IssueDescription))
		ids = append(ids, a.ID)
	}
	slices.SortFunc(ids, uuid.UUID.Compare)
	holder, err := w.pool.Begin(pgtest.Soon(t))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = holder.Rollback(context.Background()) }()
	if _, err := holder.Exec(pgtest.Soon(t), "SELECT id FROM file_assets WHERE id=$1 FOR UPDATE", ids[0]); err != nil {
		t.Fatal(err)
	}
	locked, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	go func() {
		done <- contentTx(t, w, false, func(ctx context.Context) error {
			q := w.contentRequest()
			q.ImageIDs = []uuid.UUID{ids[1], ids[0], ids[1]}
			if _, err := w.contentBinder().LockContent(ctx, q); err != nil {
				return err
			}
			close(locked)
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}()
	pgtest.WaitForLockWaitBehind(t, w.pool, holder.Conn().PgConn(), 3*time.Second)
	probe, err := w.pool.Begin(pgtest.Soon(t))
	if err != nil {
		t.Fatal(err)
	}
	_, err = probe.Exec(pgtest.Soon(t), "SELECT id FROM file_assets WHERE id=$1 FOR UPDATE NOWAIT", ids[1])
	_ = probe.Rollback(context.Background())
	if err != nil {
		t.Fatalf("high ID was locked before waiting for low ID: %v", err)
	}
	if err := holder.Commit(pgtest.Soon(t)); err != nil {
		t.Fatal(err)
	}
	select {
	case <-locked:
	case err := <-done:
		t.Fatalf("binding ended before complete lock: %v", err)
	case <-pgtest.Soon(t).Done():
		t.Fatal("binding did not acquire its complete set")
	}
	for _, id := range ids {
		probe, err := w.pool.Begin(pgtest.Soon(t))
		if err != nil {
			t.Fatal(err)
		}
		_, err = probe.Exec(pgtest.Soon(t), "SELECT id FROM file_assets WHERE id=$1 FOR UPDATE NOWAIT", id)
		_ = probe.Rollback(context.Background())
		var pe *pgconn.PgError
		if !errors.As(err, &pe) || pe.Code != "55P03" {
			t.Fatalf("asset was not held after LockContent: %v", err)
		}
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
