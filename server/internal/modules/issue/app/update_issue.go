package app

import (
	"context"
	"fmt"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// UpdateIssueDeps are what updateIssue reads and writes through.
type UpdateIssueDeps struct {
	Events    shared.EventPublisher
	Locks     Locks
	Issues    IssueUpdater
	States    StateCatalog
	Labels    LabelCatalog
	Members   ProjectMembers
	Profiles  PublicProfiles
	Clock     Clock
	Sanitizer HTMLSanitizer
}

// UpdateIssue changes a work item: PATCH /api/v0/issues/{issue_id} (M4
// design 3.5, 3.7, 5.1, 6.7).
type UpdateIssue struct {
	d UpdateIssueDeps
}

// NewUpdateIssue returns the use case.
func NewUpdateIssue(d UpdateIssueDeps) *UpdateIssue {
	return &UpdateIssue{d: d}
}

// named are the names an update's activity writes, read under the
// project's lock: the work item's state before and after, when the update
// changes it, and the new one's group; its parent before and after, when
// the update changes it; the labels it adds and removes, each in id order.
type named struct {
	state         *domain.Changed[domain.Named]
	group         domain.StateGroup
	parent        *domain.Changed[*domain.Ref]
	labelsAdded   []domain.Named
	labelsRemoved []domain.Named
}

// Execute checks p (domain.CheckIssuePatch), before any read; then Locks
// runs the write on the work item id (M4 design 3.5, 6.7, 11.11): its
// precheck, the work item read and the decision on issue.update on it, the
// guest rule first, before any lock; then the project's write gate, which
// refuses a write that may wait no longer (503 server_busy); then in one
// transaction, the project FOR NO KEY UPDATE, the work item read again and
// the decision again, and under them, in this order: an archived project
// refuses it (409 project.archived), then an archived work item (409
// issue.archived), a change of nothing too; what p changes in the work item
// as the caller is answered it, a parent he does not see counted as none
// (asAnswered, domain.ChangesOf), and when it changes nothing, nothing is
// written and the answer is the work item as read; the dependent rows it
// adds, the order of the dates it leaves and the parent it sets (check),
// every problem in one 422; the clock; the work item's fields, its completion by
// its new state's group when its state changes (domain.CompletedAt); the
// accounts and the labels it removes and adds, in id order; the accounts
// it adds subscribed; its activity (domain.UpdateActivity), each account
// by his display name, each parent by its identifier. All of it is the
// caller's at that moment. The answer is the work item as stored, read
// back in the transaction, as the caller sees it (Locks.seen), the work
// item as read when nothing is written.
func (u *UpdateIssue) Execute(ctx context.Context, id uuid.UUID, p domain.IssuePatch) (domain.Issue, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Issue{}, err
	}
	if p, err = domain.CheckIssuePatch(p); err != nil {
		return domain.Issue{}, err
	}
	w := write{issue: &id, action: domain.ActionUpdate, notFound: domain.ErrIssueNotFound}
	var updated domain.Issue
	err = u.d.Locks.run(ctx, actor, w, func(ctx context.Context, h held) error {
		switch {
		case h.project.Archived:
			return domain.ErrProjectArchived
		case h.issue.ArchivedAt != nil:
			return domain.ErrIssueArchived
		}
		answered, old, err := asAnswered(ctx, u.d.Issues, h, actor.UserID, p)
		if err != nil {
			return err
		}
		c := domain.ChangesOf(answered, p)
		if p.DescriptionHTML != nil {
			change := domain.CompareDescription(u.d.Sanitizer.Clean(h.issue.DescriptionHTML), u.d.Sanitizer.Clean(*p.DescriptionHTML), p.SkipActivity != nil && *p.SkipActivity)
			c.Description = &change
		}
		if c.None() {
			updated, err = u.d.Locks.seenOne(ctx, actor.UserID, h.grant, h.issue)
			return err
		}
		n, err := u.check(ctx, h, actor.UserID, p, c, old)
		if err != nil {
			return err
		}
		relations, err := prepareDescription(ctx, u.d.Issues, u.d.Members, h.issue, c.Description)
		if err != nil {
			return err
		}
		s := stamp{workspace: h.issue.WorkspaceID, project: h.issue.ProjectID, issue: id, by: actor.UserID, at: u.d.Clock.Now()}
		if err := u.write(ctx, s, h.issue, c, n, relations); err != nil {
			return err
		}
		if updated, err = readBack(ctx, u.d.Issues, id); err != nil {
			return err
		}
		updated, err = u.d.Locks.seenOne(ctx, actor.UserID, h.grant, updated)
		return err
	})
	if err != nil {
		return domain.Issue{}, err
	}
	return updated, nil
}

// check checks the dependent rows that c adds to the work item that h, the
// caller actor's write, holds under the project's lock (M4 design 3.6,
// 3.7, 3.9), and reads the names its activity writes: the new state, when
// c changes it, is the project's and not its triage state, by a creation's
// rule (pickState); each account added may be assigned
// (domain.Unassignable), while one the work item keeps stays whatever his
// role; each label added is the project's. The labels the work item keeps
// are on it, so they are the project's undeleted labels: deleting a label
// takes it off every work item (DetachLabel). Then the dates p leaves the
// work item with are in order, merged with those it keeps
// (domain.DatesOutOfOrder); then the parent c sets may be the work item's
// (parentChange), from its parent before, old when the comparison read it
// (asAnswered). Every problem is reported at once, in one 422, each at its
// place in p.
func (u *UpdateIssue) check(ctx context.Context, h held, actor uuid.UUID, p domain.IssuePatch, c domain.Changes, old *domain.Ref) (named, error) {
	i := h.issue
	var n named
	var problems []shared.FieldError
	if c.StateID != nil {
		states, err := u.d.States.States(ctx, i.ProjectID)
		if err != nil {
			return named{}, err
		}
		old, err := issueState(states, i)
		if err != nil {
			return named{}, err
		}
		if state, found := pickState(states, c.StateID); found {
			n.state = &domain.Changed[domain.Named]{Old: domain.Named{ID: old.ID, Name: old.Name}, New: domain.Named{ID: state.ID, Name: state.Name}}
			n.group = state.Group
		} else {
			problems = append(problems, domain.StateNotAllowed())
		}
	}
	if len(c.Assignees.Added) > 0 {
		roles, err := u.d.Members.Roles(ctx, i.ProjectID, c.Assignees.Added)
		if err != nil {
			return named{}, err
		}
		problems = append(problems, domain.Unassignable(*p.AssigneeIDs, c.Assignees.Added, roles)...)
	}
	var labels []LabelRef
	if len(c.Labels.Added) > 0 || len(c.Labels.Removed) > 0 {
		var err error
		asked := slices.SortedFunc(slices.Values(slices.Concat(c.Labels.Added, c.Labels.Removed)), uuid.UUID.Compare)
		if labels, err = u.d.Labels.Labels(ctx, i.ProjectID, asked); err != nil {
			return named{}, err
		}
		for at, id := range *p.LabelIDs {
			if slices.Contains(c.Labels.Added, id) && !slices.ContainsFunc(labels, func(l LabelRef) bool { return l.ID == id }) {
				problems = append(problems, domain.UnknownLabel(at))
			}
		}
	}
	if f := domain.DatesOutOfOrder(i, p); f != nil {
		problems = append(problems, *f)
	}
	if c.Parent != nil {
		parent, ok, err := parentChange(ctx, u.d.Issues, h, actor, c.Parent.To, old)
		if err != nil {
			return named{}, err
		}
		if ok {
			n.parent = &parent
		} else {
			problems = append(problems, domain.ParentNotAllowed())
		}
	}
	if len(problems) > 0 {
		return named{}, shared.Invalid(problems...)
	}
	var err error
	if n.labelsAdded, err = labelNames(labels, c.Labels.Added); err != nil {
		return named{}, err
	}
	if n.labelsRemoved, err = labelNames(labels, c.Labels.Removed); err != nil {
		return named{}, err
	}
	return n, nil
}

// labelNames are the labels ids, in their order, by their names in
// labels; none for none. A label not in labels is an error: the work
// item's labels, and those check let through, are the project's.
func labelNames(labels []LabelRef, ids []uuid.UUID) ([]domain.Named, error) {
	var out []domain.Named
	for _, id := range ids {
		i := slices.IndexFunc(labels, func(l LabelRef) bool { return l.ID == id })
		if i < 0 {
			return nil, fmt.Errorf("label %s is not the project's", id)
		}
		out = append(out, domain.Named{ID: id, Name: labels[i].Name})
	}
	return out, nil
}

// write writes what c changes in the work item i, all of it s's, in the
// order of M4 design 3.5: its fields; its assignees' rows, those removed
// then those added; its labels' rows, the same; its subscribers', the
// accounts added; its activity's.
func (u *UpdateIssue) write(ctx context.Context, s stamp, i domain.Issue, c domain.Changes, n named, relations descriptionRelations) error {
	if c.Description != nil && c.Description.Write && !c.Description.Changed {
		clean := c.Description.Content
		if err := u.d.Issues.WriteDescription(ctx, DescriptionWrite{IssueID: i.ID, HTML: clean.HTML, Text: clean.Text}); err != nil {
			return err
		}
		if len(relations.subscribers) > 0 {
			if err := u.d.Issues.InsertSubscribers(ctx, s.links(relations.subscribers)); err != nil {
				return err
			}
		}
		return writeMentions(ctx, u.d.Issues, s, relations.mentions)
	}
	// Preserve the lock-time description, including a legacy NULL stripped value.
	f := IssueFields{SortOrder: i.SortOrder, ID: i.ID, Name: i.Name, StateID: i.StateID, Priority: i.Priority, StartDate: i.StartDate, TargetDate: i.TargetDate,
		DescriptionHTML: i.DescriptionHTML, DescriptionStripped: i.DescriptionStripped,
		ParentID: i.ParentID, CompletedAt: i.CompletedAt, By: s.by, At: s.at}
	activity := domain.Update{Parent: n.parent, State: n.state, LabelsAdded: n.labelsAdded, LabelsRemoved: n.labelsRemoved}
	if c.Description != nil && c.Description.Write {
		clean := c.Description.Content
		f.DescriptionHTML, f.DescriptionStripped = clean.HTML, &clean.Text
		activity.Description = &domain.Changed[string]{Old: i.DescriptionHTML, New: clean.HTML}
	}
	if c.Parent != nil {
		f.ParentID = c.Parent.To
	}
	if c.Name != nil {
		f.Name, activity.Name = *c.Name, &domain.Changed[string]{Old: i.Name, New: *c.Name}
	}
	if c.Priority != nil {
		f.Priority, activity.Priority = *c.Priority, &domain.Changed[domain.Priority]{Old: i.Priority, New: *c.Priority}
	}
	if c.SortOrder != nil {
		f.SortOrder = c.SortOrder.New
	}
	if c.StateID != nil {
		f.StateID, f.CompletedAt = *c.StateID, domain.CompletedAt(n.group, s.at)
	}
	if c.StartDate != nil {
		f.StartDate, activity.StartDate = c.StartDate.To, &domain.Changed[*shared.Date]{Old: i.StartDate, New: c.StartDate.To}
	}
	if c.TargetDate != nil {
		f.TargetDate, activity.TargetDate = c.TargetDate.To, &domain.Changed[*shared.Date]{Old: i.TargetDate, New: c.TargetDate.To}
	}
	if err := u.d.Issues.UpdateIssue(ctx, f); err != nil {
		return err
	}
	if err := u.relink(ctx, s, c.Assignees, u.d.Issues.RemoveAssignees, u.d.Issues.InsertAssignees); err != nil {
		return err
	}
	if err := u.relink(ctx, s, c.Labels, u.d.Issues.RemoveLabels, u.d.Issues.InsertLabels); err != nil {
		return err
	}
	if targets := ascending(slices.Concat(c.Assignees.Added, relations.subscribers)); len(targets) > 0 {
		if err := u.d.Issues.InsertSubscribers(ctx, s.links(targets)); err != nil {
			return err
		}
	}
	if err := writeMentions(ctx, u.d.Issues, s, relations.mentions); err != nil {
		return err
	}
	accounts, err := assignees(ctx, u.d.Profiles, slices.Concat(c.Assignees.Added, c.Assignees.Removed))
	if err != nil {
		return err
	}
	activity.AssigneesAdded, activity.AssigneesRemoved = accounts[:len(c.Assignees.Added)], accounts[len(c.Assignees.Added):]
	if err := writeUpdateActivities(ctx, u.d.Issues, s, activity); err != nil {
		return err
	}
	if c.Description != nil && c.Description.Changed {
		return saveDescriptionVersion(ctx, u.d.Issues, s, c.Description.Content)
	}
	return nil
}

// relink removes the rows of d's removed ids, then inserts those of its
// added ones, each when there are any.
func (u *UpdateIssue) relink(ctx context.Context, s stamp, d domain.Diff, remove func(context.Context, UnlinkRows) error,
	insert func(context.Context, LinkRows) error) error {
	if len(d.Removed) > 0 {
		if err := remove(ctx, s.unlinks(d.Removed)); err != nil {
			return err
		}
	}
	if len(d.Added) > 0 {
		return insert(ctx, s.links(d.Added))
	}
	return nil
}
