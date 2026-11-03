package app

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

type imageLocksFunc func(context.Context, []uuid.UUID) ([]domain.Asset, error)

func (f imageLocksFunc) LockAssets(ctx context.Context, ids []uuid.UUID) ([]domain.Asset, error) {
	return f(ctx, ids)
}

func TestReadBoundImagesBatch(t *testing.T) {
	for _, count := range []int{0, 1, 1000, 1001} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			f, ref, now := imageReadFixture(t, domain.UserAvatar)
			source := f.rows[ref.AssetID]
			f.rows = make(map[uuid.UUID]domain.Asset)
			var refs []ImageReference
			for i := count; i > 0; i-- {
				var id uuid.UUID
				binary.BigEndian.PutUint32(id[12:], uint32(i))
				row := source
				row.ID = id
				f.rows[id] = row
				r := ref
				r.AssetID = id
				refs = append(refs, r, r)
			}
			refs = append(refs, ImageReference{})
			calls, clockCalls := 0, 0
			expectedLocks := (count + 999) / 1000
			u := NewImageReader(ImageReadDeps{Locks: f, RuntimeNow: func() time.Time { clockCalls++; return now }, Signer: readSignFunc(func(_ context.Context, g domain.ReadGrant) (domain.SignedRequest, error) {
				calls++
				if len(f.locks) != expectedLocks {
					t.Fatalf("signed before all blocks locked: %d", len(f.locks))
				}
				return domain.SignedRequest{URL: "image", ExpiresAt: g.ExpiresAt}, nil
			})})
			got, err := u.ReadBoundImages(t.Context(), refs)
			if err != nil || len(got) != count || calls != count || len(f.locks) != expectedLocks || clockCalls > 1 {
				t.Fatalf("batch images=%d calls=%d locks=%d clock=%d err=%v", len(got), calls, len(f.locks), clockCalls, err)
			}
			var all []uuid.UUID
			for _, block := range f.locks {
				if len(block) > 1000 {
					t.Fatalf("unbounded SQL: %d", len(block))
				}
				all = append(all, block...)
			}
			if len(all) != count || !slices.IsSortedFunc(all, uuid.UUID.Compare) || len(slices.Compact(all)) != count || len(f.writes) != 0 {
				t.Fatal("unordered/duplicated/incomplete locks or DML")
			}
		})
	}
}

func TestReadBoundImagesLaterBlockFailure(t *testing.T) {
	f, ref, now := imageReadFixture(t, domain.UserAvatar)
	var refs []ImageReference
	for i := 1; i <= 1001; i++ {
		r := ref
		binary.BigEndian.PutUint32(r.AssetID[12:], uint32(i))
		refs = append(refs, r)
	}
	boom := errors.New("second SQL block failed")
	locks := 0
	u := NewImageReader(ImageReadDeps{Locks: imageLocksFunc(func(_ context.Context, ids []uuid.UUID) ([]domain.Asset, error) {
		locks++
		if locks == 2 {
			return nil, boom
		}
		row := f.rows[ref.AssetID]
		row.ID = ids[0]
		return []domain.Asset{row}, nil
	}), RuntimeNow: func() time.Time { return now }, Signer: readSignFunc(func(context.Context, domain.ReadGrant) (domain.SignedRequest, error) {
		t.Fatal("partial batch signed before later SQL error")
		return domain.SignedRequest{}, nil
	})})
	got, err := u.ReadBoundImages(t.Context(), refs)
	if !errors.Is(err, boom) || got != nil || locks != 2 {
		t.Fatalf("got=%+v locks=%d err=%v", got, locks, err)
	}
}
