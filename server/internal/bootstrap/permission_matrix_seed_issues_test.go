package bootstrap

import (
	"context"
	"slices"
	"strings"
	"uuid"

	issuepg "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	issueapp "github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	issuedomain "github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// matrixIssue is the work item prepareMatrix seeds in each project of
// matrixProjects through the issue store, by its workspace's admin, its
// project's first: in Todo, medium, assigned to PM's account and to the
// removed member's, whose membership of acme and of its public project
// ended since, and labelled Bug. gone's is deleted with it.
const matrixIssue = "Seeded"

// issueSeed writes the prepared work items through the issue store, each
// numbered through the project store, as createIssue numbers it.
type issueSeed struct {
	projectSeed
	store *issuepg.Store
}

// issues stores each project's matrixIssue with the id sd names: its
// number, its row, its assignees' rows in id order, and its label's.
func (s issueSeed) issues(sd seeded) {
	s.t.Helper()
	ctx := context.Background()
	assignees := slices.SortedFunc(slices.Values([]uuid.UUID{s.ids[callerProjectMember], s.ids[callerRemoved]}), uuid.UUID.Compare)
	for _, p := range matrixProjects {
		slug, _, _ := strings.Cut(p.key, "/")
		sequence, err := s.projectSeed.store.NextIssueSequence(ctx, s.projects[p.key])
		if err != nil {
			s.t.Fatal(err)
		}
		id, by := sd.issue(p.key, matrixIssue), s.ids[matrixAdmins[slug]]
		if err := s.store.InsertIssue(ctx, issueapp.IssueRow{ID: id, WorkspaceID: s.workspaces[slug], ProjectID: s.projects[p.key],
			SequenceID: sequence, Name: matrixIssue, StateID: sd.state(p.key, "Todo"), Priority: issuedomain.PriorityMedium, SortOrder: 65535,
			By: by, At: s.now}); err != nil {
			s.t.Fatal(err)
		}
		for _, link := range []struct {
			insert  func(context.Context, issueapp.LinkRows) error
			targets []uuid.UUID
		}{{s.store.InsertAssignees, assignees}, {s.store.InsertLabels, []uuid.UUID{sd.label(p.key, "Bug")}}} {
			rows := issueapp.LinkRows{WorkspaceID: s.workspaces[slug], ProjectID: s.projects[p.key], IssueID: id, Targets: link.targets, By: by,
				At: s.now}
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
// the issue store by the ids sd names: each project's work item, in its
// project, its first, named matrixIssue, in Todo, medium, assigned to PM's
// account and the removed member's, in id order, labelled Bug; undeleted,
// but gone's, deleted with gone. One missing, or stored otherwise, would
// let a cell answer as it wants for another reason.
func (s issueSeed) seededIssues(sd seeded) {
	s.t.Helper()
	assignees := slices.SortedFunc(slices.Values([]uuid.UUID{s.ids[callerProjectMember], s.ids[callerRemoved]}), uuid.UUID.Compare)
	for _, p := range matrixProjects {
		gone := strings.HasPrefix(p.key, "gone/")
		i, found, err := s.store.GetIssue(context.Background(), sd.issue(p.key, matrixIssue))
		if err != nil || found == gone || !gone && (i.ProjectID != s.projects[p.key] || i.SequenceID != 1 || i.Name != matrixIssue ||
			i.StateID != sd.state(p.key, "Todo") || i.Priority != issuedomain.PriorityMedium || !slices.Equal(i.AssigneeIDs, assignees) ||
			!slices.Equal(i.LabelIDs, []uuid.UUID{sd.label(p.key, "Bug")})) {
			s.t.Fatalf("%s's work item = %+v, %v, %v; want %s, its first, in Todo, medium, assigned to %v, labelled Bug, deleted %v", p.key, i,
				found, err, matrixIssue, assignees, gone)
		}
	}
}
