package bootstrap

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
	"uuid"

	projectpg "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres"
	projectapp "github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Each member reads and changes his own display settings in a project, on
// the wired app (M3 design 3.18): alice's change of hers in Web leaves
// bob's there, and hers in Ops, as they were, and each reads back his own,
// a member without any the defaults. Bob is made a member of acme and Web
// through the stores, his settings in Web at 3.
func TestEachMemberHasHisOwnDisplaySettings(t *testing.T) {
	contract, base, pool, alice, aliceID, web, ops := twoProjects(t)
	bob := registerAccount(t, contract, base, "bob@example.com").AccessToken
	bobID := accountID(t, contract, base, bob)
	var acme uuid.UUID
	if err := pool.QueryRow(context.Background(), "SELECT workspace_id FROM projects WHERE id = $1", web).Scan(&acme); err != nil {
		t.Fatal(err)
	}
	ctx, now := context.Background(), time.Now()
	if err := workspacepg.New(pool).CreateMember(ctx, workspaceapp.MemberRow{ID: uuid.NewV7(), WorkspaceID: acme, MemberID: bobID,
		Role: shared.RoleMember, CreatedBy: aliceID, Now: now}); err != nil {
		t.Fatal(err)
	}
	projects := projectpg.New(pool)
	if err := projects.CreateMember(ctx, projectapp.MemberRow{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web, MemberID: bobID,
		Role: shared.RoleMember, CreatedBy: aliceID, Now: now}); err != nil {
		t.Fatal(err)
	}
	if err := projects.CreatePreferences(ctx, projectapp.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web, UserID: bobID,
		SortOrder: 3, CreatedBy: aliceID, Now: now}); err != nil {
		t.Fatal(err)
	}
	settings := func(token string, project uuid.UUID) string {
		t.Helper()
		status, body := call(t, contract, http.MethodGet, base+"/api/v0/me/projects/"+project.String()+"/preferences", token, "")
		if status != http.StatusOK {
			t.Fatalf("reading the settings in %s = %d %s", project, status, body)
		}
		return strings.TrimSuffix(body, "\n")
	}
	alicesOps := settings(alice, ops)

	changed := `{"navigation":{"default_tab":"cycles","hide_in_more_menu":["intake"]},"sort_order":1}`
	if status, body := call(t, contract, http.MethodPatch, base+"/api/v0/me/projects/"+web.String()+"/preferences", alice, changed); status !=
		http.StatusOK || body != changed+"\n" {
		t.Fatalf("alice's change in Web = %d %s, want 200 %s", status, body, changed)
	}

	for _, tt := range []struct {
		who, token string
		project    uuid.UUID
		want       string
	}{
		{"alice in Web", alice, web, changed},
		{"bob in Web", bob, web, `{"navigation":{"default_tab":"work_items","hide_in_more_menu":[]},"sort_order":3}`},
		{"alice in Ops", alice, ops, alicesOps},
	} {
		if got := settings(tt.token, tt.project); got != tt.want {
			t.Errorf("%s: %s, want %s", tt.who, got, tt.want)
		}
	}
}
