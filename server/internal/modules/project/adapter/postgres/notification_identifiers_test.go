package postgresadapter_test

import (
	"context"
	"errors"
	"maps"
	"sync/atomic"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// statements counts the statements its pool's connections send.
type statements struct{ n atomic.Int64 }

func (s *statements) TraceQueryStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryStartData) context.Context {
	s.n.Add(1)
	return ctx
}

func (*statements) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

// The identifiers of the projects of the work items a page of
// notifications names (M7 P1c spec 2.6; pre-flight re-84 m6), read by the
// ids it is asked, in one statement: each undeleted project's among them,
// an archived one's and another workspace's too, each once though asked
// twice; none for a deleted project, an id of no project or no ids. The
// workspaces' other projects are not read: the store answers the ids, not
// a workspace.
func TestNotificationIdentifiers(t *testing.T) {
	s, pool := newStore(t)
	maker := newAccount(t, pool, "maker@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	web, ops := newProject(t, s, acme, "Web", "WEB", maker), newProject(t, s, acme, "Ops", "OPS", maker)
	old, gone := newProject(t, s, acme, "Old", "OLD", maker), newProject(t, s, acme, "Gone", "GONE", maker)
	site := newProject(t, s, beta, "Site", "SITE", maker)
	exec(t, pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", old, now)
	exec(t, pool, "UPDATE projects SET deleted_at = $2 WHERE id = $1", gone, now)
	trace := &statements{}
	config := pool.Config()
	config.ConnConfig.Tracer = trace
	traced, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(traced.Close)
	store := postgresadapter.New(traced)
	for _, tt := range []struct {
		name string
		ids  []uuid.UUID
		want map[uuid.UUID]string
	}{
		{"Web, Old archived, Site of beta, Gone deleted and none", []uuid.UUID{site, gone, uuid.NewV7(), old, web, web},
			map[uuid.UUID]string{web: "WEB", old: "OLD", site: "SITE"}},
		{"Ops alone", []uuid.UUID{ops}, map[uuid.UUID]string{ops: "OPS"}},
		{"Gone deleted and none", []uuid.UUID{gone, uuid.NewV7()}, map[uuid.UUID]string{}},
		{"no ids", nil, map[uuid.UUID]string{}},
	} {
		trace.n.Store(0)
		got, err := store.NotificationIdentifiers(context.Background(), tt.ids)
		if err != nil || len(got) != len(tt.want) || !maps.Equal(got, tt.want) {
			t.Errorf("%s: NotificationIdentifiers() = %v, %v; want %v", tt.name, got, err, tt.want)
		}
		if n := trace.n.Load(); n != 1 {
			t.Errorf("%s: NotificationIdentifiers() sent %d statements; want one", tt.name, n)
		}
	}
}

// The identifiers are read in the transaction ctx carries: a project the
// transaction wrote is read before it commits, and none once it is rolled
// back. A failed read is its error, not none.
func TestNotificationIdentifiersReadInTheTransaction(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, pool, "acme")
	project := uuid.NewV7()
	var inside map[uuid.UUID]string
	err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
		if err := s.CreateProject(ctx, app.ProjectRow{ID: project, WorkspaceID: acme, Name: "Web", Identifier: "WEB", Network: domain.NetworkPublic,
			Timezone: "UTC", CreatedBy: alice, Now: now}); err != nil {
			return err
		}
		var err error
		inside, err = s.NotificationIdentifiers(ctx, []uuid.UUID{project})
		return errors.Join(err, errors.New("roll back"))
	})
	if want := map[uuid.UUID]string{project: "WEB"}; err == nil || !maps.Equal(inside, want) {
		t.Errorf("in the transaction: %v, %v; want %v", inside, err, want)
	}
	if got, err := s.NotificationIdentifiers(context.Background(), []uuid.UUID{project}); err != nil || len(got) != 0 {
		t.Errorf("after the rollback: %v, %v; want none", got, err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if got, err := s.NotificationIdentifiers(cancelled, []uuid.UUID{project}); !errors.Is(err, context.Canceled) || got != nil {
		t.Errorf("on a cancelled context: %v, %v; want context.Canceled", got, err)
	}
}
