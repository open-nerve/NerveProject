package app

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListFavoritesDeps are what the list of one's favorites needs: the
// workspace module's directory, the Authorizer, the favorites, the reads of
// their targets and the read-only snapshot transactions.
type ListFavoritesDeps struct {
	Workspaces Workspaces
	Authorizer shared.Authorizer
	Favorites  FavoriteReader
	Targets    Targets
	Tx         shared.TxManager
}

// ListFavorites lists one's favorites in a workspace (listFavorites).
type ListFavorites struct{ d ListFavoritesDeps }

// NewListFavorites returns the use case over d.
func NewListFavorites(d ListFavoritesDeps) *ListFavorites { return &ListFavorites{d: d} }

// Execute lists the caller's favorites in the workspace slug names (P3c
// spec 2.4, 2.5, 2.7), in one read-only snapshot transaction and without a
// lock: the workspace and favorite.read's decision on it, a slug of none
// and a workspace he does not see workspace.not_found, a role refused the
// Authorizer's shared.Forbidden; then his undeleted favorites there, those
// in his folders too, by sequence descending, then by id descending; then
// what he sees of their targets (Targets.sight). It answers every folder,
// and each favorite of a target he sees, with its target's name and icon
// now; one whose target he no longer sees is kept, not listed (M7 design
// 3.8). The projects' toggles do not matter (R5).
func (u *ListFavorites) Execute(ctx context.Context, slug string) ([]NamedFavorite, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	var out []NamedFavorite
	err = u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		workspaceID, err := decideWorkspace(ctx, u.d.Workspaces, u.d.Authorizer, actor, domain.ActionFavoriteRead, slug)
		if err != nil {
			return err
		}
		favorites, err := u.d.Favorites.Favorites(ctx, actor.UserID, workspaceID)
		if err != nil {
			return err
		}
		var targets []entity
		for _, f := range favorites {
			if e, ofTarget := entityOf(f); ofTarget {
				targets = append(targets, e)
			}
		}
		s, err := u.d.Targets.sight(ctx, actor, workspaceID, targets)
		if err != nil {
			return err
		}
		out = s.visible(favorites)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
