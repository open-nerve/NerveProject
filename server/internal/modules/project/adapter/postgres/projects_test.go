package postgresadapter_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func ptr[T any](v T) *T { return &v }

// jsonOf is v as JSON, to compare values that hold pointers.
func jsonOf(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// CreateProject stores the use case's values, the other columns at their
// defaults, and the audit columns at the clock's time, by the creator; it
// leaves every other project as it was. GetProject reads it back as
// stored, in UTC, to the microsecond.
func TestCreateProjectStoresTheRow(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	newProject(t, s, acme, "Ops", "OPS", bob)
	newProject(t, s, beta, "Web", "WEB", bob)
	row := app.ProjectRow{
		ID: uuid.NewV7(), WorkspaceID: acme, Name: "研发 Web", Description: "The web app", Identifier: "WEBÇ", Network: domain.NetworkPrivate,
		LeadID: &bob, LogoProps: domain.LogoProps{InUse: ptr("emoji"), Emoji: &domain.Emoji{Value: ptr("128640")}}, Timezone: "Asia/Shanghai",
		CreatedBy: alice, Now: now,
	}
	others := tableRows(t, pool, "projects", row.ID)

	if err := s.CreateProject(context.Background(), row); err != nil {
		t.Fatal(err)
	}

	got, found, err := s.GetProject(context.Background(), row.ID, alice)
	want := domain.Project{
		ID: row.ID, WorkspaceID: acme, Name: "研发 Web", Description: "The web app", Identifier: "WEBÇ", Network: domain.NetworkPrivate,
		LeadID: &bob, LogoProps: row.LogoProps, Timezone: "Asia/Shanghai", CreatedAt: now, UpdatedAt: now, MemberIDs: []uuid.UUID{},
	}
	if g, w := jsonOf(t, got), jsonOf(t, want); err != nil || !found || g != w || got.CreatedAt.Location() != time.UTC {
		t.Errorf("GetProject() = %s, %v, %v; want\n%s", g, found, err, w)
	}
	var createdBy, updatedBy uuid.UUID
	var sequence int
	var deleted *time.Time
	if err := pool.QueryRow(context.Background(),
		"SELECT created_by_id, updated_by_id, last_issue_sequence, deleted_at FROM projects WHERE id = $1", row.ID).
		Scan(&createdBy, &updatedBy, &sequence, &deleted); err != nil {
		t.Fatal(err)
	}
	if createdBy != alice || updatedBy != alice || sequence != 0 || deleted != nil {
		t.Errorf("row: created_by %s updated_by %s, last_issue_sequence %d, deleted %v; want alice, 0, not deleted", createdBy, updatedBy, sequence, deleted)
	}
	if after := tableRows(t, pool, "projects", row.ID); after != others {
		t.Errorf("the other projects:\n%s\nwant\n%s", after, others)
	}
}

// The four values of logo_props the CHECK accepts (M3 design 4.6), and the
// web app's create body, come back as they were stored: no icon, an emoji
// only, an icon only, every key, and an emoji in use
// (core/components/projects/create/utils.ts:14-19).
func TestCreateProjectKeepsTheLogo(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, pool, "acme")
	for i, logo := range []domain.LogoProps{
		{},
		{Emoji: &domain.Emoji{Value: ptr("128640")}},
		{Icon: &domain.Icon{Name: ptr("home"), Color: ptr("#6d7b8a")}},
		{InUse: ptr("icon"), Emoji: &domain.Emoji{Value: ptr("128640"), URL: ptr("https://example.com/e.png")},
			Icon: &domain.Icon{Name: ptr("home"), Color: ptr("#6d7b8a"), BackgroundColor: ptr("#ffffff")}},
		{InUse: ptr("emoji"), Emoji: &domain.Emoji{Value: ptr("128640")}},
	} {
		id, name := uuid.NewV7(), "P"+string(rune('A'+i))
		err := s.CreateProject(context.Background(), app.ProjectRow{
			ID: id, WorkspaceID: acme, Name: name, Identifier: name, LogoProps: logo, Timezone: "UTC", CreatedBy: alice, Now: now,
		})
		if err != nil {
			t.Fatal(err)
		}
		got, _, err := s.GetProject(context.Background(), id, alice)
		if g, w := jsonOf(t, got.LogoProps), jsonOf(t, logo); err != nil || g != w {
			t.Errorf("the logo %s came back as %s, %v", w, g, err)
		}
	}
}

// An identifier and a name are unique among the workspace's undeleted
// projects only: another workspace's project, and a deleted one, leave
// them free (M3 design 3.19).
func TestCreateProjectIdentifierOrNameTaken(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	first := newProject(t, s, acme, "Web", "WEB", alice)
	create := func(workspace uuid.UUID, name, identifier string) error {
		return s.CreateProject(context.Background(), app.ProjectRow{
			ID: uuid.NewV7(), WorkspaceID: workspace, Name: name, Identifier: identifier, Timezone: "UTC", CreatedBy: alice, Now: now,
		})
	}

	if err := create(acme, "Web 2", "WEB"); !errors.Is(err, domain.ErrIdentifierTaken) {
		t.Errorf("a taken identifier: %v, want project.identifier_taken", err)
	}
	if err := create(acme, "Web", "WEB2"); !errors.Is(err, domain.ErrNameTaken) {
		t.Errorf("a taken name: %v, want project.name_taken", err)
	}
	if err := create(beta, "Web", "WEB"); err != nil {
		t.Errorf("another workspace's name and identifier: %v, want them free", err)
	}
	exec(t, pool, "UPDATE projects SET deleted_at = $2 WHERE id = $1", first, now)
	if err := create(acme, "Web", "WEB"); err != nil {
		t.Errorf("a deleted project's name and identifier: %v, want them free", err)
	}
}

// IdentifierTaken: the workspace's undeleted projects' identifiers,
// archived or not, as stored, in upper case; another workspace's and a
// deleted project's do not count.
func TestIdentifierTaken(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	alice := newAccount(t, pool, "alice@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	newProject(t, s, acme, "Web", "WEB", alice)
	old := newProject(t, s, acme, "Old", "OLD", alice)
	exec(t, pool, "UPDATE projects SET archived_at = now() WHERE id = $1", old)
	gone := newProject(t, s, acme, "Gone", "GONE", alice)
	exec(t, pool, "UPDATE projects SET deleted_at = now() WHERE id = $1", gone)
	newProject(t, s, beta, "Ops", "OPS", alice)
	for _, tt := range []struct {
		identifier string
		want       bool
	}{{"WEB", true}, {"OLD", true}, {"web", false}, {"GONE", false}, {"OPS", false}, {"NEW", false}} {
		if got, err := s.IdentifierTaken(ctx, acme, tt.identifier); err != nil || got != tt.want {
			t.Errorf("IdentifierTaken(acme, %s) = %v, %v; want %v", tt.identifier, got, err, tt.want)
		}
	}
}

// webFixture is the workspace acme with its projects ops and web, which
// has each state of a membership once, next to the rows another predicate
// would let in: another project's rows of the same accounts, an inactive
// and a deleted membership, a deleted display settings row; and rows that
// lie in the table in another order than the answer's.
type webFixture struct {
	s                                                *postgresadapter.Store
	pool                                             *pgxpool.Pool
	acme, ops, web                                   uuid.UUID
	alice, bob, carol, dave, erin, frank, gina, hank uuid.UUID
}

// newWebFixture stores webFixture's rows, alice's. ops first: its rows come
// first in the tables, so a join that loses a predicate reads them before
// web's; everyone but erin is its admin. In web: carol's membership is
// stored first but began last; gina's began with dave's and has the
// smaller id, stored after his; hank's began with theirs too, has the
// largest id of the three and is stored last of them, so that neither the
// table's order of the three nor its reverse is the answer's; alice's
// began between theirs and carol's. bob's ended, frank's was deleted;
// carol's display settings were deleted; erin was never a member.
func newWebFixture(t *testing.T) webFixture {
	t.Helper()
	s, pool := newStore(t)
	var ids []uuid.UUID
	for _, email := range []string{"alice@corp.com", "bob@corp.com", "carol@corp.com", "dave@corp.com", "erin@corp.com", "frank@corp.com",
		"gina@corp.com", "hank@corp.com"} {
		ids = append(ids, newAccount(t, pool, email))
	}
	f := webFixture{s: s, pool: pool, alice: ids[0], bob: ids[1], carol: ids[2], dave: ids[3], erin: ids[4], frank: ids[5], gina: ids[6],
		hank: ids[7]}
	f.acme = newWorkspace(t, pool, "acme")
	f.ops = newProject(t, s, f.acme, "Ops", "OPS", f.alice)
	f.web = newProject(t, s, f.acme, "Web", "WEB", f.alice)
	member := func(id, project, user uuid.UUID, role shared.Role, at time.Time, sortOrder float64) {
		t.Helper()
		ctx := context.Background()
		if err := s.CreateMember(ctx, app.MemberRow{ID: id, WorkspaceID: f.acme, ProjectID: project, MemberID: user, Role: role,
			CreatedBy: f.alice, Now: at}); err != nil {
			t.Fatal(err)
		}
		if err := s.CreatePreferences(ctx, app.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: f.acme, ProjectID: project, UserID: user,
			SortOrder: sortOrder, CreatedBy: f.alice, Now: at}); err != nil {
			t.Fatal(err)
		}
	}
	for _, u := range []uuid.UUID{f.alice, f.bob, f.carol, f.dave, f.frank, f.gina, f.hank} {
		member(uuid.NewV7(), f.ops, u, shared.RoleAdmin, now, -1)
	}
	ginasID := uuid.NewV7()
	member(uuid.NewV7(), f.web, f.carol, shared.RoleGuest, now.Add(2*time.Minute), 30)
	member(uuid.NewV7(), f.web, f.dave, shared.RoleMember, now, 40)
	member(ginasID, f.web, f.gina, shared.RoleMember, now, 60)
	member(uuid.NewV7(), f.web, f.hank, shared.RoleMember, now, 70)
	member(uuid.NewV7(), f.web, f.alice, shared.RoleAdmin, now.Add(time.Minute), 10)
	member(uuid.NewV7(), f.web, f.bob, shared.RoleMember, now, 20)
	member(uuid.NewV7(), f.web, f.frank, shared.RoleMember, now, 50)
	exec(t, pool, "UPDATE project_members SET is_active = false WHERE project_id = $1 AND member_id = $2", f.web, f.bob)
	exec(t, pool, "UPDATE project_members SET deleted_at = $3 WHERE project_id = $1 AND member_id = $2", f.web, f.frank, now)
	exec(t, pool, "UPDATE project_user_properties SET deleted_at = $3 WHERE project_id = $1 AND user_id = $2", f.web, f.carol, now)
	return f
}

// GetProject answers the caller's view (M3 design 3.19): his role and his
// place in his sidebar only while his membership is active, and the active
// members in the order they became members, then by the membership's id;
// on webFixture's web.
func TestGetProject(t *testing.T) {
	f := newWebFixture(t)
	s, pool, web := f.s, f.pool, f.web
	alice, bob, carol, dave, erin, frank := f.alice, f.bob, f.carol, f.dave, f.erin, f.frank
	members := []uuid.UUID{f.gina, dave, f.hank, alice, carol}
	tests := []struct {
		name      string
		user      uuid.UUID
		role      *shared.Role
		sortOrder *float64
	}{
		{"alice, admin", alice, ptr(shared.RoleAdmin), ptr(10.0)},
		{"dave, member", dave, ptr(shared.RoleMember), ptr(40.0)},
		{"carol, guest without display settings", carol, ptr(shared.RoleGuest), nil},
		{"bob, membership ended", bob, nil, nil},
		{"frank, membership deleted", frank, nil, nil},
		{"erin, never a member", erin, nil, nil},
	}
	for _, tt := range tests {
		got, found, err := s.GetProject(context.Background(), web, tt.user)
		if err != nil || !found || got.ID != web || got.Name != "Web" || jsonOf(t, got.MemberRole) != jsonOf(t, tt.role) ||
			jsonOf(t, got.SortOrder) != jsonOf(t, tt.sortOrder) || !slices.Equal(got.MemberIDs, members) {
			t.Errorf("%s: GetProject() = %+v, %v, %v; want role %v, sort order %v, members %v", tt.name, got, found, err,
				jsonOf(t, tt.role), jsonOf(t, tt.sortOrder), members)
		}
	}
	// Archived: found as it is; deleted, or no project: not found.
	exec(t, pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", web, now)
	if got, found, err := s.GetProject(context.Background(), web, alice); err != nil || !found || got.ArchivedAt == nil || !got.ArchivedAt.Equal(now) {
		t.Errorf("the archived project: %+v, %v, %v; want it, archived at %v", got, found, err, now)
	}
	exec(t, pool, "UPDATE projects SET deleted_at = $2 WHERE id = $1", web, now)
	for _, id := range []uuid.UUID{web, uuid.NewV7()} {
		if got, found, err := s.GetProject(context.Background(), id, alice); err != nil || found {
			t.Errorf("GetProject(%s) = %+v, %v, %v; want not found", id, got, found, err)
		}
	}
}

// GetProject reads each column as stored, the ones a new project has at
// their defaults too: five projects, each with another of the five views
// on, so that a view read as another, or not at all, differs in one; each
// with bob as its default assignee, and an archive_in and an updated_at of
// its own.
func TestGetProjectReadsEveryColumn(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme := newWorkspace(t, pool, "acme")
	views := []string{"cycle_view", "module_view", "issue_views_view", "intake_view", "guest_view_all_features"}
	for i, view := range views {
		name := "P" + string(rune('A'+i))
		id, updated := newProject(t, s, acme, name, name, alice), now.Add(time.Duration(i+1)*time.Hour)
		exec(t, pool, "UPDATE projects SET "+view+" = true, default_assignee_id = $2, archive_in = $3, updated_at = $4 WHERE id = $1",
			id, bob, i+1, updated)

		got, found, err := s.GetProject(context.Background(), id, alice)
		on, want := []bool{got.CycleView, got.ModuleView, got.IssueViewsView, got.IntakeView, got.GuestViewAllFeatures}, make([]bool, len(views))
		want[i] = true
		if err != nil || !found || !slices.Equal(on, want) || jsonOf(t, got.DefaultAssigneeID) != jsonOf(t, bob) || got.ArchiveIn != i+1 ||
			!got.CreatedAt.Equal(now) || !got.UpdatedAt.Equal(updated) {
			t.Errorf("%s on: GetProject() = %+v, %v, %v; want the views %v, default assignee %s, archive_in %d, created %v, updated %v",
				view, got, found, err, want, bob, i+1, now, updated)
		}
	}
}
