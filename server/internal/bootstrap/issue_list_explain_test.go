package bootstrap

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func listEvidence(t *testing.T, dir, name string, v any) {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, name+".json"), data, 0644); err != nil {
		t.Fatal(err)
	}
}
func listStatementName(sql string) string {
	if strings.HasPrefix(sql, "-- name: ") {
		return strings.Fields(sql)[2]
	}
	return strings.TrimSpace(sql)
}
func listStatementClass(name string) string {
	switch name {
	case "FlatIssues", "GroupIssues", "SubgroupIssues":
		return "list"
	case "IssueListStates", "IssueListLabels", "ListAssigneeIDs", "PublicProfiles":
		return "directories"
	default:
		return "other"
	}
}
func explainListStatements(t *testing.T, pool *pgxpool.Pool, dir, request string, records []listStatement) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var user, isolation, readOnly, jit, plan string
	if err = tx.QueryRow(ctx, "SELECT current_user,current_setting('transaction_isolation'),current_setting('transaction_read_only'),current_setting('jit'),current_setting('plan_cache_mode')").Scan(&user, &isolation, &readOnly, &jit, &plan); err != nil {
		t.Fatal(err)
	}
	if isolation != "repeatable read" || readOnly != "on" || jit != "off" || plan != "force_custom_plan" {
		t.Fatal("EXPLAIN configuration", isolation, readOnly, jit, plan)
	}
	for i, r := range records {
		if !strings.HasPrefix(r.SQL, "-- name: ") {
			continue
		}
		original, err := tx.Conn().Prepare(ctx, "", r.SQL)
		if err != nil {
			t.Fatal(err)
		}
		sql := "EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) " + r.SQL
		explained, err := tx.Conn().Prepare(ctx, "", sql)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(original.ParamOIDs, explained.ParamOIDs) {
			t.Fatal("EXPLAIN changed parameter OIDs")
		}
		var raw json.RawMessage
		if err = tx.QueryRow(ctx, sql, r.Args...).Scan(&raw); err != nil {
			t.Fatal(err)
		}
		listEvidence(t, dir, request+"-"+strconvList(i)+"-"+listStatementName(r.SQL), struct {
			SQL     string
			Args    []any
			Types   []string
			OIDs    []uint32
			Runtime string
			Plan    json.RawMessage
		}{r.SQL, r.Args, r.Types, original.ParamOIDs, user, raw})
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}
func listScaleMetadata(t *testing.T, roles splitRoles, pool *pgxpool.Pool) map[string]any {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	values := map[string]any{"go": runtime.Version(), "os": runtime.GOOS, "arch": runtime.GOARCH, "num_cpu": runtime.NumCPU(), "gomaxprocs": runtime.GOMAXPROCS(0), "seed_version": "list-scale-v1", "pool_max_conns": pool.Config().MaxConns, "execution_mode": pool.Config().ConnConfig.DefaultQueryExecMode.String(), "runtime_params": pool.Config().ConnConfig.RuntimeParams, "measurement": "httptest module route + API middleware + tokenIsAccount; no network or real token authentication"}
	var version, user, owner, jit, plan, timezone string
	var utc time.Time
	if err := pool.QueryRow(ctx, "SELECT version(),current_user,current_setting('jit'),current_setting('plan_cache_mode'),current_setting('TimeZone'),now()").Scan(&version, &user, &jit, &plan, &timezone, &utc); err != nil {
		t.Fatal(err)
	}
	if user != roles.serverName || jit != "off" || plan != "force_custom_plan" || utc.Location() != time.UTC {
		t.Fatal("production pool configuration", user, jit, plan, utc)
	}
	if err := roles.owner.QueryRow(ctx, "SELECT current_user").Scan(&owner); err != nil {
		t.Fatal(err)
	}
	values["postgres"], values["runtime"], values["owner"], values["timezone"], values["utc_scan"] = version, user, owner, timezone, utc
	rows, err := pool.Query(ctx, "SELECT indexdef FROM pg_indexes WHERE schemaname='public' ORDER BY tablename,indexname")
	if err != nil {
		t.Fatal(err)
	}
	indexes, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	values["indexes"] = indexes
	return values
}
