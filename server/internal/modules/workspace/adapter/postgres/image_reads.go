package postgresadapter

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ShareImageWorkspaces reuses the same ordered, undeleted-owner share query.
func (s *Store) ShareImageWorkspaces(ctx context.Context, ids []uuid.UUID) ([]uuid.UUID, error) {
	return s.ShareAssetWorkspaces(ctx, ids)
}

func (s *Store) ShareInvitationPreview(ctx context.Context, id uuid.UUID) (domain.InvitationPreview, error) {
	r, err := s.queries(ctx).ShareInvitationPreview(ctx, id)
	if err != nil {
		return domain.InvitationPreview{}, notFound(err)
	}
	return domain.InvitationPreview{WorkspaceID: r.WorkspaceID, LogoAssetID: r.LogoAssetID, ID: r.ID, Role: shared.Role(r.Role), Declined: r.RespondedAt != nil, WorkspaceName: r.WorkspaceName, WorkspaceSlug: r.WorkspaceSlug}, nil
}
