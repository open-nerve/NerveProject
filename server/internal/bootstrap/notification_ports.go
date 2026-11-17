package bootstrap

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/access"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// notificationIssues is issue's NotificationIssues as the notification
// module's Issues (M7 P1b spec 2.4, 2.6): the same lock, reads and guest
// rule, each value converted; the lock answers the work item's workspace
// and project.
type notificationIssues struct {
	issues issue.NotificationIssues
}

func (n notificationIssues) ProjectIssuesRemain(ctx context.Context, projectID uuid.UUID) (bool, error) {
	return n.issues.ProjectIssuesRemain(ctx, projectID)
}

func (n notificationIssues) ShareIssue(ctx context.Context, id uuid.UUID) (notification.LockedIssue, bool, error) {
	i, found, err := n.issues.ShareIssue(ctx, id)
	return notification.LockedIssue{WorkspaceID: i.WorkspaceID, ProjectID: i.ProjectID}, found, err
}

func (n notificationIssues) NotificationIssue(ctx context.Context, id uuid.UUID) (notification.IssueColumns, bool, error) {
	i, found, err := n.issues.NotificationIssue(ctx, id)
	return notification.IssueColumns(i), found, err
}

func (n notificationIssues) SubscriberIDs(ctx context.Context, id uuid.UUID) ([]uuid.UUID, error) {
	return n.issues.SubscriberIDs(ctx, id)
}

func (n notificationIssues) CommentText(ctx context.Context, id uuid.UUID) (string, bool, error) {
	return n.issues.CommentText(ctx, id)
}

func (n notificationIssues) HiddenFrom(creator *uuid.UUID, account uuid.UUID, ownOnly bool) bool {
	return n.issues.HiddenFrom(creator, account, ownOnly)
}

// notificationProjects is project's NotificationNames as the notification
// module's ProjectNames (spec 2.5, 2.6): the same reads, each alone and
// without a lock, each value converted.
type notificationProjects struct {
	names project.NotificationNames
}

func (p notificationProjects) State(ctx context.Context, projectID, stateID uuid.UUID) (notification.StateName, bool, error) {
	s, found, err := p.names.NotificationState(ctx, projectID, stateID)
	return notification.StateName(s), found, err
}

func (p notificationProjects) Identifier(ctx context.Context, workspaceID, projectID uuid.UUID) (string, bool, error) {
	return p.names.NotificationIdentifier(ctx, workspaceID, projectID)
}

// notificationReaders is access's Decider as the notification module's
// Readers (spec 2.7): the same decisions, each converted.
type notificationReaders struct {
	decider *access.Decider
}

func (r notificationReaders) DecideMany(ctx context.Context, action shared.Action, workspaceID, projectID uuid.UUID, accountIDs []uuid.UUID) (map[uuid.UUID]notification.Decision, error) {
	decisions, err := r.decider.DecideMany(ctx, action, workspaceID, projectID, accountIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]notification.Decision, len(decisions))
	for id, d := range decisions {
		out[id] = notification.Decision(d)
	}
	return out, nil
}
