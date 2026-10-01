package bootstrap

import (
	"context"
	"net/http"
	"slices"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// The project module as bootstrap wires it (M3 design 6.6): alice creates
// acme, in Shanghai time, and invites bob, who joins it as a member. She
// creates a project with bob as its lead: the answer is the project as
// stored, as she sees it, her role 20, first in her sidebar, the two of
// them its members, in acme's time zone; the rows it wrote are there, each
// of them its admin with his display settings, and the six states, Backlog
// the default. Another project with its identifier, in any case, or its
// name answers the code of each and writes nothing.
func TestCreatingAProject(t *testing.T) {
	contract := apitest.Load(t)
	dbURL := pgtest.NewDatabase(t)
	base := startApp(t, testConfig(t, dbURL, false), migrations.FS())
	alice := registerAccount(t, contract, base, "alice@example.com").AccessToken
	bob := registerAccount(t, contract, base, "bob@example.com").AccessToken
	if status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces", alice,
		`{"name":"Acme","slug":"acme","timezone":"Asia/Shanghai"}`); status != http.StatusCreated {
		t.Fatalf("creating acme = %d %s", status, body)
	}
	answerInvitation(t, contract, base, bob, "accept", invite(t, contract, base, alice, "acme", "bob@example.com"), http.StatusOK)
	aliceID, bobID := accountID(t, contract, base, alice), accountID(t, contract, base, bob)

	status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces/acme/projects", alice,
		`{"name":"Web","identifier":"web","project_lead_id":"`+bobID.String()+`"}`)

	var p struct {
		ID         uuid.UUID   `json:"id"`
		Identifier string      `json:"identifier"`
		Network    int         `json:"network"`
		LeadID     *uuid.UUID  `json:"project_lead_id"`
		Timezone   string      `json:"timezone"`
		MemberRole *int        `json:"member_role"`
		SortOrder  *float64    `json:"sort_order"`
		MemberIDs  []uuid.UUID `json:"member_ids"`
	}
	decodeAnswer(t, body, &p)
	if status != http.StatusCreated || p.Identifier != "WEB" || p.Network != 2 || p.LeadID == nil || *p.LeadID != bobID ||
		p.Timezone != "Asia/Shanghai" || p.MemberRole == nil || *p.MemberRole != 20 || p.SortOrder == nil || *p.SortOrder != 65535 ||
		!slices.Equal(p.MemberIDs, []uuid.UUID{aliceID, bobID}) {
		t.Fatalf("create = %d %s; want 201 with WEB, public, bob its lead, in Asia/Shanghai, alice its admin at 65535, alice and bob its members",
			status, body)
	}
	pool := openPool(t, dbURL)
	want := aliceID.String() + " 20 65535, " + bobID.String() + " 20 65535 | " +
		"Backlog backlog default, Todo unstarted, In Progress started, Done completed, Cancelled cancelled, Triage triage | 1 project"
	if got := projectRows(t, pool, p.ID); got != want {
		t.Errorf("the rows are %s, want %s", got, want)
	}

	for _, tt := range []struct{ body, code string }{
		{`{"name":"Web 2","identifier":"Web"}`, "project.identifier_taken"},
		{`{"name":"Web","identifier":"WEB2"}`, "project.name_taken"},
	} {
		if status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces/acme/projects", bob, tt.body); status != http.StatusConflict ||
			problemCode(t, []byte(body)) != tt.code {
			t.Errorf("POST %s = %d %s, want 409 %s", tt.body, status, body, tt.code)
		}
	}
	if got := projectRows(t, pool, p.ID); got != want {
		t.Errorf("after the refusals the rows are %s, want %s", got, want)
	}
}

// accountID is the id of the account whose access token is token.
func accountID(t *testing.T, contract *apitest.Contract, base, token string) uuid.UUID {
	t.Helper()
	status, body := call(t, contract, http.MethodGet, base+"/api/v0/me", token, "")
	var me struct {
		ID uuid.UUID `json:"id"`
	}
	if status != http.StatusOK {
		t.Fatalf("GET /api/v0/me = %d %s", status, body)
	}
	decodeAnswer(t, body, &me)
	return me.ID
}

// projectRows are the undeleted rows under the project id: each
// membership's account, role and place in his sidebar; each state's name,
// group and whether it is the default, by sequence; and the number of the
// workspace's projects.
func projectRows(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) string {
	t.Helper()
	var rows string
	if err := pool.QueryRow(context.Background(), `
SELECT (SELECT string_agg(m.member_id || ' ' || m.role || ' ' || u.sort_order, ', ' ORDER BY m.id)
          FROM project_members m JOIN project_user_properties u ON u.project_id = m.project_id AND u.user_id = m.member_id
         WHERE m.project_id = $1 AND m.deleted_at IS NULL AND u.deleted_at IS NULL)
    || ' | ' || (SELECT string_agg(s.name || ' ' || s."group" || CASE WHEN s."default" THEN ' default' ELSE '' END, ', ' ORDER BY s.sequence)
                   FROM states s WHERE s.project_id = $1 AND s.deleted_at IS NULL)
    || ' | ' || (SELECT count(*) FROM projects p
                  WHERE p.workspace_id = (SELECT workspace_id FROM projects WHERE id = $1) AND p.deleted_at IS NULL) || ' project'`,
		id).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	return rows
}
