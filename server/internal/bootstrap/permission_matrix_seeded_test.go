package bootstrap

import (
	"fmt"
	"runtime"
	"slices"
	"strings"
	"testing"
	"uuid"

	projectdomain "github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The rows prepareMatrix seeds, and how a row's request names them; the
// writers that seed them are in permission_matrix_seed_test.go.

// matrixMemberships are the memberships prepareMatrix seeds, workspace by
// workspace, each workspace's creator first: acme with its admin, member,
// guest and the member later removed, and the project level's own accounts
// (matrixAccounts); gone with its admin and the member; other, whose admin
// was never a member of acme and where the removed member is still active.
var matrixMemberships = []struct {
	slug string
	c    caller
	role shared.Role
}{
	{"acme", callerAdmin, shared.RoleAdmin}, {"acme", callerMember, shared.RoleMember}, {"acme", callerGuest, shared.RoleGuest},
	{"acme", callerRemoved, shared.RoleMember}, {"acme", callerProjectAdmin, shared.RoleMember},
	{"acme", callerProjectMember, shared.RoleMember}, {"acme", callerMemberAndAdmin, shared.RoleAdmin},
	{"acme", callerGuestOnly, shared.RoleGuest}, {"acme", callerBefore, shared.RoleMember},
	{"gone", callerDeleted, shared.RoleAdmin}, {"gone", callerMember, shared.RoleMember},
	{"other", callerNever, shared.RoleAdmin}, {"other", callerRemoved, shared.RoleMember},
}

// matrixInvitations are the invitations prepareMatrix seeds, each sent by
// its workspace's admin: in acme and in gone, one to an address no account
// has; and one to each column's own address (ownInvitation), as a member.
// gone's are deleted with it.
var matrixInvitations = []struct {
	slug, email string
	role        shared.Role
}{
	{"acme", "newcomer@example.com", shared.RoleMember},
	{"gone", "newcomer@example.com", shared.RoleMember},
	{"other", emailOf(callerAdmin), shared.RoleMember},
	{"other", emailOf(callerMember), shared.RoleMember},
	{"other", emailOf(callerGuest), shared.RoleMember},
	{"acme", emailOf(callerNever), shared.RoleMember},
	{"acme", emailOf(callerRemoved), shared.RoleMember},
	{"acme", emailOf(callerDeleted), shared.RoleMember},
}

// matrixProjects are the projects prepareMatrix seeds through the project
// store, each by its workspace's admin: acme's public and private ones,
// and one archived; gone's, deleted with it; other's, which no list of
// acme's may show.
var matrixProjects = []struct {
	key, name, identifier string // key: the workspace's slug / which project
	network               projectdomain.Network
}{
	{"acme/public", "Web", "WEB", projectdomain.NetworkPublic},
	{"acme/private", "Secret", "SEC", projectdomain.NetworkPrivate},
	{"acme/archived", "Old", "OLD", projectdomain.NetworkPublic},
	{"gone/project", "Web", "WEB", projectdomain.NetworkPublic},
	{"other/project", "Other", "OTH", projectdomain.NetworkPublic},
}

// matrixProjectMembers are the project memberships prepareMatrix seeds,
// each with its display settings: in acme's public and private projects,
// the project level's members (projectColumns); the removed member, still
// an active member of the public one, so that only his membership of acme
// keeps him out; the member before in the private one, ended; WG- in both,
// and the member in the private one, for partingStates to end or delete;
// the archived project's admin; each other workspace's admin in its
// project.
var matrixProjectMembers = []struct {
	key  string
	c    caller
	role shared.Role
}{
	{"acme/public", callerProjectAdmin, shared.RoleAdmin}, {"acme/public", callerProjectMember, shared.RoleMember},
	{"acme/public", callerGuest, shared.RoleGuest}, {"acme/public", callerMemberAndAdmin, shared.RoleMember},
	{"acme/public", callerRemoved, shared.RoleMember}, {"acme/public", callerGuestOnly, shared.RoleGuest},
	{"acme/private", callerProjectAdmin, shared.RoleAdmin}, {"acme/private", callerProjectMember, shared.RoleMember},
	{"acme/private", callerGuest, shared.RoleGuest}, {"acme/private", callerMemberAndAdmin, shared.RoleMember},
	{"acme/private", callerBefore, shared.RoleMember}, {"acme/private", callerGuestOnly, shared.RoleGuest},
	{"acme/private", callerMember, shared.RoleMember},
	{"acme/archived", callerProjectAdmin, shared.RoleAdmin},
	{"gone/project", callerDeleted, shared.RoleAdmin}, {"other/project", callerNever, shared.RoleAdmin},
}

// ownInvitation is the workspace of the invitation to c's own address: one
// he is not an active member of, so accepting it makes him one. acme's
// admin, member and guest are invited to other; the others to acme, where
// the removed member's ended membership is restored. An active member's
// address stays uninvited in acme, so that createWorkspaceInvitations'
// row of an active member's address is refused as that, not as an
// invited one.
func ownInvitation(c caller) string {
	switch c {
	case callerAdmin, callerMember, callerGuest:
		return "other"
	}
	return "acme"
}

// emailOf is the address of c's account.
func emailOf(c caller) string {
	return strings.ReplaceAll(string(c), " ", "-") + "@example.com"
}

// seeded are the ids of the rows prepareMatrix seeds that a request or a
// check can name: each workspace, by its slug; each membership, by the
// workspace's slug and the column; each invitation, by the workspace's
// slug and the address; each project, by its key; and each account, by
// its name in matrixAccounts, which prepareMatrix registers. t is the test
// that asks for them (in).
type seeded struct {
	t           testing.TB
	workspaces  map[string]uuid.UUID
	memberships map[string]uuid.UUID
	invitations map[string]uuid.UUID
	projects    map[string]uuid.UUID
	accounts    map[caller]uuid.UUID
}

// newSeeded names an id for each workspace of matrixMemberships, each of
// matrixMemberships, each of matrixInvitations and each of matrixProjects
// before prepareMatrix writes them, so that matrixViolations, without a
// database, sees the keys and the targets the cells will.
func newSeeded() seeded {
	s := seeded{workspaces: map[string]uuid.UUID{}, memberships: map[string]uuid.UUID{}, invitations: map[string]uuid.UUID{},
		projects: map[string]uuid.UUID{}, accounts: map[caller]uuid.UUID{}}
	for _, m := range matrixMemberships {
		if _, named := s.workspaces[m.slug]; !named {
			s.workspaces[m.slug] = uuid.NewV7()
		}
		s.memberships[m.slug+"/"+string(m.c)] = uuid.NewV7()
	}
	for _, i := range matrixInvitations {
		s.invitations[i.slug+"/"+i.email] = uuid.NewV7()
	}
	for _, p := range matrixProjects {
		s.projects[p.key] = uuid.NewV7()
	}
	return s
}

// in is s for the test t, which a key never seeded fails.
func (s seeded) in(t testing.TB) seeded {
	s.t = t
	return s
}

// workspace is the id of the workspace slug; one never seeded fails the
// test at once.
func (s seeded) workspace(slug string) uuid.UUID {
	id, ok := s.workspaces[slug]
	if !ok {
		s.t.Helper()
		s.t.Fatalf("no workspace %s is seeded", slug)
	}
	return id
}

// membership is the id of c's membership of the workspace slug. A key that
// was never seeded fails the test at once: its id would name no row, and an
// outsider's cell would get its 404 whatever the rule.
func (s seeded) membership(slug string, c caller) uuid.UUID {
	id, ok := s.memberships[slug+"/"+string(c)]
	if !ok {
		s.t.Helper()
		s.t.Fatalf("no membership of %s by %s is seeded", slug, c)
	}
	return id
}

// invitation is the id of the invitation of email to the workspace slug. A
// key that was never seeded fails the test at once, as membership's does.
func (s seeded) invitation(slug, email string) uuid.UUID {
	id, ok := s.invitations[slug+"/"+email]
	if !ok {
		s.t.Helper()
		s.t.Fatalf("no invitation of %s to %s is seeded", email, slug)
	}
	return id
}

// project is the id of the project key; one never seeded fails the test at
// once, as membership's does.
func (s seeded) project(key string) uuid.UUID {
	id, ok := s.projects[key]
	if !ok {
		s.t.Helper()
		s.t.Fatalf("no project %s is seeded", key)
	}
	return id
}

// account is the id of the account of matrixAccounts c, which prepareMatrix
// registers; uuid.Nil until then, when nothing is registered, so that a
// request built without a database names an account still (matrixViolations).
// A caller that is no registered account fails the test at once: a column
// that calls as another's account (PG's, as the workspace's guest) is
// named by that account (accountOf).
func (s seeded) account(c caller) uuid.UUID {
	if !slices.Contains(matrixAccounts, c) {
		s.t.Helper()
		s.t.Fatalf("no account %s is registered", c)
	}
	return s.accounts[c]
}

// fatalOf runs f on a goroutine of its own with a testing.TB whose Fatalf
// records the message and ends that goroutine, as testing.T's does, without
// failing the test; it returns the message, "" when f did not fail: how a
// test sees a key never seeded fail.
func fatalOf(f func(testing.TB)) string {
	p := &fatalProbe{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		f(p)
	}()
	<-done
	return p.message
}

type fatalProbe struct {
	testing.TB
	message string
}

func (*fatalProbe) Helper() {}

func (p *fatalProbe) Fatalf(format string, args ...any) {
	p.message = fmt.Sprintf(format, args...)
	runtime.Goexit()
}

// workspaceOfRow is the slug of the workspace the seeded row id is under,
// false for an id no seeded row has.
func (s seeded) workspaceOfRow(id uuid.UUID) (string, bool) {
	for _, rows := range []map[string]uuid.UUID{s.memberships, s.invitations} {
		for key, seededID := range rows {
			if seededID == id {
				slug, _, _ := strings.Cut(key, "/")
				return slug, true
			}
		}
	}
	return "", false
}

// matrixAdmins are the creators of the prepared workspaces, their admins.
var matrixAdmins = map[string]caller{"acme": callerAdmin, "gone": callerDeleted, "other": callerNever}
