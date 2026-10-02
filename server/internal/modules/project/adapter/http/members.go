package httpadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListProjectMembers serves GET /api/v0/projects/{project_id}/members.
func (h handler) ListProjectMembers(ctx context.Context, req gen.ListProjectMembersRequestObject) (gen.ListProjectMembersResponseObject, error) {
	list, err := h.uc.ListMembers.Execute(ctx, req.ProjectID)
	if err != nil {
		return nil, err
	}
	return gen.ListProjectMembers200JSONResponse(members(list)), nil
}

// AddProjectMembers serves POST /api/v0/projects/{project_id}/members: the
// members go to the use case as given, a role outside the three too, which
// the domain refuses (422).
func (h handler) AddProjectMembers(ctx context.Context, req gen.AddProjectMembersRequestObject) (gen.AddProjectMembersResponseObject, error) {
	in := make([]domain.NewMember, len(req.Body.Members))
	for i, m := range req.Body.Members {
		in[i] = domain.NewMember{MemberID: m.MemberID, Role: shared.Role(m.Role)}
	}
	list, err := h.uc.AddMembers.Execute(ctx, req.ProjectID, in)
	if err != nil {
		return nil, err
	}
	return gen.AddProjectMembers201JSONResponse(members(list)), nil
}

// members is list as the API shows it: data an array, never null.
func members(list []domain.Member) gen.ProjectMemberList {
	out := gen.ProjectMemberList{Data: make([]gen.ProjectMember, len(list))}
	for i, m := range list {
		out.Data[i] = gen.ProjectMember{ID: m.ID, ProjectID: m.ProjectID, MemberID: m.MemberID, Role: gen.ProjectRole(m.Role), CreatedAt: m.CreatedAt}
	}
	return out
}
