package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// InvitationImagePreviewer keeps the exact invitation valid under the parent
// lock. Writers may share the workspace, so the invitation itself needs a lock.
type InvitationImagePreviewer interface {
	InvitationPreviewer
	ShareWorkspace(context.Context, uuid.UUID) error
	ShareInvitationPreview(context.Context, uuid.UUID) (domain.InvitationPreview, error)
}
type GetInvitationDeps struct {
	Invitations InvitationImagePreviewer
	Images      ImageReader
	Tx          shared.TxManager
	MAC         InvitationMAC
}

// GetWorkspaceInvitation grants only the existing MAC link's fixed preview.
type GetWorkspaceInvitation struct {
	d      GetInvitationDeps
	tokens invitationTokens
}

func NewGetWorkspaceInvitation(d GetInvitationDeps) *GetWorkspaceInvitation {
	return &GetWorkspaceInvitation{d: d, tokens: invitationTokens{mac: d.MAC}}
}

func (u *GetWorkspaceInvitation) Execute(ctx context.Context, id uuid.UUID, token string) (domain.InvitationPreview, error) {
	if !u.tokens.valid(id, token) {
		return domain.InvitationPreview{}, domain.ErrInvitationNotFound
	}
	candidate, err := u.d.Invitations.InvitationPreview(ctx, id)
	if err != nil {
		return domain.InvitationPreview{}, invitationNotFound(err)
	}
	var out domain.InvitationPreview
	err = u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := u.d.Invitations.ShareWorkspace(ctx, candidate.WorkspaceID); err != nil {
			return invitationNotFound(err)
		}
		current, err := u.d.Invitations.ShareInvitationPreview(ctx, id)
		if err != nil {
			return invitationNotFound(err)
		}
		if current.WorkspaceID != candidate.WorkspaceID {
			return domain.ErrInvitationNotFound
		}
		images, err := u.d.Images.ReadWorkspaceImages(ctx, []WorkspaceImageReference{{WorkspaceID: current.WorkspaceID, LogoAssetID: current.LogoAssetID}})
		if err != nil {
			return err
		}
		current.Logo = nil
		if current.LogoAssetID != nil {
			if image, ok := images[*current.LogoAssetID]; ok {
				current.Logo = &image
			}
		}
		out = current
		return nil
	})
	if err != nil {
		return domain.InvitationPreview{}, err
	}
	return out, nil
}
