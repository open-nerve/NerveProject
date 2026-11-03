package app

import (
	"context"
	"errors"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func TestContentBindingsFullSetRefusesLast(t *testing.T) {
	u, f, q, at := contentFixture(t, domain.ScopeIssue)
	template := f.rows[accessID(8)]
	f.rows, q.ImageIDs, q.AttachmentIDs = map[uuid.UUID]domain.Asset{}, nil, nil
	for i := 1; i <= 1001; i++ {
		a := copyContentAsset(t, template, contentID(1000+i))
		if i == 1001 {
			a.Placement.ProjectID = accessID(99)
		}
		f.rows[a.ID] = a
		q.ImageIDs = append(q.ImageIDs, a.ID)
	}
	if _, err := u.LockContent(t.Context(), q); !errors.Is(err, domain.ErrAssetNotFound) {
		t.Fatalf("invalid last asset accepted: %v", err)
	}
	if err := u.BindContent(t.Context(), q, at); !errors.Is(err, domain.ErrAssetNotFound) || len(f.writes) != 0 || len(f.clockLocks) != 0 {
		t.Fatalf("invalid last asset left partial changes: %v writes=%v clock=%v", err, f.writes, f.clockLocks)
	}
}

func TestContentBindingsEmptyAndPrepared(t *testing.T) {
	u, f, q, _ := contentFixture(t, domain.ScopeIssue)
	q.ImageIDs, q.AttachmentIDs = nil, nil
	lost := errors.New("no transaction")
	f.lockErr = lost
	if _, err := u.LockContent(t.Context(), q); !errors.Is(err, lost) {
		t.Fatalf("empty set bypassed transaction port: %v", err)
	}
	u, f, q, at := contentFixture(t, domain.ScopeNewComment)
	a := f.rows[accessID(8)]
	a.BoundAt = &at
	f.rows[a.ID] = a
	if _, err := u.LockContent(t.Context(), q); !errors.Is(err, domain.ErrAssetBound) {
		t.Fatalf("already bound preparation reused: %v", err)
	}
	q.NewComment, q.Host.DraftID = false, accessID(99)
	if _, err := u.LockContent(t.Context(), q); !errors.Is(err, domain.ErrInvalidAsset) {
		t.Fatalf("invalid content host accepted: %v", err)
	}
}

func TestDraftAssetTransferFullSetRefusesLast(t *testing.T) {
	u, f, q, at := draftAssetFixture(t)
	template := f.rows[accessID(8)]
	f.rows, q.ImageIDs = map[uuid.UUID]domain.Asset{}, nil
	q.Publish, q.NewProjectID = true, q.OldProjectID
	for i := 1; i <= 1001; i++ {
		a := copyContentAsset(t, template, contentID(1000+i))
		if i == 1001 {
			a.Uploaded = false
		}
		f.rows[a.ID] = a
	}
	if _, err := u.LockDraftAssets(t.Context(), q); !errors.Is(err, domain.ErrAssetNotUploaded) {
		t.Fatalf("pending last asset accepted: %v", err)
	}
	if err := u.MoveDraftAssets(t.Context(), q, accessID(42), at); !errors.Is(err, domain.ErrAssetNotUploaded) || len(f.writes) != 0 || len(f.clockLocks) != 0 {
		t.Fatalf("pending last asset left partial changes: %v writes=%v clock=%v", err, f.writes, f.clockLocks)
	}
}

type draftDirectoryPage []uuid.UUID

func (p draftDirectoryPage) DraftAssetIDs(context.Context, uuid.UUID, uuid.UUID, int) ([]uuid.UUID, error) {
	return p, nil
}

func TestDraftAssetTransferDirectoryBoundary(t *testing.T) {
	for _, ids := range [][]uuid.UUID{{uuid.Nil()}, {accessID(99)}} {
		u, f, q, at := draftAssetFixture(t)
		q.ImageIDs = ids
		if _, err := u.LockDraftAssets(t.Context(), q); !errors.Is(err, domain.ErrAssetNotFound) {
			t.Fatalf("invalid draft reference accepted: %v", err)
		}
		if err := u.MoveDraftAssets(t.Context(), q, uuid.Nil(), at); !errors.Is(err, domain.ErrAssetNotFound) || len(f.writes) != 0 {
			t.Fatalf("invalid draft reference left partial changes: %v", err)
		}
	}
	for _, page := range []draftDirectoryPage{{accessID(8), accessID(6)}, {accessID(6), accessID(6)}, make([]uuid.UUID, 1001)} {
		u, f, q, _ := draftAssetFixture(t)
		u.d.Reader = page
		if _, err := u.LockDraftAssets(t.Context(), q); err == nil || len(f.locks) != 0 {
			t.Fatalf("broken directory reached lock stage: %v", err)
		}
	}
}
