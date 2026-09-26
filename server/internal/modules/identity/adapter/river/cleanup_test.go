package riveradapter_test

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/riverqueue/river"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	riveradapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/river"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

var now = clocktest.At(time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)).Now()

func TestCleanupKind(t *testing.T) {
	if got := (riveradapter.CleanupArgs{}).Kind(); got != "identity.cleanup_expired_sessions" {
		t.Errorf("Kind() = %q, want identity.cleanup_expired_sessions (M2 design 3.15)", got)
	}
}

// The worker's integration test calls Work (M2 design 3.15): the expired
// sessions of both accounts go, the live ones stay.
func TestCleanupWorkerDeletesTheExpiredSessions(t *testing.T) {
	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, config.DatabaseConfig{URL: pgtest.NewDatabase(t), MaxConns: 2})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	store := postgresadapter.New(pool)
	var live []uuid.UUID
	for _, email := range []string{"alice@corp.com", "bob@corp.com"} {
		u := app.NewUser{ID: uuid.NewV7(), Email: email, PasswordHash: "$argon2id$v=19$m=64,t=1,p=1$c2FsdA$a2V5", DisplayName: "x", Now: now}
		if err := store.CreateUser(ctx, u); err != nil {
			t.Fatal(err)
		}
		for _, expires := range []time.Time{now.Add(-time.Minute), now.Add(time.Minute)} {
			id := uuid.NewV7()
			if err := store.CreateSession(ctx, app.NewSession{ID: id, UserID: u.ID, TokenHash: make([]byte, 32), ExpiresAt: expires, Now: now}); err != nil {
				t.Fatal(err)
			}
			if expires.After(now) {
				live = append(live, id)
			}
		}
	}
	w := riveradapter.NewCleanupWorker(app.NewCleanupSessions(store, clocktest.At(now), slog.New(slog.DiscardHandler)))

	if err := w.Work(ctx, &river.Job[riveradapter.CleanupArgs]{}); err != nil {
		t.Fatalf("Work() = %v", err)
	}

	rows, err := pool.Query(ctx, "SELECT id FROM auth_sessions")
	if err != nil {
		t.Fatal(err)
	}
	var left []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		left = append(left, id)
	}
	slices.SortFunc(left, uuid.UUID.Compare)
	slices.SortFunc(live, uuid.UUID.Compare)
	if !slices.Equal(left, live) {
		t.Errorf("sessions left = %v, want the live ones %v", left, live)
	}
}

type failingCleanup struct{ err error }

func (f failingCleanup) Execute(context.Context) (int, error) { return 3, f.err }

// A failed cleanup fails the job, so that River retries it.
func TestCleanupWorkerFailsWithTheCleanup(t *testing.T) {
	boom := errors.New("connection reset")

	if err := riveradapter.NewCleanupWorker(failingCleanup{boom}).Work(context.Background(), &river.Job[riveradapter.CleanupArgs]{}); !errors.Is(err, boom) {
		t.Errorf("Work() = %v, want %v", err, boom)
	}
}
