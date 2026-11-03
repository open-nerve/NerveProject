package bootstrap

import (
	"context"
	"uuid"

	assetapp "github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	assetdomain "github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type assetHostAccess struct {
	workspaces workspace.AssetAccess
	projects   project.AssetAccess
	issues     issue.AssetAccess
}

func (a assetHostAccess) Inspect(ctx context.Context, actor shared.Actor, q assetapp.HostRequest) (assetapp.Host, error) {
	wi, pi, ii, err := assetOwnerIntents(q.Intent)
	if err != nil {
		return assetapp.Host{}, err
	}
	h := assetapp.Host{Scope: q.Scope}
	switch q.Scope.Kind {
	case assetdomain.ScopeUser:
		return assetUserHost(actor, q)
	case assetdomain.ScopeWorkspace:
		got, e := a.workspaces.Inspect(ctx, actor, workspace.AssetRequest{ID: q.Scope.ID, Intent: wi})
		err = e
		h.WorkspaceID = got.WorkspaceID
	case assetdomain.ScopeProject, assetdomain.ScopeNewProject:
		got, e := a.projects.Inspect(ctx, actor, project.AssetRequest{ID: q.Scope.ID, NewProject: q.Scope.Kind == assetdomain.ScopeNewProject, Intent: pi})
		err = e
		h.WorkspaceID, h.ProjectID = got.WorkspaceID, got.ProjectID
	case assetdomain.ScopeIssue, assetdomain.ScopeComment, assetdomain.ScopeNewComment, assetdomain.ScopeDraft:
		got, e := a.issues.Inspect(ctx, actor, issue.AssetRequest{ID: q.Scope.ID, Kind: assetIssueKind(q.Scope.Kind), Intent: ii})
		err = e
		h.WorkspaceID, h.ProjectID, h.IssueID, h.CommentID, h.DraftID = got.WorkspaceID, got.ProjectID, got.IssueID, got.CommentID, got.DraftID
	default:
		return assetapp.Host{}, assetdomain.ErrInvalidAsset
	}
	if err != nil {
		return assetapp.Host{}, assetHostError(err)
	}
	return h, nil
}

func (a assetHostAccess) Check(ctx context.Context, actor shared.Actor, q assetapp.HostRequest, h assetapp.Host) error {
	wi, pi, ii, err := assetOwnerIntents(q.Intent)
	if err != nil {
		return err
	}
	if h.Scope != q.Scope {
		return assetdomain.ErrAssetScopeChanged
	}
	switch q.Scope.Kind {
	case assetdomain.ScopeUser:
		now, e := assetUserHost(actor, q)
		if e != nil {
			return e
		}
		if h != now {
			return assetdomain.ErrAssetScopeChanged
		}
		return nil
	case assetdomain.ScopeWorkspace:
		err = a.workspaces.Check(ctx, actor, workspace.AssetRequest{ID: q.Scope.ID, Intent: wi}, workspace.AssetHost{WorkspaceID: h.WorkspaceID})
	case assetdomain.ScopeProject, assetdomain.ScopeNewProject:
		err = a.projects.Check(ctx, actor, project.AssetRequest{ID: q.Scope.ID, NewProject: q.Scope.Kind == assetdomain.ScopeNewProject, Intent: pi}, project.AssetHost{WorkspaceID: h.WorkspaceID, ProjectID: h.ProjectID})
	case assetdomain.ScopeIssue, assetdomain.ScopeComment, assetdomain.ScopeNewComment, assetdomain.ScopeDraft:
		err = a.issues.Check(ctx, actor, issue.AssetRequest{ID: q.Scope.ID, Kind: assetIssueKind(q.Scope.Kind), Intent: ii}, issue.AssetHost{WorkspaceID: h.WorkspaceID, ProjectID: h.ProjectID, IssueID: h.IssueID, CommentID: h.CommentID, DraftID: h.DraftID})
	default:
		return assetdomain.ErrInvalidAsset
	}
	return assetHostError(err)
}

func assetOwnerIntents(intent assetapp.AccessIntent) (workspace.AssetIntent, project.AssetIntent, issue.AssetIntent, error) {
	switch intent {
	case assetapp.ReadAsset:
		return workspace.AssetRead, project.AssetRead, issue.AssetRead, nil
	case assetapp.PrepareAsset:
		return workspace.AssetPrepare, project.AssetPrepare, issue.AssetPrepare, nil
	case assetapp.RemoveAsset:
		return workspace.AssetRemove, project.AssetRemove, issue.AssetRemove, nil
	default:
		return 0, 0, 0, assetdomain.ErrInvalidAsset
	}
}
func assetIssueKind(kind assetdomain.ScopeKind) issue.AssetHostKind {
	switch kind {
	case assetdomain.ScopeIssue:
		return issue.AssetIssue
	case assetdomain.ScopeComment:
		return issue.AssetComment
	case assetdomain.ScopeNewComment:
		return issue.AssetNewComment
	case assetdomain.ScopeDraft:
		return issue.AssetDraft
	}
	return 0
}
func assetHostError(err error) error {
	if workspace.AssetHostNotFound(err) || project.AssetHostNotFound(err) || issue.AssetHostNotFound(err) {
		return assetdomain.ErrAssetNotFound
	}
	if workspace.AssetHostChanged(err) || project.AssetHostChanged(err) || issue.AssetHostChanged(err) {
		return assetdomain.ErrAssetScopeChanged
	}
	return err
}

// The authentication middleware identifies the current user; capability
// issuance also rechecks its credential under the account's first row lock.
func assetUserHost(actor shared.Actor, q assetapp.HostRequest) (assetapp.Host, error) {
	if actor.UserID == uuid.Nil() {
		return assetapp.Host{}, shared.Unauthenticated()
	}
	if q.Scope.ID != uuid.Nil() {
		return assetapp.Host{}, assetdomain.ErrInvalidAsset
	}
	return assetapp.Host{Scope: q.Scope}, nil
}
