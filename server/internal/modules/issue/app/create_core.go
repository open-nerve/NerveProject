package app

import (
	"context"
	"fmt"
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// IssueCreationDeps are the ports used while the caller holds the project lock.
type IssueCreationDeps struct {
	Events    shared.EventPublisher
	Issues    IssueCreator
	Sequences Sequences
	States    StateCatalog
	Labels    LabelCatalog
	Members   ProjectMembers
	Profiles  PublicProfiles
	Clock     Clock
	Sanitizer HTMLSanitizer
}

// IssueCreation writes a checked new issue within its caller's transaction.
// It owns no authorization, gate or transaction boundary.
type IssueCreation struct{ d IssueCreationDeps }

// NewIssueCreation returns the shared creation core.
func NewIssueCreation(d IssueCreationDeps) *IssueCreation { return &IssueCreation{d: d} }

// checked is what createIssue's checks of the dependent rows found, under
// the project's lock: the state, the accounts the work item is assigned
// to and its labels, each in id order, and its parent, nil for none.
type checked struct {
	state                 StateRef
	mentions, subscribers []uuid.UUID
	assignees             []uuid.UUID
	labels                []uuid.UUID
	parent                *uuid.UUID
	parentName            *domain.Ref
	labelNames            []domain.Named
}

// createLocked requires pure domain validation and the project's write lock.
// Its returned moment is the sole clock reading used by all of its writes.
func (u *IssueCreation) createLocked(ctx context.Context, actorID, projectID uuid.UUID, h held, in domain.NewIssue) (domain.Issue, time.Time, error) {
	project := h.project
	if project.Archived {
		return domain.Issue{}, time.Time{}, domain.ErrProjectArchived
	}
	c, err := u.check(ctx, h, actorID, projectID, in)
	if err != nil {
		return domain.Issue{}, time.Time{}, err
	}
	clean := u.d.Sanitizer.Clean(*in.DescriptionHTML)
	mentions, subscribers, err := descriptionTargets(ctx, u.d.Members, domain.Issue{ProjectID: projectID, CreatedByID: &actorID}, clean.Identity.MentionIDs)
	if err != nil {
		return domain.Issue{}, time.Time{}, err
	}
	c.mentions, c.subscribers = mentions, subscribers
	sequence, err := u.d.Sequences.Next(ctx, projectID)
	if err != nil {
		return domain.Issue{}, time.Time{}, err
	}
	now := u.d.Clock.Now()
	greatest, err := u.d.Issues.GreatestSortOrder(ctx, projectID, c.state.ID)
	if err != nil {
		return domain.Issue{}, time.Time{}, err
	}
	row := IssueRow{ID: uuid.NewV7(), WorkspaceID: project.WorkspaceID, ProjectID: projectID, SequenceID: sequence, Name: in.Name,
		StateID: c.state.ID, Priority: *in.Priority, StartDate: in.StartDate, TargetDate: in.TargetDate, ParentID: c.parent,
		DescriptionHTML: clean.HTML, DescriptionStripped: &clean.Text,
		SortOrder: domain.SortOrderAfter(greatest), CompletedAt: domain.CompletedAt(c.state.Group, now), By: actorID, At: now}
	if err := u.write(ctx, row, c, clean); err != nil {
		return domain.Issue{}, time.Time{}, err
	}
	created, err := readBack(ctx, u.d.Issues, row.ID)
	return created, now, err
}

// check checks the dependent rows of in, by the caller actor, under h, his
// write's lock on the project (M4 design 3.7, 3.9): the state asked for is
// the project's and not its triage state, or the project's default state
// when none is; the assignees by domain.Assignees, from the roles of the
// accounts asked for, or of the project's default assignee when none is;
// each label asked for is the project's; the parent asked for may be one
// (parentOf). Every problem is reported at once, in one 422.
func (u *IssueCreation) check(ctx context.Context, h held, actor, projectID uuid.UUID, in domain.NewIssue) (checked, error) {
	p := h.project
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
		c.labelNames, err = labelNames(known, c.labels)
		if err != nil && len(problems) == 0 {
			return checked{}, err
		}
	}
	if in.ParentID != nil {
		parent, ok, err := parentOf(ctx, u.d.Issues, h, actor, projectID, *in.ParentID)
		if err != nil {
			return checked{}, err
		}
		if !ok {
			problems = append(problems, domain.ParentNotAllowed())
		}
		c.parent = in.ParentID
		if ok {
			c.parentName = refOf(h.project, parent)
		}
	}
	if len(problems) > 0 {
		return checked{}, shared.Invalid(problems...)
	}
	return c, nil
}

// write writes the work item row and the rows under it, in the order of M4
// design 3.5: the work item, its assignees, its labels, its subscribers,
// its activity.
func (u *IssueCreation) write(ctx context.Context, row IssueRow, c checked, clean domain.RichText) error {
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
	subscribers := ascending(slices.Concat([]uuid.UUID{row.By}, c.assignees, c.subscribers))
	if err := u.d.Issues.InsertSubscribers(ctx, s.links(subscribers)); err != nil {
		return err
	}
	if len(c.mentions) > 0 {
		if err := u.d.Issues.InsertMentions(ctx, s.links(c.mentions)); err != nil {
			return err
		}
	}
	accounts, err := assignees(ctx, u.d.Profiles, c.assignees)
	if err != nil {
		return err
	}
	if err := u.d.Issues.InsertActivities(ctx, s.activity(domain.CreationActivity(accounts))); err != nil {
		return err
	}
	if err := u.d.Issues.InsertDescriptionVersion(ctx, descriptionVersion(s, clean)); err != nil {
		return err
	}
	return u.d.Events.Publish(ctx, s.event(domain.EventIssueCreated, createdChanges(row, c, accounts)))
}
