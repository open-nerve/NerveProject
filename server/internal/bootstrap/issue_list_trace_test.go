package bootstrap

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

type listRequestKey struct{}
type listStatementKey struct{}
type listStatement struct {
	Request, SQL   string
	Args           []any
	Types          []string
	PID            uint32
	Started, Ended time.Time
	Error          string
}
type listTrace struct {
	mu      sync.Mutex
	records []*listStatement
}

func (t *listTrace) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	request, _ := ctx.Value(listRequestKey{}).(string)
	if request == "" {
		return ctx
	}
	r := &listStatement{Request: request, SQL: data.SQL, PID: conn.PgConn().PID(), Started: time.Now()}
	var err error
	r.Args, err = copyListArgs(data.Args)
	if err != nil {
		r.Error = err.Error()
	}
	for _, a := range data.Args {
		r.Types = append(r.Types, fmt.Sprintf("%T", a))
	}
	t.mu.Lock()
	t.records = append(t.records, r)
	t.mu.Unlock()
	return context.WithValue(ctx, listStatementKey{}, r)
}
func (t *listTrace) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	r, _ := ctx.Value(listStatementKey{}).(*listStatement)
	if r == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	r.Ended = time.Now()
	if data.Err != nil {
		r.Error += data.Err.Error()
	}
}
func (t *listTrace) take(request string) []listStatement {
	t.mu.Lock()
	defer t.mu.Unlock()
	var out []listStatement
	for _, r := range t.records {
		if r.Request == request {
			out = append(out, *r)
		}
	}
	return out
}

// Only the concrete input types used by the real read path are accepted.
// slices.Clone preserves typed nil and non-nil empty slices.
func copyListArgs(args []any) ([]any, error) {
	out := make([]any, len(args))
	for i, a := range args {
		switch v := a.(type) {
		case nil, uuid.UUID, int32, int64, bool, string:
			out[i] = a
		case *uuid.UUID:
			out[i] = copyListPointer(v)
		case *int64:
			out[i] = copyListPointer(v)
		case *string:
			out[i] = copyListPointer(v)
		case []uuid.UUID:
			out[i] = slices.Clone(v)
		case []bool:
			out[i] = slices.Clone(v)
		case []string:
			out[i] = slices.Clone(v)
		default:
			return nil, fmt.Errorf("unsupported captured argument %d: %T", i, a)
		}
	}
	return out, nil
}
func copyListPointer[T any](p *T) *T {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func listTracePool(t *testing.T, url string, trace *listTrace) *pgxpool.Pool {
	t.Helper()
	cfg := testConfig(t, url, false).Database
	source, err := postgres.NewPool(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	copied := source.Config()
	copied.ConnConfig.Tracer = trace
	if source.Config().ConnConfig.Tracer != nil {
		t.Fatal("Config mutated source pool")
	}
	source.Close()
	pool, err := pgxpool.NewWithConfig(context.Background(), copied)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}
