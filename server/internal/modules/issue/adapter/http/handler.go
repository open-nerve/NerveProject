// Package httpadapter serves the issue module's API: it implements the
// strict server that oapi-codegen generates from api/modules/issue.yaml
// into the gen package, and translates between the generated types and the
// use cases.
package httpadapter

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)

// CreateIssueUseCase is app.CreateIssue.
type CreateIssueUseCase interface {
	Execute(ctx context.Context, projectID uuid.UUID, in domain.NewIssue) (domain.Issue, error)
}

// GetIssueUseCase is app.GetIssue.
type GetIssueUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) (domain.Issue, error)
}

// GetIssueByIdentifierUseCase is app.GetIssueByIdentifier.
type GetIssueByIdentifierUseCase interface {
	Execute(ctx context.Context, slug, identifier string) (domain.Issue, error)
}

// UpdateIssueUseCase is app.UpdateIssue.
type UpdateIssueUseCase interface {
	Execute(ctx context.Context, id uuid.UUID, p domain.IssuePatch) (domain.Issue, error)
}

// DeleteIssueUseCase is app.DeleteIssue.
type DeleteIssueUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) error
}

// ArchiveIssueUseCase is app.ArchiveIssue, and app.UnarchiveIssue.
type ArchiveIssueUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) (domain.Issue, error)
}

// ListSubIssuesUseCase is app.ListSubIssues.
type ListSubIssuesUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) ([]domain.Issue, error)
}

// AddSubIssuesUseCase is app.AddSubIssues.
type AddSubIssuesUseCase interface {
	Execute(ctx context.Context, id uuid.UUID, ids []uuid.UUID) ([]domain.Issue, error)
}

// ListIssueRelationsUseCase is app.ListIssueRelations.
type ListIssueRelationsUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) ([]app.IssueRelation, error)
}

// CreateIssueRelationsUseCase is app.CreateIssueRelations.
type CreateIssueRelationsUseCase interface {
	Execute(ctx context.Context, id uuid.UUID, t domain.RelationType, ids []uuid.UUID) ([]app.IssueRelation, error)
}

// DeleteIssueRelationUseCase is app.DeleteIssueRelation.
type DeleteIssueRelationUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) error
}

// UseCases are the use cases behind the module's operations.
type UseCases struct {
	CreateIssue          CreateIssueUseCase
	GetIssue             GetIssueUseCase
	GetIssueByIdentifier GetIssueByIdentifierUseCase
	UpdateIssue          UpdateIssueUseCase
	DeleteIssue          DeleteIssueUseCase
	ArchiveIssue         ArchiveIssueUseCase
	UnarchiveIssue       ArchiveIssueUseCase
	ListSubIssues        ListSubIssuesUseCase
	AddSubIssues         AddSubIssuesUseCase
	ListIssueRelations   ListIssueRelationsUseCase
	CreateIssueRelations CreateIssueRelationsUseCase
	DeleteIssueRelation  DeleteIssueRelationUseCase
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
