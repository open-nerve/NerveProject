package bootstrap

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

type assetSnapshotKey struct{}
type assetSnapshotTrace struct {
	gate    *gate
	mu      sync.Mutex
	queries []string
	err     error
}

func (p *assetSnapshotTrace) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	if httpserver.RequestID(ctx) != "asset-snapshot-reader" {
		return ctx
	}
	p.mu.Lock()
	p.queries = append(p.queries, data.SQL)
	p.mu.Unlock()
	if strings.HasPrefix(data.SQL, "-- name: FindAsset :one") {
		return context.WithValue(ctx, assetSnapshotKey{}, true)
	}
	return ctx
}
func (p *assetSnapshotTrace) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	if found, _ := ctx.Value(assetSnapshotKey{}).(bool); !found || data.Err != nil {
		return
	}
	err := p.gate.wait(ctx)
	p.mu.Lock()
	p.err = err
	p.mu.Unlock()
}

// Pause the real HTTP query after its asset read, delete the issue through a
// second HTTP request, then read the host. A missing snapshot sees the new
// tombstone and mixes two different database states within one metadata read.
func TestAssetReadUsesOneSnapshot(t *testing.T) {
	dbURL := pgtest.NewDatabase(t)
	pool := openPool(t, dbURL)
	cfg := testConfig(t, dbURL, false)
	trace := &assetSnapshotTrace{gate: newGate()}
	factory := func(ctx context.Context, c config.DatabaseConfig) (*pgxpool.Pool, error) {
		source, err := postgres.NewPool(ctx, c)
		if err != nil {
			return nil, err
		}
		pc := source.Config()
		source.Close()
		pc.ConnConfig.Tracer = trace
		return pgxpool.NewWithConfig(ctx, pc)
	}
	a, err := newAppWithPoolFactory(t.Context(), cfg, slog.New(slog.DiscardHandler), migrations.FS(), testWebUI, factory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(a.close)
	w := prepareIssueWorld(t, issueWorld{base: runBuiltApp(t, a), pool: pool, contract: apitest.Load(t), tokens: map[string]string{}, ids: map[string]uuid.UUID{}})
	item := w.newWorkItem(t)
	row := assetRowForHost(t, w, domain.ScopeIssue, item, item)
	req := newRequest(t, "GET", w.base+"/api/v0/assets/"+row.ID.String(), w.tokens["bob"], nil)
	req.Header.Set(httpserver.HeaderRequestID, "asset-snapshot-reader")
	var once sync.Once
	resume := func() { once.Do(func() { close(trace.gate.open) }) }
	defer resume()
	response := sendInBackground(req)
	select {
	case <-trace.gate.held:
	case got := <-response:
		t.Fatalf("metadata read ended before its database barrier: %+v", got)
	case <-time.After(5 * time.Second):
		t.Fatal("metadata read did not reach its database barrier")
	}
	w.send(t, "alice", "DELETE", "/api/v0/issues/"+item.String(), "", 204)
	resume()
	got := receiveWithin(t, response, 5*time.Second, "metadata snapshot")
	if got.err != nil || got.res.StatusCode != 200 {
		t.Fatalf("metadata mixed snapshots: %+v", got)
	}
	w.contract.CheckResponse(t, req, got.res)
	trace.mu.Lock()
	queries, traceErr := strings.Join(trace.queries, "\n"), trace.err
	trace.mu.Unlock()
	if traceErr != nil || !strings.Contains(strings.ToLower(queries), "begin isolation level repeatable read read only") {
		t.Fatalf("not the production read-only snapshot: %s; %v", queries, traceErr)
	}
	assetAPICall(t, w, "bob", "GET", "/api/v0/assets/"+row.ID.String(), "", 404)
}
