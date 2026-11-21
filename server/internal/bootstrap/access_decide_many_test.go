package bootstrap

import (
	"context"
	"errors"
	"maps"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/access"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	projectpg "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres"
	projectapp "github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// DecideMany is Authorize for each account (M7 P1b spec 2.7, R7; M7 design
// 3.3), on the permission matrix's seed (prepareMatrix), each built as
// app_ports.go builds the Authorizer, from workspace's and project's Provide: for
// each workspace of the seed, gone, deleted, among them, and each project
// of the seed, its own workspace's and every other's, one DecideMany of
// issue.read for every registered account answers each as Authorize
// answers him with the Target of that workspace and project, cell by cell:
// allowed when Authorize answers no error, own-only then as its Grant's.
// Then again once the copy holds the states, made by SQL and the project
// store, in which a batch read could part from the single one and that no
// request makes alone: the removed member's membership of acme's public
// project active again, his of acme still ended; PM's membership of acme
// deleted, his project memberships left; acme's archived project deleted
// alone, its memberships left; other deleted alone, its memberships and
// project left; and acme's member a member of other's project. Each pass
// answers some accounts allowed, some own-only and some refused.
func TestDecideManyIsAuthorizeOnTheMatrixSeed(t *testing.T) {
	d := prepareMatrix(t)
	s := d.seeded.in(t)
	pool := openPool(t, pgtest.NewDatabaseFrom(t, d.url))
	wp, pp := workspace.Provide(pool, nil), project.Provide(pool, nil)
	auth := access.New(access.Deps{WorkspaceRoles: wp.WorkspaceRoles, ProjectAccess: accessProjects{projects: pp.ProjectAccess}})
	decider := access.NewDecider(access.DeciderDeps{WorkspaceRoles: wp.WorkspaceRolesMany,
		ProjectAccess: accessProjectsMany{projects: pp.ProjectAccessMany}})
	ctx := context.Background()
	accounts := make([]uuid.UUID, len(matrixAccounts))
	for i, c := range matrixAccounts {
		accounts[i] = s.account(c)
	}
	compare := func(pass string) {
		t.Helper()
		answers := map[access.Decision]int{}
		for _, slug := range slices.Sorted(maps.Keys(s.workspaces)) {
			for _, p := range matrixProjects {
				target := shared.Target{WorkspaceID: s.workspace(slug), ProjectID: s.project(p.key)}
				got, err := decider.DecideMany(ctx, "issue.read", target.WorkspaceID, target.ProjectID, accounts)
				if err != nil || len(got) != len(accounts) {
					t.Fatalf("%s: DecideMany(%s, %s) = %v, %v; want every account answered", pass, slug, p.key, got, err)
				}
				for _, c := range matrixAccounts {
					grant, err := auth.Authorize(ctx, shared.Actor{UserID: s.account(c)}, "issue.read", target)
					if err != nil && !errors.Is(err, shared.ErrNotVisible) && !errors.Is(err, shared.Forbidden()) {
						t.Fatalf("%s: Authorize(%s, %s, %s) = %v", pass, c, slug, p.key, err)
					}
					want := access.Decision{Allowed: err == nil, OwnOnly: err == nil && grant.OwnOnly}
					if got[s.account(c)] != want {
						t.Errorf("%s: %s in %s, %s: DecideMany %+v, Authorize %+v, %v", pass, c, slug, p.key, got[s.account(c)], grant, err)
					}
					answers[want]++
				}
			}
		}
		for _, a := range []access.Decision{{}, {Allowed: true}, {Allowed: true, OwnOnly: true}} {
			if answers[a] == 0 {
				t.Errorf("%s: no cell answers %+v (%v); the comparison needs each answer", pass, a, answers)
			}
		}
	}
	compare("the seed")
	statesNoRequestMakes(t, pool, s)
	compare("the seed with the states no request makes alone")
}

// statesNoRequestMakes puts the copy pool holds of the matrix's seed s in
// the states, made by SQL and the project store, in which a read of many
// accounts' or many projects' facts could part from Authorize's read of one
// and that no request makes alone (M7 P1b spec 2.7, P1c spec 2.5): the
// removed member's membership of acme's public project active again, his
// of acme still ended; PM's membership of acme deleted, his project
// memberships left; acme's archived project deleted alone, its memberships
// left; other deleted alone, its memberships and project left; and acme's
// member a member of other's project. Each SQL statement must change one
// row (matrixSeed.exec).
func statesNoRequestMakes(t *testing.T, pool *pgxpool.Pool, s seeded) {
	t.Helper()
	now, exec := time.Now(), func(sql string, args ...any) { matrixSeed{}.exec(t, pool, sql, args...) }
	exec("UPDATE project_members SET is_active = true WHERE project_id = $1 AND member_id = $2 AND NOT is_active AND deleted_at IS NULL",
		s.project("acme/public"), s.account(callerRemoved))
	exec("UPDATE workspace_members SET deleted_at = $3 WHERE workspace_id = $1 AND member_id = $2 AND deleted_at IS NULL",
		s.workspace("acme"), s.account(callerProjectMember), now)
	exec("UPDATE projects SET deleted_at = $2 WHERE id = $1 AND deleted_at IS NULL", s.project("acme/archived"), now)
	exec("UPDATE workspaces SET deleted_at = $2 WHERE id = $1 AND deleted_at IS NULL", s.workspace("other"), now)
	if err := projectpg.New(pool).CreateMember(context.Background(), projectapp.MemberRow{ID: uuid.NewV7(), WorkspaceID: s.workspace("other"),
		ProjectID: s.project("other/project"), MemberID: s.account(callerMember), Role: shared.RoleMember, CreatedBy: s.account(callerNever),
		Now: now}); err != nil {
		t.Fatal(err)
	}
}

// fakeProjectAccessMany answers the facts it holds by project, and records
// what it was asked.
type fakeProjectAccessMany struct {
	facts map[uuid.UUID]project.AccessFactsMany
	err   error
	asked [][]uuid.UUID
}

func (f *fakeProjectAccessMany) ProjectFactsMany(_ context.Context, projectID uuid.UUID, ids []uuid.UUID) (project.AccessFactsMany, bool, error) {
	f.asked = append(f.asked, append([]uuid.UUID{projectID}, ids...))
	p, found := f.facts[projectID]
	return p, found, f.err
}

// accessProjectsMany hands access project's answer to the same question:
// the facts converted, the roles, found and the error as they came.
func TestAccessProjectsManyConvertsProjectsAnswer(t *testing.T) {
	web, gone, acme, alice, bob := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	roles := map[uuid.UUID]shared.Role{alice: shared.RoleGuest}
	fake := &fakeProjectAccessMany{facts: map[uuid.UUID]project.AccessFactsMany{web: {WorkspaceID: acme, Public: true, GuestViewAll: true,
		Roles: roles}}}
	a := accessProjectsMany{projects: fake}
	ctx := context.Background()
	f, found, err := a.ProjectFactsMany(ctx, web, []uuid.UUID{alice, bob})
	if err != nil || !found || f.WorkspaceID != acme || !f.Public || !f.GuestViewAll || !maps.Equal(f.Roles, roles) {
		t.Errorf("ProjectFactsMany(web) = %+v, %v, %v; want acme's, public, every work item shown, alice its guest", f, found, err)
	}
	if f, found, err := a.ProjectFactsMany(ctx, gone, []uuid.UUID{alice}); err != nil || found || f.Roles != nil {
		t.Errorf("ProjectFactsMany(gone) = %+v, %v, %v; want not found", f, found, err)
	}
	if want := [][]uuid.UUID{{web, alice, bob}, {gone, alice}}; !slices.EqualFunc(fake.asked, want, slices.Equal) {
		t.Errorf("project was asked %v, want %v", fake.asked, want)
	}
	failure := errors.New("connection reset")
	fake.err = failure
	if _, _, err := a.ProjectFactsMany(ctx, web, []uuid.UUID{alice}); !errors.Is(err, failure) {
		t.Errorf("ProjectFactsMany() = %v, want %v", err, failure)
	}
}
