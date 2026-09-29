package pgtest_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

func connect(t *testing.T, url string) *pgx.Conn {
	t.Helper()
	conn, err := pgx.Connect(context.Background(), url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	return conn
}

func TestDatabasesAreIsolated(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	a := connect(t, pgtest.NewDatabase(t))
	b := connect(t, pgtest.NewDatabase(t))

	if _, err := a.Exec(ctx, "CREATE TABLE only_in_a (id int)"); err != nil {
		t.Fatal(err)
	}
	var found bool
	if err := b.QueryRow(ctx, "SELECT to_regclass('only_in_a') IS NOT NULL").Scan(&found); err != nil {
		t.Fatal(err)
	}
	if found {
		t.Error("a table created in one test database is visible in another")
	}
}

// A copy holds what the prepared database held, and each copy is its own:
// a write to one reaches neither the other copy nor the prepared database.
func TestNewDatabaseFromCopiesThePreparedDatabase(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prepared := pgtest.NewDatabase(t)
	conn, err := pgx.Connect(ctx, prepared)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	_, err = conn.Exec(ctx, "CREATE TABLE marks (name text); INSERT INTO marks VALUES ('prepared')")
	_ = conn.Close(ctx) // nobody may be connected to the database copied
	if err != nil {
		t.Fatal(err)
	}
	a := connect(t, pgtest.NewDatabaseFrom(t, prepared))
	b := connect(t, pgtest.NewDatabaseFrom(t, prepared))
	if _, err := a.Exec(ctx, "INSERT INTO marks VALUES ('a')"); err != nil {
		t.Fatal(err)
	}

	marks := func(conn *pgx.Conn) string {
		var names string
		if err := conn.QueryRow(ctx, "SELECT string_agg(name, ',' ORDER BY name) FROM marks").Scan(&names); err != nil {
			t.Fatal(err)
		}
		return names
	}
	if got := [3]string{marks(a), marks(b), marks(connect(t, prepared))}; got != [3]string{"a,prepared", "prepared", "prepared"} {
		t.Errorf("marks in copy a, copy b, the prepared database = %q; want the write in a alone", got)
	}
}

func TestEmptyDatabaseHasNoTables(t *testing.T) {
	t.Parallel()
	conn := connect(t, pgtest.NewEmptyDatabase(t))

	var tables int
	err := conn.QueryRow(context.Background(),
		"SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public'").Scan(&tables)
	if err != nil {
		t.Fatal(err)
	}
	if tables != 0 {
		t.Errorf("empty database has %d tables, want 0", tables)
	}
}

func TestDatabaseIsDroppedAfterTheTest(t *testing.T) {
	ctx := context.Background()
	var url string
	var open *pgx.Conn
	t.Run("uses a database", func(t *testing.T) {
		url = pgtest.NewDatabase(t)
		// No cleanup closes this connection, so it is still open when
		// NewDatabase's cleanup drops the database: the drop must force it.
		conn, err := pgx.Connect(ctx, url)
		if err != nil {
			t.Fatalf("connect: %v", err)
		}
		open = conn
	})
	if url == "" {
		t.Skip("the subtest was skipped")
	}
	if open == nil {
		t.FailNow() // the subtest could not connect and has reported why
	}

	_, err := pgx.Connect(ctx, url)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "3D000" { // invalid_catalog_name
		t.Errorf("connect after the test = %v, want database does not exist (3D000)", err)
	}
	_ = open.Close(ctx) // only now, after the drop has been asserted
}
