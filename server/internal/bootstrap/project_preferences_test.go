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
// through the stores, without settings, and alice's in Web are deleted
// through SQL, as no operation deletes them alone: the first change of
// each inserts a row of his own, two in one project.
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
	if tag, err := pool.Exec(ctx, "UPDATE project_user_properties SET deleted_at = $3 WHERE project_id = $1 AND user_id = $2 AND "+
		"deleted_at IS NULL", web, aliceID, now); err != nil || tag.RowsAffected() != 1 {
		t.Fatalf("deleting alice's settings in Web: %v, %v; want one row", tag, err)
	}
	settings := func(token string, project uuid.UUID) string {
		t.Helper()
		status, body := call(t, contract, http.MethodGet, base+"/api/v0/me/projects/"+project.String()+"/preferences", token, "")
		if status != http.StatusOK {
			t.Fatalf("reading the settings in %s = %d %s", project, status, body)
		}
		return strings.TrimSuffix(body, "\n")
	}
	change := func(who, token, body, want string) {
		t.Helper()
		if status, got := call(t, contract, http.MethodPatch, base+"/api/v0/me/projects/"+web.String()+"/preferences", token, body); status !=
			http.StatusOK || got != want+"\n" {
			t.Fatalf("%s's change in Web = %d %s, want 200 %s", who, status, got, want)
		}
	}
	alicesOps := settings(alice, ops)
	if got, want := settings(alice, web), `{"navigation":{"default_tab":"work_items","hide_in_more_menu":[]},"sort_order":65535}`; got != want {
		t.Errorf("alice in Web, without settings: %s, want the defaults %s", got, want)
	}
	bobs := `{"navigation":{"default_tab":"work_items","hide_in_more_menu":[]},"sort_order":3}`
	// Another member's change in progress holds Web FOR SHARE; bob's does
	// not wait for it.
	other := holding(t, pool, "SELECT 1 FROM projects WHERE id = $1 FOR SHARE", web)
	a := receiveWithin(t, sendInBackground(newRequest(t, http.MethodPatch, base+"/api/v0/me/projects/"+web.String()+"/preferences", bob,
		[]byte(`{"sort_order":3}`))), 10*time.Second, "bob's change while Web is held FOR SHARE")
	if a.err != nil || a.res.StatusCode != http.StatusOK || string(a.body) != bobs+"\n" {
		t.Fatalf("bob's change in Web = %v %s, want 200 %s", a.err, a.body, bobs)
	}
	if err := other.Rollback(ctx); err != nil {
		t.Fatal(err)
	}

	// Alice's first change inserts her row; her second changes it, the
	// place kept.
	changed := `{"navigation":{"default_tab":"cycles","hide_in_more_menu":["intake"]},"sort_order":1}`
	change("alice", alice, `{"sort_order":1}`, `{"navigation":{"default_tab":"work_items","hide_in_more_menu":[]},"sort_order":1}`)
	change("alice", alice, `{"navigation":{"default_tab":"cycles","hide_in_more_menu":["intake"]}}`, changed)

	for _, tt := range []struct {
		who, token string
		project    uuid.UUID
		want       string
	}{
		{"alice in Web", alice, web, changed},
		{"bob in Web", bob, web, bobs},
		{"alice in Ops", alice, ops, alicesOps},
	} {
		if got := settings(tt.token, tt.project); got != tt.want {
			t.Errorf("%s: %s, want %s", tt.who, got, tt.want)
		}
	}
}
