package postgresadapter_test

import (
	"context"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListProjects answers each project as GetProject reads it (M3 design
// 3.12, 3.19): the members, the caller's role and his place in his
// sidebar. On webFixture, every account lists ops and web, each the row
// GetProject answers him.
func TestListProjectsAnswersEachAsGetProjectDoes(t *testing.T) {
	f := newWebFixture(t)
	ctx := context.Background()
	for _, user := range []uuid.UUID{f.alice, f.bob, f.carol, f.dave, f.erin, f.frank, f.gina, f.hank} {
		list, err := f.s.ListProjects(ctx, f.acme, user, domain.Visibility{All: true, Public: true}, false)
		if err != nil || len(list) != 2 {
			t.Errorf("ListProjects(%s) = %d projects, %v; want ops and web", user, len(list), err)
			continue
		}
		for _, p := range list {
			want, found, err := f.s.GetProject(ctx, p.ID, user)
			if err != nil || !found || jsonOf(t, p) != jsonOf(t, want) {
				t.Errorf("ListProjects(%s) answers %s; GetProject answers %s, %v, %v", user, jsonOf(t, p), jsonOf(t, want), found, err)
			}
		}
	}
}

// ListProjects: of the workspace's undeleted projects, the archived ones
// or the others, those the visibility lets the user see besides those he
// is an active member of, each as he sees it; by his place in his sidebar,
// the projects he is not a member of last, then by name. The fixture's
// projects are stored in another order than the answer's.
func TestListProjects(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	private := func(id uuid.UUID) { exec(t, pool, "UPDATE projects SET network = 0 WHERE id = $1", id) }
	join := func(project, user uuid.UUID, sortOrder float64) {
		t.Helper()
		if err := s.CreateMember(ctx, app.MemberRow{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: project, MemberID: user,
			Role: shared.RoleMember, CreatedBy: alice, Now: now}); err != nil {
			t.Fatal(err)
		}
		if err := s.CreatePreferences(ctx, app.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: project, UserID: user,
			SortOrder: sortOrder, CreatedBy: alice, Now: now}); err != nil {
			t.Fatal(err)
		}
	}
	// Alice is a member of mid (place -5) and zeta (10), both private, of
	// kilo (10), public, and of bare, private, whose display settings were
	// deleted: hers, without a place. She was one of ended, whose
	// membership ended, and of left, whose membership was deleted; apple
	// and pear are public, secret private; archived, deleted and beta's are
	// not in this list. A name is unique in a workspace, so the id decides
	// no order here; zeta's identifier comes before kilo's, so that the
	// name, not the identifier, decides their tie.
	zeta := newProject(t, s, acme, "Zeta", "AZ", alice)
	private(zeta)
	join(zeta, alice, 10)
	pear := newProject(t, s, acme, "Pear", "PEAR", alice)
	mid := newProject(t, s, acme, "Mid", "MID", alice)
	private(mid)
	join(mid, alice, -5)
	secret := newProject(t, s, acme, "Secret", "SEC", alice)
	private(secret)
	ended := newProject(t, s, acme, "Ended", "END", alice)
	private(ended)
	join(ended, alice, 1)
	exec(t, pool, "UPDATE project_members SET is_active = false WHERE project_id = $1 AND member_id = $2", ended, alice)
	kilo := newProject(t, s, acme, "Kilo", "KILO", alice)
	join(kilo, alice, 10)
	apple := newProject(t, s, acme, "Apple", "APPLE", alice)
	bare := newProject(t, s, acme, "Bare", "BARE", alice)
	private(bare)
	join(bare, alice, 2)
	exec(t, pool, "UPDATE project_user_properties SET deleted_at = now() WHERE project_id = $1 AND user_id = $2", bare, alice)
	left := newProject(t, s, acme, "Left", "LEFT", alice)
	private(left)
	join(left, alice, 3)
	exec(t, pool, "UPDATE project_members SET deleted_at = now() WHERE project_id = $1 AND member_id = $2", left, alice)
	archived := newProject(t, s, acme, "Old", "OLD", alice)
	exec(t, pool, "UPDATE projects SET archived_at = now() WHERE id = $1", archived)
	deleted := newProject(t, s, acme, "Gone", "GONE", alice)
	exec(t, pool, "UPDATE projects SET deleted_at = now() WHERE id = $1", deleted)
	newProject(t, s, beta, "Beta", "BETA", alice)

	tests := []struct {
		name     string
		user     uuid.UUID
		v        domain.Visibility
		archived bool
		want     []uuid.UUID
	}{
		{"everything", alice, domain.Visibility{All: true, Public: true}, false,
			[]uuid.UUID{mid, kilo, zeta, apple, bare, ended, left, pear, secret}},
		{"the public ones", alice, domain.Visibility{Public: true}, false, []uuid.UUID{mid, kilo, zeta, apple, bare, pear}},
		{"hers alone", alice, domain.Visibility{}, false, []uuid.UUID{mid, kilo, zeta, bare}},
		{"another's", bob, domain.Visibility{Public: true}, false, []uuid.UUID{apple, kilo, pear}},
		{"the archived ones", alice, domain.Visibility{All: true, Public: true}, true, []uuid.UUID{archived}},
	}
	for _, tt := range tests {
		list, err := s.ListProjects(ctx, acme, tt.user, tt.v, tt.archived)
		var got []uuid.UUID
		for _, p := range list {
			got = append(got, p.ID)
		}
		if err != nil || !slices.Equal(got, tt.want) {
			t.Errorf("%s: ListProjects() = %v, %v; want %v", tt.name, got, err, tt.want)
		}
	}
	// Each as the user sees it: his role and place in his own, none in the
	// others; the members.
	list, err := s.ListProjects(ctx, acme, alice, domain.Visibility{Public: true}, false)
	if err != nil || len(list) != 6 || list[0].MemberRole == nil || *list[0].MemberRole != shared.RoleMember || list[0].SortOrder == nil ||
		*list[0].SortOrder != -5 || !slices.Equal(list[0].MemberIDs, []uuid.UUID{alice}) || list[3].MemberRole != nil || list[3].SortOrder != nil {
		t.Errorf("ListProjects() = %+v, %v; want mid with her role and place, apple without", list, err)
	}
}
