package app

import (
	"context"
	"errors"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type UpdateMeUsers interface {
	UserReader
	UserUpdater
}
type UpdateMeDeps struct {
	Users    UpdateMeUsers
	Bindings ImageBindings
	Images   ImageReader
	Lock     CredentialLock
	Tx       shared.TxManager
	Clock    Clock
}

// UpdateMe atomically updates account fields and its two image bindings.
type UpdateMe struct{ d UpdateMeDeps }

func NewUpdateMe(d UpdateMeDeps) *UpdateMe { return &UpdateMe{d: d} }
func (u *UpdateMe) Execute(ctx context.Context, p domain.UserPatch) (domain.User, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.User{}, err
	}
	if err := domain.CheckUserPatch(p); err != nil {
		return domain.User{}, err
	}
	var out domain.User
	err = u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if _, err := u.d.Lock.Lock(ctx, actor, u.d.Clock.Now()); err != nil {
			return err
		}
		current, err := u.d.Users.GetUser(ctx, actor.UserID)
		if errors.Is(err, ErrNotFound) {
			return shared.Unauthenticated()
		}
		if err != nil {
			return err
		}
		change := ProfileImageChanges{UserID: actor.UserID, AvatarCurrent: current.AvatarAssetID, AvatarNext: current.AvatarAssetID, CoverCurrent: current.CoverImageAssetID, CoverNext: current.CoverImageAssetID}
		if p.SetAvatar {
			change.AvatarNext = p.AvatarAssetID
		}
		if p.SetCoverImage {
			change.CoverNext = p.CoverImageAssetID
		}
		at, err := u.d.Bindings.LockImageBindings(ctx, change)
		if err != nil {
			return err
		}
		updated, err := u.d.Users.UpdateUser(ctx, actor.UserID, p, at)
		if errors.Is(err, ErrNotFound) {
			return shared.Unauthenticated()
		}
		if err != nil {
			return err
		}
		if err := u.d.Bindings.ApplyImageBindings(ctx, change, at); err != nil {
			return err
		}
		out, err = withProfileImages(ctx, u.d.Images, updated)
		return err
	})
	if err != nil {
		return domain.User{}, err
	}
	return out, nil
}
