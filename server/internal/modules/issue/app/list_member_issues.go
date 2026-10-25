package app

import (
	"context"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// WorkspaceMemberDirectory finds an undeleted membership, active or ended.
type WorkspaceMemberDirectory interface {
	MemberExists(context.Context, uuid.UUID, uuid.UUID) (bool, error)
}

// ListMemberIssuesDeps are the caller's authorization and ordinary project scope.
type ListMemberIssuesDeps struct {
	ReadTx     shared.TxManager
	Authorizer shared.Authorizer
	Workspaces WorkspaceDirectory
	Members    WorkspaceMemberDirectory
	Projects   VisibleProjects
	Listing    *Listing
}

// ListMemberIssues reads a target account's relationships in the caller's scope.
type ListMemberIssues struct{ d ListMemberIssuesDeps }

func NewListMemberIssues(d ListMemberIssuesDeps) *ListMemberIssues { return &ListMemberIssues{d: d} }
func (u *ListMemberIssues) Execute(ctx context.Context, slug string, userID uuid.UUID, raw map[string][]string) (IssueListPage, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return IssueListPage{}, err
	}
	query, err := domain.ParseListQuery(raw, domain.MemberList, actor.UserID)
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
		_, err = u.d.Authorizer.Authorize(ctx, actor, domain.ActionListMemberIssues, shared.Target{WorkspaceID: wid})
		if errors.Is(err, shared.ErrNotVisible) {
			return domain.ErrWorkspaceNotFound
		}
		if err != nil {
			return err
		}
		found, err = u.d.Members.MemberExists(ctx, wid, userID)
		if err != nil {
			return err
		}
		if !found {
			return domain.ErrMemberNotFound
		}
		projects, err := u.d.Projects.Visible(ctx, wid, actor.UserID)
		if err != nil {
			return err
		}
		page, err = u.d.Listing.Read(ctx, ListingInput{ActorID: actor.UserID, TargetMemberID: &userID, Projects: projects, Query: query})
		return err
	})
	if err != nil {
		return IssueListPage{}, err
	}
	return page, nil
}
