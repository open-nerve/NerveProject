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

func TestNewPoolRejectsMalformedURLWithoutLeakingPassword(t *testing.T) {
	_, err := postgres.NewPool(context.Background(), config.DatabaseConfig{
		URL:      "postgres://nerve:secret@localhost:notaport/nerve",
		MaxConns: 1,
	})
	if err == nil {
		t.Fatal("NewPool() error = nil, want a parse error")
	}
	if !strings.HasPrefix(err.Error(), "database.url: ") || strings.Contains(err.Error(), "secret") {
		t.Errorf("NewPool() error = %q, want a database.url error without the password", err)
	}
}
