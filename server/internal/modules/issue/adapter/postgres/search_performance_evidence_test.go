package postgresadapter_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type searchPerfRequestKey struct{}
type searchPerfStatementKey struct{}
type searchPerfStatement struct {
	SQL, Begin, Error string
	Args              []any
	Types             []string
	PID               uint32
	Started, Ended    time.Time
}
type searchPerfTrace struct {
	mu      sync.Mutex
	begin   map[string]string
	records map[string]*searchPerfStatement
}

func (trace *searchPerfTrace) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	request, _ := ctx.Value(searchPerfRequestKey{}).(string)
	if request == "" {
		return ctx
	}
	trace.mu.Lock()
	defer trace.mu.Unlock()
	if trace.begin == nil {
		trace.begin, trace.records = map[string]string{}, map[string]*searchPerfStatement{}
	}
	if strings.HasPrefix(data.SQL, "BEGIN") {
		trace.begin[request] = data.SQL
	}
	if !strings.HasPrefix(data.SQL, "-- name: SearchIssues :many") {
		return ctx
	}
	r := &searchPerfStatement{SQL: data.SQL, Begin: trace.begin[request], PID: conn.PgConn().PID(), Started: time.Now()}
	for _, arg := range data.Args {
		r.Types = append(r.Types, fmt.Sprintf("%T", arg))
		switch v := arg.(type) {
		case nil, string, bool, int32, uuid.UUID:
			r.Args = append(r.Args, arg)
		case *uuid.UUID:
			r.Args = append(r.Args, searchPerfPointer(v))
		case *int32:
			r.Args = append(r.Args, searchPerfPointer(v))
		case []uuid.UUID:
			r.Args = append(r.Args, slices.Clone(v))
		case []bool:
			r.Args = append(r.Args, slices.Clone(v))
		case []string:
			r.Args = append(r.Args, slices.Clone(v))
		default:
			r.Error = fmt.Sprintf("unsupported actual argument %T", arg)
		}
	}
	if trace.records[request] != nil {
		r.Error = "multiple production Search statements in one sample"
	}
	trace.records[request] = r
	return context.WithValue(ctx, searchPerfStatementKey{}, r)
}

func (trace *searchPerfTrace) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	r, _ := ctx.Value(searchPerfStatementKey{}).(*searchPerfStatement)
	if r == nil {
		return
	}
	trace.mu.Lock()
	defer trace.mu.Unlock()
	r.Ended = time.Now()
	if data.Err != nil {
		r.Error += data.Err.Error()
	}
}

func searchPerfPointer[T any](p *T) *T {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func (trace *searchPerfTrace) take(t *testing.T, request string) searchPerfStatement {
	t.Helper()
	trace.mu.Lock()
	defer trace.mu.Unlock()
	r := trace.records[request]
	if r == nil || len(r.Args) != 17 || r.Begin != "BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY" || r.Ended.IsZero() {
		t.Fatalf("incomplete production capture %s: %+v", request, r)
	}
	delete(trace.records, request)
	delete(trace.begin, request)
	return *r
}

func searchPerfWrite(t *testing.T, dir, name string, value any) {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, name+".json"), append(data, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}

// This count is a correctness diagnostic, never a timed production call. Keep
// the original candidate predicates and all 17 bindings, including LIMIT's OID.
func searchPerfDiagnosticCount(t *testing.T, pool *pgxpool.Pool, r searchPerfStatement) int {
	t.Helper()
	prefix, _, ok := strings.Cut(r.SQL, "), paged AS MATERIALIZED (")
	if !ok {
		t.Fatal("unknown final production CTE boundary")
	}
	sql := prefix + ") SELECT count(*) FROM candidates WHERE $17::integer >= 0"
	var count int
	if err := pool.QueryRow(context.Background(), sql, r.Args...).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func searchPerfExplain(t *testing.T, pool *pgxpool.Pool, dir, name string, r searchPerfStatement) {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var user, isolation, readOnly, jit, mode string
	if err = tx.QueryRow(ctx, "SELECT current_user,current_setting('transaction_isolation'),current_setting('transaction_read_only'),current_setting('jit'),current_setting('plan_cache_mode')").Scan(&user, &isolation, &readOnly, &jit, &mode); err != nil {
		t.Fatal(err)
	}
	if isolation != "repeatable read" || readOnly != "on" || jit != "off" || mode != "force_custom_plan" {
		t.Fatal("EXPLAIN configuration", isolation, readOnly, jit, mode)
	}
	original, err := tx.Conn().Prepare(ctx, "", r.SQL)
	if err != nil {
		t.Fatal(err)
	}
	sql := "EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) " + r.SQL
	explained, err := tx.Conn().Prepare(ctx, "", sql)
	if err != nil || len(original.ParamOIDs) != 17 || !reflect.DeepEqual(original.ParamOIDs, explained.ParamOIDs) {
		t.Fatal("original/EXPLAIN parameter OIDs differ", original.ParamOIDs, err)
	}
	var raw json.RawMessage
	if err = tx.QueryRow(ctx, sql, r.Args...).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	searchPerfWrite(t, dir, name, map[string]any{"statement": r, "parameter_oids": original.ParamOIDs, "explain_parameter_oids": explained.ParamOIDs, "runtime_user": user, "pid": tx.Conn().PgConn().PID(), "isolation": isolation, "read_only": readOnly, "jit": jit, "plan_cache_mode": mode, "plan": raw})
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}

func searchPerfMetadata(t *testing.T, owner, pool *pgxpool.Pool) map[string]any {
	t.Helper()
	ctx := context.Background()
	values := map[string]any{"go": runtime.Version(), "os": runtime.GOOS, "arch": runtime.GOARCH, "cpus": runtime.NumCPU(), "gomaxprocs": runtime.GOMAXPROCS(0), "pool_max_conns": pool.Config().MaxConns, "runtime_params": pool.Config().ConnConfig.RuntimeParams, "execution_mode": pool.Config().ConnConfig.DefaultQueryExecMode.String(), "cache_boundary": "first-observed is not cold: seed/COPY and ANALYZE warm shared buffers and OS cache; no cache eviction; five subsequent calls are warm; candidate diagnostics and EXPLAIN follow each scenario's timings and can warm later scenarios", "measurement": "production TxManager(ReadOnlySnapshot) + Store.Search and actual pgx SQL; no HTTP/network/auth/directory latency"}
	var version, user, ownerUser, ctype, collate, encoding, locale, provider, jit, mode, seqscan, timezone string
	if err := pool.QueryRow(ctx, `SELECT version(),current_user,datctype,datcollate,current_setting('server_encoding'),coalesce(datlocale,''),datlocprovider::text,current_setting('jit'),current_setting('plan_cache_mode'),current_setting('enable_seqscan'),current_setting('TimeZone') FROM pg_database WHERE datname=current_database()`).Scan(&version, &user, &ctype, &collate, &encoding, &locale, &provider, &jit, &mode, &seqscan, &timezone); err != nil {
		t.Fatal(err)
	}
	var superuser, runtimeMember bool
	if err := pool.QueryRow(ctx, "SELECT rolsuper,pg_has_role(current_user,'nerve_runtime','MEMBER') FROM pg_roles WHERE rolname=current_user").Scan(&superuser, &runtimeMember); err != nil || superuser || !runtimeMember || jit != "off" || mode != "force_custom_plan" || seqscan != "on" {
		t.Fatal("runtime configuration", superuser, runtimeMember, jit, mode, seqscan, err)
	}
	if err := owner.QueryRow(ctx, "SELECT current_user").Scan(&ownerUser); err != nil {
		t.Fatal(err)
	}
	values["postgres"], values["runtime_role"], values["seed_analyze_role"] = version, user, ownerUser
	values["lc_ctype"], values["lc_collate"], values["encoding"], values["database_locale"], values["locale_provider"] = ctype, collate, encoding, locale, provider
	values["jit"], values["plan_cache_mode"], values["enable_seqscan"], values["timezone"] = jit, mode, seqscan, timezone
	rows, err := owner.Query(ctx, `SELECT row_to_json(s)::text FROM (SELECT relname,n_live_tup,last_analyze,analyze_count FROM pg_stat_user_tables WHERE relname IN ('issues','issue_links','issue_assignees','issue_labels','issue_relations') ORDER BY relname) s`)
	if err != nil {
		t.Fatal(err)
	}
	stats, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	values["analyze_stats"] = stats
	rows, err = pool.Query(ctx, "SELECT indexdef FROM pg_indexes WHERE schemaname='public' AND tablename IN ('issues','issue_links','issue_assignees','issue_labels','issue_relations') ORDER BY tablename,indexname")
	if err != nil {
		t.Fatal(err)
	}
	indexes, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	values["indexes"] = indexes
	var matrix []map[string]any
	for _, name := range []string{"lo", "login", "登录", "修复登录"} {
		for _, term := range []string{"lo", "login", "登录", "修复登录"} {
			var score float32
			if err := pool.QueryRow(ctx, "SELECT similarity($1::text,$2::text)", name, term).Scan(&score); err != nil {
				t.Fatal(err)
			}
			matrix = append(matrix, map[string]any{"name": name, "query": term, "native_float32": score})
		}
	}
	values["diagnostic_similarity_matrix_not_topN_oracle"] = matrix
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../../.."))
	for _, args := range [][]string{{"rev-parse", "HEAD"}, {"status", "--porcelain"}} {
		cmd := osexec.Command("git", args...)
		cmd.Dir = root
		out, err := cmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		values[strings.Join(args, " ")] = strings.TrimSpace(string(out))
	}
	hashes := map[string]string{}
	for _, path := range []string{"search_performance_test.go", "search_performance_fixture_test.go", "search_performance_evidence_test.go", "search_issues.go", "store.go", "store_test.go", "queries/search_issues.sql", "gen/search_issues.sql.go", "../../../../platform/postgres/pool.go", "../../../../platform/postgres/tx.go"} {
		path = filepath.Clean(filepath.Join(filepath.Dir(file), path))
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			t.Fatal(err)
		}
		hashes[rel] = fmt.Sprintf("%x", sha256.Sum256(data))
	}
	values["source_sha256"] = hashes
	return values
}
