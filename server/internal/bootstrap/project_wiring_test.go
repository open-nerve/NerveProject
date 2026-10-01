package bootstrap

import (
	"context"
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// createProject as bootstrap wires it (M3 design 3.6, 6.6): its rows carry
// the time of the request, from the clock project.New takes; and it writes
// in the one transaction of the TxManager project.New takes, so while every
// insert of a state fails, a create answers 500 and leaves no project,
// membership or display setting behind.
func TestCreateProjectRunsOnTheWiredClockAndTransaction(t *testing.T) {
	contract := apitest.Load(t)
	dbURL := pgtest.NewDatabase(t)
	base := startApp(t, testConfig(t, dbURL, false), migrations.FS())
	pool := openPool(t, dbURL)
	alice := registerAccount(t, contract, base, "alice@example.com").AccessToken
	if status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces", alice,
		`{"name":"Acme","slug":"acme"}`); status != http.StatusCreated {
		t.Fatalf("creating acme = %d %s", status, body)
	}
	before := time.Now().Truncate(time.Microsecond)
	status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces/acme/projects", alice, `{"name":"Web","identifier":"WEB"}`)
	after := time.Now()
	if status != http.StatusCreated {
		t.Fatalf("creating Web = %d %s", status, body)
	}
	var web struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, body, &web)
	var createdAt time.Time
	if err := pool.QueryRow(context.Background(), "SELECT created_at FROM projects WHERE id = $1", web.ID).Scan(&createdAt); err != nil {
		t.Fatal(err)
	}
	if createdAt.Before(before) || createdAt.After(after) {
		t.Errorf("Web was created at %v, want within the request, %v to %v", createdAt, before, after)
	}

	rows := func() string {
		t.Helper()
		var n string
		if err := pool.QueryRow(context.Background(), `SELECT (SELECT count(*) FROM projects) || ' ' || (SELECT count(*) FROM project_members)
			|| ' ' || (SELECT count(*) FROM project_user_properties) || ' ' || (SELECT count(*) FROM states)`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	want := rows()
	if _, err := pool.Exec(context.Background(), "ALTER TABLE states ADD CONSTRAINT no_states CHECK (false) NOT VALID"); err != nil {
		t.Fatal(err)
	}
	status, body = call(t, contract, http.MethodPost, base+"/api/v0/workspaces/acme/projects", alice, `{"name":"Ops","identifier":"OPS"}`)
	if got := rows(); status != http.StatusInternalServerError || got != want {
		t.Errorf("creating Ops while the states fail = %d %s, rows %s; want 500 and the rows as they were, %s", status, body, got, want)
	}
}
