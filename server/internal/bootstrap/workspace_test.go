package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// The workspace module as bootstrap wires it (M3 design 6.6), with
// workspace.creation_enabled both ways (3.11). On: the caller creates a
// workspace, through identity's account lock, and is its admin and only
// member; his list holds it; he reads it, through the Authorizer, as its
// admin (20). Another account, the admin of a workspace of his own, reads
// it as not found: the role read is the caller's, in this workspace. Off:
// the API answers workspace.creation_disabled and nothing is created.
func TestCreatingAWorkspaceAsConfigured(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		t.Run(fmt.Sprintf("creation_enabled %v", enabled), func(t *testing.T) {
			contract := apitest.Load(t)
			cfg := testConfig(t, pgtest.NewDatabase(t), false)
			cfg.Workspace.CreationEnabled = enabled
			base := startApp(t, cfg, migrations.FS())
			alice := registerAccount(t, contract, base, "alice@example.com").AccessToken
			bob := registerAccount(t, contract, base, "bob@example.com").AccessToken
			if enabled {
				if status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces", bob, `{"name":"Bob's","slug":"bobs"}`); status != http.StatusCreated {
					t.Fatalf("bob's own workspace = %d %s, want 201", status, body)
				}
			}

			created, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces", alice, `{"name":"Acme","slug":"acme"}`)
			_, list := call(t, contract, http.MethodGet, base+"/api/v0/workspaces", alice, "")
			aliceReads, aliceBody := call(t, contract, http.MethodGet, base+"/api/v0/workspaces/acme", alice, "")
			bobReads, bobBody := call(t, contract, http.MethodGet, base+"/api/v0/workspaces/acme", bob, "")

			var w, read struct {
				Slug         string `json:"slug"`
				Role         int    `json:"role"`
				TotalMembers int    `json:"total_members"`
			}
			var l struct {
				Data []struct {
					Slug string `json:"slug"`
				} `json:"data"`
			}
			if err := json.Unmarshal([]byte(list), &l); err != nil {
				t.Fatalf("decode the list %s: %v", list, err)
			}
			if !enabled {
				if created != http.StatusForbidden || problemCode(t, []byte(body)) != "workspace.creation_disabled" ||
					len(l.Data) != 0 || aliceReads != http.StatusNotFound {
					t.Errorf("create = %d %s, list %s, read %d; want 403 workspace.creation_disabled, nothing created", created, body, list, aliceReads)
				}
				return
			}
			if err := json.Unmarshal([]byte(body), &w); err != nil || created != http.StatusCreated || w.Slug != "acme" || w.Role != 20 || w.TotalMembers != 1 {
				t.Errorf("create = %d %s, want 201 with the caller its admin and only member", created, body)
			}
			if len(l.Data) != 1 || l.Data[0].Slug != "acme" {
				t.Errorf("alice's list %s, want acme alone", list)
			}
			if err := json.Unmarshal([]byte(aliceBody), &read); err != nil || aliceReads != http.StatusOK || read.Slug != "acme" || read.Role != 20 {
				t.Errorf("alice's read = %d %s, want 200 with acme and her role 20", aliceReads, aliceBody)
			}
			if bobReads != http.StatusNotFound || problemCode(t, []byte(bobBody)) != "workspace.not_found" {
				t.Errorf("bob's read = %d %s, want 404 workspace.not_found", bobReads, bobBody)
			}
		})
	}
}

// The composed createWorkspace reads the caller's account through
// identity's Accounts, under its lock (M3 design 3.6 convention 6, 6.6).
// The request has passed authentication when it waits on the account row
// that a deactivation holds; once that commits, it reads the account
// deactivated and answers the use case's 401, with a plain Bearer challenge
// rather than the authenticator's invalid_token. Nothing is created.
func TestAnAccountDeactivatedMeanwhileCannotCreateAWorkspace(t *testing.T) {
	contract := apitest.Load(t)
	base, pool := sessionApp(t)
	alice := registerAccount(t, contract, base, "alice@example.com").AccessToken

	deactivation, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = deactivation.Rollback(context.Background()) }()
	if _, err := deactivation.Exec(context.Background(), "UPDATE users SET is_active = false WHERE email = 'alice@example.com'"); err != nil {
		t.Fatal(err)
	}
	req := newRequest(t, http.MethodPost, base+"/api/v0/workspaces", alice, []byte(`{"name":"Acme","slug":"acme"}`))
	contract.CheckRequest(t, req)
	answered := sendInBackground(req)
	// The creation waits on alice's row. The app runs River's jobs on the
	// same database, so only a wait for a row of users counts.
	pgtest.WaitForLockWaitOn(t, pool, "users", 5*time.Second)
	if err := deactivation.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}

	a := receiveWithin(t, answered, 10*time.Second, "answer to the creation")
	if a.err != nil {
		t.Fatal(a.err)
	}
	contract.CheckResponse(t, req, a.res)
	var workspaces int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM workspaces").Scan(&workspaces); err != nil {
		t.Fatal(err)
	}
	if a.res.StatusCode != http.StatusUnauthorized || a.res.Header.Get("WWW-Authenticate") != "Bearer" ||
		problemCode(t, a.body) != "unauthorized" || workspaces != 0 {
		t.Errorf("create = %d %q %s, %d workspaces; want 401 with a plain Bearer challenge, none created",
			a.res.StatusCode, a.res.Header.Get("WWW-Authenticate"), a.body, workspaces)
	}
}

// answer is what a request sent in the background got.
type answer struct {
	res  *http.Response
	body []byte
	err  error
}

// sendInBackground sends req with the tests' client and hands over its
// answer, the body read and put back.
func sendInBackground(req *http.Request) <-chan answer {
	answered := make(chan answer, 1)
	go func() {
		res, err := client.Do(req)
		if err != nil {
			answered <- answer{err: err}
			return
		}
		body, err := io.ReadAll(res.Body)
		_ = res.Body.Close()
		res.Body = io.NopCloser(bytes.NewReader(body))
		answered <- answer{res, body, err}
	}()
	return answered
}

// The composed updateWorkspace decides after it has locked the workspace
// row (M3 design 3.6 convention 2). A transaction holds the row FOR NO KEY
// UPDATE and demotes alice, acme's admin, to member; her PATCH, authenticated,
// waits on the row. Once the demotion commits she is refused forbidden and
// the name stays: a decision taken before the lock would have read her
// uncommitted role, admin, and renamed the workspace.
func TestAnAdminDemotedMeanwhileCannotUpdateTheWorkspace(t *testing.T) {
	contract := apitest.Load(t)
	base, pool := sessionApp(t)
	alice := registerAccount(t, contract, base, "alice@example.com").AccessToken
	if status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces", alice, `{"name":"Acme","slug":"acme"}`); status != http.StatusCreated {
		t.Fatalf("create = %d %s", status, body)
	}

	demotion, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = demotion.Rollback(context.Background()) }()
	for _, sql := range []string{
		"SELECT id FROM workspaces WHERE slug = 'acme' FOR NO KEY UPDATE",
		"UPDATE workspace_members SET role = 15 WHERE member_id = (SELECT id FROM users WHERE email = 'alice@example.com')",
	} {
		if _, err := demotion.Exec(context.Background(), sql); err != nil {
			t.Fatal(err)
		}
	}
	req := newRequest(t, http.MethodPatch, base+"/api/v0/workspaces/acme", alice, []byte(`{"name":"Renamed"}`))
	contract.CheckRequest(t, req)
	answered := sendInBackground(req)
	// The app runs River's jobs on the same database: only a wait for a row
	// of workspaces counts.
	pgtest.WaitForLockWaitOn(t, pool, "workspaces", 5*time.Second)
	if err := demotion.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}

	a := receiveWithin(t, answered, 10*time.Second, "answer to the update")
	if a.err != nil {
		t.Fatal(a.err)
	}
	contract.CheckResponse(t, req, a.res)
	var name string
	if err := pool.QueryRow(context.Background(), "SELECT name FROM workspaces WHERE slug = 'acme'").Scan(&name); err != nil {
		t.Fatal(err)
	}
	if a.res.StatusCode != http.StatusForbidden || problemCode(t, a.body) != "forbidden" || name != "Acme" {
		t.Errorf("update = %d %s, name %q; want 403 forbidden and the name unchanged", a.res.StatusCode, a.body, name)
	}
}
