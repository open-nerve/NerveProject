package postgresadapter

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func (s *Store) DeleteScopeRows(ctx context.Context, kind domain.ScopeKind, id uuid.UUID, stamp app.DeletionStamp) error {
	if id == uuid.Nil() || stamp.At.IsZero() || (stamp.By != nil && *stamp.By == uuid.Nil()) {
		return domain.ErrInvalidAsset
	}
	q := s.queries(ctx)
	switch kind {
	case domain.ScopeIssue:
		return q.DeleteIssueAssets(ctx, gen.DeleteIssueAssetsParams{ParentID: &id, At: &stamp.At, By: stamp.By})
	case domain.ScopeComment:
		return q.DeleteCommentAssets(ctx, gen.DeleteCommentAssetsParams{ParentID: &id, At: &stamp.At, By: stamp.By})
	case domain.ScopeDraft:
		return q.DeleteDraftAssets(ctx, gen.DeleteDraftAssetsParams{ParentID: &id, At: &stamp.At, By: stamp.By})
	default:
		return domain.ErrInvalidAsset
	}
}
