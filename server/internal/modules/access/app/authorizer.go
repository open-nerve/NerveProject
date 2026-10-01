package app

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/access/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Authorizer implements shared.Authorizer (M3 design 3.4): every call reads
// the facts afresh, nothing is cached, so a removal or a change of role
// takes effect at the next request.
type Authorizer struct {
	roles    WorkspaceRoles
	projects ProjectAccess
}

// NewAuthorizer returns the Authorizer that reads through roles and
// projects.
func NewAuthorizer(roles WorkspaceRoles, projects ProjectAccess) *Authorizer {
	return &Authorizer{roles: roles, projects: projects}
}

// Authorize decides whether actor may do action on t. An action without a
// row in the rule table is an error before anything is read. The facts are
// the caller's membership of t's workspace and, when t names a project, the
// project's: a project that is not there, or is another workspace's than
// t's, is none, and a project-level rule sees nothing (M3 design 3.4).
func (a *Authorizer) Authorize(ctx context.Context, actor shared.Actor, action shared.Action, t shared.Target) (shared.Grant, error) {
	rule, ok := domain.RuleFor(action)
	if !ok {
		return shared.Grant{}, fmt.Errorf("access: no rule for action %q", action)
	}
	role, active, err := a.roles.ActiveRole(ctx, t.WorkspaceID, actor.UserID)
	if err != nil {
		return shared.Grant{}, fmt.Errorf("access: read the workspace role: %w", err)
	}
	facts := domain.Facts{Workspace: domain.Membership{Active: active, Role: role}}
	if t.ProjectID != (uuid.UUID{}) {
		p, found, err := a.projects.ProjectFacts(ctx, t.ProjectID, actor.UserID)
		if err != nil {
			return shared.Grant{}, fmt.Errorf("access: read the project: %w", err)
		}
		if found && p.WorkspaceID == t.WorkspaceID {
			facts.Project = &domain.Project{Public: p.Public, Member: domain.Membership{Active: p.Member, Role: p.Role}}
		}
	}
	return domain.Decide(rule, facts)
}
