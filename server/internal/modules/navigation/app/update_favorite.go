package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// UpdateFavorite changes one's favorite by its id (updateFavorite).
type UpdateFavorite struct{ d FavoriteWriteDeps }

// NewUpdateFavorite returns the use case over d.
func NewUpdateFavorite(d FavoriteWriteDeps) *UpdateFavorite { return &UpdateFavorite{d: d} }

// Execute changes the caller's favorite id by p (P3c spec 2.4, 2.5, 2.7).
// Its refusals, in order: his favorite read without a lock and decided on
// (FavoriteLocks.own), every favorite he may not have
// navigation.favorite_not_found, the one value, whose target he no longer
// sees among them, a role refused the Authorizer's shared.Forbidden; then
// every problem of p as a change of a favorite of its kind in one 422
// (domain.CheckFavoritePatch), checked only once the favorite is read, and
// before any lock; then, under its locks and decision (FavoriteLocks.run),
// the favorite read again, the same 404 when it was deleted meanwhile or
// he no longer sees its target; then a parent p names that is none of his
// undeleted folders in the workspace, 422 parent_id not_allowed. A p that
// leaves the favorite as its lock read it, {} among them, answers it as it
// is, with nothing written and its updated_at kept; else each field p
// gives replaces its own, last changed by him at the moment read after the
// last lock, and a folder's name, in any case, that another of his folders
// there has is 409, from the store, its own in another case not. It is
// answered with its target's name and icon now.
func (u *UpdateFavorite) Execute(ctx context.Context, id uuid.UUID, p domain.FavoritePatch) (NamedFavorite, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return NamedFavorite{}, err
	}
	w, f, err := u.d.Locks.own(ctx, actor, id)
	if err != nil {
		return NamedFavorite{}, err
	}
	p, problems := domain.CheckFavoritePatch(p, f.Kind)
	if len(problems) > 0 {
		return NamedFavorite{}, shared.Invalid(problems...)
	}
	if p.SetParentID {
		w.parentID = p.ParentID
	}
	var out NamedFavorite
	err = u.d.Locks.run(ctx, actor, w, func(ctx context.Context, h favoriteHeld) error {
		if !h.parent {
			return shared.Invalid(parentNotAllowed)
		}
		next, changed := p.Apply(h.row)
		if changed {
			next.UpdatedAt = u.d.Clock.Now()
			if err := u.d.Favorites.UpdateFavorite(ctx, next, actor.UserID); err != nil {
				return err
			}
		}
		out, _ = h.sight.shows(next)
		return nil
	})
	if err != nil {
		return NamedFavorite{}, err
	}
	return out, nil
}
