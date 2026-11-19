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

// IssueUpdateDeps are the ports the update core reads and writes through
// while its caller holds the work item's write lock. They carry no clock:
// each caller gives updateLocked its own.
type IssueUpdateDeps struct {
	ContentBindings ContentBinding
	Events          shared.EventPublisher
	Locks           Locks
	Issues          IssueUpdater
	States          StateCatalog
	Labels          LabelCatalog
	Members         ProjectMembers
	Planning        PlanningTargets
	Profiles        PublicProfiles
	Sanitizer       HTMLSanitizer
}

// IssueUpdate changes a work item within its caller's transaction, under
// his write's lock (updateLocked). It owns no authorization, gate or
// transaction boundary.
type IssueUpdate struct{ d IssueUpdateDeps }

// NewIssueUpdate returns the shared update core.
func NewIssueUpdate(d IssueUpdateDeps) *IssueUpdate { return &IssueUpdate{d: d} }

// named are the names an update's activity writes, read under the
// project's lock: the work item's state before and after, when the update
// changes it, and the new one's group; its parent before and after, when
// the update changes it; the labels it adds and removes, each in id order;
// its cycle before and after, when the update changes it, and the modules
// it joins and leaves (membership, M6 P3a spec R1).
type named struct {
	state         *domain.Changed[domain.Named]
	group         domain.StateGroup
	parent        *domain.Changed[*domain.Ref]
	labelsAdded   []domain.Named
	labelsRemoved []domain.Named
	membership
}

// updateLocked is the update core (M7 P2a spec 2.7, R6): the update p,
// checked (domain.CheckIssuePatch), by the caller actorID, of the work item
// that h, his write's lock and decision, holds, in its transaction. It
// owns no authorization, gate or transaction boundary, and no guard of its
// caller's write (write.check): an update's and a triage's accept each
// decide their own action before it. Under the lock, in this order: an
// archived project refuses it (409 project.archived), then an archived work
// item (409 issue.archived), a change of nothing too; what p changes in
// the work item as the caller is answered it, a parent he does not see
// counted as none (asAnswered, domain.ChangesOf), and when it changes
// nothing, nothing is written, the clock is not read, and the answer is
// the work item as read; the dependent rows it adds, the order of the
// dates it leaves, the parent it sets, the cycle and the modules it joins
// (check), every problem in one 422; the moment, read once after the last
// lock: the content's lock's (ContentBinding.LockContent) when p's
// description binds images, else clock's; whether the cycle it joins has
// ended then (M6 P3a spec R2); the work item's fields, its completion by
// its new state's group when its state changes (domain.CompletedAt); the
// accounts and the labels it removes and adds, in id order; its cycle's
// rows (cycleRows); the modules it leaves and joins, the same; the
// accounts it adds subscribed; its activity (domain.UpdateActivity), each
// account by his display name, each parent by its identifier; issue.updated.
// All of it is the caller's at that moment. An update passes its use
// case's clock; a triage's accept, whose patch binds no content, passes
// the triage's one moment (moment), so that the whole request writes at
// it. The answer is the work item as stored, read back, as the caller
// sees it then (Locks.seenAt); when nothing is written, as read, at the
// moment of Locks' clock (Locks.seenOne).
func (u *IssueUpdate) updateLocked(ctx context.Context, actorID uuid.UUID, h held, p domain.IssuePatch, clock Clock) (domain.Issue, error) {
	switch {
	case h.project.Archived:
		return domain.Issue{}, domain.ErrProjectArchived
	case h.issue.ArchivedAt != nil:
		return domain.Issue{}, domain.ErrIssueArchived
	}
	id := h.issue.ID
	answered, old, err := asAnswered(ctx, u.d.Issues, h, actorID, p)
	if err != nil {
		return domain.Issue{}, err
	}
	c := domain.ChangesOf(answered, p)
	q := ContentBindRequest{ActorID: actorID, Host: ContentHost{WorkspaceID: h.issue.WorkspaceID, ProjectID: h.issue.ProjectID, IssueID: id}}
	if p.DescriptionHTML != nil {
		change := domain.CompareDescription(u.d.Sanitizer.Clean(h.issue.DescriptionHTML), u.d.Sanitizer.Clean(*p.DescriptionHTML), p.SkipActivity != nil && *p.SkipActivity)
		c.Description = &change
		q.ImageIDs = change.Content.Identity.AssetImageIDs()
	}
	if c.None() {
		// An unchanged legacy body can still contain an unbound reference.
		// Validate it even though no issue version or activity is written.
		if len(q.ImageIDs) > 0 {
			at, err := u.d.ContentBindings.LockContent(ctx, q)
			if err != nil {
				return domain.Issue{}, err
			}
			if err := u.d.ContentBindings.BindContent(ctx, q, at); err != nil {
				return domain.Issue{}, err
			}
		}
		return u.d.Locks.seenOne(ctx, actorID, h.grant, h.issue)
	}
	n, err := u.check(ctx, h, actorID, p, c, old)
	if err != nil {
		return domain.Issue{}, err
	}
	relations, err := prepareDescription(ctx, u.d.Issues, u.d.Members, h.issue, c.Description)
	if err != nil {
		return domain.Issue{}, err
	}
	var at time.Time
	if len(q.ImageIDs) > 0 {
		at, err = u.d.ContentBindings.LockContent(ctx, q)
		if err != nil {
			return domain.Issue{}, err
		}
	} else {
		at = clock.Now()
	}
	s := stamp{workspace: h.issue.WorkspaceID, project: h.issue.ProjectID, issue: id, by: actorID, at: at}
	event, err := u.write(ctx, s, h.issue, c, n, relations)
	if err != nil {
		return domain.Issue{}, err
	}
	if len(q.ImageIDs) > 0 {
		if err := u.d.ContentBindings.BindContent(ctx, q, at); err != nil {
			return domain.Issue{}, err
		}
	}
	if event != nil {
		if err := u.d.Events.Publish(ctx, *event); err != nil {
			return domain.Issue{}, err
		}
	}
	updated, err := readBack(ctx, u.d.Issues, id)
	if err != nil {
		return domain.Issue{}, err
	}
	return u.d.Locks.seenAt(ctx, actorID, h.grant, updated, moment(at))
}

// check checks the dependent rows that c adds to the work item that h, the
// caller actor's write, holds under the project's lock (M4 design 3.6,
// 3.7, 3.9), and reads the names its activity writes: first the cycle it
// leaves and the one it joins and the modules it joins and leaves, when c
// changes them, whose refusals come before any other's (readMembership, M6
// P3a spec R2); the new state,
// when c changes it, is the project's and not its triage state, by a creation's
// rule (pickState); each account added may be assigned
// (domain.Unassignable), while one the work item keeps stays whatever his
// role; each label added is the project's. The labels the work item keeps
// are on it, so they are the project's undeleted labels: deleting a label
// takes it off every work item (DetachLabel). Then the dates p leaves the
// work item with are in order, merged with those it keeps
// (domain.DatesOutOfOrder); then the parent c sets may be the work item's
// (parentChange), from its parent before, old when the comparison read it
// (asAnswered); the cycle and each module joined are the project's. Every
// problem is reported at once, in one 422, each at its place in p.
func (u *IssueUpdate) check(ctx context.Context, h held, actor uuid.UUID, p domain.IssuePatch, c domain.Changes, old *domain.Ref) (named, error) {
	i := h.issue
	var n named
	var problems []shared.FieldError
	var listed []uuid.UUID
	if p.ModuleIDs != nil {
		listed = *p.ModuleIDs
	}
	m, late, err := readMembership(ctx, u.d.Planning, i.ProjectID, c.Cycle, c.Modules, listed)
	if err != nil {
		return named{}, err
	}
	n.membership = m
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
	problems = append(problems, late...)
	if len(problems) > 0 {
		return named{}, shared.Invalid(problems...)
	}
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
// order of M4 design 3.5: first, the cycle it joins may not have ended on
// the day of s's moment, the write's clock (refuseEnded, M6 P3a spec R2);
// its fields; its assignees' rows, those removed then those added; its
// labels' rows, the same; its cycle's (cycleRows); its modules', the same
// as its labels'; its subscribers', the accounts added; its activity's.
func (u *IssueUpdate) write(ctx context.Context, s stamp, i domain.Issue, c domain.Changes, n named, relations descriptionRelations) (*shared.Event, error) {
	if n.cycle != nil {
		if err := refuseEnded(ctx, u.d.Planning, i.ProjectID, n.cycle.New, s.at); err != nil {
			return nil, err
		}
	}
	if c.Description != nil && c.Description.Write && !c.Description.Changed {
		clean := c.Description.Content
		if err := u.d.Issues.WriteDescription(ctx, DescriptionWrite{IssueID: i.ID, HTML: clean.HTML, Text: clean.Text}); err != nil {
			return nil, err
		}
		if len(relations.subscribers) > 0 {
			if err := u.d.Issues.InsertSubscribers(ctx, s.links(relations.subscribers)); err != nil {
				return nil, err
			}
		}
		return nil, writeMentions(ctx, u.d.Issues, s, relations.mentions)
	}
	// Preserve the lock-time description, including a legacy NULL stripped value.
	f := IssueFields{SortOrder: i.SortOrder, ID: i.ID, Name: i.Name, StateID: i.StateID, Priority: i.Priority, StartDate: i.StartDate, TargetDate: i.TargetDate,
		DescriptionHTML: i.DescriptionHTML, DescriptionStripped: i.DescriptionStripped,
		ParentID: i.ParentID, CompletedAt: i.CompletedAt, By: s.by, At: s.at}
	activity := domain.Update{Parent: n.parent, State: n.state, LabelsAdded: n.labelsAdded, LabelsRemoved: n.labelsRemoved, Cycle: n.cycle,
		ModulesAdded: n.modulesJoined, ModulesRemoved: n.modulesLeft}
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
		return nil, err
	}
	if err := u.relink(ctx, s, c.Assignees, u.d.Issues.RemoveAssignees, u.d.Issues.InsertAssignees); err != nil {
		return nil, err
	}
	if err := u.relink(ctx, s, c.Labels, u.d.Issues.RemoveLabels, u.d.Issues.InsertLabels); err != nil {
		return nil, err
	}
	if n.cycle != nil {
		if err := u.relink(ctx, s, cycleRows(*n.cycle), u.d.Issues.RemoveCycle, u.d.Issues.InsertCycle); err != nil {
			return nil, err
		}
	}
	if err := u.relink(ctx, s, c.Modules, u.d.Issues.RemoveModules, u.d.Issues.InsertModules); err != nil {
		return nil, err
	}
	if targets := ascending(slices.Concat(c.Assignees.Added, relations.subscribers)); len(targets) > 0 {
		if err := u.d.Issues.InsertSubscribers(ctx, s.links(targets)); err != nil {
			return nil, err
		}
	}
	if err := writeMentions(ctx, u.d.Issues, s, relations.mentions); err != nil {
		return nil, err
	}
	accounts, err := assignees(ctx, u.d.Profiles, slices.Concat(c.Assignees.Added, c.Assignees.Removed))
	if err != nil {
		return nil, err
	}
	activity.AssigneesAdded, activity.AssigneesRemoved = accounts[:len(c.Assignees.Added)], accounts[len(c.Assignees.Added):]
	if err := writeUpdateActivities(ctx, u.d.Issues, s, activity); err != nil {
		return nil, err
	}
	if c.Description != nil && c.Description.Changed {
		if err := saveDescriptionVersion(ctx, u.d.Issues, s, c.Description.Content); err != nil {
			return nil, err
		}
	}
	event := s.event(domain.EventIssueUpdated, updatedChanges(activity, c.SortOrder, relations.mentions.Added))
	return &event, nil
}

// relink removes the rows of d's removed ids, then inserts those of its
// added ones, each when there are any.
func (u *IssueUpdate) relink(ctx context.Context, s stamp, d domain.Diff, remove func(context.Context, UnlinkRows) error,
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
