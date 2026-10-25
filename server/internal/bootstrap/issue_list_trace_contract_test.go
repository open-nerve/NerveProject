package bootstrap

import (
	"context"
	"math"
	"net/http"
	"reflect"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// A shallow copy, float conversion or loss of typed nil would corrupt replay.
func TestIssueListTraceArguments(t *testing.T) {
	id := uuid.NewV7()
	n, s := int64(math.MaxInt64), "cursor"
	input := []any{nil, id, &id, &n, &s, int32(51), true, "text", []uuid.UUID{id}, []bool{true}, []string{"A"},
		(*uuid.UUID)(nil), (*int64)(nil), (*string)(nil), []uuid.UUID(nil), []bool(nil), []string(nil), []uuid.UUID{}, []bool{}, []string{}}
	copy, err := copyListArgs(input)
	if err != nil || !reflect.DeepEqual(input, copy) {
		t.Fatalf("typed replay %v: %#v", err, copy)
	}
	id, n, s = uuid.NewV7(), 0, "changed"
	input[8].([]uuid.UUID)[0] = id
	input[9].([]bool)[0] = false
	input[10].([]string)[0] = s
	if *copy[2].(*uuid.UUID) == id || *copy[3].(*int64) != math.MaxInt64 || *copy[4].(*string) != "cursor" || copy[8].([]uuid.UUID)[0] == id || !copy[9].([]bool)[0] || copy[10].([]string)[0] != "A" {
		t.Fatal("captured pointers/arrays alias the caller")
	}
	if _, err := copyListArgs([]any{struct{ Value int }{1}}); err == nil {
		t.Fatal("unknown parameter accepted")
	}
}

// TraceEnd must follow consumption, and outer request tags must reach every SQL.
func TestIssueListTraceBoundary(t *testing.T) {
	roles := newSplitRoles(t)
	trace := &listTrace{}
	pool := listTracePool(t, roles.serverURL, trace)
	w := newIssueWorldOn(t, roles.serverURL, pool, func(*config.Config) {})
	w.newWorkItem(t)
	ctx := context.WithValue(context.Background(), listRequestKey{}, "rows-open")
	rows, err := pool.Query(ctx, "SELECT $1::bigint", int64(9007199254740993))
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	if records := trace.take("rows-open"); len(records) != 1 || !records[0].Ended.IsZero() {
		t.Fatal("query ended before rows closed", records)
	}
	rows.Close()
	if records := trace.take("rows-open"); len(records) != 1 || records[0].Ended.IsZero() {
		t.Fatal("missing rows close", records)
	}
	route := newIssueRoute(t, pool, postgres.NewTxManager(pool, 0))
	ctx = context.WithValue(context.Background(), listRequestKey{}, "member")
	_, rec := route.sendContext(ctx, http.MethodGet, memberIssuesPath(w, "alice")+"?relation=created", w.ids["bob"], "")
	if rec.Code != 200 {
		t.Fatal(rec.Code, rec.Body)
	}
	p := checkListEnvelope(t, rec.Body.String(), false, false)
	if p.Total != 1 {
		t.Fatal(p)
	}
	records := trace.take("member")
	if len(records) < 8 {
		t.Fatal("missing route queries", records)
	}
	pid := records[0].PID
	for _, r := range records {
		if r.PID != pid || r.Ended.IsZero() || r.Error != "" {
			t.Fatal("incomplete request trace", r)
		}
	}
}
