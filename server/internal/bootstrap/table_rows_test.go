package bootstrap

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	workspacedomain "github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// tableRows is every table of the database, as the catalog lists them
// (pg_tables, the system's schemas left out), by its schema-qualified name,
// with its rows as row_to_json writes them, one a line, in the order of
// that text. No list is kept here: a table a later phase adds, or one no
// module owns, is in it. skip, when set, leaves out the tables it reports.
func tableRows(t *testing.T, pool *pgxpool.Pool, skip func(table string) bool) map[string]string {
	t.Helper()
	rows, err := pool.Query(context.Background(), `SELECT format('%I.%I', schemaname, tablename) FROM pg_tables
		WHERE schemaname NOT IN ('pg_catalog', 'information_schema') ORDER BY 1`)
	if err != nil {
		t.Fatal(err)
	}
	tables, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	all := map[string]string{}
	for _, table := range tables {
		if skip != nil && skip(table) {
			continue
		}
		var text string
		if err := pool.QueryRow(context.Background(), `SELECT coalesce(string_agg(t, E'\n' ORDER BY t), '')
			FROM (SELECT row_to_json(r)::text AS t FROM `+table+` r) s`).Scan(&text); err != nil {
			t.Fatalf("%s: %v", table, err)
		}
		all[table] = text
	}
	if len(all) == 0 {
		t.Fatal("the catalog lists no table: a check over it would see nothing")
	}
	return all
}

// riversOwn reports whether table is one of River's own (river_job,
// river_leader, …): the running apps' job clients write them on their own
// schedule (the leader's election, the periodic session cleanup), whatever
// a request does, so no request's writes can be told from them there.
func riversOwn(table string) bool {
	return strings.HasPrefix(table, "public.river_")
}

// The two checks over tableRows see a table no list names. In a table the
// test creates, a link's token written in any form expectNoTokenStored
// looks for is found (tokensIn), and a row written changes registrationRows.
// A token in one of River's tables is found too: registrationRows leaves
// them out, the token's check does not.
func TestTheRowChecksSeeEveryTable(t *testing.T) {
	pool := openPool(t, pgtest.NewDatabase(t))
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec("CREATE TABLE scratch (note text, tag bytea)")
	id := uuid.NewV7()
	link := invitationLink{id, invitationToken(t, id)}
	tag, _ := workspacedomain.ParseToken(link.token)
	before := registrationRows(t, pool)
	if found := tokensIn(t, tableRows(t, pool, nil), []invitationLink{link}); len(found) != 0 {
		t.Fatalf("before the token is written: %q, want nothing", found)
	}
	for _, tt := range []struct {
		name, table, sql string
		arg              any
	}{
		{"the token", "public.scratch", "INSERT INTO scratch (note) VALUES ($1)", link.token},
		{"its 22 characters", "public.scratch", "INSERT INTO scratch (note) VALUES ($1)", strings.TrimPrefix(link.token, "nrv_inv_")},
		{"its tag, a bytea", "public.scratch", "INSERT INTO scratch (tag) VALUES ($1)", tag[:]},
		{"its tag in standard base64", "public.scratch", "INSERT INTO scratch (note) VALUES ($1)", base64.RawStdEncoding.EncodeToString(tag[:])},
		{"its tag in hex", "public.scratch", "INSERT INTO scratch (note) VALUES ($1)", hex.EncodeToString(tag[:])},
		{"the token as a queue of River's", "public.river_queue", "INSERT INTO river_queue (name, updated_at) VALUES ($1, now())", link.token},
	} {
		exec("TRUNCATE scratch, river_queue")
		exec(tt.sql, tt.arg)
		found := tokensIn(t, tableRows(t, pool, nil), []invitationLink{link})
		if len(found) == 0 || !strings.HasPrefix(found[0], tt.table+" holds the token of "+id.String()) {
			t.Errorf("%s written into %s: %q, want it found there", tt.name, tt.table, found)
		}
	}
	exec("TRUNCATE river_queue")
	if registrationRows(t, pool) != before {
		t.Fatal("with scratch and River's queue empty again, registrationRows is not as it was")
	}
	exec("INSERT INTO scratch (note) VALUES ('a row')")
	if registrationRows(t, pool) == before {
		t.Error("a row written into scratch leaves registrationRows as it was")
	}
}
