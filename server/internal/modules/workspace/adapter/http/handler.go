// Package httpadapter serves the workspace module's API: it implements the
// strict server that oapi-codegen generates from api/modules/workspace.yaml
// into the gen package, and translates between the generated types and the
// use cases.
package httpadapter

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListWorkspacesUseCase is app.ListWorkspaces.
type ListWorkspacesUseCase interface {
	Execute(ctx context.Context) ([]domain.Workspace, error)
}

// CreateWorkspaceUseCase is app.CreateWorkspace's entry for the API.
type CreateWorkspaceUseCase interface {
	Execute(ctx context.Context, w domain.NewWorkspace) (domain.Workspace, error)
}

// GetWorkspaceUseCase is app.GetWorkspace.
type GetWorkspaceUseCase interface {
	Execute(ctx context.Context, slug string) (domain.Workspace, error)
}

// UpdateWorkspaceUseCase is app.UpdateWorkspace.
type UpdateWorkspaceUseCase interface {
	Execute(ctx context.Context, slug string, p domain.WorkspacePatch) (domain.Workspace, error)
}

// DeleteWorkspaceUseCase is app.DeleteWorkspace.
type DeleteWorkspaceUseCase interface {
	Execute(ctx context.Context, slug string) error
}

// LeaveUseCase is app.LeaveWorkspace.
type LeaveUseCase interface {
	Execute(ctx context.Context, slug string) error
}

// ListMembersUseCase is app.ListWorkspaceMembers.
type ListMembersUseCase interface {
	Execute(ctx context.Context, slug string) ([]domain.Member, error)
}

// UpdateMemberUseCase is app.UpdateWorkspaceMember.
type UpdateMemberUseCase interface {
	Execute(ctx context.Context, id uuid.UUID, role shared.Role) (domain.Member, error)
}

// RemoveMemberUseCase is app.RemoveWorkspaceMember.
type RemoveMemberUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) error
}

// GetPreferencesUseCase is app.GetWorkspacePreferences.
type GetPreferencesUseCase interface {
	Execute(ctx context.Context, slug string) (domain.Preferences, error)
}

// UpdatePreferencesUseCase is app.UpdateWorkspacePreferences.
type UpdatePreferencesUseCase interface {
	Execute(ctx context.Context, slug string, p domain.PreferencesPatch) (domain.Preferences, error)
}

// ListInvitationsUseCase is app.ListWorkspaceInvitations.
type ListInvitationsUseCase interface {
	Execute(ctx context.Context, slug string) ([]domain.InvitationWithToken, error)
}

// CreateInvitationsUseCase is app.CreateWorkspaceInvitations.
type CreateInvitationsUseCase interface {
	Execute(ctx context.Context, slug string, batch []domain.NewInvitation) ([]domain.InvitationWithToken, error)
}

// GetInvitationUseCase is app.GetWorkspaceInvitation.
type GetInvitationUseCase interface {
	Execute(ctx context.Context, id uuid.UUID, token string) (domain.InvitationPreview, error)
}

// UpdateInvitationUseCase is app.UpdateWorkspaceInvitation.
type UpdateInvitationUseCase interface {
	Execute(ctx context.Context, id uuid.UUID, role shared.Role) (domain.InvitationWithToken, error)
}

// DeleteInvitationUseCase is app.DeleteWorkspaceInvitation.
type DeleteInvitationUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) error
}

// AcceptInvitationUseCase is app.AcceptWorkspaceInvitation.
type AcceptInvitationUseCase interface {
	Execute(ctx context.Context, id uuid.UUID, token string) (domain.Workspace, error)
}

// DeclineInvitationUseCase is app.DeclineWorkspaceInvitation.
type DeclineInvitationUseCase interface {
	Execute(ctx context.Context, id uuid.UUID, token string) error
}

// CheckSlugUseCase is app.CheckSlug.
type CheckSlugUseCase interface {
	Execute(ctx context.Context, slug string) (domain.SlugReason, error)
}

// UseCases are the use cases behind the module's operations.
type UseCases struct {
	ListWorkspaces    ListWorkspacesUseCase
	CreateWorkspace   CreateWorkspaceUseCase
	GetWorkspace      GetWorkspaceUseCase
	UpdateWorkspace   UpdateWorkspaceUseCase
	DeleteWorkspace   DeleteWorkspaceUseCase
	Leave             LeaveUseCase
	ListMembers       ListMembersUseCase
	UpdateMember      UpdateMemberUseCase
	RemoveMember      RemoveMemberUseCase
	CheckSlug         CheckSlugUseCase
	GetPreferences    GetPreferencesUseCase
	UpdatePreferences UpdatePreferencesUseCase
	ListInvitations   ListInvitationsUseCase
	CreateInvitations CreateInvitationsUseCase
	GetInvitation     GetInvitationUseCase
	UpdateInvitation  UpdateInvitationUseCase
	DeleteInvitation  DeleteInvitationUseCase
	AcceptInvitation  AcceptInvitationUseCase
	DeclineInvitation DeclineInvitationUseCase
}

// PublicOperations are the module's routes that need no token (M3 design
// 3.8), as the generated code registers them.
func PublicOperations() []string {
	return []string{"GET /api/v0/workspace-invitations/{invitation_id}"}
}

// Register mounts the module's routes on router behind api's per-route
// middlewares; api.Errors answers binding, decoding and handler errors.
// Every operation but PublicOperations needs a token.
func Register(router *httpserver.Router, api *httpserver.API, uc UseCases) {
	strict := gen.NewStrictHandlerWithOptions(handler{uc: uc}, nil, gen.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  api.Errors.BodyError,
		ResponseErrorHandlerFunc: api.Errors.Write,
	})
	var middlewares []gen.MiddlewareFunc
	for _, m := range api.Middlewares(gen.BodyShapes()) {
		middlewares = append(middlewares, m)
	}
	gen.HandlerWithOptions(strict, gen.StdHTTPServerOptions{
		BaseRouter:       router,
		Middlewares:      middlewares,
		ErrorHandlerFunc: api.Errors.BadRequest,
	})
}

// handler implements gen.StrictServerInterface.
type handler struct {
	uc UseCases
}
