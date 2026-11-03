package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	assetapp "github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	assetdomain "github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
)

type assetParentReader struct {
	workspaces workspace.AssetParents
	projects   project.AssetParents
	issues     issue.AssetParents
}
type assetParentFact struct {
	id uuid.UUID
	at *time.Time
	by *uuid.UUID
}

func (r assetParentReader) Deletions(ctx context.Context, hosts []assetapp.Host) (map[assetapp.ParentKey]assetapp.DeletionStamp, error) {
	if len(hosts) > 1000 {
		return nil, errors.New("asset parent fact batch exceeds limit")
	}
	var w, p, i, c, d []uuid.UUID
	for _, h := range hosts {
		w = append(w, h.WorkspaceID)
		p = append(p, h.ProjectID)
		i = append(i, h.IssueID)
		c = append(c, h.CommentID)
		d = append(d, h.DraftID)
	}
	result := map[assetapp.ParentKey]assetapp.DeletionStamp{}
	for _, batch := range []struct {
		kind assetdomain.ScopeKind
		ids  []uuid.UUID
	}{{assetdomain.ScopeWorkspace, w}, {assetdomain.ScopeProject, p}, {assetdomain.ScopeIssue, i}, {assetdomain.ScopeComment, c}, {assetdomain.ScopeDraft, d}} {
		ids := sortedAssetParents(batch.ids)
		if len(ids) == 0 {
			continue
		}
		rows, err := r.read(ctx, batch.kind, ids)
		if err != nil {
			return nil, err
		}
		pending := make(map[uuid.UUID]bool, len(ids))
		for _, id := range ids {
			pending[id] = true
		}
		for _, row := range rows {
			if !pending[row.id] {
				return nil, fmt.Errorf("unexpected or repeated asset %s parent fact", batch.kind)
			}
			delete(pending, row.id)
			if row.at != nil {
				if row.at.IsZero() {
					return nil, errors.New("asset parent has invalid deletion time")
				}
				result[assetapp.ParentKey{Kind: batch.kind, ID: row.id}] = assetapp.DeletionStamp{At: *row.at, By: row.by}
			}
		}
		if len(pending) > 0 {
			return nil, fmt.Errorf("asset %s parent facts missing %d rows", batch.kind, len(pending))
		}
	}
	return result, nil
}
func (r assetParentReader) read(ctx context.Context, kind assetdomain.ScopeKind, ids []uuid.UUID) ([]assetParentFact, error) {
	out := make([]assetParentFact, 0, len(ids))
	switch kind {
	case assetdomain.ScopeWorkspace:
		rows, err := r.workspaces.AssetWorkspaceFacts(ctx, ids)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			out = append(out, assetParentFact{row.ID, row.DeletedAt, row.DeletedByID})
		}
	case assetdomain.ScopeProject:
		rows, err := r.projects.AssetProjectFacts(ctx, ids)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			out = append(out, assetParentFact{row.ID, row.DeletedAt, row.DeletedByID})
		}
	case assetdomain.ScopeIssue:
		rows, err := r.issues.AssetIssueFacts(ctx, ids)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			out = append(out, assetParentFact{row.ID, row.DeletedAt, row.DeletedByID})
		}
	case assetdomain.ScopeComment:
		rows, err := r.issues.AssetCommentFacts(ctx, ids)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			out = append(out, assetParentFact{row.ID, row.DeletedAt, row.DeletedByID})
		}
	case assetdomain.ScopeDraft:
		rows, err := r.issues.AssetDraftFacts(ctx, ids)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			out = append(out, assetParentFact{row.ID, row.DeletedAt, row.DeletedByID})
		}
	default:
		return nil, errors.New("invalid asset parent fact kind")
	}
	return out, nil
}
