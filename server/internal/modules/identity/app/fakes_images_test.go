package app_test

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
)

// Existing no-image account tests still require the complete transaction
// protocol. This fake refuses an unexpected image instead of hiding it.
type noProfileImages struct{}

func (noProfileImages) LockImageBindings(_ context.Context, q app.ProfileImageChanges) (time.Time, error) {
	if q.AvatarCurrent != nil || q.AvatarNext != nil || q.CoverCurrent != nil || q.CoverNext != nil {
		return time.Time{}, fmt.Errorf("unexpected image in no-image fixture")
	}
	return now, nil
}
func (noProfileImages) ApplyImageBindings(ctx context.Context, q app.ProfileImageChanges, _ time.Time) error {
	_, err := (noProfileImages{}).LockImageBindings(ctx, q)
	return err
}
func (noProfileImages) ReadProfileImages(_ context.Context, refs []app.ProfileImageReference) (map[uuid.UUID]app.ImageURL, error) {
	for _, r := range refs {
		if r.AvatarAssetID != nil || r.CoverImageAssetID != nil {
			return nil, fmt.Errorf("unexpected image in no-image fixture")
		}
	}
	return nil, nil
}
func getMe(users app.UserReader) *app.GetMe {
	f := newCredentialFixture()
	return app.NewGetMe(app.GetMeDeps{Users: users, Images: noProfileImages{}, Lock: f.lock(), Tx: f.tx, Clock: clocktest.At(now)})
}
func updateMe(users app.UpdateMeUsers) *app.UpdateMe {
	f := newCredentialFixture()
	return app.NewUpdateMe(app.UpdateMeDeps{Users: users, Bindings: noProfileImages{}, Images: noProfileImages{}, Lock: f.lock(), Tx: f.tx, Clock: clocktest.At(now)})
}
