package postgresadapter_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// Each request follows the public typed cursor, takes one row and checks the
// lookahead plus the original total. The expected IDs come from fixed fixtures.
func checkFlatPages(t *testing.T, s *postgresadapter.Store, in app.ListRead, want []uuid.UUID) {
	t.Helper()
	in.Query.Limit = 1
	var seen []uuid.UUID
	for n := 0; n <= len(want); n++ {
		rows, err := s.Flat(context.Background(), in)
		if err != nil {
			t.Fatal(err)
		}
		if n == len(want) {
			if len(rows) != 0 {
				t.Fatalf("past end returned %v", rows)
			}
			break
		}
		if len(rows) != min(2, len(want)-n) {
			t.Fatalf("page %d: len %d; want %d", n, len(rows), min(2, len(want)-n))
		}
		if rows[0].Total != int64(len(want)) {
			t.Fatalf("page %d: total %d; want %d", n, rows[0].Total, len(want))
		}
		if rows[0].Issue.ID != want[n] {
			t.Fatalf("page %d ID %s; want %s", n, rows[0].Issue.ID, want[n])
		}
		seen = append(seen, rows[0].Issue.ID)
		encoded, err := domain.EncodeListCursor(rows[0].Position)
		if err != nil {
			t.Fatal(err)
		}
		cursor, err := domain.DecodeListCursor(encoded, in.Query.Order)
		if err != nil {
			t.Fatal(err)
		}
		in.Query.Cursor = &cursor
	}
	if !slices.Equal(seen, want) {
		t.Fatalf("pagination %v; want %v", seen, want)
	}
}

func finiteValues() []float64 {
	normal := math.Float64frombits(0x0010000000000000)
	return []float64{-math.MaxFloat64, math.Nextafter(-math.MaxFloat64, 0), -9007199254740994, -9007199254740992, -9007199254740991,
		math.Nextafter(-1, math.Inf(-1)), -1, math.Nextafter(-1, 0), -math.Nextafter(normal, math.Inf(1)), -normal, -math.Nextafter(normal, 0),
		-math.SmallestNonzeroFloat64, math.Copysign(0, -1), 0, math.SmallestNonzeroFloat64, math.Nextafter(normal, 0), normal, math.Nextafter(normal, math.Inf(1)),
		math.Nextafter(1, 0), 1, math.Nextafter(1, math.Inf(1)), 9007199254740991, 9007199254740992, 9007199254740994, math.Nextafter(math.MaxFloat64, 0), math.MaxFloat64}
}

func TestFlatIssueListFinitePrecision(t *testing.T) {
	_, pool := newStore(t)
	actor := newAccount(t, pool, "precision")
	wid := newWorkspace(t, pool, "precision")
	p := newProject(t, pool, wid, "FLOAT")
	values := finiteValues()
	want := make([]uuid.UUID, len(values))
	for n, value := range values {
		// Reverse IDs oppose numeric order. Zeros deliberately use their UUID order.
		id := uuid.MustParse(fmt.Sprintf("00000000-0000-0000-0000-%012x", 1000-n))
		want[n] = id
		newIssue(t, pool, p, id, n+1, actor)
		exec(t, pool, "UPDATE issues SET sort_order=$2 WHERE id=$1", id, value)
	}
	want[12], want[13] = want[13], want[12]
	in := app.ListRead{ActorID: actor, Directories: app.ListDirectories{Projects: []app.VisibleProject{{ID: p.id}}, States: []app.ListState{{ID: p.state, Group: "backlog", Key: "backlog"}}}, Query: domain.ListQuery{Order: "sort_order", Limit: 100}}
	for _, digits := range []string{"-15", "0", "3"} {
		t.Run("extra_float_digits="+digits, func(t *testing.T) {
			pc := pool.Config()
			pc.ConnConfig.RuntimeParams["extra_float_digits"] = digits
			pc.MaxConns = 1
			other, err := pgxpool.NewWithConfig(context.Background(), pc)
			if err != nil {
				t.Fatal(err)
			}
			defer other.Close()
			var actual string
			if err := other.QueryRow(context.Background(), "SHOW extra_float_digits").Scan(&actual); err != nil || actual != digits {
				t.Fatalf("startup parameter %q, %v", actual, err)
			}
			store := postgresadapter.New(other)
			rows, err := store.Flat(context.Background(), in)
			if err != nil {
				t.Fatal(err)
			}
			checkFlatIDs(t, rows, want)
			for _, r := range rows {
				var value float64
				if err := json.Unmarshal(r.Position.Key, &value); err != nil || value != r.Issue.SortOrder {
					t.Fatalf("public float %s / %g, %v", r.Position.Key, r.Issue.SortOrder, err)
				}
				if value == 0 && math.Signbit(value) {
					t.Fatal("noncanonical negative zero")
				}
			}
			checkFlatPages(t, store, in, want)
		})
	}
	// Native PostgreSQL float order is independent of the production bit key.
	rows, err := pool.Query(context.Background(), "SELECT id FROM issues WHERE project_id=$1 ORDER BY sort_order,id", p.id)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var natural []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		natural = append(natural, id)
	}
	if rows.Err() != nil || !slices.Equal(natural, want) {
		t.Fatalf("natural float order %v: %v", natural, rows.Err())
	}
}

func TestFlatIssueListWideDates(t *testing.T) {
	for _, field := range []string{"created_at", "updated_at", "start_date", "target_date"} {
		t.Run(field, func(t *testing.T) {
			s, pool := newStore(t)
			actor := newAccount(t, pool, "wide")
			wid := newWorkspace(t, pool, "wide")
			p := newProject(t, pool, wid, "WIDE")
			// Adjacent microseconds around 2^53, a negative epoch and both supported
			// four-digit year bounds expose UnixNano/duration and float rounding.
			times := []time.Time{time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(1600, 1, 1, 0, 0, 0, 1000, time.UTC), time.UnixMicro(-1).UTC(), time.UnixMicro(0).UTC(), time.UnixMicro(9007199254740991).UTC(), time.UnixMicro(9007199254740992).UTC(), time.UnixMicro(9007199254740993).UTC(), time.Date(9999, 12, 31, 23, 59, 59, 999999000, time.UTC)}
			isDate := strings.HasSuffix(field, "_date")
			if isDate {
				times = []time.Time{times[0], times[1], time.Date(1969, 12, 31, 0, 0, 0, 0, time.UTC), time.Unix(0, 0).UTC(), time.Date(9999, 12, 30, 0, 0, 0, 0, time.UTC), time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)}
			}
			want := make([]uuid.UUID, len(times))
			for n, at := range times {
				id := uuid.MustParse(fmt.Sprintf("00000000-0000-0000-0000-%012x", 100-n))
				want[n] = id
				newIssue(t, pool, p, id, n+1, actor)
				exec(t, pool, "UPDATE issues SET "+field+"=$2 WHERE id=$1", id, at)
			}
			for _, direction := range []string{"", "-"} {
				ids := slices.Clone(want)
				if direction == "-" {
					slices.Reverse(ids)
				}
				in := app.ListRead{ActorID: actor, Directories: app.ListDirectories{Projects: []app.VisibleProject{{ID: p.id}}, States: []app.ListState{{ID: p.state, Group: "backlog", Key: "b"}}}, Query: domain.ListQuery{Order: domain.ListOrder(direction + field), Limit: 1}}
				checkFlatPages(t, s, in, ids)
			}
		})
	}
}

// Read the exact numeric CASE from the production SQL, rather than a second
// handwritten approximation. Only its source row is a synthetic one: enormous
// child counts cannot be seeded. This probe does not claim a 2^53-child page.
func numericExpression(t *testing.T, file string) string {
	t.Helper()
	data, err := os.ReadFile("queries/" + file)
	if err != nil {
		t.Fatal(err)
	}
	_, tail, ok := strings.Cut(string(data), "), numeric_keys AS (")
	if !ok {
		t.Fatal("numeric CTE missing")
	}
	start := strings.Index(tail, "CASE sqlc.arg(order_key)::text")
	end := strings.Index(tail, "END AS k_num")
	if start < 0 || end < start {
		t.Fatal("numeric CASE missing")
	}
	return strings.ReplaceAll(tail[start:end+3], "sqlc.arg(order_key)::text", "$1::text")
}

func TestIssueListNumericExpressions(t *testing.T) {
	for _, file := range []string{"list_issues.sql", "list_issue_groups.sql", "list_issue_subgroups.sql"} {
		t.Run(file, func(t *testing.T) { checkListNumericExpression(t, file) })
	}
}

func checkListNumericExpression(t *testing.T, file string) {
	t.Helper()
	_, pool := newStore(t)
	expression := numericExpression(t, file)
	query := "SELECT pg_typeof(k)::text,k::text,k=$6::numeric FROM (SELECT " + expression + " AS k FROM (SELECT pg_catalog.float8send($2::float8) AS bits,$3::timestamptz AS created_at,$3::timestamptz AS updated_at,$4::bigint AS child_count,$5::date AS start_date,$5::date AS target_date,'urgent'::text AS priority) s) probe"
	probe := func(order string, value float64, at time.Time, count int64, date, want string) {
		t.Helper()
		var kind, actual string
		var equal bool
		err := pool.QueryRow(context.Background(), query, order, value, at, count, date, want).Scan(&kind, &actual, &equal)
		if err != nil || kind != "numeric" || !equal {
			t.Fatalf("%s key=%s type=%s want=%s: %v", order, actual, kind, want, err)
		}
	}
	for _, value := range finiteValues() {
		magnitude := int64(math.Float64bits(value) & 0x7fffffffffffffff)
		if value < 0 {
			magnitude = -magnitude
		}
		probe("sort_order", value, now, 0, "1970-01-01", strconv.FormatInt(magnitude, 10))
	}
	for _, count := range []int64{0, 1, 9007199254740991, 9007199254740992, 9007199254740993, math.MaxInt64} {
		probe("sub_issues_count", 0, now, count, "1970-01-01", strconv.FormatInt(count, 10))
	}
	for _, micros := range []int64{-1, 0, 9007199254740991, 9007199254740992, 9007199254740993} {
		for _, order := range []string{"created_at", "updated_at"} {
			probe(order, 0, time.UnixMicro(micros), 0, "1970-01-01", strconv.FormatInt(micros, 10))
		}
	}
	for _, date := range []struct{ value, key string }{{"0001-01-01", "-719162"}, {"1969-12-31", "-1"}, {"9999-12-31", "2932896"}} {
		for _, order := range []string{"start_date", "target_date"} {
			probe(order, 0, now, 0, date.value, date.key)
		}
	}
}
