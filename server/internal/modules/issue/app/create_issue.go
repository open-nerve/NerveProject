package app

import (
	"context"
	"fmt"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CreateIssueDeps are what createIssue reads and writes through.
type CreateIssueDeps struct {
	Locks     Locks
	Issues    IssueCreator
	Sequences Sequences
	States    StateCatalog
	Labels    LabelCatalog
	Members   ProjectMembers
	Profiles  PublicProfiles
	Clock     Clock
}

// CreateIssue creates a work item in a project: POST
// /api/v0/projects/{project_id}/issues (M4 design 3.5, 3.7, 5.1).
type CreateIssue struct {
	d CreateIssueDeps
}

// NewCreateIssue returns the use case.
func NewCreateIssue(d CreateIssueDeps) *CreateIssue {
	return &CreateIssue{d: d}
}

// checked is what createIssue's checks of the dependent rows found, under
// the project's lock: the state, the accounts the work item is assigned
// to and its labels, each in id order.
type checked struct {
	state     StateRef
	assignees []uuid.UUID
	labels    []uuid.UUID
}

// Execute checks in (domain.CheckNewIssue), before any read; then Locks
// runs the write (M4 design 3.5): its precheck, before any lock; then in
// one transaction, the project FOR NO KEY UPDATE and the decision on
// issue.create, and under them, in this order: an archived project refuses
// it (409 project.archived); the dependent rows (check), every problem in
// one 422; the project's next number; the clock; the sort order after the
// greatest of the state's work items; the work item; its assignees, the
// default one too; its labels; its subscribers, the caller and the
// assignees, in id order; its activity: the creation, then each assignee
// by his display name (domain.CreationActivity). All of it is the
// caller's at that moment. The answer is the work item as stored, read
// back in the transaction.
func (u *CreateIssue) Execute(ctx context.Context, projectID uuid.UUID, in domain.NewIssue) (domain.Issue, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Issue{}, err
	}
	if in, err = domain.CheckNewIssue(in); err != nil {
		return domain.Issue{}, err
	}
	w := write{project: projectID, action: domain.ActionCreate, notFound: domain.ErrProjectNotFound}
	var created domain.Issue
	err = u.d.Locks.run(ctx, actor, w, func(ctx context.Context, h held) error {
		project := h.project
		if project.Archived {
			return domain.ErrProjectArchived
		}
		c, err := u.check(ctx, projectID, project, in)
		if err != nil {
			return err
		}
		sequence, err := u.d.Sequences.Next(ctx, projectID)
		if err != nil {
			return err
		}
		now := u.d.Clock.Now()
		greatest, err := u.d.Issues.GreatestSortOrder(ctx, projectID, c.state.ID)
		if err != nil {
			return err
		}
		row := IssueRow{ID: uuid.NewV7(), WorkspaceID: project.WorkspaceID, ProjectID: projectID, SequenceID: sequence, Name: in.Name,
			StateID: c.state.ID, Priority: *in.Priority, SortOrder: domain.SortOrderAfter(greatest),
			CompletedAt: domain.CompletedAt(c.state.Group, now), By: actor.UserID, At: now}
		if err := u.write(ctx, row, c); err != nil {
			return err
		}
		created, err = readBack(ctx, u.d.Issues, row.ID)
		return err
	})
	if err != nil {
		return domain.Issue{}, err
	}
	return created, nil
}

// check checks the dependent rows of in under the project's lock (M4
// design 3.7): the state asked for is the project's and not its triage
// state, or the project's default state when none is; the assignees by
// domain.Assignees, from the roles of the accounts asked for, or of the
// project's default assignee when none is; each label asked for is the
// project's. Every problem is reported at once, in one 422.
func (u *CreateIssue) check(ctx context.Context, projectID uuid.UUID, p LockedProject, in domain.NewIssue) (checked, error) {
	var c checked
	var problems []shared.FieldError
	states, err := u.d.States.States(ctx, projectID)
	if err != nil {
		return checked{}, err
	}
	var found bool
	c.state, found = pickState(states, in.StateID)
	switch {
	case !found && in.StateID != nil:
		problems = append(problems, domain.StateNotAllowed())
	case !found:
		return checked{}, fmt.Errorf("project %s has no default state", projectID)
	}
	asked := in.AssigneeIDs
	if len(asked) == 0 && p.DefaultAssigneeID != nil {
		asked = []uuid.UUID{*p.DefaultAssigneeID}
	}
	roles := map[uuid.UUID]shared.Role{}
	if len(asked) > 0 {
		if roles, err = u.d.Members.Roles(ctx, projectID, asked); err != nil {
			return checked{}, err
		}
	}
	var refused []shared.FieldError
	c.assignees, refused = domain.Assignees(in.AssigneeIDs, p.DefaultAssigneeID, roles)
	problems = append(problems, refused...)
	if len(in.LabelIDs) > 0 {
		known, err := u.d.Labels.Labels(ctx, projectID, in.LabelIDs)
		if err != nil {
			return checked{}, err
		}
		for i, id := range in.LabelIDs {
			if !slices.ContainsFunc(known, func(l LabelRef) bool { return l.ID == id }) {
				problems = append(problems, domain.UnknownLabel(i))
			}
		}
		c.labels = slices.SortedFunc(slices.Values(in.LabelIDs), uuid.UUID.Compare)
	}
	if len(problems) > 0 {
		return checked{}, shared.Invalid(problems...)
	}
	return c, nil
}

// write writes the work item row and the rows under it, in the order of M4
// design 3.5: the work item, its assignees, its labels, its subscribers,
// its activity.
func (u *CreateIssue) write(ctx context.Context, row IssueRow, c checked) error {
	if err := u.d.Issues.InsertIssue(ctx, row); err != nil {
		return err
	}
	s := stamp{workspace: row.WorkspaceID, project: row.ProjectID, issue: row.ID, by: row.By, at: row.At}
	if len(c.assignees) > 0 {
		if err := u.d.Issues.InsertAssignees(ctx, s.links(c.assignees)); err != nil {
			return err
		}
	}
	if len(c.labels) > 0 {
		if err := u.d.Issues.InsertLabels(ctx, s.links(c.labels)); err != nil {
			return err
		}
	}
	subscribers := slices.Compact(slices.SortedFunc(slices.Values(append([]uuid.UUID{row.By}, c.assignees...)), uuid.UUID.Compare))
	if err := u.d.Issues.InsertSubscribers(ctx, s.links(subscribers)); err != nil {
		return err
	}
	accounts, err := assignees(ctx, u.d.Profiles, c.assignees)
	if err != nil {
		return err
	}
	return u.d.Issues.InsertActivities(ctx, s.activity(domain.CreationActivity(accounts)))
}
