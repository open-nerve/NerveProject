package bootstrap

import (
	"context"
	"slices"
	"strings"
	"uuid"

	issuepg "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	issueapp "github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	issuedomain "github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	projectdomain "github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// matrixIssue is the work item prepareMatrix seeds in each project of
// matrixProjects through the issue store, by its workspace's admin, its
// project's first. gone's is deleted with it.
const matrixIssue = "Seeded"

// ownIssues are the work items prepareMatrix seeds besides, each the one
// a column's cells name (issueOf), whose maker the column's case is about:
// made by the column's account (accountOf) in its project (projectOf), in
// this order, after its project's matrixIssue. PM's, the member who made
// it; the workspace guest's, stored with him as its maker: a guest makes
// none (createIssue), so a guest's own work item is one he made as a
// member before; no decision reads how it came to be, only its creator and
// his role now; the workspace admin's who is the project's guest, who may
// make one as an admin of it (M4 design 9.2); and, in the private project,
// the member before's, made before he left it (endings): 9.2's member who
// made it and has left the project.
var ownIssues = []struct {
	c    caller
	name string
}{{callerCreator, "Member's"}, {callerGuestOwn, "Guest's"}, {callerAdminGuestOwn, "Admin guest's"}, {callerBefore, "Before's"}}

// guestViewAll is the project that shows its guests every work item
// (guest_view_all_features), as its admin chose: acme's private one, for
// the guest who sees them all (callerGuestAll); acme's public one does
// not, as a project starts (M4 design 3.4).
const guestViewAll = "acme/private"

// seededItem is a work item issueSeed stores: its project's key, its name
// and its maker.
type seededItem struct {
	key, name string
	by        caller
}

// seededItems are the work items issueSeed stores, in order: each
// project's matrixIssue, by its workspace's admin, then ownIssues.
func seededItems() []seededItem {
	var items []seededItem
	for _, p := range matrixProjects {
		slug, _, _ := strings.Cut(p.key, "/")
		items = append(items, seededItem{p.key, matrixIssue, matrixAdmins[slug]})
	}
	for _, o := range ownIssues {
		items = append(items, seededItem{projectOf(o.c), o.name, accountOf(o.c)})
	}
	return items
}

// numberOf is the number of the seeded work item name of the project key,
// its place among its project's in seededItems' order, and its maker; 0
// and none for a name never seeded.
func numberOf(key, name string) (int, caller) {
	n := 0
	for _, i := range seededItems() {
		if i.key == key {
			n++
			if i.name == name {
				return n, i.by
			}
		}
	}
	return 0, ""
}

// nextNumber is the number of the project key's next work item, the one a
// creation makes: one past the seeded ones.
func nextNumber(key string) int {
	n := 1
	for _, i := range seededItems() {
		if i.key == key {
			n++
		}
	}
	return n
}

// issueSeed writes the prepared work items through the issue store, each
// numbered through the project store, as createIssue numbers it.
type issueSeed struct {
	projectSeed
	store *issuepg.Store
}

// issues makes guestViewAll show its guests every work item, through the
// project store, by acme's admin; then stores each of seededItems with the
// id sd names, by its maker: its number, its row, its assignees' rows in
// id order, and its label's. Each is in Todo, medium, assigned to PM's
// account and to the removed member's, whose membership of acme and of its
// public project ended since, and labelled Bug.
func (s issueSeed) issues(sd seeded) {
	s.t.Helper()
	ctx := context.Background()
	on := true
	if err := s.projectSeed.store.UpdateProject(ctx, s.projects[guestViewAll], projectdomain.ProjectPatch{GuestViewAllFeatures: &on},
		s.ids[callerAdmin], s.now); err != nil {
		s.t.Fatal(err)
	}
	assignees := slices.SortedFunc(slices.Values([]uuid.UUID{s.ids[callerProjectMember], s.ids[callerRemoved]}), uuid.UUID.Compare)
	for _, item := range seededItems() {
		slug, _, _ := strings.Cut(item.key, "/")
		sequence, err := s.projectSeed.store.NextIssueSequence(ctx, s.projects[item.key])
		if err != nil {
			s.t.Fatal(err)
		}
		id, by := sd.issue(item.key, item.name), s.ids[item.by]
		if err := s.store.InsertIssue(ctx, issueapp.IssueRow{ID: id, WorkspaceID: s.workspaces[slug], ProjectID: s.projects[item.key],
			SequenceID: sequence, Name: item.name, StateID: sd.state(item.key, "Todo"), Priority: issuedomain.PriorityMedium, SortOrder: 65535,
			By: by, At: s.now}); err != nil {
			s.t.Fatal(err)
		}
		for _, link := range []struct {
			insert  func(context.Context, issueapp.LinkRows) error
			targets []uuid.UUID
		}{{s.store.InsertAssignees, assignees}, {s.store.InsertLabels, []uuid.UUID{sd.label(item.key, "Bug")}}} {
			rows := issueapp.LinkRows{WorkspaceID: s.workspaces[slug], ProjectID: s.projects[item.key], IssueID: id, Targets: link.targets,
				By: by, At: s.now}
			for range link.targets {
				rows.IDs = append(rows.IDs, uuid.NewV7())
			}
			if err := link.insert(ctx, rows); err != nil {
				s.t.Fatal(err)
			}
		}
	}
}

// seededIssues checks the work items the cells rest on, read back through
// the issue store by the ids sd names: each of seededItems in its project,
// of its number and name, made by its maker, in Todo, medium, assigned to
// PM's account and the removed member's, in id order, labelled Bug;
// undeleted, but gone's, deleted with gone. Then what the guest rule reads
// of the issue table's guests, through the project store and the
// workspace store: the workspace's guest an active guest of acme's two
// projects, of which guestViewAll alone shows its guests every work item,
// and the workspace admin who is a guest an active guest of the public one
// and acme's active admin. One missing, or stored otherwise, would let a
// cell answer as it wants for another reason.
func (s issueSeed) seededIssues(sd seeded) {
	s.t.Helper()
	ctx := context.Background()
	assignees := slices.SortedFunc(slices.Values([]uuid.UUID{s.ids[callerProjectMember], s.ids[callerRemoved]}), uuid.UUID.Compare)
	for _, item := range seededItems() {
		gone := strings.HasPrefix(item.key, "gone/")
		number, _ := numberOf(item.key, item.name)
		i, found, err := s.store.GetIssue(ctx, sd.issue(item.key, item.name))
		if err != nil || found == gone || !gone && (i.ProjectID != s.projects[item.key] || i.SequenceID != number || i.Name != item.name ||
			i.CreatedByID == nil || *i.CreatedByID != s.ids[item.by] || i.StateID != sd.state(item.key, "Todo") ||
			i.Priority != issuedomain.PriorityMedium || !slices.Equal(i.AssigneeIDs, assignees) ||
			!slices.Equal(i.LabelIDs, []uuid.UUID{sd.label(item.key, "Bug")})) {
			s.t.Fatalf("%s's work item %s = %+v, %v, %v; want its %d, by %s, in Todo, medium, assigned to %v, labelled Bug, deleted %v", item.key,
				item.name, i, found, err, number, item.by, assignees, gone)
		}
	}
	for _, tt := range []struct {
		key     string
		c       caller
		viewAll bool
	}{{"acme/public", callerGuest, false}, {guestViewAll, callerGuest, true}, {"acme/public", callerAdminGuest, false}} {
		f, found, err := s.projectSeed.store.ProjectFacts(ctx, s.projects[tt.key], s.ids[tt.c])
		if err != nil || !found || !f.Member || f.Role != shared.RoleGuest || f.GuestViewAll != tt.viewAll {
			s.t.Fatalf("%s's facts of %s = %+v, %v, %v; want him its active guest, every work item shown %v", tt.c, tt.key, f, found, err,
				tt.viewAll)
		}
	}
	if role, active, err := s.matrixSeed.store.ActiveRole(ctx, sd.workspace("acme"), s.ids[callerAdminGuest]); err != nil || !active ||
		role != shared.RoleAdmin {
		s.t.Fatalf("%s's role in acme = %d, %v, %v; want its active admin", callerAdminGuest, role, active, err)
	}
}
