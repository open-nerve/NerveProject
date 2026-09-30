package httpadapter

import (
	"context"
	"time"
	"uuid"

	"github.com/oapi-codegen/nullable"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListWorkspaceInvitations serves GET /api/v0/workspaces/{slug}/invitations.
func (h handler) ListWorkspaceInvitations(ctx context.Context, req gen.ListWorkspaceInvitationsRequestObject) (gen.ListWorkspaceInvitationsResponseObject, error) {
	list, err := h.uc.ListInvitations.Execute(ctx, req.Slug)
	if err != nil {
		return nil, err
	}
	return gen.ListWorkspaceInvitations200JSONResponse{Data: invitations(list)}, nil
}

// CreateWorkspaceInvitations serves POST
// /api/v0/workspaces/{slug}/invitations.
func (h handler) CreateWorkspaceInvitations(ctx context.Context, req gen.CreateWorkspaceInvitationsRequestObject) (gen.CreateWorkspaceInvitationsResponseObject, error) {
	batch := make([]domain.NewInvitation, len(req.Body.Invitations))
	for i, inv := range req.Body.Invitations {
		batch[i] = domain.NewInvitation{Email: inv.Email, Role: shared.Role(inv.Role)}
	}
	list, err := h.uc.CreateInvitations.Execute(ctx, req.Slug, batch)
	if err != nil {
		return nil, err
	}
	return gen.CreateWorkspaceInvitations201JSONResponse{Data: invitations(list)}, nil
}

// GetWorkspaceInvitation serves the public GET
// /api/v0/workspace-invitations/{invitation_id}.
func (h handler) GetWorkspaceInvitation(ctx context.Context, req gen.GetWorkspaceInvitationRequestObject) (gen.GetWorkspaceInvitationResponseObject, error) {
	p, err := h.uc.GetInvitation.Execute(ctx, req.InvitationID, req.Params.Token)
	if err != nil {
		return nil, err
	}
	return gen.GetWorkspaceInvitation200JSONResponse{
		ID:               p.ID,
		Role:             gen.WorkspaceRole(p.Role),
		Declined:         p.Declined,
		WorkspaceName:    p.WorkspaceName,
		WorkspaceSlug:    p.WorkspaceSlug,
		WorkspaceLogoURL: nullable.NewNullNullable[string](), // M5
	}, nil
}

// UpdateWorkspaceInvitation serves PATCH
// /api/v0/workspace-invitations/{invitation_id}.
func (h handler) UpdateWorkspaceInvitation(ctx context.Context, req gen.UpdateWorkspaceInvitationRequestObject) (gen.UpdateWorkspaceInvitationResponseObject, error) {
	inv, err := h.uc.UpdateInvitation.Execute(ctx, req.InvitationID, shared.Role(req.Body.Role))
	if err != nil {
		return nil, err
	}
	return gen.UpdateWorkspaceInvitation200JSONResponse(invitation(inv)), nil
}

// DeleteWorkspaceInvitation serves DELETE
// /api/v0/workspace-invitations/{invitation_id}.
func (h handler) DeleteWorkspaceInvitation(ctx context.Context, req gen.DeleteWorkspaceInvitationRequestObject) (gen.DeleteWorkspaceInvitationResponseObject, error) {
	if err := h.uc.DeleteInvitation.Execute(ctx, req.InvitationID); err != nil {
		return nil, err
	}
	return gen.DeleteWorkspaceInvitation204Response{}, nil
}

// AcceptWorkspaceInvitation serves POST
// /api/v0/workspace-invitations/{invitation_id}/accept.
func (h handler) AcceptWorkspaceInvitation(ctx context.Context, req gen.AcceptWorkspaceInvitationRequestObject) (gen.AcceptWorkspaceInvitationResponseObject, error) {
	w, err := h.uc.AcceptInvitation.Execute(ctx, req.InvitationID, req.Body.Token)
	if err != nil {
		return nil, err
	}
	return gen.AcceptWorkspaceInvitation200JSONResponse(workspace(w)), nil
}

// DeclineWorkspaceInvitation serves POST
// /api/v0/workspace-invitations/{invitation_id}/decline.
func (h handler) DeclineWorkspaceInvitation(ctx context.Context, req gen.DeclineWorkspaceInvitationRequestObject) (gen.DeclineWorkspaceInvitationResponseObject, error) {
	if err := h.uc.DeclineInvitation.Execute(ctx, req.InvitationID, req.Body.Token); err != nil {
		return nil, err
	}
	return gen.DeclineWorkspaceInvitation204Response{}, nil
}

// invitations is list as the API shows it.
func invitations(list []domain.InvitationWithToken) []gen.WorkspaceInvitation {
	out := make([]gen.WorkspaceInvitation, len(list))
	for i, inv := range list {
		out[i] = invitation(inv)
	}
	return out
}

// invitation is inv as the API shows it. Its nullable fields are required:
// the zero Nullable is "unspecified", so null is set explicitly.
func invitation(inv domain.InvitationWithToken) gen.WorkspaceInvitation {
	respondedAt := nullable.NewNullNullable[time.Time]()
	if inv.RespondedAt != nil {
		respondedAt = nullable.NewNullableWithValue(*inv.RespondedAt)
	}
	createdBy := nullable.NewNullNullable[uuid.UUID]()
	if inv.CreatedByID != nil {
		createdBy = nullable.NewNullableWithValue(*inv.CreatedByID)
	}
	return gen.WorkspaceInvitation{
		ID:          inv.ID,
		WorkspaceID: inv.WorkspaceID,
		Email:       inv.Email,
		Role:        gen.WorkspaceRole(inv.Role),
		Accepted:    inv.Accepted,
		RespondedAt: respondedAt,
		CreatedAt:   inv.CreatedAt,
		CreatedByID: createdBy,
		Token:       inv.Token,
	}
}
