package postgresadapter

import (
	"context"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

func (s *Store) ListDraftPreparations(ctx context.Context, q app.DraftPreparationQuery) ([]domain.Asset, error) {
	if _, ok := postgres.Tx(ctx); !ok {
		return nil, errors.New("draft preparation read requires the caller's transaction")
	}
	h := q.Host
	if q.ActorID == uuid.Nil() || h.WorkspaceID == uuid.Nil() || h.DraftID == uuid.Nil() ||
		h.Scope != (domain.Scope{Kind: domain.ScopeDraft, ID: h.DraftID}) ||
		h.IssueID != uuid.Nil() || h.CommentID != uuid.Nil() || q.Limit < 1 || q.Limit > 101 {
		return nil, domain.ErrInvalidAsset
	}
	rows, err := s.queries(ctx).ListDraftPreparations(ctx, gen.ListDraftPreparationsParams{
		ActorID: q.ActorID, WorkspaceID: h.WorkspaceID, ProjectID: optionalID(h.ProjectID),
		DraftID: h.DraftID, AfterID: q.After, PageLimit: int32(q.Limit),
	})
	if err != nil {
		return nil, err
	}
	return decodeAssets(rows)
}
