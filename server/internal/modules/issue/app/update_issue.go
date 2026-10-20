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
	Locks    Locks
	Issues   IssueUpdater
	States   StateCatalog
	Labels   LabelCatalog
	Members  ProjectMembers
	Profiles PublicProfiles
	Clock    Clock
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
// changes it, and the new one's group; the labels it adds and removes,
// each in id order.
type named struct {
	state         *domain.Changed[domain.Named]
	group         domain.StateGroup
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
// refuses it (409 project.archived); what p changes in the work item
// (domain.ChangesOf), and when it changes nothing, nothing is written and
// the answer is the work item as read; the dependent rows it adds (check),
// every problem in one 422; the clock; the work item's fields, its
// completion by its new state's group when its state changes
// (domain.CompletedAt); the accounts and the labels it removes and adds,
// in id order; the accounts it adds subscribed; its activity
// (domain.UpdateActivity), each account by his display name. All of it is
// the caller's at that moment. The answer is the work item as stored, read
// back in the transaction.
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
		if h.project.Archived {
			return domain.ErrProjectArchived
		}
		c := domain.ChangesOf(h.issue, p)
		if c.None() {
			updated = h.issue
			return nil
		}
		n, err := u.check(ctx, h.issue, p, c)
		if err != nil {
			return err
		}
		s := stamp{workspace: h.issue.WorkspaceID, project: h.issue.ProjectID, issue: id, by: actor.UserID, at: u.d.Clock.Now()}
		if err := u.write(ctx, s, h.issue, c, n); err != nil {
			return err
		}
		updated, err = readBack(ctx, u.d.Issues, id)
		return err
	})
	if err != nil {
		return domain.Issue{}, err
	}
	return updated, nil
}

// check checks the dependent rows that c adds to the work item i under the
// project's lock (M4 design 3.6, 3.7), and reads the names its activity
// writes: the new state, when c changes it, is the project's and not its
// triage state, by a creation's rule (pickState); each account added may
// be assigned (domain.Unassignable), while one the work item keeps stays
// whatever his role; each label added is the project's. The labels the
// work item keeps are on it, so they are the project's undeleted labels:
// deleting a label takes it off every work item (DetachLabel). Every
// problem is reported at once, in one 422, each at its place in p.
func (u *UpdateIssue) check(ctx context.Context, i domain.Issue, p domain.IssuePatch, c domain.Changes) (named, error) {
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

// issueState is the state of states, its project's, that the work item i is
// in. A state that is none of them is an error, not a refusal: a work
// item's state is its project's.
func issueState(states []StateRef, i domain.Issue) (StateRef, error) {
	at := slices.IndexFunc(states, func(s StateRef) bool { return s.ID == i.StateID })
	if at < 0 {
		return StateRef{}, fmt.Errorf("work item %s is in state %s, which is not its project's", i.ID, i.StateID)
	}
	return states[at], nil
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
func (u *UpdateIssue) write(ctx context.Context, s stamp, i domain.Issue, c domain.Changes, n named) error {
	f := IssueFields{ID: i.ID, Name: i.Name, StateID: i.StateID, Priority: i.Priority, CompletedAt: i.CompletedAt, By: s.by, At: s.at}
	activity := domain.Update{State: n.state, LabelsAdded: n.labelsAdded, LabelsRemoved: n.labelsRemoved}
	if c.Name != nil {
		f.Name, activity.Name = *c.Name, &domain.Changed[string]{Old: i.Name, New: *c.Name}
	}
	if c.Priority != nil {
		f.Priority, activity.Priority = *c.Priority, &domain.Changed[domain.Priority]{Old: i.Priority, New: *c.Priority}
	}
	if c.StateID != nil {
		f.StateID, f.CompletedAt = *c.StateID, domain.CompletedAt(n.group, s.at)
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
	if len(c.Assignees.Added) > 0 {
		if err := u.d.Issues.InsertSubscribers(ctx, s.links(c.Assignees.Added)); err != nil {
			return err
		}
	}
	accounts, err := assignees(ctx, u.d.Profiles, slices.Concat(c.Assignees.Added, c.Assignees.Removed))
	if err != nil {
		return err
	}
	activity.AssigneesAdded, activity.AssigneesRemoved = accounts[:len(c.Assignees.Added)], accounts[len(c.Assignees.Added):]
	return u.d.Issues.InsertActivities(ctx, s.activity(domain.UpdateActivity(activity)))
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
