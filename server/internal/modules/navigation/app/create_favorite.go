package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// FavoriteWriteDeps are what the writes of one's favorites need: their
// locks and decision, the favorites' writes and the clock.
type FavoriteWriteDeps struct {
	Locks     FavoriteLocks
	Favorites FavoriteWriter
	Clock     Clock
}

// The problems of a target and a parent one may not have (P3c spec 2.5,
// R7): each one value, whatever the reason, so its answer tells nothing of
// what is there.
var (
	targetNotAllowed = shared.FieldError{Field: "entity_id", Code: shared.FieldNotAllowed, Message: "is no target you may favor in this workspace"}
	parentNotAllowed = shared.FieldError{Field: "parent_id", Code: shared.FieldNotAllowed, Message: "is none of your folders in this workspace"}
)

// CreateFavorite creates one's favorite in a workspace (createFavorite).
type CreateFavorite struct{ d FavoriteWriteDeps }

// NewCreateFavorite returns the use case over d.
func NewCreateFavorite(d FavoriteWriteDeps) *CreateFavorite { return &CreateFavorite{d: d} }

// Execute creates the caller's favorite n in the workspace slug names, and
// tells whether it created it (P3c spec 2.4, 2.5, 2.7). Its refusals, in
// order: every problem of n's body in one 422, before any read
// (domain.CheckNewFavorite); a slug of none and a workspace he does not see
// workspace.not_found, a role refused the Authorizer's shared.Forbidden,
// before any lock (FavoriteLocks.creation); then, under its locks and
// decision (FavoriteLocks.run), one 422 with each of its target and its
// parent he may not have: a target not there, another workspace's, one he
// does not see and an archived one alike (sight.eligible); a parent none of
// his undeleted folders there.
// Then, his undeleted favorite of the target answered as it is, not
// created, its folder and sequence kept; else, at the moment read after
// the last lock, his favorite of the target's project (the project itself
// for a project's), one step past his greatest sequence there
// (domain.SequenceAfter), made and last changed by him; a folder whose
// name another of his folders there has is 409, from the store. It is
// answered with its target's name and icon.
func (u *CreateFavorite) Execute(ctx context.Context, slug string, n domain.NewFavorite) (NamedFavorite, bool, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return NamedFavorite{}, false, err
	}
	n, problems := domain.CheckNewFavorite(n)
	if len(problems) > 0 {
		return NamedFavorite{}, false, shared.Invalid(problems...)
	}
	w, err := u.d.Locks.creation(ctx, actor, slug, n)
	if err != nil {
		return NamedFavorite{}, false, err
	}
	var out NamedFavorite
	var created bool
	err = u.d.Locks.run(ctx, actor, w, func(ctx context.Context, h favoriteHeld) error {
		var a aim
		var refused []shared.FieldError
		if w.target != nil {
			var eligible bool
			if a, eligible = h.sight.eligible(*w.target); !eligible {
				refused = append(refused, targetNotAllowed)
			}
		}
		if !h.parent {
			refused = append(refused, parentNotAllowed)
		}
		if len(refused) > 0 {
			return shared.Invalid(refused...)
		}
		now := u.d.Clock.Now()
		greatest, err := u.d.Favorites.GreatestFavoriteSequence(ctx, actor.UserID, w.workspaceID)
		if err != nil {
			return err
		}
		f := domain.Favorite{ID: uuid.NewV7(), WorkspaceID: w.workspaceID, ProjectID: a.projectID, UserID: actor.UserID, Kind: n.Kind, EntityID: n.EntityID,
			ParentID: n.ParentID, Sequence: domain.SequenceAfter(greatest), CreatedAt: now, UpdatedAt: now}
		if n.Name != nil {
			f.Name = *n.Name
		}
		stored, isNew, err := u.d.Favorites.InsertFavorite(ctx, f, actor.UserID)
		if err != nil {
			return err
		}
		out, created = answered(stored, a), isNew
		return nil
	})
	if err != nil {
		return NamedFavorite{}, false, err
	}
	return out, created, nil
}
