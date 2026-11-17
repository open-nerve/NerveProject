package app

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/access/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Decider decides an action for many accounts at once (M7 P1b spec 2.7, M7
// design 3.3): it reads the facts of every account in one call to each
// port, and decides each account's with the domain.Decide that Authorize
// calls, so the two cannot part (R7). Nothing is cached.
type Decider struct {
	roles    WorkspaceRolesMany
	projects ProjectAccessMany
}

// NewDecider returns the Decider that reads through roles and projects.
func NewDecider(roles WorkspaceRolesMany, projects ProjectAccessMany) *Decider {
	return &Decider{roles: roles, projects: projects}
}

// Decision is DecideMany's answer for an account: Allowed when Authorize
// would answer him no error, and OwnOnly then as its Grant's: he sees only
// the work items he created. A refused account's is the zero Decision.
type Decision struct {
	Allowed bool
	OwnOnly bool
}

// DecideMany decides action on the project projectID of the workspace
// workspaceID for each of accountIDs, as Authorize decides it for each with
// the Target of that workspace and project, which names no creator: from
// each account's membership of the workspace and the project's facts, a
// project that is not there, or is another workspace's, being none. An
// action without a rule, and one whose rule allows the creator
// (AllowCreator), are errors before anything is read: the creator, whom
// DecideMany does not know, would change the decision. Every account asked
// is answered, once.
func (d *Decider) DecideMany(ctx context.Context, action shared.Action, workspaceID, projectID uuid.UUID,
	accountIDs []uuid.UUID) (map[uuid.UUID]Decision, error) {
	rule, ok := domain.RuleFor(action)
	if !ok {
		return nil, fmt.Errorf("access: no rule for action %q", action)
	}
	if rule.AllowCreator {
		return nil, fmt.Errorf("access: the rule of %q allows the creator, whom a decision for many accounts does not know", action)
	}
	roles, err := d.roles.ActiveRoles(ctx, workspaceID, accountIDs)
	if err != nil {
		return nil, fmt.Errorf("access: read the workspace roles: %w", err)
	}
	p, found, err := d.projects.ProjectFactsMany(ctx, projectID, accountIDs)
	if err != nil {
		return nil, fmt.Errorf("access: read the project: %w", err)
	}
	inWorkspace := found && p.WorkspaceID == workspaceID
	decisions := make(map[uuid.UUID]Decision, len(accountIDs))
	for _, id := range accountIDs {
		role, active := roles[id]
		facts := domain.Facts{Workspace: domain.Membership{Active: active, Role: role}}
		if inWorkspace {
			projectRole, member := p.Roles[id]
			facts.Project = &domain.Project{Public: p.Public, Member: domain.Membership{Active: member, Role: projectRole},
				GuestViewAll: p.GuestViewAll}
		}
		grant, err := domain.Decide(rule, facts)
		switch {
		case err == nil:
			decisions[id] = Decision{Allowed: true, OwnOnly: grant.OwnOnly}
		case errors.Is(err, shared.ErrNotVisible) || errors.Is(err, shared.Forbidden()):
			decisions[id] = Decision{}
		default:
			return nil, err
		}
	}
	return decisions, nil
}
