package app_test

import (
	"context"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
)

type assetPurgeRows struct {
	page  app.PurgePage
	calls int
}

func (r *assetPurgeRows) PurgeAssets(context.Context, time.Time, uuid.UUID, int) (app.PurgePage, error) {
	r.calls++
	return r.page, nil
}
func TestAssetPurgeRejectsInvalidProgress(t *testing.T) {
	id := uuid.NewV7()
	for _, page := range []app.PurgePage{{Scanned: -1}, {Scanned: 2, After: id}, {Scanned: 0, Deleted: 1}, {Scanned: 1, Deleted: 2, After: id}, {Scanned: 1, After: uuid.Nil()}, {Scanned: 0, After: id}} {
		rows := &assetPurgeRows{page: page}
		if _, err := app.NewPurger(rows).Page(t.Context(), time.Now(), 1, app.PurgeCursor{}); err == nil {
			t.Fatalf("accepted invalid progress: %+v", page)
		}
	}
	for _, limit := range []int{0, -1, 1 << 32} {
		rows := &assetPurgeRows{}
		if _, err := app.NewPurger(rows).Page(t.Context(), time.Now(), limit, app.PurgeCursor{}); err == nil || rows.calls != 0 {
			t.Fatalf("invalid batch reached repository: limit=%d calls=%d err=%v", limit, rows.calls, err)
		}
	}
}
