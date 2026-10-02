package bootstrap

import (
	"context"
	"maps"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	projectpg "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres"
	projectapp "github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	projectdomain "github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// endedMembers prepares the database at url: through the command line,
// the accounts of alice, bob, carol and dave, and acme and gone, alice the
// admin of each; through the stores, bob acme's admin and carol its guest,
// bob the admin of Web and Ops, carol Web's guest; then bob's and carol's
// memberships of acme and of its projects ended by alice at one moment, as
// a removal ends them (M3 design 3.6, 3.7); gone deleted; carol's account
// deactivated. dave was never a member of acme.
func endedMembers(t *testing.T, url string) *pgxpool.Pool {
	t.Helper()
	pool := openPool(t, url)
	ids := map[string]uuid.UUID{}
	for _, name := range []string{"alice", "bob", "carol", "dave"} {
		ids[name] = uuid.MustParse(createdAccount(t, url, pool, name+"@corp.com"))
	}
	for _, slug := range []string{"acme", "gone"} {
		if _, _, err := runWorkspaces(t, url, CreateWorkspace(slug, slug, "alice@corp.com")); err != nil {
			t.Fatal(err)
		}
	}
	workspaces, projects, ctx, now := workspacepg.New(pool), projectpg.New(pool), context.Background(), time.Now()
	workspaceID := func(slug string) uuid.UUID {
		w, err := workspaces.WorkspaceBySlug(ctx, slug)
		if err != nil {
			t.Fatal(err)
		}
		return w.ID
	}
	acme, gone := workspaceID("acme"), workspaceID("gone")
	var acmes []uuid.UUID
	for _, p := range []struct{ name, identifier string }{{"Web", "WEB"}, {"Ops", "OPS"}} {
		id := uuid.NewV7()
		if err := projects.CreateProject(ctx, projectapp.ProjectRow{ID: id, WorkspaceID: acme, Name: p.name, Identifier: p.identifier,
			Network: projectdomain.NetworkPublic, Timezone: "UTC", CreatedBy: ids["alice"], Now: now}); err != nil {
			t.Fatal(err)
		}
		acmes = append(acmes, id)
	}
	for _, m := range []struct {
		name     string
		role     shared.Role
		projects []uuid.UUID
	}{{"bob", shared.RoleAdmin, acmes}, {"carol", shared.RoleGuest, acmes[:1]}} {
		if err := workspaces.CreateMember(ctx, workspaceapp.MemberRow{ID: uuid.NewV7(), WorkspaceID: acme, MemberID: ids[m.name], Role: m.role,
			CreatedBy: ids["alice"], Now: now}); err != nil {
			t.Fatal(err)
		}
		for _, p := range m.projects {
			if err := projects.CreateMember(ctx, projectapp.MemberRow{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: p, MemberID: ids[m.name],
				Role: m.role, CreatedBy: ids["alice"], Now: now}); err != nil {
				t.Fatal(err)
			}
		}
		if err := workspaces.EndMember(ctx, acme, ids[m.name], ids["alice"], now); err != nil {
			t.Fatal(err)
		}
		if err := projects.EndMemberships(ctx, m.projects, ids[m.name], ids["alice"], now); err != nil {
			t.Fatal(err)
		}
	}
	if err := workspaces.DeleteWorkspace(ctx, gone, ids["alice"], now); err != nil {
		t.Fatal(err)
	}
	if _, _, err := runUsers(t, url, DeactivateUser("carol@corp.com")); err != nil {
		t.Fatal(err)
	}
	return pool
}

// memberStates are the memberships of acme's members and of its projects,
// one line each, in byte order: the account, the workspace's or the
// project's, the role and whether it is active.
func memberStates(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var rows string
	if err := pool.QueryRow(context.Background(), `SELECT string_agg(s, E'\n' ORDER BY s COLLATE "C") FROM (
		SELECT u.email || ' acme ' || m.role || ' ' || m.is_active AS s FROM workspace_members m JOIN users u ON u.id = m.member_id
		JOIN workspaces w ON w.id = m.workspace_id WHERE w.slug = 'acme'
		UNION ALL SELECT u.email || ' ' || p.name || ' ' || m.role || ' ' || m.is_active FROM project_members m
		JOIN users u ON u.id = m.member_id JOIN projects p ON p.id = m.project_id) r`).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	return rows
}

// `nerve workspaces reactivate-member` runs on the minimal composition (M3
// design 3.11, 6.6), the address normalized: bob's membership of acme is
// active again, an admin's still, and his memberships of Web and Ops stay
// ended, as the line says; the reactivation is logged once. Run again, it
// says the membership is active and changes nothing. carol's, her account
// deactivated, is reactivated all the same, and the line says what is
// next; run again, it says so after the membership is reported active.
func TestWorkspacesReactivateMember(t *testing.T) {
	url := pgtest.NewDatabase(t)
	pool := endedMembers(t, url)
	if got, want := memberStates(t, pool), strings.Join([]string{"alice@corp.com acme 20 true", "bob@corp.com Ops 20 false",
		"bob@corp.com Web 20 false", "bob@corp.com acme 20 false", "carol@corp.com Web 5 false", "carol@corp.com acme 5 false"}, "\n"); got != want {
		t.Fatalf("the memberships before:\n%s\nwant\n%s", got, want)
	}

	out, logs, err := runWorkspaces(t, url, ReactivateMember("acme", " Bob@Corp.COM "))

	if want := "reactivated bob@corp.com in acme as admin; project memberships still ended: 2, each restored when the member joins or is added to " +
		"its project\n"; err != nil || out != want {
		t.Fatalf("reactivate-member = %q, %v; want %q", out, err, want)
	}
	if strings.Count(logs, `msg="workspace member reactivated"`) != 1 || !strings.Contains(logs, "by=cli") {
		t.Errorf("logs = %s, want the reactivation logged once, by cli", logs)
	}
	if got, want := memberStates(t, pool), strings.Join([]string{"alice@corp.com acme 20 true", "bob@corp.com Ops 20 false",
		"bob@corp.com Web 20 false", "bob@corp.com acme 20 true", "carol@corp.com Web 5 false", "carol@corp.com acme 5 false"}, "\n"); got != want {
		t.Errorf("the memberships after:\n%s\nwant\n%s", got, want)
	}
	before := tableRows(t, pool, riversOwn)
	if out, _, err := runWorkspaces(t, url, ReactivateMember("acme", "bob@corp.com")); err != nil ||
		out != "bob@corp.com is an active member of acme already; nothing changed\n" {
		t.Errorf("reactivate-member again = %q, %v; want the membership reported active", out, err)
	}
	if after := tableRows(t, pool, riversOwn); !maps.Equal(after, before) {
		t.Errorf("the tables after reactivating an active membership changed:\n%v\nwant them as they were:\n%v", after, before)
	}
	out, _, err = runWorkspaces(t, url, ReactivateMember("acme", "carol@corp.com"))
	if want := "reactivated carol@corp.com in acme as guest; project memberships still ended: 1, each restored when the member joins or is " +
		"added to its project; the account is deactivated: run nerve users activate --email carol@corp.com next\n"; err != nil || out != want {
		t.Errorf("reactivate-member of carol = %q, %v; want %q", out, err, want)
	}
	out, _, err = runWorkspaces(t, url, ReactivateMember("acme", "carol@corp.com"))
	if want := "carol@corp.com is an active member of acme already; nothing changed; the account is deactivated: run nerve users activate " +
		"--email carol@corp.com next\n"; err != nil || out != want {
		t.Errorf("reactivate-member of carol again = %q, %v; want %q", out, err, want)
	}
}

// A refused reactivation prints no line, says why in one line, and leaves
// every table as it was (M3 design 3.11): no such account, no such
// workspace, a deleted one, an account never its member; and a
// reactivation refused at its commit, after every statement ran, so that
// no step wrote in a transaction of its own.
func TestWorkspacesReactivateMemberErrors(t *testing.T) {
	url := pgtest.NewDatabase(t)
	pool := endedMembers(t, url)
	before := tableRows(t, pool, riversOwn)
	tests := []struct {
		name, slug, email string
		refuse            bool // the commit of a write of workspace_members
		want              string
	}{
		{"an unknown account", "acme", "nobody@corp.com", false, "No account has this e-mail address."},
		{"an unknown workspace", "beta", "bob@corp.com", false, "No workspace has this slug."},
		{"a deleted workspace", "gone", "alice@corp.com", false, "No workspace has this slug."},
		{"never a member", "acme", "dave@corp.com", false, "The account has never been a member of this workspace."},
		{"the commit refused", "acme", "bob@corp.com", true, "the commit is refused"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.refuse {
				defer refusingCommits(t, pool, "workspace_members")()
			}
			out, _, err := runWorkspaces(t, url, ReactivateMember(tt.slug, tt.email))
			if err == nil || !strings.Contains(err.Error(), tt.want) || out != "" {
				t.Errorf("= %q, %v; want no line and %q", out, err, tt.want)
			}
			if after := tableRows(t, pool, riversOwn); !maps.Equal(after, before) {
				t.Errorf("the tables changed:\n%v\nwant them as they were:\n%v", after, before)
			}
		})
	}
}
