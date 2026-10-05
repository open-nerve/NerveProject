package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListStates lists a project's states: GET
// /api/v0/projects/{project_id}/states.
type ListStates struct {
	states StateLister
	auth   shared.Authorizer
}

// NewListStates returns the use case.
func NewListStates(states StateLister, auth shared.Authorizer) *ListStates {
	return &ListStates{states: states, auth: auth}
}

// Execute lists the project's states but its triage state, by sequence,
// none while it is archived (M3 design 3.12, 3.17), after the decision on
// state.list. A read opens no transaction.
func (u *ListStates) Execute(ctx context.Context, projectID uuid.UUID) ([]domain.State, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	if err := findAndDecide(ctx, u.states, u.auth, actor, projectID, domain.ActionStateList); err != nil {
		return nil, err
	}
	return u.states.ListStates(ctx, projectID)
}
