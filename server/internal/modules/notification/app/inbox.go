package app

import (
	"context"
	"errors"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/notification/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// InboxDeps open a receiver's inbox in a workspace (P1c spec 2.2, 2.5):
// the workspace by its slug, the decision of the action on it, the
// projects he reads in it, and the business clock.
type InboxDeps struct {
	Workspaces WorkspaceDirectory
	Authorizer shared.Authorizer
	Readable   ReadableProjects
	Clock      Clock
}

// open is actor's inbox in the workspace slug names, for action, at the
// moment it reads: the workspace's id, the projects he reads in it, each
// once, and the clock's moment, read after them. A slug that names none,
// and a workspace in which Authorize does not let him see, answer
// ErrWorkspaceNotFound, the same value, and read nothing more; another
// refusal is the Authorizer's. Each read is in the transaction ctx
// carries.
func (d InboxDeps) open(ctx context.Context, actor shared.Actor, slug string, action shared.Action) (Inbox, error) {
	workspaceID, found, err := d.Workspaces.WorkspaceBySlug(ctx, slug)
	if err != nil {
		return Inbox{}, err
	}
	if !found {
		return Inbox{}, domain.ErrWorkspaceNotFound
	}
	_, err = d.Authorizer.Authorize(ctx, actor, action, shared.Target{WorkspaceID: workspaceID})
	if errors.Is(err, shared.ErrNotVisible) {
		return Inbox{}, domain.ErrWorkspaceNotFound
	}
	if err != nil {
		return Inbox{}, err
	}
	grants, err := d.Readable.Readable(ctx, actor, workspaceID)
	if err != nil {
		return Inbox{}, err
	}
	return Inbox{ReceiverID: actor.UserID, WorkspaceID: workspaceID, Grants: grants, Now: d.Clock.Now()}, nil
}

// NotificationView is a notification as its receiver is answered (spec
// 2.4): its work item, its sender and the account that sent it, nil for
// nerve, its data's view, its three states and its event's moment.
type NotificationView struct {
	ID, WorkspaceID, ProjectID, IssueID uuid.UUID
	Sender                              domain.Sender
	TriggeredByID                       *uuid.UUID
	Data                                domain.View
	State                               domain.State
	CreatedAt                           time.Time
}

// viewsOf are the notifications of page as reader is answered them, in
// their order, none for none (spec 2.4, 2.6, R7): the work items their
// changes name, asked for once with the projects of grants and only when
// they name any, in the transaction ctx carries.
func viewsOf(ctx context.Context, refs IssueRefs, reader uuid.UUID, grants []Grant, page []Notification) ([]NotificationView, error) {
	data := make([]domain.Data, len(page))
	for n, row := range page {
		data[n] = row.Data
	}
	var named map[uuid.UUID]domain.IssueRef
	if ids := domain.RefIDs(data); len(ids) > 0 {
		var err error
		if named, err = refs.IssueRefs(ctx, reader, grants, ids); err != nil {
			return nil, err
		}
	}
	views := make([]NotificationView, len(page))
	for n, row := range page {
		views[n] = NotificationView{ID: row.ID, WorkspaceID: row.WorkspaceID, ProjectID: row.ProjectID, IssueID: row.IssueID, Sender: row.Sender,
			TriggeredByID: row.TriggeredByID, Data: domain.ViewOf(row.Data, named), State: row.State, CreatedAt: row.CreatedAt}
	}
	return views, nil
}
