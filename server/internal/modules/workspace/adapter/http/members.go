package httpadapter

import (
	"context"

	"github.com/oapi-codegen/nullable"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListWorkspaceMembers serves GET /api/v0/workspaces/{slug}/members.
func (h handler) ListWorkspaceMembers(ctx context.Context, req gen.ListWorkspaceMembersRequestObject) (gen.ListWorkspaceMembersResponseObject, error) {
	list, err := h.uc.ListMembers.Execute(ctx, req.Slug)
	if err != nil {
		return nil, err
	}
	out := gen.ListWorkspaceMembers200JSONResponse{Data: make([]gen.WorkspaceMember, len(list))}
	for i, m := range list {
		out.Data[i] = member(m)
	}
	return out, nil
}

// LeaveWorkspace serves POST /api/v0/workspaces/{slug}/leave.
func (h handler) LeaveWorkspace(ctx context.Context, req gen.LeaveWorkspaceRequestObject) (gen.LeaveWorkspaceResponseObject, error) {
	if err := h.uc.Leave.Execute(ctx, req.Slug); err != nil {
		return nil, err
	}
	return gen.LeaveWorkspace204Response{}, nil
}

// UpdateWorkspaceMember serves PATCH /api/v0/workspace-members/{workspace_member_id}.
func (h handler) UpdateWorkspaceMember(ctx context.Context, req gen.UpdateWorkspaceMemberRequestObject) (gen.UpdateWorkspaceMemberResponseObject, error) {
	m, err := h.uc.UpdateMember.Execute(ctx, req.WorkspaceMemberID, shared.Role(req.Body.Role))
	if err != nil {
		return nil, err
	}
	return gen.UpdateWorkspaceMember200JSONResponse(member(m)), nil
}

// RemoveWorkspaceMember serves DELETE /api/v0/workspace-members/{workspace_member_id}.
func (h handler) RemoveWorkspaceMember(ctx context.Context, req gen.RemoveWorkspaceMemberRequestObject) (gen.RemoveWorkspaceMemberResponseObject, error) {
	if err := h.uc.RemoveMember.Execute(ctx, req.WorkspaceMemberID); err != nil {
		return nil, err
	}
	return gen.RemoveWorkspaceMember204Response{}, nil
}

// member is m as the API shows it.
func member(m domain.Member) gen.WorkspaceMember {
	// Required and nullable: the zero Nullable is "unspecified" and would
	// marshal as "", so null is set explicitly.
	email := nullable.NewNullNullable[string]()
	if m.User.Email != nil {
		email = nullable.NewNullableWithValue(*m.User.Email)
	}
	return gen.WorkspaceMember{
		ID:          m.ID,
		WorkspaceID: m.WorkspaceID,
		Role:        gen.WorkspaceRole(m.Role),
		IsActive:    m.IsActive,
		CreatedAt:   m.CreatedAt,
		Member: gen.MemberUser{
			ID:          m.User.ID,
			DisplayName: m.User.DisplayName,
			FirstName:   m.User.FirstName,
			LastName:    m.User.LastName,
			AvatarURL:   nullable.NewNullNullable[string](), // M5
			Email:       email,
		},
	}
}
