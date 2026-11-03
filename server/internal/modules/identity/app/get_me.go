package app

import (
	"context"
	"errors"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type GetMeDeps struct {
	Users  UserReader
	Images ImageReader
	Lock   CredentialLock
	Tx     shared.TxManager
	Clock  Clock
}

// GetMe signs profile images under the caller's credential/account lock.
type GetMe struct{ d GetMeDeps }

func NewGetMe(d GetMeDeps) *GetMe { return &GetMe{d: d} }
func (g *GetMe) Execute(ctx context.Context) (domain.User, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.User{}, err
	}
	var out domain.User
	err = g.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if _, err := g.d.Lock.Lock(ctx, actor, g.d.Clock.Now()); err != nil {
			return err
		}
		u, err := g.d.Users.GetUser(ctx, actor.UserID)
		if errors.Is(err, ErrNotFound) {
			return shared.Unauthenticated()
		}
		if err != nil {
			return err
		}
		out, err = withProfileImages(ctx, g.d.Images, u)
		return err
	})
	if err != nil {
		return domain.User{}, err
	}
	return out, nil
}
