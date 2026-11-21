package httpadapter

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
)

// ListFavoritesUseCase is app.ListFavorites.
type ListFavoritesUseCase interface {
	Execute(ctx context.Context, slug string) ([]app.NamedFavorite, error)
}

// CreateFavoriteUseCase is app.CreateFavorite: created tells a new
// favorite from the one the caller had already.
type CreateFavoriteUseCase interface {
	Execute(ctx context.Context, slug string, n domain.NewFavorite) (f app.NamedFavorite, created bool, err error)
}

// UpdateFavoriteUseCase is app.UpdateFavorite.
type UpdateFavoriteUseCase interface {
	Execute(ctx context.Context, id uuid.UUID, p domain.FavoritePatch) (app.NamedFavorite, error)
}

// DeleteFavoriteUseCase is app.DeleteFavorite.
type DeleteFavoriteUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) error
}

// ListFavorites serves GET /api/v0/workspaces/{slug}/favorites.
func (h handler) ListFavorites(ctx context.Context, req gen.ListFavoritesRequestObject) (gen.ListFavoritesResponseObject, error) {
	list, err := h.uc.ListFavorites.Execute(ctx, req.Slug)
	if err != nil {
		return nil, err
	}
	out := gen.FavoriteList{Data: make([]gen.Favorite, len(list))}
	for i, f := range list {
		out.Data[i] = favoriteOf(f)
	}
	return gen.ListFavorites200JSONResponse(out), nil
}

// CreateFavorite serves POST /api/v0/workspaces/{slug}/favorites: the
// body's values go to the use case as given (newFavorite), which checks
// them; a new favorite answers 201, the one the caller had already 200
// (P3c spec 2.4).
func (h handler) CreateFavorite(ctx context.Context, req gen.CreateFavoriteRequestObject) (gen.CreateFavoriteResponseObject, error) {
	f, created, err := h.uc.CreateFavorite.Execute(ctx, req.Slug, newFavorite(*req.Body))
	switch {
	case err != nil:
		return nil, err
	case created:
		return gen.CreateFavorite201JSONResponse(favoriteOf(f)), nil
	}
	return gen.CreateFavorite200JSONResponse(favoriteOf(f)), nil
}

// UpdateFavorite serves PATCH /api/v0/favorites/{favorite_id}: the fields
// the body gives go to the use case, those it leaves out nil
// (favoritePatch).
func (h handler) UpdateFavorite(ctx context.Context, req gen.UpdateFavoriteRequestObject) (gen.UpdateFavoriteResponseObject, error) {
	f, err := h.uc.UpdateFavorite.Execute(ctx, req.FavoriteID, favoritePatch(*req.Body))
	if err != nil {
		return nil, err
	}
	return gen.UpdateFavorite200JSONResponse(favoriteOf(f)), nil
}

// DeleteFavorite serves DELETE /api/v0/favorites/{favorite_id}.
func (h handler) DeleteFavorite(ctx context.Context, req gen.DeleteFavoriteRequestObject) (gen.DeleteFavoriteResponseObject, error) {
	if err := h.uc.DeleteFavorite.Execute(ctx, req.FavoriteID); err != nil {
		return nil, err
	}
	return gen.DeleteFavorite204Response{}, nil
}

// newFavorite is b as the use case takes it, unchecked (spec 2.4): its kind
// as given, each other field nil when it is left out; the use case checks
// each by the kind.
func newFavorite(b gen.FavoriteCreate) domain.NewFavorite {
	return domain.NewFavorite{Kind: domain.FavoriteKind(b.EntityType), EntityID: b.EntityID, Name: b.Name, ParentID: b.ParentID}
}

// favoritePatch is b as the use case takes it, unchecked (spec 2.4): a
// parent_id given, null among them, apart from one left out; a sequence and
// a name nil when they are left out.
func favoritePatch(b gen.FavoriteUpdate) domain.FavoritePatch {
	p := domain.FavoritePatch{SetParentID: b.ParentID.IsSpecified(), Sequence: b.Sequence, Name: b.Name}
	if p.SetParentID && !b.ParentID.IsNull() {
		parent := b.ParentID.MustGet()
		p.ParentID = &parent
	}
	return p
}

// favoriteOf is f as the API shows it (spec 2.4, R8): flat, with the name
// and the icon the use case answers it with, {} for none, and null for each
// of its project, its target and its folder it lacks.
func favoriteOf(f app.NamedFavorite) gen.Favorite {
	v := f.Favorite
	return gen.Favorite{ID: v.ID, WorkspaceID: v.WorkspaceID, ProjectID: orNull(v.ProjectID), EntityType: gen.FavoriteKind(v.Kind),
		EntityID: orNull(v.EntityID), ParentID: orNull(v.ParentID), Sequence: v.Sequence, Name: f.Name, LogoProps: logoOut(f.LogoProps),
		CreatedAt: v.CreatedAt}
}
