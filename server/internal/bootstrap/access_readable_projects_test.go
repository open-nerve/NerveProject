package bootstrap

import (
	"context"
	"errors"
	"maps"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/access"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Readable is Authorize's issue.read on each project (M7 P1c spec 2.5, R1),
// on the permission matrix's seed (prepareMatrix), each built as app_ports.go
// builds the Authorizer, from workspace's and project's Provide: for each
// registered account and each workspace of the seed, gone, deleted, among
// them, Readable answers, in id order, exactly the projects of the seed,
// its own workspace's and every other's, on which Authorize allows him
// issue.read with the Target of that workspace and project, each own-only
// as its Grant is, cell by cell. Then again once the copy holds the states
// no request makes alone (statesNoRequestMakes), P1b's, and acme's public
// project archived: those states delete the seed's archived project, and
// an archived project's members must still read it. Each pass answers some
// cells allowed, some own-only and some refused.
func TestReadableIsAuthorizeOnTheMatrixSeed(t *testing.T) {
	d := prepareMatrix(t)
	s := d.seeded.in(t)
	pool := openPool(t, pgtest.NewDatabaseFrom(t, d.url))
	wp, pp := workspace.Provide(pool, nil), project.Provide(pool, nil)
	auth := access.New(access.Deps{WorkspaceRoles: wp.WorkspaceRoles, ProjectAccess: accessProjects{projects: pp.ProjectAccess}})
	readable := access.NewReadableProjects(access.ReadableDeps{WorkspaceRoles: wp.WorkspaceRoles,
		Projects: accessMemberProjects{projects: pp.MemberProjectAccess}})
	ctx := context.Background()
	keys := map[uuid.UUID]string{}
	for _, p := range matrixProjects {
		keys[s.project(p.key)] = p.key
	}
	projects := slices.SortedFunc(maps.Keys(keys), uuid.UUID.Compare)
	named := func(grants []access.Grant) []string {
		var out []string
		for _, g := range grants {
			name, ok := keys[g.ProjectID]
			if !ok {
				name = "no project of the seed " + g.ProjectID.String()
			}
			if g.OwnOnly {
				name += " (own only)"
			}
			out = append(out, name)
		}
		return out
	}
	compare := func(pass string) {
		t.Helper()
		answers := map[string]int{}
		for _, slug := range slices.Sorted(maps.Keys(s.workspaces)) {
			for _, c := range matrixAccounts {
				actor := shared.Actor{UserID: s.account(c)}
				got, err := readable.Readable(ctx, actor, s.workspace(slug))
				if err != nil {
					t.Fatalf("%s: Readable(%s, %s) = %v", pass, c, slug, err)
				}
				var want []access.Grant
				for _, p := range projects {
					grant, err := auth.Authorize(ctx, actor, "issue.read", shared.Target{WorkspaceID: s.workspace(slug), ProjectID: p})
					switch {
					case err == nil && grant.OwnOnly:
						answers["own-only"]++
					case err == nil:
						answers["allowed"]++
					case errors.Is(err, shared.ErrNotVisible) || errors.Is(err, shared.Forbidden()):
						answers["refused"]++
						continue
					default:
						t.Fatalf("%s: Authorize(%s, %s, %s) = %v", pass, c, slug, keys[p], err)
					}
					want = append(want, access.Grant{ProjectID: p, OwnOnly: grant.OwnOnly})
				}
				if !slices.Equal(got, want) {
					t.Errorf("%s: %s in %s: Readable %q, Authorize allows %q", pass, c, slug, named(got), named(want))
				}
			}
		}
		for _, a := range []string{"allowed", "own-only", "refused"} {
			if answers[a] == 0 {
				t.Errorf("%s: no cell answers %s (%v); the comparison needs each answer", pass, a, answers)
			}
		}
	}
	compare("the seed")
	statesNoRequestMakes(t, pool, s)
	matrixSeed{}.exec(t, pool, "UPDATE projects SET archived_at = $2 WHERE id = $1 AND archived_at IS NULL AND deleted_at IS NULL",
		s.project("acme/public"), time.Now())
	compare("the seed with the states no request makes alone, acme's public project archived")
}

// fakeMemberProjectAccess answers the facts it holds by workspace and
// account, and records what it was asked.
type fakeMemberProjectAccess struct {
	facts map[[2]uuid.UUID][]project.MemberAccessFacts
	err   error
	asked [][2]uuid.UUID
}

func (f *fakeMemberProjectAccess) MemberProjectFacts(_ context.Context, workspaceID, userID uuid.UUID) ([]project.MemberAccessFacts, error) {
	f.asked = append(f.asked, [2]uuid.UUID{workspaceID, userID})
	if f.err != nil {
		return nil, f.err
	}
	return f.facts[[2]uuid.UUID{workspaceID, userID}], nil
}

// accessMemberProjects hands access project's answer to the same question:
// each project's facts converted, in its order; the error as it came, with
// no facts.
func TestAccessMemberProjectsConvertsProjectsAnswer(t *testing.T) {
	acme, web, secret, alice, bob := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	fake := &fakeMemberProjectAccess{facts: map[[2]uuid.UUID][]project.MemberAccessFacts{{acme, alice}: {
		{ProjectID: web, Public: true, Role: shared.RoleGuest, GuestViewAll: true}, {ProjectID: secret, Role: shared.RoleAdmin}}}}
	a := accessMemberProjects{projects: fake}
	ctx := context.Background()
	want := []access.MemberProjectFacts{{ProjectID: web, Public: true, Role: shared.RoleGuest, GuestViewAll: true},
		{ProjectID: secret, Role: shared.RoleAdmin}}
	if got, err := a.MemberProjectFacts(ctx, acme, alice); err != nil || !slices.Equal(got, want) {
		t.Errorf("MemberProjectFacts(acme, alice) = %+v, %v; want %+v", got, err, want)
	}
	if got, err := a.MemberProjectFacts(ctx, acme, bob); err != nil || len(got) != 0 {
		t.Errorf("MemberProjectFacts(acme, bob) = %+v, %v; want none", got, err)
	}
	if want := [][2]uuid.UUID{{acme, alice}, {acme, bob}}; !slices.Equal(fake.asked, want) {
		t.Errorf("project was asked %v, want %v", fake.asked, want)
	}
	failure := errors.New("connection reset")
	fake.err = failure
	if got, err := a.MemberProjectFacts(ctx, acme, alice); !errors.Is(err, failure) || got != nil {
		t.Errorf("MemberProjectFacts() = %+v, %v; want %v and none", got, err, failure)
	}
}
