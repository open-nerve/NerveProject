// Package httpadapter serves the project module's API: it implements the
// strict server that oapi-codegen generates from api/modules/project.yaml
// into the gen package, and translates between the generated types and the
// use cases.
package httpadapter

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListProjectsUseCase is app.ListProjects.
type ListProjectsUseCase interface {
	Execute(ctx context.Context, slug string, archived bool) ([]domain.Project, error)
}

// CreateProjectUseCase is app.CreateProject.
type CreateProjectUseCase interface {
	Execute(ctx context.Context, slug string, in domain.NewProject) (domain.Project, error)
}

// GetProjectUseCase is app.GetProject.
type GetProjectUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) (domain.Project, error)
}

// CheckIdentifierUseCase is app.CheckProjectIdentifier.
type CheckIdentifierUseCase interface {
	Execute(ctx context.Context, slug, identifier string) (bool, error)
}

// UpdateProjectUseCase is app.UpdateProject.
type UpdateProjectUseCase interface {
	Execute(ctx context.Context, id uuid.UUID, p domain.ProjectPatch) (domain.Project, error)
}

// ArchiveProjectUseCase is app.ArchiveProject, which archives or
// unarchives.
type ArchiveProjectUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) (domain.Project, error)
}

// DeleteProjectUseCase is app.DeleteProject.
type DeleteProjectUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) error
}

// GetPreferencesUseCase is app.GetProjectPreferences.
type GetPreferencesUseCase interface {
	Execute(ctx context.Context, projectID uuid.UUID) (domain.Preferences, error)
}

// UpdatePreferencesUseCase is app.UpdateProjectPreferences.
type UpdatePreferencesUseCase interface {
	Execute(ctx context.Context, projectID uuid.UUID, p domain.PreferencesPatch) (domain.Preferences, error)
}

// ListMembersUseCase is app.ListProjectMembers.
type ListMembersUseCase interface {
	Execute(ctx context.Context, projectID uuid.UUID) ([]domain.Member, error)
}

// AddMembersUseCase is app.AddProjectMembers.
type AddMembersUseCase interface {
	Execute(ctx context.Context, projectID uuid.UUID, in []domain.NewMember) ([]domain.Member, error)
}

// JoinProjectUseCase is app.JoinProject.
type JoinProjectUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) (domain.Project, error)
}

// UpdateMemberUseCase is app.UpdateProjectMember.
type UpdateMemberUseCase interface {
	Execute(ctx context.Context, id uuid.UUID, role shared.Role) (domain.Member, error)
}

// RemoveMemberUseCase is app.RemoveProjectMember.
type RemoveMemberUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) error
}

// LeaveProjectUseCase is app.LeaveProject.
type LeaveProjectUseCase interface {
	Execute(ctx context.Context, projectID uuid.UUID) error
}

// ListStatesUseCase is app.ListStates.
type ListStatesUseCase interface {
	Execute(ctx context.Context, projectID uuid.UUID) ([]domain.State, error)
}

// CreateStateUseCase is app.CreateState.
type CreateStateUseCase interface {
	Execute(ctx context.Context, projectID uuid.UUID, in domain.StateCreate) (domain.State, error)
}

// UpdateStateUseCase is app.UpdateState.
type UpdateStateUseCase interface {
	Execute(ctx context.Context, id uuid.UUID, p domain.StatePatch) (domain.State, error)
}

// DeleteStateUseCase is app.DeleteState.
type DeleteStateUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) error
}

// MarkDefaultStateUseCase is app.MarkDefaultState.
type MarkDefaultStateUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) error
}

// ListWorkspaceStatesUseCase is app.ListWorkspaceStates.
type ListWorkspaceStatesUseCase interface {
	Execute(ctx context.Context, slug string) ([]domain.State, error)
}

// UseCases are the use cases behind the module's operations.
type UseCases struct {
	ListProjects        ListProjectsUseCase
	CreateProject       CreateProjectUseCase
	GetProject          GetProjectUseCase
	CheckIdentifier     CheckIdentifierUseCase
	UpdateProject       UpdateProjectUseCase
	ArchiveProject      ArchiveProjectUseCase
	UnarchiveProject    ArchiveProjectUseCase
	DeleteProject       DeleteProjectUseCase
	GetPreferences      GetPreferencesUseCase
	UpdatePreferences   UpdatePreferencesUseCase
	ListMembers         ListMembersUseCase
	AddMembers          AddMembersUseCase
	JoinProject         JoinProjectUseCase
	UpdateMember        UpdateMemberUseCase
	RemoveMember        RemoveMemberUseCase
	LeaveProject        LeaveProjectUseCase
	ListStates          ListStatesUseCase
	CreateState         CreateStateUseCase
	UpdateState         UpdateStateUseCase
	DeleteState         DeleteStateUseCase
	MarkDefaultState    MarkDefaultStateUseCase
	ListWorkspaceStates ListWorkspaceStatesUseCase
}

// Register mounts the module's routes on router behind api's per-route
// middlewares; api.Errors answers binding, decoding and handler errors.
// Every operation needs a token.
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
