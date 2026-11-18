package bootstrap

import (
	"context"
	"maps"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/access"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
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

// notificationWorkspaces is workspace's WorkspaceDirectory as the
// notification module's (M7 P1c spec 2.2): the same lookup by slug,
// without a lock, answering the workspace's id.
type notificationWorkspaces struct {
	directory workspace.WorkspaceDirectory
}

func (d notificationWorkspaces) WorkspaceBySlug(ctx context.Context, slug string) (uuid.UUID, bool, error) {
	w, found, err := d.directory.WorkspaceBySlug(ctx, slug)
	return w.ID, found, err
}

// notificationReadable is access's ReadableProjects as the notification
// module's (spec 2.5, R1): the same projects, in their order, each grant
// converted.
type notificationReadable struct {
	readable *access.ReadableProjects
}

func (r notificationReadable) Readable(ctx context.Context, actor shared.Actor, workspaceID uuid.UUID) ([]notification.Grant, error) {
	grants, err := r.readable.Readable(ctx, actor, workspaceID)
	if err != nil {
		return nil, err
	}
	out := make([]notification.Grant, len(grants))
	for i, g := range grants {
		out[i] = notification.Grant(g)
	}
	return out, nil
}

// notificationIssueRefs is issue's IssueRefs and project's
// NotificationIdentifiers composed as the notification module's IssueRefs
// (M7 P1c spec 2.6, R4; pre-flight re-84 m6): what it names the work items
// a page of notifications names with, its changes' parents and relations'
// other ends, in one IssueRefs of the page's ids, with the reader's grants
// converted, and, only when it answered any, one read of the identifiers
// of their projects, each project once.
type notificationIssueRefs struct {
	issues   issue.IssueRefs
	projects project.NotificationIdentifiers
}

// IssueRefs answers, by id, those of the work items ids that reader sees
// now by grants, as issue's IssueRefs answers them, each named with its
// project's identifier; one whose project has none, deleted since it was
// read, is not among them. None for none.
func (r notificationIssueRefs) IssueRefs(ctx context.Context, reader uuid.UUID, grants []notification.Grant, ids []uuid.UUID) (map[uuid.UUID]notification.IssueRef, error) {
	projectGrants := make([]issue.ProjectGrant, len(grants))
	for i, g := range grants {
		projectGrants[i] = issue.ProjectGrant(g)
	}
	refs, err := r.issues.IssueRefs(ctx, reader, projectGrants, ids)
	if err != nil || len(refs) == 0 {
		return nil, err
	}
	projects := make(map[uuid.UUID]struct{}, len(refs))
	for _, ref := range refs {
		projects[ref.ProjectID] = struct{}{}
	}
	identifiers, err := r.projects.NotificationIdentifiers(ctx, slices.SortedFunc(maps.Keys(projects), uuid.UUID.Compare))
	if err != nil {
		return nil, err
	}
	named := make(map[uuid.UUID]notification.IssueRef, len(refs))
	for id, ref := range refs {
		if identifier, ok := identifiers[ref.ProjectID]; ok {
			named[id] = notification.IssueRef{ID: ref.ID, SequenceID: ref.SequenceID, Identifier: identifier, Name: ref.Name}
		}
	}
	return named, nil
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
