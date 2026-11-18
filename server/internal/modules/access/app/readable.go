package app

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/access/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// readAction is the action Readable decides on each project: reading its
// work items (M7 P1c spec 2.5).
const readAction shared.Action = "issue.read"

// ReadableProjects answers which projects of a workspace a caller may read
// the work items of, and whether only his own (M7 P1c spec 2.5, R1; M7
// design 6.3): it reads his facts in one call to each port, and decides
// each project with the domain.Decide that Authorize calls, so the two
// cannot part. Nothing is cached.
type ReadableProjects struct {
	roles    WorkspaceRoles
	projects MemberProjectAccess
}

// NewReadableProjects returns the ReadableProjects that reads through roles
// and projects.
func NewReadableProjects(roles WorkspaceRoles, projects MemberProjectAccess) *ReadableProjects {
	return &ReadableProjects{roles: roles, projects: projects}
}

// Grant is Readable's answer for a project: the caller may read its work
// items, only those he created when OwnOnly, as Authorize's Grant says.
type Grant struct {
	ProjectID uuid.UUID
	OwnOnly   bool
}

// Readable answers, in project id order, the projects of the workspace
// workspaceID on which Authorize allows actor issue.read with the Target of
// that workspace and project, which names no creator, each OwnOnly as its
// Grant is; a project it refuses him is not among them. With his membership
// of the workspace it decides only the projects he is an active member of,
// archived or not: issue.read's rule allows no one else (memberRule). So the
// workspace's admin who is a member of none reads none, and a project's
// guest reads only his own work items unless the project shows its guests
// every one, by his project role, also when he is the workspace's admin.
func (r *ReadableProjects) Readable(ctx context.Context, actor shared.Actor, workspaceID uuid.UUID) ([]Grant, error) {
	rule, err := memberRule(readAction)
	if err != nil {
		return nil, err
	}
	role, active, err := r.roles.ActiveRole(ctx, workspaceID, actor.UserID)
	if err != nil {
		return nil, fmt.Errorf("access: read the workspace role: %w", err)
	}
	projects, err := r.projects.MemberProjectFacts(ctx, workspaceID, actor.UserID)
	if err != nil {
		return nil, fmt.Errorf("access: read the projects the caller is a member of: %w", err)
	}
	grants := make([]Grant, 0, len(projects))
	for _, p := range projects {
		grant, err := domain.Decide(rule, domain.Facts{Workspace: domain.Membership{Active: active, Role: role},
			Project: &domain.Project{Public: p.Public, Member: domain.Membership{Active: true, Role: p.Role}, GuestViewAll: p.GuestViewAll}})
		switch {
		case err == nil:
			grants = append(grants, Grant{ProjectID: p.ProjectID, OwnOnly: grant.OwnOnly})
		case errors.Is(err, shared.ErrNotVisible) || errors.Is(err, shared.Forbidden()):
		default:
			return nil, err
		}
	}
	return grants, nil
}

// memberRule is the rule of action when it allows the active members of the
// target project alone, alike on every work item of it: a project-level
// rule without AllowCreator. Any other is an error, so that Readable, which
// reads only the projects the caller is a member of and no creator, answers
// what Authorize would: a LevelVisible rule allows the workspace's admin,
// and its member on a public project, though neither is a member of it; a
// workspace-level rule names no project; and AllowCreator lets the creator
// of the work item change the decision.
func memberRule(action shared.Action) (domain.Rule, error) {
	rule, ok := domain.RuleFor(action)
	switch {
	case !ok:
		return domain.Rule{}, fmt.Errorf("access: no rule for action %q", action)
	case rule.Level != domain.LevelProject || rule.AllowCreator:
		return domain.Rule{}, fmt.Errorf("access: the rule of %q allows more than a project's active members, alike on each work item", action)
	}
	return rule, nil
}
