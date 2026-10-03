package bootstrap

import (
	"context"
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
	workspacedomain "github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The writers of prepareMatrix's rows (permission_matrix_seeded_test.go):
// the workspace store's and the project store's, the SQL that makes the
// states no store makes alone, and the checks that the rows the cells rest
// on are there.

// matrixSeed writes the prepared workspaces, memberships and settings
// through the workspace store, and keeps the workspaces' ids by slug; exec
// runs the SQL that makes the states no store makes alone (partingStates).
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
	if _, err := s.store.CreateInvitations(context.Background(), []workspaceapp.InvitationRow{
		{ID: id, WorkspaceID: s.workspaces[slug], Email: email, Role: role, CreatedBy: s.ids[matrixAdmins[slug]], Now: s.now},
	}); err != nil {
		s.t.Fatal(err)
	}
}

// exec runs sql for the test tb, which sql must change one row of: a
// statement that matched none would leave the seed as it was, and the
// cells that need the change would test another case.
func (s matrixSeed) exec(tb testing.TB, pool *pgxpool.Pool, sql string, args ...any) {
	tb.Helper()
	tag, err := pool.Exec(context.Background(), sql, args...)
	if err != nil {
		tb.Fatal(err)
	}
	if tag.RowsAffected() != 1 {
		tb.Fatalf("%s changed %d rows, want 1", sql, tag.RowsAffected())
	}
}

// projectSeed writes the prepared projects and their memberships through
// the project store, each by its workspace's admin, and keeps the
// projects' ids by key.
type projectSeed struct {
	matrixSeed
	store    *projectpg.Store
	projects map[string]uuid.UUID
}

// project stores the project id with key.
func (s projectSeed) project(id uuid.UUID, key, name, identifier string, network projectdomain.Network) {
	s.t.Helper()
	slug, _, _ := strings.Cut(key, "/")
	if err := s.store.CreateProject(context.Background(), projectapp.ProjectRow{
		ID: id, WorkspaceID: s.workspaces[slug], Name: name, Identifier: identifier, Network: network, Timezone: "UTC",
		CreatedBy: s.ids[matrixAdmins[slug]], Now: s.now,
	}); err != nil {
		s.t.Fatal(err)
	}
	s.projects[key] = id
}

// join makes c a member of the project key with role, the membership id,
// and stores his display settings in it.
func (s projectSeed) join(id uuid.UUID, key string, c caller, role shared.Role) {
	s.t.Helper()
	slug, _, _ := strings.Cut(key, "/")
	ctx, by := context.Background(), s.ids[matrixAdmins[slug]]
	if err := s.store.CreateMember(ctx, projectapp.MemberRow{
		ID: id, WorkspaceID: s.workspaces[slug], ProjectID: s.projects[key], MemberID: s.ids[c], Role: role, CreatedBy: by, Now: s.now,
	}); err != nil {
		s.t.Fatal(err)
	}
	if err := s.store.CreatePreferences(ctx, projectapp.PreferencesRow{
		ID: uuid.NewV7(), WorkspaceID: s.workspaces[slug], ProjectID: s.projects[key], UserID: s.ids[c], SortOrder: 65535, CreatedBy: by,
		Now: s.now,
	}); err != nil {
		s.t.Fatal(err)
	}
}

// archive archives the project key, by its workspace's admin.
func (s projectSeed) archive(key string) {
	s.t.Helper()
	slug, _, _ := strings.Cut(key, "/")
	if err := s.store.SetArchived(context.Background(), s.projects[key], true, s.ids[matrixAdmins[slug]], s.now); err != nil {
		s.t.Fatal(err)
	}
}

// endings ends, through the project store, the project memberships that
// P5b's writes end, as their statement ends one (EndMember), at the seed's
// moment: the member before's of the private project, removed by acme's
// admin (removeProjectMember), and WG-'s of the public one, which he left
// (leaveProject).
func (s projectSeed) endings() {
	s.t.Helper()
	for _, e := range []struct {
		key   string
		c, by caller
	}{{"acme/private", callerBefore, matrixAdmins["acme"]}, {"acme/public", callerGuestOnly, callerGuestOnly}} {
		if err := s.store.EndMember(context.Background(), s.projects[e.key], s.ids[e.c], s.ids[e.by], s.now); err != nil {
			s.t.Fatal(err)
		}
	}
}

// partingStates puts memberships of matrixProjectMembers in the states in
// which a list and reading could part (TestListingProjectsIsReadingEach),
// beside WG-'s membership of acme's public project, which he left
// (endings): his membership of its private one deleted, the member's
// deleted too, and PM's display settings in it deleted while his membership
// stays active. SQL makes these, which no store makes alone: the two
// deleted states only a deleted project or workspace makes, which the list
// must still read as reading does, and display settings gone from a live
// membership; exec fails a statement that changes no row, and names it,
// which it checks first. Each state is then read back, the ended one too:
// one missing would let a list that counts an ended or a deleted
// membership, or takes display settings for a membership, agree with
// reading for every account.
func (s projectSeed) partingStates(pool *pgxpool.Pool) {
	s.t.Helper()
	const none = "UPDATE project_members SET is_active = false WHERE false"
	if failed, want := fatalOf(func(tb testing.TB) { s.exec(tb, pool, none) }), none+" changed 0 rows, want 1"; failed != want {
		s.t.Errorf("exec of a statement that changes no row: failed with %q, want %q", failed, want)
	}
	public, private := s.projects["acme/public"], s.projects["acme/private"]
	for _, c := range []caller{callerGuestOnly, callerMember} {
		s.exec(s.t, pool, "UPDATE project_members SET deleted_at = $3 WHERE project_id = $1 AND member_id = $2", private, s.ids[c], s.now)
	}
	s.exec(s.t, pool, "UPDATE project_user_properties SET deleted_at = $3 WHERE project_id = $1 AND user_id = $2", private,
		s.ids[callerProjectMember], s.now)
	// A deleted membership with a live one beside it would be no deleted
	// state at all: the live one is what the list and reading would see. The
	// row stays active, as cascade.sql leaves it, so only its deleted_at
	// keeps it out.
	deleted := "m.is_active AND m.deleted_at IS NOT NULL AND NOT EXISTS (SELECT 1 FROM project_members o " +
		"WHERE o.project_id = m.project_id AND o.member_id = m.member_id AND o.deleted_at IS NULL)"
	for _, st := range []struct {
		project uuid.UUID
		c       caller
		holds   string // of m, his membership of the project
	}{
		{public, callerGuestOnly, "NOT m.is_active AND m.deleted_at IS NULL"},
		{private, callerGuestOnly, deleted},
		{private, callerMember, deleted},
		{private, callerProjectMember, "m.is_active AND m.deleted_at IS NULL AND NOT EXISTS (SELECT 1 FROM project_user_properties u " +
			"WHERE u.project_id = m.project_id AND u.user_id = m.member_id AND u.deleted_at IS NULL)"},
	} {
		var holds bool
		if err := pool.QueryRow(context.Background(), "SELECT EXISTS (SELECT 1 FROM project_members m WHERE m.project_id = $1 AND "+
			"m.member_id = $2 AND "+st.holds+")", st.project, s.ids[st.c]).Scan(&holds); err != nil || !holds {
			s.t.Fatalf("%s's membership of %s: %v, %v; want %s", st.c, st.project, holds, err, st.holds)
		}
	}
}

// removal ends the removed member's membership of acme, then his active
// memberships of acme's projects, found when they are ended, through the
// stores, as removeWorkspaceMember's statements end them (M3 design 3.6's
// lock table, convention 6): by acme's admin, at one moment. Its other
// statement, the pending invitations', is left out: prepareMatrix makes
// the invitation to his address after it, as one sent once he was removed
// (3.8).
func (s projectSeed) removal(sd seeded) {
	s.t.Helper()
	ctx, acme, removed, by := context.Background(), sd.workspace("acme"), s.ids[callerRemoved], s.ids[matrixAdmins["acme"]]
	if err := s.matrixSeed.store.EndMember(ctx, acme, removed, by, s.now); err != nil {
		s.t.Fatal(err)
	}
	projects, err := s.store.LockActiveMemberProjects(ctx, []uuid.UUID{acme}, removed)
	if err != nil {
		s.t.Fatal(err)
	}
	if err := s.store.EndMemberships(ctx, projects, removed, by, s.now); err != nil {
		s.t.Fatal(err)
	}
}

// preconditions checks the seeded rows that a cell's answer rests on and
// that no answer shows: one missing would let a mutant pass the cells.
func (s projectSeed) preconditions(sd seeded) {
	s.t.Helper()
	ctx := context.Background()
	// The removed member's membership of the project his column aims at,
	// public, is ended, as the removal left it, not deleted: his cell's 404
	// is a removed member's, which a member of acme would not get there.
	public, removed := s.projects[projectOf(callerRemoved)], s.ids[callerRemoved]
	if f, found, err := s.store.ProjectFacts(ctx, public, removed); err != nil || !found || f.Member {
		s.t.Fatalf("the removed member's facts of %s = %+v, %v, %v; want him no active member of it", projectOf(callerRemoved), f, found, err)
	}
	if ms, err := s.store.Memberships(ctx, public, []uuid.UUID{removed}); err != nil || len(ms) != 1 || ms[removed].Active {
		s.t.Fatalf("the removed member's memberships of %s = %+v, %v; want his ended one", projectOf(callerRemoved), ms, err)
	}
	// other's admin is its only active admin, beside its active member, so
	// that his leaving's 409 is the rule's (3.7 rule 1); acme's admin has
	// another, PM+WA, so that his leaving's 204 is no other case.
	for _, tt := range []struct {
		slug  string
		c     caller
		other bool
	}{{"other", callerNever, false}, {"acme", callerAdmin, true}} {
		role, active, err := s.matrixSeed.store.ActiveRole(ctx, sd.workspace(tt.slug), s.ids[tt.c])
		if err != nil || !active || role != shared.RoleAdmin {
			s.t.Fatalf("%s's role in %s = %d, %v, %v; want its active admin", tt.c, tt.slug, role, active, err)
		}
		if other, err := s.matrixSeed.store.HasOtherAdmin(ctx, sd.workspace(tt.slug), s.ids[tt.c]); err != nil || other != tt.other {
			s.t.Fatalf("another admin of %s than %s: %v, %v; want %v", tt.slug, tt.c, other, err, tt.other)
		}
	}
	if role, active, err := s.matrixSeed.store.ActiveRole(ctx, sd.workspace("other"), s.ids[callerRemoved]); err != nil || !active ||
		role != shared.RoleMember {
		s.t.Fatalf("the removed member's role in other = %d, %v, %v; want its active member", role, active, err)
	}
	// other's project is the one no list of acme's may show: were it not
	// there, undeleted in a workspace of its own, a list of every
	// workspace's projects would pass the matrix and
	// TestListingProjectsIsReadingEach alike.
	if f, found, err := s.store.ProjectFacts(ctx, s.projects["other/project"], s.ids[callerNever]); err != nil || !found ||
		f.WorkspaceID != sd.workspace("other") {
		s.t.Fatalf("other's project's facts = %+v, %v, %v; want it undeleted in other (%s), not acme (%s)", f, found, err,
			sd.workspace("other"), sd.workspace("acme"))
	}
	s.targets(sd)
	s.memberships(sd)
}

// targets checks the accounts addProjectMembers' rows add
// (permission_matrix_members_test.go): the workspace's member an active
// member of acme and of neither project his row adds him to, so that its
// 201 is an addition; X's account and the removed member no active members
// of acme (the removed member's membership ended: removal), WG-'s its
// active guest, WA-'s its active admin and PM's an active member of acme's
// public project, so that each 422 is the refusal its row names. The
// workspace's member, WM-公, and its admin, WA-, are no active members of
// acme's public project either, so that joinProject's row makes each a
// member with his workspace role, and the add of WA- as a member is refused
// for his role, not as a duplicate. Nor has either an ended membership of
// a project his rows add him to or he joins: the store's Memberships of him
// there are none, so that the add's 201 and the join's 200 make a new
// membership and restore no ended one, whose role 9.1's table would decide,
// and which, ended as an admin's, would answer as a new one.
func (s projectSeed) targets(sd seeded) {
	s.t.Helper()
	ctx := context.Background()
	for _, tt := range []struct {
		c      caller
		role   shared.Role
		active bool
	}{{callerMember, shared.RoleMember, true}, {callerNever, 0, false}, {callerRemoved, 0, false}, {callerGuestOnly, shared.RoleGuest, true},
		{callerAdmin, shared.RoleAdmin, true}} {
		if role, active, err := s.matrixSeed.store.ActiveRole(ctx, sd.workspace("acme"), s.ids[tt.c]); err != nil || active != tt.active ||
			active && role != tt.role {
			s.t.Fatalf("%s's role in acme = %d, %v, %v; want %d, %v", tt.c, role, active, err, tt.role, tt.active)
		}
	}
	for _, tt := range []struct {
		key string
		c   caller
	}{{"acme/public", callerMember}, {"acme/archived", callerMember}, {"acme/public", callerAdmin}} {
		if f, found, err := s.store.ProjectFacts(ctx, s.projects[tt.key], s.ids[tt.c]); err != nil || !found || f.Member {
			s.t.Fatalf("%s's facts of %s = %+v, %v, %v; want him no active member of it", tt.c, tt.key, f, found, err)
		}
		if ms, err := s.store.Memberships(ctx, s.projects[tt.key], []uuid.UUID{s.ids[tt.c]}); err != nil || len(ms) != 0 {
			s.t.Fatalf("%s's memberships of %s = %+v, %v; want none, ended or active", tt.c, tt.key, ms, err)
		}
	}
	if f, found, err := s.store.ProjectFacts(ctx, s.projects["acme/public"], s.ids[callerProjectMember]); err != nil || !found || !f.Member {
		s.t.Fatalf("%s's facts of acme/public = %+v, %v, %v; want him its active member", callerProjectMember, f, found, err)
	}
}
