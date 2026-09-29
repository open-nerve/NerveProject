package app

import (
	"context"
	"fmt"

	"github.com/open-nerve/NerveProject/server/internal/modules/access/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Authorizer implements shared.Authorizer (M3 design 3.4): every call reads
// the facts afresh, nothing is cached, so a removal or a change of role
// takes effect at the next request.
type Authorizer struct {
	roles WorkspaceRoles
}

// NewAuthorizer returns the Authorizer that reads through roles.
func NewAuthorizer(roles WorkspaceRoles) *Authorizer {
	return &Authorizer{roles: roles}
}

// Authorize decides whether actor may do action on t. An action without a
// row in the rule table is an error before anything is read. The facts are
// the caller's workspace membership; no project is read, since the rule
// table has no project-level row yet, so a project-level rule would see no
// project and answer shared.ErrNotVisible. The ProjectAccess port adds the
// project's facts with the projects (M3 design 6.5).
func (a *Authorizer) Authorize(ctx context.Context, actor shared.Actor, action shared.Action, t shared.Target) (shared.Grant, error) {
	rule, ok := domain.RuleFor(action)
	if !ok {
		return shared.Grant{}, fmt.Errorf("access: no rule for action %q", action)
	}
	role, active, err := a.roles.ActiveRole(ctx, t.WorkspaceID, actor.UserID)
	if err != nil {
		return shared.Grant{}, fmt.Errorf("access: read the workspace role: %w", err)
	}
	return domain.Decide(rule, domain.Facts{Workspace: domain.Membership{Active: active, Role: role}})
}
