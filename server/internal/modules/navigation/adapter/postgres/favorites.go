package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// folderNameKey is the partial unique index of one's undeleted folders'
// names in a workspace, in any case (P3c spec 2.3).
const folderNameKey = "user_favorites_user_id_workspace_id_name_key"

// Favorites lists user's undeleted favorites in workspace, those in his
// folders too, by sequence descending, then by id descending (favorites.sql;
// spec 2.4).
func (s *Store) Favorites(ctx context.Context, user, workspace uuid.UUID) ([]domain.Favorite, error) {
	rows, err := s.queries(ctx).ListFavorites(ctx, gen.ListFavoritesParams{UserID: user, WorkspaceID: workspace})
	if err != nil {
		return nil, fmt.Errorf("list the favorites of user %s in workspace %s: %w", user, workspace, err)
	}
	out := make([]domain.Favorite, len(rows))
	for i, r := range rows {
		out[i] = favoriteOf(gen.GetFavoriteRow(r))
	}
	return out, nil
}

// Favorite reads user's undeleted favorite id without a lock; found is false
// when there is none, as when id is another's.
func (s *Store) Favorite(ctx context.Context, user, id uuid.UUID) (domain.Favorite, bool, error) {
	r, err := s.queries(ctx).GetFavorite(ctx, gen.GetFavoriteParams{ID: id, UserID: user})
	return foundFavorite(r, err, "read", id)
}

// LockFavorite reads user's undeleted favorite id locked FOR UPDATE until the
// transaction ctx carries ends, its new parent's FOR SHARE taken first when
// it has one (spec 2.8); found is false when there is none, as when id is
// another's, which it does not lock.
func (s *Store) LockFavorite(ctx context.Context, user, id uuid.UUID) (domain.Favorite, bool, error) {
	r, err := s.queries(ctx).LockFavorite(ctx, gen.LockFavoriteParams{ID: id, UserID: user})
	return foundFavorite(gen.GetFavoriteRow(r), err, "lock", id)
}

// ShareFolder locks FOR SHARE, until the transaction ctx carries ends, the
// parent a creation or a change names, before the favorite's own row (spec
// 2.5, 2.8): found is true when id is user's own undeleted folder in
// workspace. Any other id is no parent, and is not locked.
func (s *Store) ShareFolder(ctx context.Context, user, workspace, id uuid.UUID) (bool, error) {
	_, err := s.queries(ctx).ShareFolder(ctx, gen.ShareFolderParams{ID: id, UserID: user, WorkspaceID: workspace})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("lock folder %s: %w", id, err)
	}
	return true, nil
}

// LockFolderChildren locks FOR UPDATE, in id order, the undeleted favorites
// the folder holds, after the folder's own FOR UPDATE (spec 2.8): a
// folder's deletion's last locks.
func (s *Store) LockFolderChildren(ctx context.Context, folder uuid.UUID) error {
	if err := s.queries(ctx).LockFolderChildren(ctx, folder); err != nil {
		return fmt.Errorf("lock the favorites of folder %s: %w", folder, err)
	}
	return nil
}

// GreatestFavoriteSequence is the greatest sequence of user's undeleted
// favorites in workspace, nil when he has none (domain.SequenceAfter).
func (s *Store) GreatestFavoriteSequence(ctx context.Context, user, workspace uuid.UUID) (*float64, error) {
	greatest, err := s.queries(ctx).GreatestFavoriteSequence(ctx, gen.GreatestFavoriteSequenceParams{UserID: user, WorkspaceID: workspace})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("read the greatest favorite sequence of user %s in workspace %s: %w", user, workspace, err)
	}
	return &greatest, nil
}

// InsertFavorite stores the new favorite f, made by by at its CreatedAt, and
// answers it, created (spec 2.5). A favorite of a target whose owner has an
// undeleted favorite of it stores nothing: it answers that one as it is,
// not created. A folder whose name, in any case, another of its owner's
// undeleted folders in the workspace has is
// domain.ErrFavoriteFolderNameTaken. The domain and the use case checked
// every value, so a CHECK's or a foreign key's violation is a bug: an
// internal error.
func (s *Store) InsertFavorite(ctx context.Context, f domain.Favorite, by uuid.UUID) (domain.Favorite, bool, error) {
	row := gen.InsertFavoriteItemParams{ID: f.ID, WorkspaceID: f.WorkspaceID, ProjectID: f.ProjectID, UserID: f.UserID, EntityType: string(f.Kind),
		EntityIdentifier: f.EntityID, Name: nameColumn(f), ParentID: f.ParentID, Sequence: f.Sequence, By: by, CreatedAt: f.CreatedAt, UpdatedAt: f.UpdatedAt}
	q := s.queries(ctx)
	if f.Kind == domain.FavoriteFolder {
		if err := q.InsertFavoriteFolder(ctx, gen.InsertFavoriteFolderParams(row)); err != nil {
			return domain.Favorite{}, false, folderTaken(fmt.Sprintf("create folder %s", f.ID), err)
		}
		return f, true, nil
	}
	inserted, err := q.InsertFavoriteItem(ctx, row)
	switch {
	case err != nil:
		return domain.Favorite{}, false, fmt.Errorf("create favorite %s: %w", f.ID, err)
	case inserted == 1:
		return f, true, nil
	}
	// The undeleted favorite the insertion met was committed before it went
	// on, so this statement's snapshot holds it, unless its deletion was
	// committed in between: then the creation fails, to be retried.
	r, err := q.GetFavoriteOfTarget(ctx, gen.GetFavoriteOfTargetParams{UserID: f.UserID, EntityType: row.EntityType, EntityIdentifier: f.EntityID})
	if err != nil {
		return domain.Favorite{}, false, fmt.Errorf("read the favorite of %s %v that user %s has: %w", f.Kind, f.EntityID, f.UserID, err)
	}
	return favoriteOf(gen.GetFavoriteRow(r)), false, nil
}

// UpdateFavorite writes the parent, the sequence and, a folder's, the name
// of the undeleted favorite f.ID as f has them, last changed by by at
// f.UpdatedAt (spec 2.4). A folder's name, in any case, that another of its
// owner's undeleted folders in the workspace has is
// domain.ErrFavoriteFolderNameTaken; its own in another case is not.
func (s *Store) UpdateFavorite(ctx context.Context, f domain.Favorite, by uuid.UUID) error {
	err := s.queries(ctx).UpdateFavorite(ctx, gen.UpdateFavoriteParams{ID: f.ID, ParentID: f.ParentID, Sequence: f.Sequence, Name: nameColumn(f),
		UpdatedAt: f.UpdatedAt, By: by})
	if err != nil {
		return folderTaken(fmt.Sprintf("change favorite %s", f.ID), err)
	}
	return nil
}

// DeleteFavorite soft-deletes the undeleted favorite id and, a folder, the
// undeleted favorites it holds, every row deleted and last changed by by at
// at (spec 2.5), under the folder's lock and then its favorites' in id
// order (LockFavorite, LockFolderChildren). One deleted already keeps its
// deletion.
func (s *Store) DeleteFavorite(ctx context.Context, id uuid.UUID, at time.Time, by uuid.UUID) error {
	if err := s.queries(ctx).DeleteFavorite(ctx, gen.DeleteFavoriteParams{ID: id, At: at, By: by}); err != nil {
		return fmt.Errorf("delete favorite %s: %w", id, err)
	}
	return nil
}

// DeleteFavoritesOfTarget soft-deletes, in the transaction ctx carries,
// every account's undeleted favorites of the target of kind with id id,
// locked in id order, deleted and last changed by by at at: the cascade of
// the target's deletion or archiving (spec 2.6).
func (s *Store) DeleteFavoritesOfTarget(ctx context.Context, kind domain.FavoriteKind, id uuid.UUID, at time.Time, by uuid.UUID) error {
	err := s.queries(ctx).DeleteFavoritesOfTarget(ctx, gen.DeleteFavoritesOfTargetParams{EntityType: string(kind), EntityIdentifier: id, At: at, By: by})
	if err != nil {
		return fmt.Errorf("delete the favorites of %s %s: %w", kind, id, err)
	}
	return nil
}

// DeleteFavoritesOfProject soft-deletes, in the transaction ctx carries,
// every account's undeleted favorites whose project is project, the
// project's own and its cycles', modules' and views', locked in id order,
// deleted and last changed by by at at: the cascade of the project's
// archiving (spec 2.6).
func (s *Store) DeleteFavoritesOfProject(ctx context.Context, project uuid.UUID, at time.Time, by uuid.UUID) error {
	if err := s.queries(ctx).DeleteFavoritesOfProject(ctx, gen.DeleteFavoritesOfProjectParams{ProjectID: project, At: at, By: by}); err != nil {
		return fmt.Errorf("delete the favorites of project %s: %w", project, err)
	}
	return nil
}

// folderTaken is err, a write's of a favorite, as a folder's name, in any
// case, that another of its owner's undeleted folders in the workspace
// has: domain.ErrFavoriteFolderNameTaken; any other error is the write's.
func folderTaken(write string, err error) error {
	if postgres.UniqueViolation(err, folderNameKey) {
		return domain.ErrFavoriteFolderNameTaken
	}
	return fmt.Errorf("%s: %w", write, err)
}

// nameColumn is f's name as its column holds it: a folder's, NULL for a
// favorite of a target, whose name is "".
func nameColumn(f domain.Favorite) *string {
	if f.Name == "" {
		return nil
	}
	return &f.Name
}

// foundFavorite is the favorite a read of id gave, found when there was one;
// err is the read's.
func foundFavorite(r gen.GetFavoriteRow, err error, what string, id uuid.UUID) (domain.Favorite, bool, error) {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.Favorite{}, false, nil
	case err != nil:
		return domain.Favorite{}, false, fmt.Errorf("%s favorite %s: %w", what, id, err)
	}
	return favoriteOf(r), true, nil
}

// favoriteOf is the favorite a row of favorites.sql's reads holds.
func favoriteOf(r gen.GetFavoriteRow) domain.Favorite {
	f := domain.Favorite{ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, UserID: r.UserID, Kind: domain.FavoriteKind(r.EntityType),
		EntityID: r.EntityIdentifier, ParentID: r.ParentID, Sequence: r.Sequence, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
	if r.Name != nil {
		f.Name = *r.Name
	}
	return f
}
