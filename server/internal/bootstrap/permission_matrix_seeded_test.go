package bootstrap

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	workspacedomain "github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The rows prepareMatrix seeds, and how a row's request names them.

// matrixMemberships are the memberships prepareMatrix seeds, workspace by
// workspace, each workspace's creator first: acme with its admin, member,
// guest and the member later removed; gone with its admin and the member;
// other, whose admin was never a member of acme and where the removed
// member is still active.
var matrixMemberships = []struct {
	slug string
	c    caller
	role shared.Role
}{
	{"acme", callerAdmin, shared.RoleAdmin}, {"acme", callerMember, shared.RoleMember}, {"acme", callerGuest, shared.RoleGuest},
	{"acme", callerRemoved, shared.RoleMember},
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
// workspace's slug and the column; and each invitation, by the workspace's
// slug and the address. t is the test that asks for them (in).
type seeded struct {
	t           testing.TB
	workspaces  map[string]uuid.UUID
	memberships map[string]uuid.UUID
	invitations map[string]uuid.UUID
}

// newSeeded names an id for each workspace of matrixMemberships, each of
// matrixMemberships and each of matrixInvitations before prepareMatrix
// writes them, so that matrixViolations, without a database, sees the keys
// and the workspaces the cells will.
func newSeeded() seeded {
	s := seeded{workspaces: map[string]uuid.UUID{}, memberships: map[string]uuid.UUID{}, invitations: map[string]uuid.UUID{}}
	for _, m := range matrixMemberships {
		if _, named := s.workspaces[m.slug]; !named {
			s.workspaces[m.slug] = uuid.NewV7()
		}
		s.memberships[m.slug+"/"+string(m.c)] = uuid.NewV7()
	}
	for _, i := range matrixInvitations {
		s.invitations[i.slug+"/"+i.email] = uuid.NewV7()
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

// matrixSeed writes the prepared workspaces, memberships and settings
// through the workspace store, and keeps the workspaces' ids by slug; exec
// runs the SQL that stands in for the store P5 adds.
type matrixSeed struct {
	t          *testing.T
	store      *workspacepg.Store
	ids        map[caller]uuid.UUID
	now        time.Time
	workspaces map[string]uuid.UUID // by slug
}

func (s matrixSeed) workspace(id uuid.UUID, slug string, admin caller) {
	s.t.Helper()
	w, err := s.store.CreateWorkspace(context.Background(), workspaceapp.WorkspaceRow{
		ID: id, Name: slug, Slug: slug, Timezone: "UTC", CreatedBy: s.ids[admin], Now: s.now,
	})
	if err != nil {
		s.t.Fatal(err)
	}
	s.workspaces[slug] = w.ID
}

func (s matrixSeed) join(id uuid.UUID, slug string, c caller, role shared.Role) {
	s.t.Helper()
	if err := s.store.CreateMember(context.Background(), workspaceapp.MemberRow{
		ID: id, WorkspaceID: s.workspaces[slug], MemberID: s.ids[c], Role: role, CreatedBy: s.ids[c], Now: s.now,
	}); err != nil {
		s.t.Fatal(err)
	}
}

func (s matrixSeed) preferences(slug string, c caller, p workspacedomain.PreferencesPatch) {
	s.t.Helper()
	if _, err := s.store.UpsertPreferences(context.Background(), workspaceapp.PreferencesRow{
		ID: uuid.NewV7(), WorkspaceID: s.workspaces[slug], UserID: s.ids[c], Patch: p, Now: s.now,
	}); err != nil {
		s.t.Fatal(err)
	}
}

// invite stores the invitation id of email to the workspace slug, by its
// admin.
func (s matrixSeed) invite(id uuid.UUID, slug, email string, role shared.Role) {
	s.t.Helper()
	admin := map[string]caller{"acme": callerAdmin, "gone": callerDeleted, "other": callerNever}[slug]
	if _, err := s.store.CreateInvitations(context.Background(), []workspaceapp.InvitationRow{
		{ID: id, WorkspaceID: s.workspaces[slug], Email: email, Role: role, CreatedBy: s.ids[admin], Now: s.now},
	}); err != nil {
		s.t.Fatal(err)
	}
}

func (s matrixSeed) exec(pool *pgxpool.Pool, sql string, args ...any) {
	s.t.Helper()
	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		s.t.Fatal(err)
	}
}
