package app_test

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Legacy fixtures have no cover rows. Never manufacture a signed success if
// one appears: actual image binding and bytes are tested through the full API.
type noProjectImages struct{ clock app.Clock }

func (f noProjectImages) LockImageBindings(ctx context.Context, q app.CoverImageChange) (time.Time, error) {
	if ctx.Value(inTxKey{}) != true || q.Current != nil || q.Next != nil {
		return time.Time{}, errors.New("invalid no-image fixture binding")
	}
	return f.clock.Now(), nil
}
func (noProjectImages) ApplyImageBindings(ctx context.Context, q app.CoverImageChange, _ time.Time) error {
	if ctx.Value(inTxKey{}) != true || q.Current != nil || q.Next != nil {
		return errors.New("invalid no-image fixture apply")
	}
	return nil
}
func (noProjectImages) ReadProjectImages(ctx context.Context, refs []app.ProjectImageReference) (map[uuid.UUID]app.ImageURL, error) {
	if ctx.Value(inTxKey{}) != true {
		return nil, errors.New("fixture signing outside transaction")
	}
	for _, ref := range refs {
		if ref.CoverImageAssetID != nil {
			return nil, errors.New("unexpected image in legacy fixture")
		}
	}
	return nil, nil
}

type imageCaller struct{ log *callLog }

func (f imageCaller) LockCaller(ctx context.Context, a shared.Actor, _ time.Time) error {
	f.log.add(ctx, "LockCaller %s", a.UserID)
	return nil
}

type imageGate struct{ log *callLog }

func (f imageGate) Enter(ctx context.Context, ids ...uuid.UUID) (func(), error) {
	f.log.add(ctx, "Enter %v", ids)
	return func() { f.log.calls = append(f.log.calls, fmt.Sprintf("Leave %v", ids)) }, nil
}

// Only image-returning writes add the account lock before the old sequence.
func withImageCaller(user uuid.UUID, calls []string) []string {
	return append([]string{"Begin", "Now", fmt.Sprintf("LockCaller %s", user)}, calls[1:]...)
}
