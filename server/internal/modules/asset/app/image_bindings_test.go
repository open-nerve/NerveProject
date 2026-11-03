package app

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

type imageRows struct {
	rows              map[uuid.UUID]domain.Asset
	locks             [][]uuid.UUID
	writes            []uuid.UUID
	lockErr, writeErr error
}

func (f *imageRows) LockAssets(_ context.Context, ids []uuid.UUID) ([]domain.Asset, error) {
	f.locks = append(f.locks, slices.Clone(ids))
	if f.lockErr != nil {
		return nil, f.lockErr
	}
	var out []domain.Asset
	for _, id := range ids {
		if a, ok := f.rows[id]; ok {
			out = append(out, a)
		}
	}
	return out, nil
}
func (f *imageRows) SaveAsset(_ context.Context, a domain.Asset) (domain.Asset, error) {
	if f.writeErr != nil {
		return domain.Asset{}, f.writeErr
	}
	f.rows[a.ID] = a
	f.writes = append(f.writes, a.ID)
	return a, nil
}
func (f *imageRows) BindImage(_ context.Context, in ImageBindingWrite) (domain.Asset, error) {
	if f.writeErr != nil {
		return domain.Asset{}, f.writeErr
	}
	a := f.rows[in.AssetID]
	if a.StorageRevision != in.Revision {
		return domain.Asset{}, domain.ErrAssetScopeChanged
	}
	a.BoundAt = &in.At
	a.UpdatedAt = in.At
	a.UpdatedByID = &in.By
	a.StorageRevision++
	if a.Placement.Scope.Kind == domain.ScopeNewProject {
		a.Placement.Scope = domain.Scope{Kind: domain.ScopeProject, ID: in.ProjectID}
		a.Placement.ProjectID = in.ProjectID
	}
	f.rows[a.ID] = a
	f.writes = append(f.writes, a.ID)
	return a, nil
}

type imageClockFunc func() time.Time

func (f imageClockFunc) Now() time.Time { return f() }
func imageFixture(t *testing.T) (*ImageBinder, *imageRows, ImageBindingRequest, time.Time) {
	t.Helper()
	at := time.Date(2026, 11, 3, 0, 0, 0, 123456000, time.UTC)
	actor := accessID(1)
	req := ImageBindingRequest{Host: ImageBindingHost{ApplicantID: actor}, Changes: []ImageChange{
		{Field: "avatar_asset_id", Purpose: domain.UserAvatar, CurrentID: imgPtr(accessID(8)), NextID: imgPtr(accessID(6))},
		{Field: "cover_image_asset_id", Purpose: domain.UserCover, CurrentID: imgPtr(accessID(7)), NextID: imgPtr(accessID(5))},
	}}
	f := &imageRows{rows: map[uuid.UUID]domain.Asset{}}
	for _, c := range req.Changes {
		for _, id := range []*uuid.UUID{c.CurrentID, c.NextID} {
			key, err := domain.NewObjectKey('o', *id, accessID(90))
			if err != nil {
				t.Fatal(err)
			}
			a := domain.Asset{ID: *id, Purpose: c.Purpose, Placement: domain.Placement{UserID: actor, Scope: domain.Scope{Kind: domain.ScopeUser}},
				Name: "image.png", Size: 3, ContentType: "image/png", Uploaded: true, CreatedAt: at.Add(-24 * time.Hour), UpdatedAt: at.Add(-time.Hour),
				Metadata: domain.StorageMetadata{SchemaVersion: 1, Origin: domain.OriginCopy, Candidates: []domain.Candidate{},
					Final: &domain.ObjectRecord{Key: key, Phase: domain.ObjectActive, SafeAfter: at, Size: 3, ContentType: "image/png", ETag: "tag", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}}
			if *id == *c.CurrentID {
				old := at.Add(-10 * 24 * time.Hour)
				a.CreatedAt = old
				a.BoundAt = &old
			}
			f.rows[a.ID] = a
		}
	}
	u := NewImageBindings(ImageBindingDeps{Locks: f, Writer: f, Bindings: f, Clock: imageClockFunc(func() time.Time {
		if len(f.locks) != 1 {
			t.Fatalf("clock before complete asset lock or read twice: %v", f.locks)
		}
		return at
	})})
	return u, f, req, at
}
func imgPtr[T any](v T) *T { return &v }

func TestImageBindingsReplace(t *testing.T) {
	u, f, req, want := imageFixture(t)
	at, err := u.LockImageBindings(t.Context(), req)
	if err != nil || !at.Equal(want) || len(f.writes) != 0 {
		t.Fatalf("lock=%v %v writes=%v", at, err, f.writes)
	}
	if err := u.ApplyImageBindings(t.Context(), req, at); err != nil {
		t.Fatal(err)
	}
	expected := []uuid.UUID{accessID(5), accessID(6), accessID(7), accessID(8)}
	if !reflect.DeepEqual(f.locks, [][]uuid.UUID{expected, expected}) {
		t.Fatalf("old/new complete ascending locks=%v", f.locks)
	}
	for _, c := range req.Changes {
		old, next := f.rows[*c.CurrentID], f.rows[*c.NextID]
		if old.DeletedAt == nil || !old.DeletedAt.Equal(at) || old.DeletedByID == nil || *old.DeletedByID != req.Host.ApplicantID {
			t.Fatalf("old stamp=%+v", old)
		}
		if next.BoundAt == nil || !next.BoundAt.Equal(at) || next.DeletedAt != nil || next.UpdatedByID == nil || *next.UpdatedByID != req.Host.ApplicantID {
			t.Fatalf("next stamp=%+v", next)
		}
		if !old.UpdatedAt.Equal(at) || !next.UpdatedAt.Equal(at) || old.Placement.UserID != req.Host.ApplicantID || next.Placement.UserID != req.Host.ApplicantID {
			t.Fatal("audit/applicant changed")
		}
	}
}
func TestImageBindingsReject(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*domain.Asset)
		want   error
		field  string
	}{
		{"other applicant", func(a *domain.Asset) { a.Placement.UserID = accessID(2) }, domain.ErrAssetNotFound, ""},
		{"scope", func(a *domain.Asset) {
			a.Placement.Scope = domain.Scope{Kind: domain.ScopeWorkspace, ID: accessID(2)}
			a.Placement.WorkspaceID = accessID(2)
		}, domain.ErrAssetNotFound, ""},
		{"extra parent", func(a *domain.Asset) { a.Placement.CommentID = accessID(2) }, domain.ErrAssetNotFound, ""},
		{"deleted", func(a *domain.Asset) { a.DeletedAt = &a.CreatedAt }, domain.ErrAssetNotFound, ""},
		{"expired", func(a *domain.Asset) { a.CreatedAt = a.CreatedAt.Add(-6*24*time.Hour - time.Microsecond) }, domain.ErrAssetNotFound, ""},
		{"purpose", func(a *domain.Asset) { a.Purpose = domain.UserCover }, nil, "avatar_asset_id"},
		{"pending", func(a *domain.Asset) { a.Uploaded = false }, domain.ErrAssetNotUploaded, ""},
		{"no final", func(a *domain.Asset) { a.Metadata.Final = nil }, domain.ErrAssetNotUploaded, ""},
		{"reclaiming", func(a *domain.Asset) { a.Metadata.Final.Phase = domain.ObjectReclaiming }, domain.ErrAssetNotUploaded, ""},
		{"bound", func(a *domain.Asset) { a.BoundAt = &a.CreatedAt }, domain.ErrAssetBound, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u, f, req, _ := imageFixture(t)
			a := f.rows[*req.Changes[0].NextID]
			tc.change(&a)
			f.rows[a.ID] = a
			_, err := u.LockImageBindings(t.Context(), req)
			if tc.field != "" {
				assertImageField(t, err, tc.field)
			} else if !errors.Is(err, tc.want) {
				t.Fatalf("error=%v want=%v", err, tc.want)
			}
			if len(f.writes) != 0 {
				t.Fatalf("refusal wrote %v", f.writes)
			}
		})
	}
}
func TestImageBindingsHiddenPrecedence(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		t.Run(map[bool]string{false: "forward", true: "reverse"}[reverse], func(t *testing.T) {
			u, f, req, _ := imageFixture(t)
			a := f.rows[*req.Changes[0].NextID]
			a.Uploaded = false
			f.rows[a.ID] = a
			delete(f.rows, *req.Changes[1].NextID)
			if reverse {
				slices.Reverse(req.Changes)
			}
			_, err := u.LockImageBindings(t.Context(), req)
			if !errors.Is(err, domain.ErrAssetNotFound) {
				t.Fatalf("hidden leaked behind conflict: %v", err)
			}
		})
	}
}
func TestImageBindingsExactSevenDaysAndCurrent(t *testing.T) {
	t.Run("exact cutoff and expired upload grant", func(t *testing.T) {
		u, f, req, at := imageFixture(t)
		for _, c := range req.Changes {
			a := f.rows[*c.NextID]
			a.CreatedAt = at.Add(-7 * 24 * time.Hour)
			past := at.Add(-time.Hour)
			a.UploadExpiresAt = &past
			f.rows[a.ID] = a
		}
		got, err := u.LockImageBindings(t.Context(), req)
		if err != nil {
			t.Fatal(err)
		}
		if err := u.ApplyImageBindings(t.Context(), req, got); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("current IDs do not rewrite old audit", func(t *testing.T) {
		u, f, req, _ := imageFixture(t)
		for i := range req.Changes {
			req.Changes[i].NextID = req.Changes[i].CurrentID
		}
		before := map[uuid.UUID]domain.Asset{}
		for id, a := range f.rows {
			before[id] = a
		}
		at, err := u.LockImageBindings(t.Context(), req)
		if err != nil {
			t.Fatal(err)
		}
		if err := u.ApplyImageBindings(t.Context(), req, at); err != nil {
			t.Fatal(err)
		}
		if len(f.writes) != 0 || !reflect.DeepEqual(before, f.rows) {
			t.Fatal("idempotent binding changed asset")
		}
	})
}
func TestImageBindingsAdminReplacement(t *testing.T) {
	for _, deleted := range []bool{false, true} {
		t.Run(map[bool]string{false: "live", true: "already deleted"}[deleted], func(t *testing.T) {
			u, f, req, want := imageFixture(t)
			req.Changes = req.Changes[:1]
			req.Changes[0].Field = "logo_asset_id"
			req.Changes[0].Purpose = domain.WorkspaceLogo
			req.Host.WorkspaceID = accessID(30)
			for _, id := range []*uuid.UUID{req.Changes[0].CurrentID, req.Changes[0].NextID} {
				a := f.rows[*id]
				a.Purpose = domain.WorkspaceLogo
				a.Placement.Scope = domain.Scope{Kind: domain.ScopeWorkspace, ID: req.Host.WorkspaceID}
				a.Placement.WorkspaceID = req.Host.WorkspaceID
				if *id == *req.Changes[0].CurrentID {
					a.Placement.UserID = accessID(2)
					if deleted {
						past := want.Add(-time.Hour)
						a.DeletedAt = &past
						a.DeletedByID = imgPtr(accessID(2))
					}
				}
				f.rows[*id] = a
			}
			old := f.rows[*req.Changes[0].CurrentID]
			at, err := u.LockImageBindings(t.Context(), req)
			if err != nil {
				t.Fatal(err)
			}
			if err := u.ApplyImageBindings(t.Context(), req, at); err != nil {
				t.Fatal(err)
			}
			got := f.rows[old.ID]
			if got.Placement.UserID != accessID(2) {
				t.Fatal("replacement changed old applicant")
			}
			if deleted && !reflect.DeepEqual(old, got) {
				t.Fatal("older deletion audit was overwritten")
			}
		})
	}
}
