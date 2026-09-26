package postgresadapter_test

import (
	"context"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// sessionUntil inserts a session of u that expires at expires.
func sessionUntil(t *testing.T, s *postgresadapter.Store, u app.NewUser, expires time.Time) uuid.UUID {
	t.Helper()
	n := app.NewSession{ID: uuid.NewV7(), UserID: u.ID, TokenHash: secretHash[:], ExpiresAt: expires, Now: now}
	if err := s.CreateSession(context.Background(), n); err != nil {
		t.Fatal(err)
	}
	return n.ID
}

// sessionIDs lists every session left, sorted.
func sessionIDs(t *testing.T, pool *pgxpool.Pool) []uuid.UUID {
	t.Helper()
	rows, err := pool.Query(context.Background(), "SELECT id FROM auth_sessions")
	if err != nil {
		t.Fatal(err)
	}
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	slices.SortFunc(ids, uuid.UUID.Compare)
	return ids
}

func sorted(ids ...uuid.UUID) []uuid.UUID {
	slices.SortFunc(ids, uuid.UUID.Compare)
	return ids
}

// The cleanup deletes the sessions that expired before now, any account's,
// revoked or not, at most limit a call; what expires at now or later stays.
func TestDeleteExpiredSessions(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newUser("alice@corp.com"), newUser("bob@corp.com")
	mustCreate(t, s, alice)
	mustCreate(t, s, bob)
	sessionUntil(t, s, alice, now.Add(-time.Hour))
	revokedExpired := sessionUntil(t, s, alice, now.Add(-time.Microsecond))
	sessionUntil(t, s, bob, now.Add(-time.Minute))
	atNow := sessionUntil(t, s, alice, now)
	live := sessionUntil(t, s, alice, now.Add(time.Hour))
	revokedLive := sessionUntil(t, s, alice, now.Add(time.Hour))
	bobsLive := sessionUntil(t, s, bob, now.Add(time.Hour))
	exec(t, pool, `UPDATE auth_sessions SET revoked_at = $2, revoke_reason = 'logout' WHERE id = ANY($1)`,
		[]uuid.UUID{revokedExpired, revokedLive}, now.Add(-time.Hour))

	var deleted []int
	for range 3 {
		n, err := s.DeleteExpiredSessions(context.Background(), now, 2)
		if err != nil {
			t.Fatal(err)
		}
		deleted = append(deleted, n)
	}

	if !slices.Equal(deleted, []int{2, 1, 0}) {
		t.Errorf("DeleteExpiredSessions(limit 2) deleted %v in three calls, want [2 1 0]", deleted)
	}
	if got, want := sessionIDs(t, pool), sorted(atNow, live, revokedLive, bobsLive); !slices.Equal(got, want) {
		t.Errorf("sessions left = %v, want %v: those from now on", got, want)
	}
}

// A session another transaction holds, e.g. being refreshed or revoked, is
// skipped rather than waited for (FOR UPDATE SKIP LOCKED); the next run
// deletes it. lock_timeout turns a wait into a failure.
func TestDeleteExpiredSessionsSkipsLockedRows(t *testing.T) {
	s, pool := newStore(t)
	alice := newUser("alice@corp.com")
	mustCreate(t, s, alice)
	held, free := sessionUntil(t, s, alice, now.Add(-time.Hour)), sessionUntil(t, s, alice, now.Add(-time.Hour))
	tx := postgres.NewTxManager(pool, 2*time.Second)
	locked, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		done <- tx.WithinTx(context.Background(), func(ctx context.Context) error {
			if _, err := postgres.DB(ctx, pool).Exec(ctx, "UPDATE auth_sessions SET updated_at = $2 WHERE id = $1", held, later); err != nil {
				return err
			}
			close(locked)
			<-release
			return nil
		})
	}()
	select {
	case <-locked:
	case err := <-done:
		t.Fatalf("locking the session: %v", err)
	}

	var n int
	err := tx.WithinTx(context.Background(), func(ctx context.Context) error {
		if _, err := postgres.DB(ctx, pool).Exec(ctx, "SET LOCAL lock_timeout = '500ms'"); err != nil {
			return err
		}
		var err error
		n, err = s.DeleteExpiredSessions(ctx, now, 1000)
		return err
	})
	close(release)
	if err := <-done; err != nil {
		t.Fatalf("the transaction holding the session: %v", err)
	}

	if err != nil || n != 1 || !slices.Equal(sessionIDs(t, pool), []uuid.UUID{held}) {
		t.Errorf("DeleteExpiredSessions() = %d, %v leaving %v; want %v deleted without waiting, %v left", n, err, sessionIDs(t, pool), free, held)
	}
	if n, err := s.DeleteExpiredSessions(context.Background(), now, 1000); err != nil || n != 1 || len(sessionIDs(t, pool)) != 0 {
		t.Errorf("the next run = %d, %v; want the released session deleted", n, err)
	}
}
