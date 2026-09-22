package postgres_test

import (
	"context"
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

func TestNewPoolAppliesMaxConns(t *testing.T) {
	pool, err := postgres.NewPool(context.Background(), config.DatabaseConfig{
		URL:      "postgres://nerve:secret@127.0.0.1:1/nerve",
		MaxConns: 3,
	})
	if err != nil {
		t.Fatalf("NewPool() error = %v", err)
	}
	defer pool.Close()
	if got := pool.Config().MaxConns; got != 3 {
		t.Errorf("MaxConns = %d, want 3", got)
	}
}

// pgx quotes the connection string in its parse error and masks the password
// only on a best-effort basis, e.g. not in the legal key/value form
// "password = secret"; NewPool shows none of that text, whether the string is
// malformed or names a file that cannot be read.
func TestNewPoolRejectsUnusableURLWithoutLeakingPassword(t *testing.T) {
	const want = "database.url: pgx cannot use it; check its syntax, the files it names (sslrootcert, sslcert, sslkey) " +
		"and any PG* environment variables (details not shown, as they may contain the password)"
	for _, url := range []string{
		"postgres://nerve:secret@localhost:notaport/nerve",
		"host=localhost port=1 password = secret sslmode=bogus",
		`host=localhost password=sec\ secret sslmode=bogus`,
		"postgres://nerve:secret@localhost/nerve?sslmode=verify-full&sslrootcert=/nonexistent/ca.pem",
	} {
		_, err := postgres.NewPool(context.Background(), config.DatabaseConfig{URL: url, MaxConns: 1})
		if err == nil || err.Error() != want || strings.Contains(err.Error(), "secret") {
			t.Errorf("NewPool(%q) error = %v, want %q", url, err, want)
		}
	}
}
