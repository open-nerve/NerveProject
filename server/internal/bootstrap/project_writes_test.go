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
	web := createdProject(t, contract, base, alice, "acme", "Web", "WEB")
	for _, w := range []struct {
		name, method, path, body string
		status                   int
		stamps                   string // the rows written: the time each took, and whether alice wrote it as the write does
	}{
		{"updateProject", http.MethodPatch, "/api/v0/projects/" + web.String(), `{"name":"Site"}`, http.StatusOK,
			"SELECT updated_at, updated_by_id = $2 FROM projects WHERE id = $1"},
		{"archiveProject", http.MethodPost, "/api/v0/projects/" + web.String() + "/archive", "", http.StatusOK,
			"SELECT archived_at, updated_by_id = $2 AND updated_at = archived_at FROM projects WHERE id = $1"},
		// Archived again, it takes the new time.
		{"archiveProject again", http.MethodPost, "/api/v0/projects/" + web.String() + "/archive", "", http.StatusOK,
			"SELECT archived_at, updated_by_id = $2 AND updated_at = archived_at FROM projects WHERE id = $1"},
		{"unarchiveProject", http.MethodPost, "/api/v0/projects/" + web.String() + "/unarchive", "", http.StatusOK,
			"SELECT updated_at, updated_by_id = $2 AND archived_at IS NULL FROM projects WHERE id = $1"},
		{"updateProjectPreferences", http.MethodPatch, "/api/v0/me/projects/" + web.String() + "/preferences", `{"sort_order":5}`, http.StatusOK,
			"SELECT updated_at, updated_by_id = $2 FROM project_user_properties WHERE project_id = $1 AND user_id = $2 AND deleted_at IS NULL"},
	} {
		before := time.Now().Truncate(time.Microsecond)
		status, body := call(t, contract, w.method, base+w.path, alice, w.body)
		after := time.Now()
		if status != w.status {
			t.Fatalf("%s = %d %s, want %d", w.name, status, body, w.status)
		}
		rows, err := pool.Query(context.Background(), w.stamps, web, aliceID)
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

// createdProject creates the project name with identifier in the workspace
// slug through the API, by the caller of token, and returns its id.
func createdProject(t *testing.T, contract *apitest.Contract, base, token, slug, name, identifier string) uuid.UUID {
	t.Helper()
	status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces/"+slug+"/projects", token,
		`{"name":"`+name+`","identifier":"`+identifier+`"}`)
	if status != http.StatusCreated {
		t.Fatalf("creating %s in %s = %d %s", name, slug, status, body)
	}
	var p struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, body, &p)
	return p.ID
}
