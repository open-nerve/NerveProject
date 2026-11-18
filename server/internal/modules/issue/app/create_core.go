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
	Attachments     AttachmentCounter
	ContentBindings ContentBinding
	Events          shared.EventPublisher
	Issues          IssueCreator
	Sequences       Sequences
	States          StateCatalog
	Labels          LabelCatalog
	Members         ProjectMembers
	Planning        PlanningTargets
	Profiles        PublicProfiles
	Clock           Clock
	Sanitizer       HTMLSanitizer
}

// IssueCreation writes a checked new issue within its caller's transaction.
// It owns no authorization, gate or transaction boundary.
type IssueCreation struct{ d IssueCreationDeps }

// NewIssueCreation returns the shared creation core.
func NewIssueCreation(d IssueCreationDeps) *IssueCreation { return &IssueCreation{d: d} }

// checked is what createIssue's checks of the dependent rows found, under
// the project's lock: the state, the accounts the work item is assigned
// to and its labels, each in id order, its parent, nil for none, the cycle
// it joins, by its name, nil for none, and the modules it joins, by their
// names, in id order (M6 P3a spec R1).
type checked struct {
	state                 StateRef
	mentions, subscribers []uuid.UUID
	assignees             []uuid.UUID
	labels                []uuid.UUID
	parent                *uuid.UUID
	parentName            *domain.Ref
	labelNames            []domain.Named
	cycle                 *domain.Named
	modules               []domain.Named
}

// preparedIssue has its final identity and checked graph, but no write time.
// Both ordinary creation and draft publication consume this one preparation.
type preparedIssue struct {
	row     IssueRow
	checked checked
	content domain.RichText
}

// placement is the state a creation puts its work item in (M7 P2a spec
// 2.4).
type placement int

const (
	// asAsked is the state the creation asks for, or the project's
	// default state when it asks for none (pickState): an ordinary
	// creation's and a draft's publication's.
	asAsked placement = iota
	// inTriage is the project's triage state, whatever the creation asks
	// for: a submission's to the project's intake, which asks for none.
	inTriage
)

// state is the state among states, the project's, that p puts a work item
// in, which asks for the state asked, nil for none; found is false when
// there is none.
func (p placement) state(states []StateRef, asked *uuid.UUID) (StateRef, bool) {
	if p == asAsked {
		return pickState(states, asked)
	}
	at := slices.IndexFunc(states, func(s StateRef) bool { return s.Group == domain.GroupTriage })
	if at < 0 {
		return StateRef{}, false
	}
	return states[at], true
}

// createLocked requires pure domain validation and the project's write lock.
// Its returned moment is the sole clock reading used by all of its writes.
// The work item is put in the state place picks.
func (u *IssueCreation) createLocked(ctx context.Context, actorID, projectID uuid.UUID, h held, in domain.NewIssue, place placement) (domain.Issue, time.Time, error) {
	p, err := u.prepareLocked(ctx, actorID, projectID, h, in, place)
	if err != nil {
		return domain.Issue{}, time.Time{}, err
	}
	q := ContentBindRequest{ActorID: actorID, Host: ContentHost{WorkspaceID: p.row.WorkspaceID, ProjectID: projectID, IssueID: p.row.ID}, ImageIDs: p.content.Identity.AssetImageIDs()}
	var at time.Time
	if len(q.ImageIDs) > 0 {
		at, err = u.d.ContentBindings.LockContent(ctx, q)
		if err != nil {
			return domain.Issue{}, time.Time{}, err
		}
	} else {
		at = u.d.Clock.Now()
	}
	created, err := u.writePreparedAt(ctx, p, at)
	if err != nil {
		return domain.Issue{}, time.Time{}, err
	}
	if len(q.ImageIDs) > 0 {
		if err := u.d.ContentBindings.BindContent(ctx, q, at); err != nil {
			return domain.Issue{}, time.Time{}, err
		}
	}
	answer, err := u.answer(ctx, actorID, created)
	if err != nil {
		return domain.Issue{}, time.Time{}, err
	}
	return answer, at, nil
}

// answer is the work item created, as read back, as its creator actorID is
// answered it, in the creation's transaction, once every row is written:
// its attachments counted (projectAttachments), and whether he subscribes
// to it, read last (M7 P1a spec 2.3), which the creation's subscribing him
// makes so. A creation and a draft's publication answer it alike.
func (u *IssueCreation) answer(ctx context.Context, actorID uuid.UUID, created domain.Issue) (domain.Issue, error) {
	items := []domain.Issue{created}
	if err := projectAttachments(ctx, u.d.Attachments, items); err != nil {
		return domain.Issue{}, err
	}
	subscribed, err := u.d.Issues.IsSubscribed(ctx, created.ID, actorID)
	if err != nil {
		return domain.Issue{}, err
	}
	items[0].Subscribed = subscribed
	return items[0], nil
}

func (u *IssueCreation) prepareLocked(ctx context.Context, actorID, projectID uuid.UUID, h held, in domain.NewIssue, place placement) (preparedIssue, error) {
	project := h.project
	if project.Archived {
		return preparedIssue{}, domain.ErrProjectArchived
	}
	c, err := u.check(ctx, h, actorID, projectID, in, place)
	if err != nil {
		return preparedIssue{}, err
	}
	clean := u.d.Sanitizer.Clean(*in.DescriptionHTML)
	mentions, subscribers, err := descriptionTargets(ctx, u.d.Members, domain.Issue{ProjectID: projectID, CreatedByID: &actorID}, clean.Identity.MentionIDs)
	if err != nil {
		return preparedIssue{}, err
	}
	c.mentions, c.subscribers = mentions, subscribers
	sequence, err := u.d.Sequences.Next(ctx, projectID)
	if err != nil {
		return preparedIssue{}, err
	}
	greatest, err := u.d.Issues.GreatestSortOrder(ctx, projectID, c.state.ID)
	if err != nil {
		return preparedIssue{}, err
	}
	row := IssueRow{ID: uuid.NewV7(), WorkspaceID: project.WorkspaceID, ProjectID: projectID, SequenceID: sequence, Name: in.Name,
		StateID: c.state.ID, Priority: *in.Priority, StartDate: in.StartDate, TargetDate: in.TargetDate, ParentID: c.parent,
		DescriptionHTML: clean.HTML, DescriptionStripped: &clean.Text,
		SortOrder: domain.SortOrderAfter(greatest), By: actorID}
	return preparedIssue{row: row, checked: c, content: clean}, nil
}

// writePreparedAt writes p at the moment at, the write's clock read after
// its last lock: first, the cycle it joins may not have ended on the day
// at falls on (refuseEnded, M6 P3a spec R2), the one judgement of it for a
// creation and a draft's publication alike.
func (u *IssueCreation) writePreparedAt(ctx context.Context, p preparedIssue, at time.Time) (domain.Issue, error) {
	if err := refuseEnded(ctx, u.d.Planning, p.row.ProjectID, p.checked.cycle, at); err != nil {
		return domain.Issue{}, err
	}
	p.row.At, p.row.CompletedAt = at, domain.CompletedAt(p.checked.state.Group, at)
	if err := u.write(ctx, p.row, p.checked, p.content); err != nil {
		return domain.Issue{}, err
	}
	return readBack(ctx, u.d.Issues, p.row.ID)
}

// check checks the dependent rows of in, by the caller actor, under h, his
// write's lock on the project (M4 design 3.7, 3.9): first the cycle and the
// modules it joins, when it joins any, whose refusals come before any
// other's (readMembership, M6 P3a spec R2); the state place picks
// (placement): the one asked for, the project's and not its triage state,
// or the project's default state when none is; or, for a submission, the
// project's triage state, a project without which is an error, as one
// without a default state is; the assignees by domain.Assignees, from the
// roles of the accounts asked for, or of the project's default assignee
// when none is;
// each label asked for is the project's; the parent asked for may be one
// (parentOf); the cycle and each module are the project's, each module's
// problem at its place in module_ids. Every problem is reported at once,
// in one 422.
func (u *IssueCreation) check(ctx context.Context, h held, actor, projectID uuid.UUID, in domain.NewIssue, place placement) (checked, error) {
	p := h.project
	var c checked
	var problems []shared.FieldError
	var joining *domain.CycleChange
	if in.CycleID != nil {
		joining = &domain.CycleChange{To: in.CycleID}
	}
	joins, late, err := readMembership(ctx, u.d.Planning, projectID, joining, domain.Diff{Added: ascending(in.ModuleIDs)}, in.ModuleIDs)
	if err != nil {
		return checked{}, err
	}
	if joins.cycle != nil {
		c.cycle = joins.cycle.New
	}
	c.modules = joins.modulesJoined
	states, err := u.d.States.States(ctx, projectID)
	if err != nil {
		return checked{}, err
	}
	var found bool
	c.state, found = place.state(states, in.StateID)
	switch {
	case !found && place == inTriage:
		return checked{}, fmt.Errorf("project %s has no triage state", projectID)
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
	problems = append(problems, late...)
	if len(problems) > 0 {
		return checked{}, shared.Invalid(problems...)
	}
	return c, nil
}

// write writes the work item row and the rows under it, in the order of M4
// design 3.5: the work item, its assignees, its labels, its cycle's and its
// modules' (M6 P3a spec R4), its subscribers, its activity: the creation's
// rows, then the cycle and the modules it joins (domain.CreationJoins, R5).
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
	if c.cycle != nil {
		if err := u.d.Issues.InsertCycle(ctx, s.links([]uuid.UUID{c.cycle.ID})); err != nil {
			return err
		}
	}
	if len(c.modules) > 0 {
		if err := u.d.Issues.InsertModules(ctx, s.links(idsOf(c.modules))); err != nil {
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
	if err := u.d.Issues.InsertActivities(ctx, s.activity(slices.Concat(domain.CreationActivity(accounts), domain.CreationJoins(c.cycle, c.modules...)))); err != nil {
		return err
	}
	if err := u.d.Issues.InsertDescriptionVersion(ctx, descriptionVersion(s, clean)); err != nil {
		return err
	}
	return u.d.Events.Publish(ctx, s.event(domain.EventIssueCreated, createdChanges(row, c, accounts)))
}
