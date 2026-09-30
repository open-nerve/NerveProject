package httpadapter

import (
	"context"
	"time"
	"uuid"

	"github.com/oapi-codegen/nullable"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

// ListWorkspaceInvitations serves GET /api/v0/workspaces/{slug}/invitations.
func (h handler) ListWorkspaceInvitations(ctx context.Context, req gen.ListWorkspaceInvitationsRequestObject) (gen.ListWorkspaceInvitationsResponseObject, error) {
	list, err := h.uc.ListInvitations.Execute(ctx, req.Slug)
	if err != nil {
		return nil, err
	}
	return gen.ListWorkspaceInvitations200JSONResponse{Data: invitations(list)}, nil
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
