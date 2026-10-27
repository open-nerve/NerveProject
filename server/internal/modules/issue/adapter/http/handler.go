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

// CreateIssueCommentUseCase creates a comment on the named issue.
type CreateIssueCommentUseCase interface {
	Execute(context.Context, uuid.UUID, string) (domain.Comment, error)
}

// ListIssueCommentsUseCase lists a page, retaining absent query values.
type ListIssueCommentsUseCase interface {
	Execute(context.Context, uuid.UUID, *int, *string) (app.CommentPage, error)
}

// UpdateIssueCommentUseCase edits a comment.
type UpdateIssueCommentUseCase interface {
	Execute(context.Context, uuid.UUID, string) (domain.Comment, error)
}

// DeleteIssueCommentUseCase deletes a comment.
type DeleteIssueCommentUseCase interface {
	Execute(context.Context, uuid.UUID) error
}

// AddCommentReactionUseCase adds the caller's reaction on a comment.
type AddCommentReactionUseCase interface {
	Execute(context.Context, uuid.UUID, string) (domain.CommentReaction, error)
}

// AddIssueReactionUseCase adds the caller's reaction.
type AddIssueReactionUseCase interface {
	Execute(context.Context, uuid.UUID, string) (domain.IssueReaction, error)
}

// RemoveIssueReactionUseCase removes the caller's reaction.
type RemoveIssueReactionUseCase interface {
	Execute(context.Context, uuid.UUID, string) error
}

// ListIssueReactionsUseCase reads live reactions.
type ListIssueReactionsUseCase interface {
	Execute(context.Context, uuid.UUID) ([]domain.IssueReaction, error)
}

// ListDescriptionVersionsUseCase retains optional page query values.
type ListDescriptionVersionsUseCase interface {
	Execute(context.Context, uuid.UUID, *int, *string) (app.DescriptionVersionPage, error)
}

// GetDescriptionVersionUseCase reads one saved description.
type GetDescriptionVersionUseCase interface {
	Execute(context.Context, uuid.UUID) (domain.DescriptionVersion, error)
}

// ListIssueActivitiesUseCase reads a page with historical reference visibility.
type ListIssueActivitiesUseCase interface {
	Execute(context.Context, uuid.UUID, *int, *string) (app.ActivityPage, error)
}

// ListMemberIssuesUseCase reads target relationships in the caller scope.
type ListMemberIssuesUseCase interface {
	Execute(context.Context, string, uuid.UUID, map[string][]string) (app.IssueListPage, error)
}

// ListIssuesUseCase reads the explicitly authorized project scope.
type ListIssuesUseCase interface {
	Execute(context.Context, uuid.UUID, map[string][]string) (app.IssueListPage, error)
}

// UseCases are the use cases behind the module's operations.
type UseCases struct {
	PublishDraft            PublishDraftUseCase
	ListDrafts              ListDraftsUseCase
	GetDraft                GetDraftUseCase
	CreateDraft             CreateDraftUseCase
	UpdateDraft             UpdateDraftUseCase
	DeleteDraft             DeleteDraftUseCase
	UpdateIssueLink         UpdateIssueLinkUseCase
	DeleteIssueLink         DeleteIssueLinkUseCase
	CreateIssueLink         CreateIssueLinkUseCase
	ListIssueLinks          ListIssueLinksUseCase
	ListMemberIssues        ListMemberIssuesUseCase
	ListIssues              ListIssuesUseCase
	ListIssueActivities     ListIssueActivitiesUseCase
	ListDescriptionVersions ListDescriptionVersionsUseCase
	GetDescriptionVersion   GetDescriptionVersionUseCase
	AddCommentReaction      AddCommentReactionUseCase
	RemoveCommentReaction   RemoveIssueReactionUseCase
	AddIssueReaction        AddIssueReactionUseCase
	RemoveIssueReaction     RemoveIssueReactionUseCase
	ListIssueReactions      ListIssueReactionsUseCase
	UpdateIssueComment      UpdateIssueCommentUseCase
	DeleteIssueComment      DeleteIssueCommentUseCase
	CreateIssueComment      CreateIssueCommentUseCase
	ListIssueComments       ListIssueCommentsUseCase
	CreateIssue             CreateIssueUseCase
	GetIssue                GetIssueUseCase
	GetIssueByIdentifier    GetIssueByIdentifierUseCase
	UpdateIssue             UpdateIssueUseCase
	DeleteIssue             DeleteIssueUseCase
	ArchiveIssue            ArchiveIssueUseCase
	UnarchiveIssue          ArchiveIssueUseCase
	ListSubIssues           ListSubIssuesUseCase
	AddSubIssues            AddSubIssuesUseCase
	ListIssueRelations      ListIssueRelationsUseCase
	CreateIssueRelations    CreateIssueRelationsUseCase
	DeleteIssueRelation     DeleteIssueRelationUseCase
}

// Register mounts the module's routes on router behind api's per-route
// middlewares; api.Errors answers binding, decoding and handler errors.
// Every operation needs a token.
func Register(router *httpserver.Router, api *httpserver.API, uc UseCases) {
	strict := gen.NewStrictHandlerWithOptions(handler{uc: uc}, []gen.StrictMiddlewareFunc{withListQuery}, gen.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  api.Errors.BodyError,
		ResponseErrorHandlerFunc: api.Errors.Write,
	})
	middlewares := []gen.MiddlewareFunc{draftPublishBody(api)}
	for _, m := range api.Middlewares(gen.BodyShapes()) {
		middlewares = append(middlewares, m)
	}
	gen.HandlerWithOptions(strict, gen.StdHTTPServerOptions{
		BaseRouter:       router,
		Middlewares:      middlewares,
		ErrorHandlerFunc: listQueryBindingError(api),
	})
}

// handler implements gen.StrictServerInterface.
type handler struct {
	uc UseCases
}
