package app

import (
	"context"
	"errors"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListWorkspaceInvitations lists a workspace's invitations: GET
// /api/v0/workspaces/{slug}/invitations.
type ListWorkspaceInvitations struct {
	invitations InvitationLister
	auth        shared.Authorizer
	tokens      invitationTokens
}

// NewListWorkspaceInvitations returns the use case.
func NewListWorkspaceInvitations(invitations InvitationLister, auth shared.Authorizer, mac InvitationMAC) *ListWorkspaceInvitations {
	return &ListWorkspaceInvitations{invitations: invitations, auth: auth, tokens: invitationTokens{mac: mac}}
}

// Execute returns the workspace's undeleted invitations, pending or
// declined, in the store's order, each with its token, which the caller
// needs to hand out the link (M3 design 3.8). Only the workspace's admins
// may list them (M3 decision 4). A read opens no transaction and decides
// directly (M3 design 3.4).
func (u *ListWorkspaceInvitations) Execute(ctx context.Context, slug string) ([]domain.InvitationWithToken, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	w, err := u.invitations.WorkspaceBySlug(ctx, slug)
	switch {
	case errors.Is(err, ErrNotFound):
		return nil, domain.ErrNotFound
	case err != nil:
		return nil, err
	}
	if _, err := decide(ctx, u.auth, actor, domain.ActionInvitationList, w.ID, domain.ErrNotFound); err != nil {
		return nil, err
	}
	invitations, err := u.invitations.ListInvitations(ctx, w.ID)
	if err != nil {
		return nil, err
	}
	return u.tokens.withTokens(invitations), nil
}
