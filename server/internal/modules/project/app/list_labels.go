package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListLabels lists a project's labels: GET
// /api/v0/projects/{project_id}/labels.
type ListLabels struct {
	labels LabelLister
	auth   shared.Authorizer
}

// NewListLabels returns the use case.
func NewListLabels(labels LabelLister, auth shared.Authorizer) *ListLabels {
	return &ListLabels{labels: labels, auth: auth}
}

// Execute lists the project's labels, parents and children alike, by sort
// order, an archived project's as any other's (M3 design 3.16, 3.19), after
// the decision on label.list. A read opens no transaction.
func (u *ListLabels) Execute(ctx context.Context, projectID uuid.UUID) ([]domain.Label, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	if err := findAndDecide(ctx, u.labels, u.auth, actor, projectID, domain.ActionLabelList); err != nil {
		return nil, err
	}
	return u.labels.ListLabels(ctx, projectID)
}
