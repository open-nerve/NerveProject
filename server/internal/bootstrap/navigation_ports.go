package bootstrap

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/access"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// navigationProjects is project's NavigationProjects as navigation's
// Projects (M7 P3b spec 2.1, 2.6): the same read and lock, the facts
// converted.
type navigationProjects struct {
	projects project.NavigationProjects
}

func (p navigationProjects) Facts(ctx context.Context, id uuid.UUID) (navigation.ProjectFacts, bool, error) {
	f, found, err := p.projects.NavigationFacts(ctx, id)
	return navigation.ProjectFacts(f), found, err
}

func (p navigationProjects) Share(ctx context.Context, id uuid.UUID) (navigation.ProjectFacts, bool, error) {
	f, found, err := p.projects.ShareNavigation(ctx, id)
	return navigation.ProjectFacts(f), found, err
}

// navigationWorkspaces is workspace's WorkspaceDirectory as navigation's
// Workspaces: the same read and locks, the workspace's id alone.
type navigationWorkspaces struct {
	directory workspace.WorkspaceDirectory
}

func (d navigationWorkspaces) WorkspaceBySlug(ctx context.Context, slug string) (uuid.UUID, bool, error) {
	w, found, err := d.directory.WorkspaceBySlug(ctx, slug)
	return w.ID, found, err
}

func (d navigationWorkspaces) ShareWorkspaceBySlug(ctx context.Context, slug string) (uuid.UUID, bool, error) {
	w, found, err := d.directory.ShareWorkspaceBySlug(ctx, slug)
	return w.ID, found, err
}

func (d navigationWorkspaces) ShareWorkspaceByID(ctx context.Context, id uuid.UUID) (bool, error) {
	_, found, err := d.directory.ShareWorkspaceByID(ctx, id)
	return found, err
}

// navigationReadable is access's ReadableProjects as navigation's Readable
// (M7 P3c spec 2.5): the same projects, in their order, each grant
// converted.
type navigationReadable struct {
	readable *access.ReadableProjects
}

func (r navigationReadable) Readable(ctx context.Context, actor shared.Actor, workspaceID uuid.UUID) ([]navigation.ProjectGrant, error) {
	grants, err := r.readable.Readable(ctx, actor, workspaceID)
	if err != nil {
		return nil, err
	}
	out := make([]navigation.ProjectGrant, len(grants))
	for i, g := range grants {
		out[i] = navigation.ProjectGrant(g)
	}
	return out, nil
}

// navigationProjectTargets is project's NavigationTargets as navigation's
// ProjectTargets (spec 2.5): the same read, each target converted.
type navigationProjectTargets struct {
	targets project.NavigationTargets
}

func (p navigationProjectTargets) ProjectTargets(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]navigation.ProjectTarget, error) {
	targets, err := p.targets.NavigationTargets(ctx, ids)
	return convertedTargets(targets, err, func(t project.NavigationTarget) navigation.ProjectTarget { return navigation.ProjectTarget(t) })
}

// navigationPlanningTargets is planning's NavigationTargets as navigation's
// PlanningTargets (spec 2.5): the same reads, each target converted.
type navigationPlanningTargets struct {
	targets planning.NavigationTargets
}

func (p navigationPlanningTargets) Cycles(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]navigation.PlanningTarget, error) {
	targets, err := p.targets.NavigationCycles(ctx, ids)
	return convertedTargets(targets, err, planningTarget)
}

func (p navigationPlanningTargets) Modules(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]navigation.PlanningTarget, error) {
	targets, err := p.targets.NavigationModules(ctx, ids)
	return convertedTargets(targets, err, planningTarget)
}

func planningTarget(t planning.NavigationTarget) navigation.PlanningTarget {
	return navigation.PlanningTarget(t)
}

// navigationIssues is issue's NavigationIssues as navigation's Issues (M7
// P3d spec 2.5, R3): the same reads, the grants handed over and the
// answers each converted.
type navigationIssues struct {
	issues issue.NavigationIssues
}

func (i navigationIssues) IssuePlaces(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]navigation.IssuePlace, error) {
	places, err := i.issues.NavigationIssuePlaces(ctx, ids)
	return convertedTargets(places, err, func(p issue.NavigationIssuePlace) navigation.IssuePlace { return navigation.IssuePlace(p) })
}

func (i navigationIssues) IssueSummaries(ctx context.Context, reader uuid.UUID, grants []navigation.ProjectGrant,
	ids []uuid.UUID) (map[uuid.UUID]navigation.IssueSummary, error) {
	issueGrants := make([]issue.ProjectGrant, len(grants))
	for n, g := range grants {
		issueGrants[n] = issue.ProjectGrant(g)
	}
	summaries, err := i.issues.NavigationIssues(ctx, reader, issueGrants, ids)
	return convertedTargets(summaries, err, func(s issue.NavigationIssue) navigation.IssueSummary { return navigation.IssueSummary(s) })
}

// convertedTargets are targets, a read's answer by id, each converted by
// convert; err is the read's.
func convertedTargets[A, B any](targets map[uuid.UUID]A, err error, convert func(A) B) (map[uuid.UUID]B, error) {
	if err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]B, len(targets))
	for id, t := range targets {
		out[id] = convert(t)
	}
	return out, nil
}
