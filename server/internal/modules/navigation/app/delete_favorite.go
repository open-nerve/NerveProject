package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// DeleteFavorite deletes one's favorite by its id (deleteFavorite).
type DeleteFavorite struct{ d FavoriteWriteDeps }

// NewDeleteFavorite returns the use case over d.
func NewDeleteFavorite(d FavoriteWriteDeps) *DeleteFavorite { return &DeleteFavorite{d: d} }

// Execute soft-deletes the caller's favorite id (P3c spec 2.5, 2.7). Its
// refusals, in order: his favorite read without a lock and decided on
// (FavoriteLocks.own), every favorite he may not have
// navigation.favorite_not_found, the one value, whose target he no longer
// sees among them, a role refused the Authorizer's shared.Forbidden; then,
// under its locks and decision (FavoriteLocks.run), a folder's taking the
// favorites it holds after it, the favorite read again, the same 404 when
// it was deleted meanwhile or he no longer sees its target. It is deleted,
// and a folder's undeleted favorites with it in the one write whatever
// their targets, a folder being always his to reach, each deleted and last
// changed by him at the moment read after the last lock.
func (u *DeleteFavorite) Execute(ctx context.Context, id uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	w, f, err := u.d.Locks.own(ctx, actor, id)
	if err != nil {
		return err
	}
	w.children = f.Kind == domain.FavoriteFolder
	return u.d.Locks.run(ctx, actor, w, func(ctx context.Context, h favoriteHeld) error {
		return u.d.Favorites.DeleteFavorite(ctx, h.row.ID, u.d.Clock.Now(), actor.UserID)
	})
}
