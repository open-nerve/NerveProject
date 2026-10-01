package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/access"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// The Authorizer takes a project's facts within the target's workspace only
// (M3 design 3.4; carry 8), on the real stores: the test builds its own,
// wired as app.go wires it, from workspace's and project's Provide. alice
// is the admin of acme and of beta, and the admin of acme's private project
// Web. Through acme she reads Web as its admin; through beta, where every
// project is hers to see, she does not see it.
func TestTheAuthorizerSeesNoProjectOfAnotherWorkspace(t *testing.T) {
	contract := apitest.Load(t)
	dbURL := pgtest.NewDatabase(t)
	base := startApp(t, testConfig(t, dbURL, false), migrations.FS())
	alice := registerAccount(t, contract, base, "alice@example.com").AccessToken
	for _, slug := range []string{"acme", "beta"} {
		if status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces", alice,
			`{"name":"`+slug+`","slug":"`+slug+`"}`); status != http.StatusCreated {
			t.Fatalf("creating %s = %d %s", slug, status, body)
		}
	}
	status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces/acme/projects", alice,
		`{"name":"Web","identifier":"WEB","network":0}`)
	if status != http.StatusCreated {
		t.Fatalf("creating Web = %d %s", status, body)
	}
	var web struct {
		ID          uuid.UUID `json:"id"`
		WorkspaceID uuid.UUID `json:"workspace_id"`
	}
	decodeAnswer(t, body, &web)
	pool := openPool(t, dbURL)
	var beta uuid.UUID
	if err := pool.QueryRow(context.Background(), "SELECT id FROM workspaces WHERE slug = 'beta'").Scan(&beta); err != nil {
		t.Fatal(err)
	}
	authorizer := access.New(access.Deps{
		WorkspaceRoles: workspace.Provide(pool).WorkspaceRoles,
		ProjectAccess:  accessProjects{projects: project.Provide(pool).ProjectAccess},
	})
	actor, read := shared.Actor{UserID: accountID(t, contract, base, alice)}, shared.Action("project.read")

	g, err := authorizer.Authorize(context.Background(), actor, read, shared.Target{WorkspaceID: web.WorkspaceID, ProjectID: web.ID})
	if err != nil || g.ProjectRole != shared.RoleAdmin {
		t.Errorf("project.read on Web through acme = %+v, %v; want her grant as its admin", g, err)
	}
	g, err = authorizer.Authorize(context.Background(), actor, read, shared.Target{WorkspaceID: beta, ProjectID: web.ID})
	if !errors.Is(err, shared.ErrNotVisible) {
		t.Errorf("project.read on Web through beta = %+v, %v; want %v", g, err, shared.ErrNotVisible)
	}
}
