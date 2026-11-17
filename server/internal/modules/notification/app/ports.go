package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/notification/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// DeletionStamp is a deletion's moment and deleter, which every row it
// deletes is stamped with: By is nil for nerve's own deletion, which no
// account asked for (M7 P1b spec 2.2, 2.8).
type DeletionStamp struct {
	At time.Time
	By *uuid.UUID
}

// IssueNotifications soft-deletes, in the transaction ctx carries, the
// undeleted notifications of the work item issueID of the workspace, locked
// FOR NO KEY UPDATE in id order before it writes them; a notification
// deleted before keeps its deletion (spec 2.8).
type IssueNotifications interface {
	DeleteIssueNotifications(ctx context.Context, workspaceID, issueID uuid.UUID, stamp DeletionStamp) error
}

// RowBatch is one batch of a deleted project's notifications: at most Limit
// of its undeleted ones whose ids are greater than After, in id order,
// soft-deleted at At by By.
type RowBatch struct {
	ProjectID, After, By uuid.UUID
	Limit                int
	At                   time.Time
}

// ProjectNotifications soft-deletes a batch of a deleted project's
// notifications in the transaction ctx carries, locked FOR NO KEY UPDATE in
// id order before it writes them, and answers their ids (spec 2.8).
type ProjectNotifications interface {
	DeleteProjectNotifications(ctx context.Context, b RowBatch) ([]uuid.UUID, error)
}

// ProjectIssues tells, without a lock, whether a project has an undeleted
// work item: the issue module's NotificationIssues (spec 2.6), which
// bootstrap hands over.
type ProjectIssues interface {
	ProjectIssuesRemain(ctx context.Context, projectID uuid.UUID) (bool, error)
}

// RowDeletions inserts the deletion of a deleted project's notifications in
// the publishing transaction, ctx's.
type RowDeletions interface {
	Enqueue(ctx context.Context, projectID uuid.UUID, at time.Time, by uuid.UUID) error
}

// Fanouts enqueue the fan-outs of events' notifications in the publishing
// transaction, ctx's, all at once (spec 2.9, R12): a job for each event, in
// their order, which carries it as it is handed over and rebuilds it to run
// the fan-out; none for none.
type Fanouts interface {
	Enqueue(ctx context.Context, events []shared.Event) error
}

// IssueRowLock locks a work item's row, the fan-out's one lock (spec 2.6,
// 2.9): the issue module's ShareIssue, which bootstrap converts. It locks
// the undeleted work item id, archived or not, FOR SHARE, its row alone, in
// the transaction ctx carries, and answers it as its row has it; found is
// false when there is none, also when it was deleted while the lock
// waited. A work item's deletion's UPDATE of the row waits for the lock,
// and the lock for it.
type IssueRowLock interface {
	ShareIssue(ctx context.Context, id uuid.UUID) (i LockedIssue, found bool, err error)
}

// LockedIssue is a work item as its locked row has it: its workspace and
// its project, which its notifications are of and the recheck decides on.
type LockedIssue struct {
	WorkspaceID, ProjectID uuid.UUID
}

// IssueReads are the issue module's reads of a work item the fan-out makes
// after its row's lock, none of which locks (spec 2.6): bootstrap converts
// its NotificationIssues. Each runs in the transaction ctx carries.
type IssueReads interface {
	// NotificationIssue reads the undeleted work item's own columns,
	// archived or not; found is false when there is none.
	NotificationIssue(ctx context.Context, issueID uuid.UUID) (i IssueColumns, found bool, err error)
	// SubscriberIDs reads the accounts of the work item's undeleted
	// subscribers, each once, in id order, none for none.
	SubscriberIDs(ctx context.Context, issueID uuid.UUID) ([]uuid.UUID, error)
	// CommentText reads the plain text the undeleted comment stores; found
	// is false when there is none.
	CommentText(ctx context.Context, commentID uuid.UUID) (text string, found bool, err error)
}

// IssueColumns are a work item's own columns its notifications keep (spec
// 2.4, 2.5): its name, its number, its state by id, its creator, nil when
// not known, and the accounts of its current assignees, each once, in id
// order.
type IssueColumns struct {
	Name        string
	SequenceID  int
	StateID     uuid.UUID
	CreatedByID *uuid.UUID
	AssigneeIDs []uuid.UUID
}

// ProjectNames complete a work item's snapshot with its project's names
// (spec 2.5, 2.6): the project module's reads, which bootstrap converts.
// Neither locks: the fan-out takes no lock of a project or a workspace,
// whose writes lock the project's row before a work item's (spec 2.6).
type ProjectNames interface {
	// State reads the name and the group of the project's undeleted state
	// stateID; found is false when there is none.
	State(ctx context.Context, projectID, stateID uuid.UUID) (s StateName, found bool, err error)
	// Identifier reads the identifier of the workspace's undeleted project
	// projectID, archived or not; found is false when there is none.
	Identifier(ctx context.Context, workspaceID, projectID uuid.UUID) (identifier string, found bool, err error)
}

// StateName is a state as a notification shows it: its name and its group.
type StateName struct {
	Name, Group string
}

// Readers decide, for many accounts at once and at the moment it asks,
// whether each may do the action on the project of the workspace, and
// whether only on the work items he created (spec 2.7): the access
// module's DecideMany, which bootstrap converts. Every account asked is
// answered; it reads without a lock.
type Readers interface {
	DecideMany(ctx context.Context, action shared.Action, workspaceID, projectID uuid.UUID, accountIDs []uuid.UUID) (map[uuid.UUID]domain.Decision, error)
}

// GuestRule is the guest rule the recheck applies (spec 2.4, M4 design
// 3.4): whether a work item whose creator is creator, nil when not known, is
// hidden from the account, whose grant on its project is ownOnly. It is the
// issue module's one predicate, Issue.HiddenFrom, which its
// NotificationIssues hands over and bootstrap converts; it reads nothing.
type GuestRule interface {
	HiddenFrom(creator *uuid.UUID, account uuid.UUID, ownOnly bool) bool
}

// NotificationWriter inserts a fan-out's notifications, in one statement in
// the transaction ctx carries (spec 2.9): a notification of the same event,
// change and receiver as an undeleted one is not inserted, so a retried or
// a repeated job writes none twice (R4).
type NotificationWriter interface {
	InsertNotifications(ctx context.Context, n Notifications) error
}

// Notifications are the notifications a fan-out sends of an event: all of
// the work item IssueID, whose creator is IssueCreatedByID, of its
// workspace and its project, sent by TriggeredByID, nil for nerve's own
// write, of the event EventID, at At, the event's moment; each Row its
// receiver's notification of one change.
type Notifications struct {
	WorkspaceID, ProjectID, IssueID, EventID uuid.UUID
	IssueCreatedByID, TriggeredByID          *uuid.UUID
	At                                       time.Time
	Rows                                     []NotificationRow
}

// NotificationRow is a notification of Notifications: its id, its
// receiver, why he receives it, the position of its change in the event's
// changes (CreationSeq for the creation itself) and its snapshot.
type NotificationRow struct {
	ID, ReceiverID uuid.UUID
	Sender         domain.Sender
	ChangeSeq      int16
	Data           domain.Data
}
