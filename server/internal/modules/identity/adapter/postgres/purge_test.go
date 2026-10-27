package postgresadapter_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

func purgeToken(t *testing.T, s *postgresadapter.Store, pool *pgxpool.Pool, user uuid.UUID, label string, id uuid.UUID, deleted *time.Time) uuid.UUID {
	t.Helper()
	n := newToken(user, label, now)
	n.ID = id
	mustCreateToken(t, s, n)
	if deleted != nil {
		exec(t, pool, "UPDATE api_tokens SET deleted_at=$2 WHERE id=$1", id, *deleted)
	}
	return id
}

// Compare complete persisted rows: housekeeping must not edit accounts,
// profiles, or sessions, including sessions already expired.
func accountRows(t *testing.T, pool *pgxpool.Pool) [3]string {
	t.Helper()
	var rows [3]string
	for i, query := range []string{
		"SELECT jsonb_agg(to_jsonb(u) ORDER BY id)::text FROM users u",
		"SELECT jsonb_agg(to_jsonb(p) ORDER BY id)::text FROM profiles p",
		"SELECT jsonb_agg(to_jsonb(s) ORDER BY id)::text FROM auth_sessions s",
	} {
		if err := pool.QueryRow(context.Background(), query).Scan(&rows[i]); err != nil {
			t.Fatal(err)
		}
	}
	return rows
}

// <= instead of < would delete the exact-cutoff token; ignoring the limit,
// id cursor, or deleted_at would delete a control token too.
func TestAPITokenPurgeBatches(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	u := newUser("purge@corp.com")
	mustCreate(t, s, u)
	if err := s.CreateDefaultProfile(ctx, uuid.NewV7(), u.ID, now); err != nil {
		t.Fatal(err)
	}
	sessionUntil(t, s, u, now.Add(-24*time.Hour))
	cutoff := now.Add(-60 * 24 * time.Hour)
	old := now.Add(-61 * 24 * time.Hour)
	recent := now.Add(-59 * 24 * time.Hour)
	// Deliberately insert in the opposite order to the keyset.
	high := purgeToken(t, s, pool, u.ID, "old high", uuid.MustParse("0199a2b4-0000-7000-8000-000000000030"), &old)
	low := purgeToken(t, s, pool, u.ID, "old low", uuid.MustParse("0199a2b4-0000-7000-8000-000000000010"), &old)
	atCutoff := purgeToken(t, s, pool, u.ID, "at cutoff", uuid.MustParse("0199a2b4-0000-7000-8000-000000000020"), &cutoff)
	recentID := purgeToken(t, s, pool, u.ID, "recent", uuid.MustParse("0199a2b4-0000-7000-8000-000000000040"), &recent)
	live := purgeToken(t, s, pool, u.ID, "live", uuid.MustParse("0199a2b4-0000-7000-8000-000000000050"), nil)
	before := accountRows(t, pool)
	first, err := s.PurgeAPITokens(ctx, app.PurgeBatch{Cutoff: cutoff, Limit: 1})
	if err != nil || !slices.Equal(first, []uuid.UUID{low}) {
		t.Fatalf("batch=1: %v,%v; want only lowest 61d token %s", first, err, low)
	}
	// A cursor after the remaining eligible token cannot delete it.
	skipped, err := s.PurgeAPITokens(ctx, app.PurgeBatch{Cutoff: cutoff, After: high, Limit: 1})
	if err != nil || len(skipped) != 0 {
		t.Fatalf("id cursor: %v,%v; want empty", skipped, err)
	}
	purger := identity.Provide(pool).Purger
	deleted, err := purger.Purge(ctx, cutoff, 1)
	if err != nil {
		t.Fatalf("remaining purge=%d,%v", deleted, err)
	}
	if !slices.Equal(purger.Tables(), []string{"api_tokens"}) {
		t.Fatalf("provided tables=%v", purger.Tables())
	}
	var left []uuid.UUID
	rows, err := pool.Query(ctx, "SELECT id FROM api_tokens ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		left = append(left, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if deleted+len(first) != 2 || !slices.Equal(left, []uuid.UUID{atCutoff, recentID, live}) || accountRows(t, pool) != before {
		t.Fatalf("deleted=%d atCutoffExists=%v recentExists=%v liveExists=%v left=%v accountsUnchanged=%v; want 2,true,true,true and accounts unchanged", deleted+len(first), slices.Contains(left, atCutoff), slices.Contains(left, recentID), slices.Contains(left, live), left, accountRows(t, pool) == before)
	}
	for _, limit := range []int{0, 2147483648} {
		if _, err := postgresadapter.New(nil).PurgeAPITokens(ctx, app.PurgeBatch{Cutoff: cutoff, Limit: limit}); err == nil {
			t.Fatalf("invalid limit %d accepted", limit)
		}
	}
}

// Joining s.queries(ctx) makes a successful batch disappear when the caller
// rolls back. A later database error must preserve only the committed count.
func TestPurgeDoesNotJoinCallerTransaction(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	u := newUser("rollback@corp.com")
	mustCreate(t, s, u)
	if err := s.CreateDefaultProfile(ctx, uuid.NewV7(), u.ID, now); err != nil {
		t.Fatal(err)
	}
	sessionUntil(t, s, u, now.Add(-time.Hour))
	cutoff := now.Add(-60 * 24 * time.Hour)
	old := now.Add(-61 * 24 * time.Hour)
	first := purgeToken(t, s, pool, u.ID, "first", uuid.MustParse("0199a2b4-0000-7000-8000-000000000010"), &old)
	second := purgeToken(t, s, pool, u.ID, "fail", uuid.MustParse("0199a2b4-0000-7000-8000-000000000020"), &old)
	before := accountRows(t, pool)
	exec(t, pool, `CREATE FUNCTION fail_purge() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF OLD.label='fail' THEN RAISE EXCEPTION 'purge fixture failure'; END IF; RETURN OLD; END $$`)
	exec(t, pool, `CREATE TRIGGER fail_purge BEFORE DELETE ON api_tokens FOR EACH ROW EXECUTE FUNCTION fail_purge()`)
	observer, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer observer.Release()
	rollback := errors.New("outer rollback")
	var deleted int
	var purgeErr error
	err = postgres.NewTxManager(pool, 2*time.Second).WithinTx(ctx, func(txctx context.Context) error {
		deleted, purgeErr = identity.Provide(pool).Purger.Purge(txctx, cutoff, 1)
		return rollback
	})
	var pe *pgconn.PgError
	if !errors.Is(err, rollback) || deleted != 1 || !errors.As(purgeErr, &pe) || pe.Message != "purge fixture failure" {
		t.Fatalf("purge=%d,%v outer=%v; want one committed batch, original PG error and rollback", deleted, purgeErr, err)
	}
	// The observer was held before the caller transaction started, so it
	// cannot be the caller's or the independent DELETE's connection.
	var firstExists, secondExists bool
	if err := observer.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM api_tokens WHERE id=$1),EXISTS(SELECT 1 FROM api_tokens WHERE id=$2)", first, second).Scan(&firstExists, &secondExists); err != nil {
		t.Fatal(err)
	}
	if firstExists || !secondExists || accountRows(t, pool) != before {
		t.Fatalf("after caller rollback: firstExists=%v failedBatchExists=%v accountsUnchanged=%v; want false,true,true", firstExists, secondExists, accountRows(t, pool) == before)
	}
}
