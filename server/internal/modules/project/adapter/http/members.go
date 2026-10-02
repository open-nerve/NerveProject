package httpadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

// ListProjectMembers serves GET /api/v0/projects/{project_id}/members.
func (h handler) ListProjectMembers(ctx context.Context, req gen.ListProjectMembersRequestObject) (gen.ListProjectMembersResponseObject, error) {
	list, err := h.uc.ListMembers.Execute(ctx, req.ProjectID)
	if err != nil {
		return nil, err
	}
	return gen.ListProjectMembers200JSONResponse(members(list)), nil
}

// members is list as the API shows it: data an array, never null.
func members(list []domain.Member) gen.ProjectMemberList {
	out := gen.ProjectMemberList{Data: make([]gen.ProjectMember, len(list))}
	for i, m := range list {
		out.Data[i] = gen.ProjectMember{ID: m.ID, ProjectID: m.ProjectID, MemberID: m.MemberID, Role: gen.ProjectRole(m.Role), CreatedAt: m.CreatedAt}
	}
	return out
}
