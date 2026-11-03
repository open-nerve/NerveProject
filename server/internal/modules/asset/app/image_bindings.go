package app

import (
	"context"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

type ImageBindingDeps struct {
	Locks    AssetLocker
	Writer   MetadataWriter
	Bindings ImageBindingWriter
	Clock    Clock
}
type ImageBinder struct{ d ImageBindingDeps }

func NewImageBindings(d ImageBindingDeps) *ImageBinder { return &ImageBinder{d: d} }
func (u *ImageBinder) LockImageBindings(ctx context.Context, request ImageBindingRequest) (time.Time, error) {
	rows, err := u.lock(ctx, request)
	if err != nil {
		return time.Time{}, err
	}
	at := u.d.Clock.Now()
	if err := checkImageBindings(request, rows, at); err != nil {
		return time.Time{}, err
	}
	return at, nil
}

func (u *ImageBinder) ApplyImageBindings(ctx context.Context, request ImageBindingRequest, at time.Time) error {
	if at.IsZero() {
		return domain.ErrInvalidAsset
	}
	rows, err := u.lock(ctx, request)
	if err != nil {
		return err
	}
	if err := checkImageBindings(request, rows, at); err != nil {
		return err
	}
	for _, change := range request.Changes {
		if sameImageID(change.CurrentID, change.NextID) {
			continue
		}
		if change.CurrentID != nil {
			old := rows[*change.CurrentID]
			if old.DeletedAt == nil {
				markDeleted(&old, DeletionStamp{At: at, By: &request.Host.ApplicantID})
				if _, err := u.d.Writer.SaveAsset(ctx, old); err != nil {
					return err
				}
			}
		}
		if change.NextID != nil {
			next := rows[*change.NextID]
			if _, err := u.d.Bindings.BindImage(ctx, ImageBindingWrite{AssetID: next.ID,
				ProjectID: request.Host.ProjectID, Revision: next.StorageRevision, At: at, By: request.Host.ApplicantID}); err != nil {
				return err
			}
		}
	}
	return nil
}
