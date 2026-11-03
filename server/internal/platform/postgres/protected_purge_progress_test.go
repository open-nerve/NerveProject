package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

type protectedProbe struct {
	candidates []uuid.UUID
	remove     func(context.Context, []uuid.UUID) ([]uuid.UUID, error)
	calls      int
}

func (q *protectedProbe) Candidates(context.Context, time.Time, uuid.UUID, int) ([]uuid.UUID, error) {
	return q.candidates, nil
}
func (q *protectedProbe) Delete(ctx context.Context, _ time.Time, ids []uuid.UUID) ([]uuid.UUID, error) {
	q.calls++
	return q.remove(ctx, ids)
}

func TestProtectedPurgeKeepsCommittedProgressOnError(t *testing.T) {
	for _, failure := range []error{&pgconn.PgError{Code: "23503", SchemaName: "public", TableName: "unrelated", ConstraintName: "unknown"}, &pgconn.PgError{Code: "23001"}, &pgconn.PgError{Code: "40P01"}, context.Canceled} {
		q := &protectedProbe{candidates: []uuid.UUID{protectedID(1), protectedID(2), protectedID(3)}}
		q.remove = func(_ context.Context, ids []uuid.UUID) ([]uuid.UUID, error) {
			if len(ids) > 1 {
				return nil, protectedError()
			}
			if ids[0] == protectedID(1) {
				return ids, nil
			}
			if ids[0] != protectedID(2) {
				t.Fatal("continued after unknown error")
			}
			return nil, failure
		}
		got, err := postgres.PurgeProtected(t.Context(), q, protectedRefs(), time.Now(), uuid.Nil(), 3)
		want := postgres.PurgePage{Scanned: 1, Deleted: 1, After: protectedID(1)}
		if got != want || !errors.Is(err, failure) || q.calls != 3 {
			t.Fatalf("partial=%+v %v calls=%d", got, err, q.calls)
		}
	}
}

func TestProtectedPurgeDoesNotFallbackForUnknownFailure(t *testing.T) {
	lost := &pgconn.PgError{Code: "23503", SchemaName: "public", TableName: "probe_asset", ConstraintName: "unknown"}
	q := &protectedProbe{candidates: []uuid.UUID{protectedID(3)}, remove: func(context.Context, []uuid.UUID) ([]uuid.UUID, error) { return nil, lost }}
	got, err := postgres.PurgeProtected(t.Context(), q, protectedRefs(), time.Now(), protectedID(2), 3)
	if got != (postgres.PurgePage{After: protectedID(2)}) || err != lost || q.calls != 1 {
		t.Fatalf("unknown=%+v %v calls=%d", got, err, q.calls)
	}
}

func TestProtectedPurgeCountsMissingCandidateAsScanned(t *testing.T) {
	q := &protectedProbe{candidates: []uuid.UUID{protectedID(1), protectedID(2)}, remove: func(context.Context, []uuid.UUID) ([]uuid.UUID, error) { return nil, nil }}
	got, err := postgres.PurgeProtected(t.Context(), q, protectedRefs(), time.Now(), uuid.Nil(), 2)
	if err != nil || got != (postgres.PurgePage{Scanned: 2, After: protectedID(2)}) || q.calls != 1 {
		t.Fatalf("concurrent absence=%+v %v calls=%d", got, err, q.calls)
	}
}
