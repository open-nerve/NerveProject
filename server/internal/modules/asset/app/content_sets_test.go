package app

import (
	"encoding/binary"
	"errors"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func contentID(n int) uuid.UUID {
	var id uuid.UUID
	binary.BigEndian.PutUint64(id[8:], uint64(n))
	return id
}
func copyContentAsset(t *testing.T, a domain.Asset, id uuid.UUID) domain.Asset {
	t.Helper()
	a.ID = id
	key, err := domain.NewObjectKey('o', id, accessID(90))
	if err != nil {
		t.Fatal(err)
	}
	final := *a.Metadata.Final
	final.Key = key
	a.Metadata.Final = &final
	return a
}

func TestContentBindingsFullOrderedSet(t *testing.T) {
	u, f, q, _ := contentFixture(t, domain.ScopeIssue)
	template := f.rows[accessID(8)]
	f.rows = map[uuid.UUID]domain.Asset{}
	q.ImageIDs, q.AttachmentIDs = nil, nil
	for i := 1001; i > 0; i-- {
		id := contentID(1000 + i)
		f.rows[id] = copyContentAsset(t, template, id)
		q.ImageIDs = append(q.ImageIDs, id, id)
	}
	at, err := u.LockContent(t.Context(), q)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.locks) != 2 || len(f.locks[0]) != 1000 || len(f.locks[1]) != 1 || f.locks[0][0] != contentID(1001) || f.locks[0][999] != contentID(2000) || f.locks[1][0] != contentID(2001) || !slices.Equal(f.clockLocks, []int{2}) {
		t.Fatalf("1001 unique lock boundary lost: batches=%d clock=%v", len(f.locks), f.clockLocks)
	}
	if err := u.BindContent(t.Context(), q, at); err != nil {
		t.Fatal(err)
	}
	if len(f.writes) != 1001 || f.writes[0] != contentID(1001) || f.writes[1000] != contentID(2001) {
		t.Fatal("batch truncated or writes unsorted")
	}
}

func TestContentBindingsMissingAndPortErrors(t *testing.T) {
	for _, id := range []uuid.UUID{uuid.Nil(), accessID(99)} {
		u, f, q, at := contentFixture(t, domain.ScopeIssue)
		q.ImageIDs = append(q.ImageIDs, id)
		if _, err := u.LockContent(t.Context(), q); !errors.Is(err, domain.ErrAssetNotFound) {
			t.Fatalf("bad ID accepted: %v", err)
		}
		if err := u.BindContent(t.Context(), q, at); !errors.Is(err, domain.ErrAssetNotFound) || len(f.writes) != 0 {
			t.Fatalf("bad ID left partial changes: %v %v", err, f.writes)
		}
	}
	for _, op := range []string{"lock", "write"} {
		u, f, q, at := contentFixture(t, domain.ScopeIssue)
		lost := errors.New("content port failed")
		if op == "lock" {
			f.lockErr = lost
			if _, err := u.LockContent(t.Context(), q); !errors.Is(err, lost) {
				t.Fatalf("lock error lost: %v", err)
			}
		} else {
			f.writeErr = lost
		}
		if err := u.BindContent(t.Context(), q, at); !errors.Is(err, lost) {
			t.Fatalf("%s error lost: %v", op, err)
		}
	}
}

func TestDraftAssetTransferCompleteSetAndLockOrder(t *testing.T) {
	t.Run("1001 scope rows are collected before batched locks", func(t *testing.T) {
		u, f, q, _ := draftAssetFixture(t)
		template := f.rows[accessID(8)]
		f.rows, q.ImageIDs = map[uuid.UUID]domain.Asset{}, nil
		for i := 1001; i > 0; i-- {
			id := contentID(1000 + i)
			f.rows[id] = copyContentAsset(t, template, id)
		}
		at, err := u.LockDraftAssets(t.Context(), q)
		if err != nil {
			t.Fatal(err)
		}
		if len(f.pages) != 2 || len(f.locks) != 2 || len(f.locks[0]) != 1000 || len(f.locks[1]) != 1 || f.locks[0][0] != contentID(1001) || f.locks[1][0] != contentID(2001) || !slices.Equal(f.clockLocks, []int{2}) {
			t.Fatalf("directory or lock batch lost: pages=%d locks=%d clock=%v", len(f.pages), len(f.locks), f.clockLocks)
		}
		if err := u.MoveDraftAssets(t.Context(), q, uuid.Nil(), at); err != nil {
			t.Fatal(err)
		}
		if len(f.writes) != 1001 || f.rows[contentID(2001)].Placement.ProjectID != *q.NewProjectID {
			t.Fatal("draft move truncated last page")
		}
	})
	t.Run("foreign low reference locks before own high scope and refuses without writing", func(t *testing.T) {
		u, f, q, _ := draftAssetFixture(t)
		low := copyContentAsset(t, f.rows[accessID(8)], accessID(1))
		low.Placement.DraftID, low.Placement.Scope.ID = accessID(99), accessID(99)
		f.rows[low.ID] = low
		q.ImageIDs = append(q.ImageIDs, low.ID)
		if _, err := u.LockDraftAssets(t.Context(), q); !errors.Is(err, domain.ErrAssetNotFound) || len(f.writes) != 0 {
			t.Fatalf("foreign image=%v writes=%v", err, f.writes)
		}
		if len(f.locks) != 1 || !slices.Equal(f.locks[0], []uuid.UUID{accessID(1), accessID(6), accessID(8), accessID(9), accessID(10)}) {
			t.Fatalf("reference locked after scope: %v", f.locks)
		}
	})
	t.Run("deleted reference refuses even though scope directory omits it", func(t *testing.T) {
		u, f, q, at := draftAssetFixture(t)
		a := f.rows[accessID(8)]
		a.DeletedAt = &at
		f.rows[a.ID] = a
		if _, err := u.LockDraftAssets(t.Context(), q); !errors.Is(err, domain.ErrAssetNotFound) {
			t.Fatalf("deleted reference=%v", err)
		}
	})
	for _, op := range []string{"list", "lock", "write"} {
		t.Run(op+" error", func(t *testing.T) {
			u, f, q, at := draftAssetFixture(t)
			lost := errors.New("draft asset port failed")
			switch op {
			case "list":
				f.listErr = lost
			case "lock":
				f.lockErr = lost
			case "write":
				f.writeErr = lost
			}
			if err := u.MoveDraftAssets(t.Context(), q, uuid.Nil(), at); !errors.Is(err, lost) {
				t.Fatalf("%s error lost: %v", op, err)
			}
		})
	}
}
