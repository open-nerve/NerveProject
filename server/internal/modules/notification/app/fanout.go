package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/notification/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// recheck is the action the recipients of a work item's notifications are
// decided on when they are sent (spec 2.4).
const recheck shared.Action = "issue.read"

// FanoutDeps supply the fan-out's lock, reads and insert, and its
// transactions: read-write, READ COMMITTED, each lock wait bounded as a
// work item's write's (spec 2.9).
type FanoutDeps struct {
	Lock      IssueRowLock
	Issues    IssueReads
	Projects  ProjectNames
	Readers   Readers
	GuestRule GuestRule
	Rows      NotificationWriter
	Tx        shared.TxManager
	Logger    *slog.Logger
}

// Fanout sends the notifications of an event of a work item (M7 P1b spec
// 2.4, 2.9): in one transaction, under the work item's row FOR SHARE, the
// first and only lock it takes, to its recipients the recheck allows, in
// one insert.
type Fanout struct{ d FanoutDeps }

// NewFanout constructs the fan-out.
func NewFanout(d FanoutDeps) *Fanout { return &Fanout{d: d} }

// Execute sends the notifications of the event e. An event that does not
// name its work item, its id, its moment or a real actor is refused. The
// work item is locked first (ShareIssue), and one deleted writes nothing;
// then only reads that take no lock (spec 2.6): the work item's columns and
// subscribers, the recipients of what e notifies of (domain.Recipients),
// the names of its state and its project, the recheck of issue.read on its
// row's workspace and project, and the text of a comment e is of; and the
// insert of the rows the recheck allows, by the issue module's guest rule
// over the creator the work item's columns name (domain.Visible), each of
// e's change at its position, by e's actor at e's moment, in the workspace
// and the project of the work item's row, which an event naming others is
// logged for, but a relation's naming another project alone, which it may
// (domain.MayNameAnotherProject). A project deleted since, whose state or
// identifier is not read any more, is read by no one: nothing is written.
// A row already sent is not written again, so a retry writes none twice.
func (u *Fanout) Execute(ctx context.Context, e shared.Event) error {
	item, err := workItemOf(e)
	if err != nil {
		return err
	}
	return u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		row, found, err := u.d.Lock.ShareIssue(ctx, item)
		if err != nil || !found {
			return err
		}
		if row.WorkspaceID != e.WorkspaceID || (row.ProjectID != e.ProjectID && !domain.MayNameAnotherProject(e)) {
			u.d.Logger.WarnContext(ctx, "notification event names another workspace or project than its work item's row, which is taken",
				slog.String("event", e.ID.String()), slog.String("work_item", item.String()),
				slog.String("event_workspace", e.WorkspaceID.String()), slog.String("event_project", e.ProjectID.String()),
				slog.String("workspace", row.WorkspaceID.String()), slog.String("project", row.ProjectID.String()))
		}
		columns, found, err := u.d.Issues.NotificationIssue(ctx, item)
		if err != nil {
			return err
		}
		if !found {
			return fmt.Errorf("work item %s is locked but not read", item)
		}
		subscribers, err := u.d.Issues.SubscriberIDs(ctx, item)
		if err != nil {
			return err
		}
		issue := domain.Issue{CreatedByID: columns.CreatedByID, AssigneeIDs: columns.AssigneeIDs}
		rows := domain.Recipients(domain.NotifiableOf(e), e.ActorID, subscribers, issue)
		if len(rows) == 0 {
			return nil
		}
		snapshot, found, err := u.snapshot(ctx, item, row, columns)
		if err != nil || !found {
			return err
		}
		decisions, err := u.d.Readers.DecideMany(ctx, recheck, row.WorkspaceID, row.ProjectID, domain.Receivers(rows))
		if err != nil {
			return err
		}
		rows = domain.Visible(rows, decisions, issue, u.d.GuestRule.HiddenFrom)
		if len(rows) == 0 {
			return nil
		}
		text, err := u.commentText(ctx, rows)
		if err != nil {
			return err
		}
		n := Notifications{WorkspaceID: row.WorkspaceID, ProjectID: row.ProjectID, IssueID: item, EventID: e.ID,
			IssueCreatedByID: columns.CreatedByID, TriggeredByID: e.ActorID, At: e.OccurredAt, Rows: make([]NotificationRow, len(rows))}
		for i, r := range rows {
			seq, err := changeSeq(r.Item.Seq)
			if err != nil {
				return err
			}
			n.Rows[i] = NotificationRow{ID: uuid.NewV7(), ReceiverID: r.Receiver, Sender: r.Sender, ChangeSeq: seq,
				Data: domain.NewData(snapshot, r.Item, text)}
		}
		return u.d.Rows.InsertNotifications(ctx, n)
	})
}

// workItemOf is the work item the event e is about (domain.WorkItem), once
// e names it, its id, its moment and, unless nerve wrote it, a real actor:
// the shape the fan-out runs on, and the subscriber enqueues.
func workItemOf(e shared.Event) (uuid.UUID, error) {
	item, ok := domain.WorkItem(e)
	if !ok || e.ID == (uuid.UUID{}) || e.OccurredAt.IsZero() || (e.ActorID != nil && *e.ActorID == (uuid.UUID{})) {
		return uuid.Nil(), errors.New("invalid notification event shape")
	}
	return item, nil
}

// snapshot is the work item item as its notifications show it: its own
// columns, and the names of its state and its project; found is false when
// either is not there any more, its project deleted.
func (u *Fanout) snapshot(ctx context.Context, item uuid.UUID, row LockedIssue, columns IssueColumns) (domain.IssueSnapshot, bool, error) {
	state, found, err := u.d.Projects.State(ctx, row.ProjectID, columns.StateID)
	if err != nil || !found {
		return domain.IssueSnapshot{}, false, err
	}
	identifier, found, err := u.d.Projects.Identifier(ctx, row.WorkspaceID, row.ProjectID)
	if err != nil || !found {
		return domain.IssueSnapshot{}, false, err
	}
	return domain.IssueSnapshot{ID: item, Name: columns.Name, SequenceID: columns.SequenceID, Identifier: identifier,
		StateName: state.Name, StateGroup: state.Group}, true, nil
}

// commentText is the stored text of the comment rows are of, which they
// keep an excerpt of (domain.NewData): nil when they are of none, or the
// comment is deleted.
func (u *Fanout) commentText(ctx context.Context, rows []domain.Row) (*string, error) {
	for _, r := range rows {
		if r.Item.CommentID == nil {
			continue
		}
		text, found, err := u.d.Issues.CommentText(ctx, *r.Item.CommentID)
		if err != nil || !found {
			return nil, err
		}
		return &text, nil
	}
	return nil, nil
}

// changeSeq is the position seq of a change as notifications.change_seq, a
// smallint from domain.CreationSeq on (R4); a position past it fails.
func changeSeq(seq int) (int16, error) {
	if seq < domain.CreationSeq || seq > math.MaxInt16 {
		return 0, fmt.Errorf("change position %d is out of notifications.change_seq's range", seq)
	}
	return int16(seq), nil
}
