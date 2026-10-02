package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListProjectMembers lists a project's members: GET
// /api/v0/projects/{project_id}/members.
type ListProjectMembers struct {
	members MemberLister
	auth    shared.Authorizer
}

// NewListProjectMembers returns the use case.
func NewListProjectMembers(members MemberLister, auth shared.Authorizer) *ListProjectMembers {
	return &ListProjectMembers{members: members, auth: auth}
}

// Execute lists the project's active members, in the order they became
// members (M3 design 3.12), after the decision on project_member.list. A
// read opens no transaction.
func (u *ListProjectMembers) Execute(ctx context.Context, projectID uuid.UUID) ([]domain.Member, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	if err := findAndDecide(ctx, u.members, u.auth, actor, projectID, domain.ActionMemberList); err != nil {
		return nil, err
	}
	return u.members.ListMembers(ctx, projectID)
}
