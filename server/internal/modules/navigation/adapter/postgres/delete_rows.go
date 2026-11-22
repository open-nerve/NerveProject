package postgresadapter

import (
	"context"
	"fmt"
	"math"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/app"
)

// DeleteProjectViews soft-deletes a batch of a deleted project's views in
// the transaction ctx carries and answers their ids (app.ViewDeletions,
// delete_rows.sql).
func (s *Store) DeleteProjectViews(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	if err := checkBatch(b); err != nil {
		return nil, err
	}
	ids, err := s.queries(ctx).DeleteProjectViews(ctx, gen.DeleteProjectViewsParams{At: b.At, DeletedBy: b.By, ProjectID: b.ParentID, After: b.After, Batch: int32(b.Limit)})
	if err != nil {
		return nil, fmt.Errorf("delete a batch of the project's views: %w", err)
	}
	return ids, nil
}

// DeleteWorkspaceViews soft-deletes a batch of a deleted workspace's own
// views, those without a project, in the transaction ctx carries and
// answers their ids.
func (s *Store) DeleteWorkspaceViews(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	if err := checkBatch(b); err != nil {
		return nil, err
	}
	ids, err := s.queries(ctx).DeleteWorkspaceViews(ctx, gen.DeleteWorkspaceViewsParams{At: b.At, DeletedBy: b.By, WorkspaceID: b.ParentID, After: b.After, Batch: int32(b.Limit)})
	if err != nil {
		return nil, fmt.Errorf("delete a batch of the workspace's views: %w", err)
	}
	return ids, nil
}

// DeleteProjectFavorites soft-deletes a batch of a deleted project's
// favorites, every account's, in the transaction ctx carries and answers
// their ids (app.FavoriteDeletions).
func (s *Store) DeleteProjectFavorites(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	if err := checkBatch(b); err != nil {
		return nil, err
	}
	ids, err := s.queries(ctx).DeleteProjectFavorites(ctx, gen.DeleteProjectFavoritesParams{At: b.At, DeletedBy: b.By, ProjectID: b.ParentID, After: b.After, Batch: int32(b.Limit)})
	if err != nil {
		return nil, fmt.Errorf("delete a batch of the project's favorites: %w", err)
	}
	return ids, nil
}

// DeleteProjectRecentVisits soft-deletes a batch of a deleted project's
// recent visits, every account's, in the transaction ctx carries and
// answers their ids (app.RecentVisitDeletions).
func (s *Store) DeleteProjectRecentVisits(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	if err := checkBatch(b); err != nil {
		return nil, err
	}
	ids, err := s.queries(ctx).DeleteProjectRecentVisits(ctx, gen.DeleteProjectRecentVisitsParams{At: b.At, DeletedBy: b.By, ProjectID: b.ParentID, After: b.After,
		Batch: int32(b.Limit)})
	if err != nil {
		return nil, fmt.Errorf("delete a batch of the project's recent visits: %w", err)
	}
	return ids, nil
}

// DeleteWorkspaceFavoriteFolders soft-deletes a batch of a deleted
// workspace's folders in the transaction ctx carries and answers their ids.
func (s *Store) DeleteWorkspaceFavoriteFolders(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	if err := checkBatch(b); err != nil {
		return nil, err
	}
	ids, err := s.queries(ctx).DeleteWorkspaceFavoriteFolders(ctx, gen.DeleteWorkspaceFavoriteFoldersParams{At: b.At, DeletedBy: b.By, WorkspaceID: b.ParentID, After: b.After,
		Batch: int32(b.Limit)})
	if err != nil {
		return nil, fmt.Errorf("delete a batch of the workspace's folders: %w", err)
	}
	return ids, nil
}

// DeleteWorkspaceFavoriteItems soft-deletes a batch of a deleted
// workspace's own favorites that are no folders, those without a project,
// in the transaction ctx carries and answers their ids.
func (s *Store) DeleteWorkspaceFavoriteItems(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	if err := checkBatch(b); err != nil {
		return nil, err
	}
	ids, err := s.queries(ctx).DeleteWorkspaceFavoriteItems(ctx, gen.DeleteWorkspaceFavoriteItemsParams{At: b.At, DeletedBy: b.By, WorkspaceID: b.ParentID, After: b.After,
		Batch: int32(b.Limit)})
	if err != nil {
		return nil, fmt.Errorf("delete a batch of the workspace's favorites: %w", err)
	}
	return ids, nil
}

func checkBatch(b app.RowBatch) error {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return fmt.Errorf("deletion batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	return nil
}
