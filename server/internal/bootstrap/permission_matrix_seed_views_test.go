package bootstrap

import (
	"context"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	navigationpg "github.com/open-nerve/NerveProject/server/internal/modules/navigation/adapter/postgres"
	navigationdomain "github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	projectdomain "github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The saved views prepareMatrix seeds (M7 P3b spec R7), and how a row's
// request names them; their rows are in permission_matrix_views_test.go.

// matrixView is the view prepareMatrix seeds in each project of
// matrixProjects through the navigation store, by its workspace's admin,
// as matrixIssue: the view a column's cells name unless ownViews gives it
// its own. gone's is deleted with it.
const matrixView = "Saved"

// ownViews are the project views prepareMatrix seeds besides, each the one
// a column's cells name (viewOf), whose owner the column's case is about:
// owned by the column's account (accountOf) in its project (projectOf),
// in this order, after its project's matrixView. PM's, the member who made
// it, who alone changes it (D29); the workspace guest's, stored with him
// as its owner: a guest creates none (createProjectView), so a guest's own
// view is one he made as a member before; the workspace admin's who is the
// project's guest, who may make one as an admin of it; and, in the private
// project, the member before's, made before he left it: its owner sees it
// no more.
var ownViews = []struct {
	c    caller
	name string
}{{callerCreator, "Member's"}, {callerGuestOwn, "Guest's"}, {callerAdminGuestOwn, "Admin guest's"}, {callerBefore, "Before's"}}

// seededViews are the project views prepareMatrix seeds, in order: each
// project's matrixView, by its workspace's admin, then ownViews; each as a
// seededItem, its project's key, its name and its owner.
func seededViews() []seededItem {
	var views []seededItem
	for _, p := range matrixProjects {
		slug, _, _ := strings.Cut(p.key, "/")
		views = append(views, seededItem{p.key, matrixView, matrixAdmins[slug]})
	}
	for _, o := range ownViews {
		views = append(views, seededItem{projectOf(o.c), o.name, accountOf(o.c)})
	}
	return views
}

// viewOf is the name of the view of its project (projectOf) that a
// column's cells name: its own, owned by its account, for a column of
// ownViews; its project's matrixView, its workspace admin's, for every
// other.
func viewOf(c caller) string {
	for _, o := range ownViews {
		if o.c == c {
			return o.name
		}
	}
	return matrixView
}

// The two workspace views of each workspace that a row names, by whose
// they are: workspaceColumns has no column of an owner, so a row of a
// workspace's view names its member's or its admin's.
const (
	aMembersView = "a member's"
	anAdminsView = "an admin's"
)

// matrixWorkspaceViews are the workspace's own views, without a project,
// that prepareMatrix seeds through the navigation store: in acme and in
// gone, deleted with it, one owned by a member of the workspace, one by
// its admin.
var matrixWorkspaceViews = []struct {
	slug, whose string
	owner       caller
}{
	{"acme", aMembersView, callerMember}, {"acme", anAdminsView, callerAdmin},
	{"gone", aMembersView, callerMember}, {"gone", anAdminsView, callerDeleted},
}

// projectView is the id of the view name of the project key; one never
// seeded fails the test at once, as membership's does.
func (s seeded) projectView(key, name string) uuid.UUID {
	id, ok := s.projectViews[key+"|"+name]
	if !ok {
		s.t.Helper()
		s.t.Fatalf("no view %s of %s is seeded", name, key)
	}
	return id
}

// workspaceView is the id of whose view of the workspace slug's own; one
// never seeded fails the test at once, as membership's does.
func (s seeded) workspaceView(slug, whose string) uuid.UUID {
	id, ok := s.workspaceViews[slug+"/"+whose]
	if !ok {
		s.t.Helper()
		s.t.Fatalf("no view of %s's own, %s, is seeded", slug, whose)
	}
	return id
}

// workspaceOfView is the slug of the workspace whose own view the id raw
// is; "" when raw is no workspace view seeded.
func (s seeded) workspaceOfView(raw string) string {
	id, err := uuid.Parse(raw)
	if err != nil {
		return ""
	}
	key, ok := keyOfRow(s.workspaceViews, id)
	if !ok {
		return ""
	}
	slug, _, _ := strings.Cut(key, "/")
	return slug
}

// viewsOn turns the views of acme's public project on through the project
// store, by acme's admin (M7 P3b spec 2.6: they are off by default): there
// the rows' cells that pass the decision write. The other projects' stay
// off, as a project starts: in acme's private one the only column that
// sees it, the guest who sees every work item, is refused as a guest
// before the toggle is looked at, and reads its views as ever; no cell
// gets so far in gone's or other's, and no row asks the archived one's.
func (s projectSeed) viewsOn() {
	s.t.Helper()
	on := true
	if err := s.store.UpdateProject(context.Background(), s.projects["acme/public"], projectdomain.ProjectPatch{IssueViewsView: &on},
		s.ids[callerAdmin], s.now); err != nil {
		s.t.Fatal(err)
	}
}

// seedMatrixViews writes each of seededViews and matrixWorkspaceViews
// through the navigation store, with the id newSeeded named, by its owner,
// at at: named, with no description, icon or filter, the shared display
// settings. prepareMatrix writes them before it deletes gone, whose
// deletion deletes gone's with it and its project (spec 2.7).
func seedMatrixViews(t *testing.T, pool *pgxpool.Pool, s seeded, at time.Time) {
	t.Helper()
	store := navigationpg.New(pool)
	write := func(id, workspace uuid.UUID, project *uuid.UUID, name string, owner caller) {
		by := s.account(owner)
		v := navigationdomain.View{ID: id, WorkspaceID: workspace, ProjectID: project, OwnedByID: by, CreatedAt: at, UpdatedAt: at,
			Content: navigationdomain.Content{Name: name, DisplayFilters: shared.DefaultListDisplayFilters(),
				DisplayProperties: shared.DefaultListDisplayProperties()}}
		if err := store.InsertView(pgtest.Soon(t), v, by); err != nil {
			t.Fatalf("seeding the view %s: %v", name, err)
		}
	}
	for _, v := range seededViews() {
		slug, _, _ := strings.Cut(v.key, "/")
		project := s.project(v.key)
		write(s.projectView(v.key, v.name), s.workspace(slug), &project, v.name, v.by)
	}
	for _, v := range matrixWorkspaceViews {
		write(s.workspaceView(v.slug, v.whose), s.workspace(v.slug), nil, v.slug+", "+v.whose, v.owner)
	}
}

// seededViewRows checks the views the cells rest on, read back by the ids
// newSeeded named: each in its workspace and its project, or of the
// workspace's own, owned by its owner; undeleted, but gone's, which gone's
// deletion deleted. One missing, or stored otherwise, would let a cell
// answer as it wants for another reason.
func seededViewRows(t *testing.T, pool *pgxpool.Pool, s seeded) {
	t.Helper()
	stored := func(id, workspace uuid.UUID, project *uuid.UUID, owner caller, gone bool) {
		var ok bool
		if err := pool.QueryRow(pgtest.Soon(t), `SELECT EXISTS (SELECT 1 FROM issue_views WHERE id = $1 AND workspace_id = $2
			AND project_id IS NOT DISTINCT FROM $3 AND owned_by_id = $4 AND (deleted_at IS NOT NULL) = $5)`, id, workspace, project,
			s.account(owner), gone).Scan(&ok); err != nil || !ok {
			t.Fatalf("the view %s: %v, %v; want %s's, in %s, project %v, deleted %v", id, ok, err, owner, workspace, project, gone)
		}
	}
	for _, v := range seededViews() {
		slug, _, _ := strings.Cut(v.key, "/")
		project := s.project(v.key)
		stored(s.projectView(v.key, v.name), s.workspace(slug), &project, v.by, slug == "gone")
	}
	for _, v := range matrixWorkspaceViews {
		stored(s.workspaceView(v.slug, v.whose), s.workspace(v.slug), nil, v.owner, v.slug == "gone")
	}
}
