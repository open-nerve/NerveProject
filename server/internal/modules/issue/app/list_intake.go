package app

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// IntakeIssuePage is a page of a project's intake list (M7 P2b spec 2.3,
// DP-11): the number of work items its filters select, the cursor aside;
// its work items, each a list row with its intake row as the caller is
// answered it; and the cursor of the page after it, empty on the last.
type IntakeIssuePage struct {
	Total      int64
	Data       []domain.Issue
	NextCursor string
}

// ListIntakeIssuesDeps supplies the read snapshot, the decision and the
// reads Locks make, the intake list's store, the references of the work
// items a page's duplicates duplicate, and the clock.
type ListIntakeIssuesDeps struct {
	ReadTx shared.TxManager
	Locks  Locks
	Rows   IntakeLister
	Refs   *IssueRefs
	Clock  Clock
}

// ListIntakeIssues lists a project's intake: GET
// /api/v0/projects/{project_id}/intake-issues (M7 P2b spec 2.3).
type ListIntakeIssues struct{ d ListIntakeIssuesDeps }

// NewListIntakeIssues returns the use case.
func NewListIntakeIssues(d ListIntakeIssuesDeps) *ListIntakeIssues { return &ListIntakeIssues{d: d} }

// Execute parses raw (domain.ParseIntakeQuery), its 400 or 422 before any
// read; then, in one read snapshot, the project projectID read and the
// decision on intake.read on it (Locks.seeProject), archived or not and
// whatever its intake's toggle: a project not there or not seen is 404
// project.not_found, a role the rule does not allow 403; the clock, read
// once, the request's one moment; the page and the row after it, at that
// moment, the caller's grant on the project telling whether he sees only
// his own work items (IntakeLister.ListIntake); and the page's work items
// as he sees them then (issues). A cursor past the last row reads the
// total again without it, as the project list does (Listing.Read).
func (u *ListIntakeIssues) Execute(ctx context.Context, projectID uuid.UUID, raw map[string][]string) (IntakeIssuePage, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return IntakeIssuePage{}, err
	}
	q, err := domain.ParseIntakeQuery(raw, actor.UserID)
	if err != nil {
		return IntakeIssuePage{}, err
	}
	var page IntakeIssuePage
	err = u.d.ReadTx.WithinTx(ctx, func(ctx context.Context) error {
		grant, err := u.d.Locks.seeProject(ctx, actor, domain.ActionReadIntake, projectID, domain.ErrProjectNotFound)
		if err != nil {
			return err
		}
		page, err = u.page(ctx, IntakeListRead{ProjectID: projectID, ActorID: actor.UserID, OwnOnly: grant.OwnOnly, Query: q, Now: u.d.Clock.Now()})
		return err
	})
	if err != nil {
		return IntakeIssuePage{}, err
	}
	return page, nil
}

// page reads the page in asks for, in the read snapshot ctx carries: its
// total the store's, its work items those before the row after them
// (issues), its cursor the last one's position; none past the last row,
// its total read again then, one row asked for without the cursor.
func (u *ListIntakeIssues) page(ctx context.Context, in IntakeListRead) (IntakeIssuePage, error) {
	rows, err := u.d.Rows.ListIntake(ctx, in)
	if err != nil {
		return IntakeIssuePage{}, err
	}
	page := IntakeIssuePage{Data: make([]domain.Issue, 0)}
	if len(rows) == 0 {
		if in.Query.Cursor == nil {
			return page, nil
		}
		in.Query.Cursor, in.Query.Limit = nil, 1
		if rows, err = u.d.Rows.ListIntake(ctx, in); err != nil {
			return IntakeIssuePage{}, err
		}
		if len(rows) > 0 {
			page.Total = rows[0].Total
		}
		return page, nil
	}
	page.Total = rows[0].Total
	if len(rows) > in.Query.Limit {
		if page.NextCursor, err = domain.EncodeIntakeCursor(rows[in.Query.Limit-1].Position); err != nil {
			return IntakeIssuePage{}, err
		}
		rows = rows[:in.Query.Limit]
	}
	if page.Data, err = u.issues(ctx, in, rows); err != nil {
		return IntakeIssuePage{}, err
	}
	return page, nil
}

// issues are the work items of rows as the caller in.ActorID sees them,
// by his grant on the project, which shows him only his own when
// in.OwnOnly (R10): each as the project list answers its row (Listing),
// its sub-items as the store counted them, its parent none when the guest
// rule hides it from him (hideParents), its attachments counted
// (projectAttachments); and each with its intake row as his detail
// answers it at in.Now (intakeSeen): its effective status then, the moment
// given and the clock not read again (domain.IntakeFacts.AsOf), and the
// work item it duplicates only when that is an undeleted work item of the
// project he sees, the targets of the page read once (IssueRefs, which
// asks domain.Issue.HiddenFrom). A row without its intake row is an
// error: the store lists none.
func (u *ListIntakeIssues) issues(ctx context.Context, in IntakeListRead, rows []IntakeRecord) ([]domain.Issue, error) {
	items := make([]domain.Issue, len(rows))
	var guarded []int
	var targets []uuid.UUID
	for n, r := range rows {
		if r.Issue.Intake == nil {
			return nil, fmt.Errorf("work item %s listed in the intake of project %s without its intake row", r.Issue.ID, in.ProjectID)
		}
		f := r.Issue.Intake.AsOf(moment(in.Now).Now)
		items[n] = r.Issue
		items[n].Intake = &f
		if in.OwnOnly && r.Issue.ParentID != nil {
			guarded = append(guarded, n)
		}
		if f.DuplicateToID != nil {
			targets = append(targets, *f.DuplicateToID)
		}
	}
	if err := hideParents(ctx, u.d.Locks.issues, in.ActorID, items, guarded); err != nil {
		return nil, err
	}
	if err := projectAttachments(ctx, u.d.Locks.attachments, items); err != nil {
		return nil, err
	}
	seen, err := u.d.Refs.IssueRefs(ctx, in.ActorID, []ProjectGrant{{ProjectID: in.ProjectID, OwnOnly: in.OwnOnly}}, ascending(targets))
	if err != nil {
		return nil, err
	}
	for n := range items {
		if target := items[n].Intake.DuplicateToID; target != nil {
			if _, ok := seen[*target]; !ok {
				items[n].Intake.DuplicateToID = nil
			}
		}
	}
	return items, nil
}
