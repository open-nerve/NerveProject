package app

import (
	"context"
	"errors"
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
	Tx        shared.TxManager
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

// Execute checks in (domain.CheckNewIssue) and takes Locks' precheck,
// before any lock; then in one transaction, in the order of M4 design 3.5:
// the project FOR NO KEY UPDATE and the decision on issue.create; an
// archived project refuses it (409 project.archived); the dependent rows
// (check), every problem in one 422; the project's next number; the clock;
// the sort order after the greatest of the state's work items; the work
// item; its assignees, the default one too; its labels; its subscribers,
// the caller and the assignees, in id order; its activity: the creation,
// then each assignee by his display name (domain.CreationActivity). All
// of it is the caller's at that moment. The answer is the work item as
// stored, read back in the transaction.
func (u *CreateIssue) Execute(ctx context.Context, projectID uuid.UUID, in domain.NewIssue) (domain.Issue, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Issue{}, err
	}
	if in, err = domain.CheckNewIssue(in); err != nil {
		return domain.Issue{}, err
	}
	w := write{project: projectID, action: domain.ActionCreate, notFound: domain.ErrProjectNotFound}
	if err := u.d.Locks.precheck(ctx, actor, w); err != nil {
		return domain.Issue{}, err
	}
	var created domain.Issue
	err = u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		project, err := u.d.Locks.lockAndDecide(ctx, actor, w)
		if err != nil {
			return err
		}
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
		created, err = u.answer(ctx, row.ID)
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
			if !slices.Contains(known, id) {
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

// pickState is the state of states asked for, when it is there and not
// the triage state, or the default state when none is asked for; found is
// false when there is no such state.
func pickState(states []StateRef, asked *uuid.UUID) (StateRef, bool) {
	for _, s := range states {
		if asked == nil && s.Default || asked != nil && s.ID == *asked && s.Group != domain.GroupTriage {
			return s, true
		}
	}
	return StateRef{}, false
}

// write writes the work item row and the rows under it, in the order of M4
// design 3.5: the work item, its assignees, its labels, its subscribers,
// its activity.
func (u *CreateIssue) write(ctx context.Context, row IssueRow, c checked) error {
	if err := u.d.Issues.InsertIssue(ctx, row); err != nil {
		return err
	}
	links := func(targets []uuid.UUID) LinkRows {
		r := LinkRows{WorkspaceID: row.WorkspaceID, ProjectID: row.ProjectID, IssueID: row.ID, Targets: targets, By: row.By, At: row.At}
		for range targets {
			r.IDs = append(r.IDs, uuid.NewV7())
		}
		return r
	}
	if len(c.assignees) > 0 {
		if err := u.d.Issues.InsertAssignees(ctx, links(c.assignees)); err != nil {
			return err
		}
	}
	if len(c.labels) > 0 {
		if err := u.d.Issues.InsertLabels(ctx, links(c.labels)); err != nil {
			return err
		}
	}
	subscribers := slices.Compact(slices.SortedFunc(slices.Values(append([]uuid.UUID{row.By}, c.assignees...)), uuid.UUID.Compare))
	if err := u.d.Issues.InsertSubscribers(ctx, links(subscribers)); err != nil {
		return err
	}
	assignees, err := u.named(ctx, c.assignees)
	if err != nil {
		return err
	}
	var rows []ActivityRow
	for _, a := range domain.CreationActivity(assignees) {
		rows = append(rows, ActivityRow{ID: uuid.NewV7(), WorkspaceID: row.WorkspaceID, ProjectID: row.ProjectID, IssueID: row.ID, Activity: a,
			ActorID: row.By, At: row.At})
	}
	return u.d.Issues.InsertActivities(ctx, rows)
}

// named is the accounts ids with their display names, in their order. An
// account without one is an error: an assignee is a member of the project,
// and accounts are never deleted.
func (u *CreateIssue) named(ctx context.Context, ids []uuid.UUID) ([]domain.Assignee, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	names, err := u.d.Profiles.DisplayNames(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Assignee, len(ids))
	for i, id := range ids {
		name, ok := names[id]
		if !ok {
			return nil, fmt.Errorf("assignee %s has no profile", id)
		}
		out[i] = domain.Assignee{ID: id, DisplayName: name}
	}
	return out, nil
}

// answer reads the work item id back as stored, in the transaction ctx
// carries: the write's answer. A work item not there after its write is an
// internal error, not a 404.
func (u *CreateIssue) answer(ctx context.Context, id uuid.UUID) (domain.Issue, error) {
	i, found, err := u.d.Issues.GetIssue(ctx, id)
	switch {
	case err != nil:
		return domain.Issue{}, err
	case !found:
		return domain.Issue{}, errors.New("work item " + id.String() + " is not there after its write")
	}
	return i, nil
}
