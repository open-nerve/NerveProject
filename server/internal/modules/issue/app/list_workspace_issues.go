package app

import (
	"context"
	"errors"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListWorkspaceIssuesDeps are the caller's authorization and ordinary project scope.
type ListWorkspaceIssuesDeps struct {
	ReadTx     shared.TxManager
	Authorizer shared.Authorizer
	Workspaces WorkspaceDirectory
	Projects   VisibleProjects
	Listing    *Listing
}

// ListWorkspaceIssues lists the work items of a workspace's projects the
// caller sees: GET /api/v0/workspaces/{slug}/issues (M7 P3a spec 2.2, 2.3).
// It is the member's list without its target member and relation.
type ListWorkspaceIssues struct{ d ListWorkspaceIssuesDeps }

// NewListWorkspaceIssues returns the use case.
func NewListWorkspaceIssues(d ListWorkspaceIssuesDeps) *ListWorkspaceIssues {
	return &ListWorkspaceIssues{d: d}
}

// Execute parses the query first, flat only (domain.WorkspaceList), so a
// bad one is 422 before anything is read. Then, in one read-only snapshot:
// the workspace slug names, not there or not seen alike 404
// workspace.not_found; the decision on workspace_issue.list in it; the
// projects the caller sees there, each with its OwnOnly
// (VisibleProjects.Visible: an active membership of an undeleted,
// unarchived project, which a workspace admin's role does not widen); and
// the engine's flat page of them. No project seen is the empty page.
func (u *ListWorkspaceIssues) Execute(ctx context.Context, slug string, raw map[string][]string) (IssueListPage, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return IssueListPage{}, err
	}
	query, err := domain.ParseListQuery(raw, domain.WorkspaceList, actor.UserID)
	if err != nil {
		return IssueListPage{}, err
	}
	var page IssueListPage
	err = u.d.ReadTx.WithinTx(ctx, func(ctx context.Context) error {
		wid, found, err := u.d.Workspaces.WorkspaceBySlug(ctx, slug)
		if err != nil {
			return err
		}
		if !found {
			return domain.ErrWorkspaceNotFound
		}
		_, err = u.d.Authorizer.Authorize(ctx, actor, domain.ActionListWorkspaceIssues, shared.Target{WorkspaceID: wid})
		if errors.Is(err, shared.ErrNotVisible) {
			return domain.ErrWorkspaceNotFound
		}
		if err != nil {
			return err
		}
		projects, err := u.d.Projects.Visible(ctx, wid, actor.UserID)
		if err != nil {
			return err
		}
		page, err = u.d.Listing.Read(ctx, ListingInput{ActorID: actor.UserID, Projects: projects, Query: query})
		return err
	})
	if err != nil {
		return IssueListPage{}, err
	}
	return page, nil
}
