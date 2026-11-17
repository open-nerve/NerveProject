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
// module's Issues (M7 P1b spec 2.6): the same lock and reads, each value
// converted; the lock answers the work item's workspace and project.
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

// notificationProjectReads are the two reads of project's IssueProjects a
// notification's snapshot is completed with, neither of which locks: the
// fan-out, which holds a work item's row, takes no lock of a project, which
// the writes take before a work item's (spec 2.6).
type notificationProjectReads interface {
	IssueStates(ctx context.Context, projectID uuid.UUID) ([]project.IssueState, error)
	ProjectIdentifiers(ctx context.Context, workspaceID uuid.UUID) ([]project.IssueProjectIdentifier, error)
}

// notificationProjects is project's reads as the notification module's
// ProjectNames: a state among the project's undeleted states, its triage
// state too; a project's identifier among the workspace's undeleted
// projects, archived ones too.
type notificationProjects struct {
	projects notificationProjectReads
}

func (p notificationProjects) State(ctx context.Context, projectID, stateID uuid.UUID) (notification.StateName, bool, error) {
	states, err := p.projects.IssueStates(ctx, projectID)
	if err != nil {
		return notification.StateName{}, false, err
	}
	for _, s := range states {
		if s.ID == stateID {
			return notification.StateName{Name: s.Name, Group: s.Group}, true, nil
		}
	}
	return notification.StateName{}, false, nil
}

func (p notificationProjects) Identifier(ctx context.Context, workspaceID, projectID uuid.UUID) (string, bool, error) {
	projects, err := p.projects.ProjectIdentifiers(ctx, workspaceID)
	if err != nil {
		return "", false, err
	}
	for _, pr := range projects {
		if pr.ID == projectID {
			return pr.Identifier, true, nil
		}
	}
	return "", false, nil
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
