package app_test

import (
	"context"
	"errors"
	"math"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
)

type assetPurgeRows struct {
	ids, deleted []uuid.UUID
	calls, limit int
	inTx         bool
	steps        []string
	fail         string
	boom         error
}

func (r *assetPurgeRows) PurgeAssetCandidates(_ context.Context, _ time.Time, _ uuid.UUID, limit int) ([]uuid.UUID, error) {
	r.calls++
	r.limit = limit
	r.steps = append(r.steps, "candidates")
	if r.inTx {
		return nil, errors.New("candidate query held transaction")
	}
	if r.fail == "candidate" {
		return nil, r.boom
	}
	return r.ids, nil
}
func (r *assetPurgeRows) DetachAssetReferences(context.Context, []uuid.UUID) error {
	if !r.inTx {
		return errors.New("unprotected detachment")
	}
	r.steps = append(r.steps, "references")
	if r.fail == "references" {
		return r.boom
	}
	return nil
}
func (r *assetPurgeRows) DeletePurgeAssets(context.Context, time.Time, []uuid.UUID) ([]uuid.UUID, error) {
	if !r.inTx {
		return nil, errors.New("unprotected delete")
	}
	r.steps = append(r.steps, "delete")
	if r.fail == "delete" {
		return nil, r.boom
	}
	return r.deleted, nil
}
func (r *assetPurgeRows) WithinTx(ctx context.Context, f func(context.Context) error) error {
	r.inTx = true
	r.steps = append(r.steps, "begin")
	defer func() { r.inTx = false }()
	if err := f(ctx); err != nil {
		r.steps = append(r.steps, "rollback")
		return err
	}
	if r.fail == "commit" {
		r.steps = append(r.steps, "rollback")
		return r.boom
	}
	r.steps = append(r.steps, "commit")
	return nil
}
func (r *assetPurgeRows) purger() *app.Purger {
	return app.NewPurger(app.PurgeDeps{Candidates: r, Deleter: r, References: r, Tx: r})
}
func TestAssetPurgeRejectsInvalidProgress(t *testing.T) {
	a, b := uuid.NewV7(), uuid.NewV7()
	for _, tc := range []struct {
		ids   []uuid.UUID
		limit int
	}{{[]uuid.UUID{uuid.Nil()}, 1}, {[]uuid.UUID{b, a}, 2}, {[]uuid.UUID{a, a}, 2}, {[]uuid.UUID{a, b}, 1}} {
		rows := &assetPurgeRows{ids: tc.ids}
		if _, err := rows.purger().Page(t.Context(), time.Now(), tc.limit, app.PurgeCursor{}); err == nil {
			t.Fatalf("accepted invalid candidates: %v", tc.ids)
		}
		if len(rows.steps) != 1 {
			t.Fatal("invalid candidates reached transaction")
		}
	}
	for _, ids := range [][]uuid.UUID{{b}, {a, a}} {
		rows := &assetPurgeRows{ids: []uuid.UUID{a}, deleted: ids}
		if _, err := rows.purger().Page(t.Context(), time.Now(), 1, app.PurgeCursor{}); err == nil {
			t.Fatalf("accepted invalid deleted result: %v", ids)
		}
		if rows.steps[len(rows.steps)-1] != "rollback" {
			t.Fatal("bad result committed")
		}
	}
	for _, limit := range []int{0, -1, 1 << 32} {
		rows := &assetPurgeRows{}
		if _, err := rows.purger().Page(t.Context(), time.Now(), limit, app.PurgeCursor{}); err == nil || rows.calls != 0 {
			t.Fatal("invalid batch reached repository")
		}
	}
}
func TestImageReferencePurgeProtocol(t *testing.T) {
	id := uuid.NewV7()
	rows := &assetPurgeRows{ids: []uuid.UUID{id}}
	page, err := rows.purger().Page(t.Context(), time.Now(), math.MaxInt32, app.PurgeCursor{})
	if err != nil || page.Done || page.Scanned != 1 || page.Deleted != 0 || page.Cursor.After != id || rows.limit != 1000 {
		t.Fatalf("zero deletion did not progress: %+v %v limit=%d", page, err, rows.limit)
	}
	if !reflect.DeepEqual(rows.steps, []string{"candidates", "begin", "references", "delete", "commit"}) {
		t.Fatal(rows.steps)
	}
	rows.ids = nil
	rows.steps = nil
	done, err := rows.purger().Page(t.Context(), time.Now(), 1, page.Cursor)
	if err != nil || !done.Done || done.Cursor != page.Cursor || !reflect.DeepEqual(rows.steps, []string{"candidates"}) {
		t.Fatalf("empty page=%+v %v steps=%v", done, err, rows.steps)
	}
}
func TestImageReferencePurgeErrorsKeepCursor(t *testing.T) {
	for _, stage := range []string{"candidate", "references", "delete", "commit"} {
		t.Run(stage, func(t *testing.T) {
			rows := &assetPurgeRows{ids: []uuid.UUID{uuid.NewV7()}, fail: stage, boom: errors.New(stage)}
			original := app.PurgeCursor{}
			got, err := rows.purger().Page(t.Context(), time.Now(), 1, original)
			if !errors.Is(err, rows.boom) || got.Cursor != original || got.Scanned != 0 || got.Deleted != 0 || got.Done {
				t.Fatalf("failed page advanced: %+v %v", got, err)
			}
		})
	}
}
