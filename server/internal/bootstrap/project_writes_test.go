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

// Each write on a project as bootstrap wires it stamps the rows it writes
// with the time of its request, from the clock project.New takes, and with
// its caller (M3 design 3.6): alice, acme's admin, writes on her project
// Web, one write after another; the statement of each reads the rows it
// wrote, by $1 Web's id and $2 alice's.
func TestTheWritesOnAProjectStampTheirRequest(t *testing.T) {
	contract := apitest.Load(t)
	dbURL := pgtest.NewDatabase(t)
	base := startApp(t, testConfig(t, dbURL, false), migrations.FS())
	pool := openPool(t, dbURL)
	alice := registerAccount(t, contract, base, "alice@example.com").AccessToken
	aliceID := accountID(t, contract, base, alice)
	if status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces", alice, `{"name":"Acme","slug":"acme"}`); status != http.StatusCreated {
		t.Fatalf("creating acme = %d %s", status, body)
	}
	status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces/acme/projects", alice, `{"name":"Web","identifier":"WEB"}`)
	if status != http.StatusCreated {
		t.Fatalf("creating Web = %d %s", status, body)
	}
	var web struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, body, &web)
	for _, w := range []struct {
		name, method, path, body string
		status                   int
		stamps                   string // the rows written: their updated_at, and whether alice wrote them
	}{
		{"updateProject", http.MethodPatch, "/api/v0/projects/" + web.ID.String(), `{"name":"Site"}`, http.StatusOK,
			"SELECT updated_at, updated_by_id = $2 FROM projects WHERE id = $1"},
	} {
		before := time.Now().Truncate(time.Microsecond)
		status, body := call(t, contract, w.method, base+w.path, alice, w.body)
		after := time.Now()
		if status != w.status {
			t.Fatalf("%s = %d %s, want %d", w.name, status, body, w.status)
		}
		rows, err := pool.Query(context.Background(), w.stamps, web.ID, aliceID)
		if err != nil {
			t.Fatal(err)
		}
		n := 0
		for rows.Next() {
			var at time.Time
			var hers bool
			if err := rows.Scan(&at, &hers); err != nil {
				t.Fatal(err)
			}
			if n++; at.Before(before) || at.After(after) || !hers {
				t.Errorf("%s wrote a row at %v, by alice %v; want within the request, %v to %v, by alice", w.name, at, hers, before, after)
			}
		}
		if err := rows.Err(); err != nil || n == 0 {
			t.Errorf("%s: %d rows written, %v; want one at least", w.name, n, err)
		}
	}
}
