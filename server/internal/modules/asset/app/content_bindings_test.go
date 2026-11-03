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

type contentRows struct {
	*imageRows
	clockLocks []int
}

func (f *contentRows) BindContentAsset(_ context.Context, in ContentBindingWrite) (domain.Asset, error) {
	if f.writeErr != nil {
		return domain.Asset{}, f.writeErr
	}
	a := f.rows[in.AssetID]
	if a.StorageRevision != in.Revision {
		return domain.Asset{}, domain.ErrAssetScopeChanged
	}
	a.BoundAt, a.UpdatedAt, a.UpdatedByID = &in.At, in.At, &in.By
	a.StorageRevision++
	if a.Placement.Scope.Kind == domain.ScopeNewComment {
		a.Placement.Scope = domain.Scope{Kind: domain.ScopeComment, ID: in.CommentID}
		a.Placement.CommentID = in.CommentID
	}
	f.rows[a.ID] = a
	f.writes = append(f.writes, a.ID)
	return a, nil
}

func contentFixture(t *testing.T, kind domain.ScopeKind) (*ContentBinder, *contentRows, ContentBindRequest, time.Time) {
	t.Helper()
	_, old, _, at := imageFixture(t)
	f := &contentRows{imageRows: &imageRows{rows: map[uuid.UUID]domain.Asset{}}}
	q := ContentBindRequest{ActorID: accessID(1), Host: ContentHost{WorkspaceID: accessID(2), ProjectID: accessID(3), IssueID: accessID(4)},
		ImageIDs: []uuid.UUID{accessID(8), accessID(8)}, AttachmentIDs: []uuid.UUID{accessID(6)}}
	if kind != domain.ScopeIssue {
		q.Host.CommentID = accessID(5)
		q.NewComment = kind == domain.ScopeNewComment
	}
	for i, id := range []uuid.UUID{accessID(8), accessID(6)} {
		a := old.rows[id]
		a.BoundAt = nil
		a.CreatedAt = at.Add(-time.Hour)
		a.Placement = domain.Placement{UserID: q.ActorID, WorkspaceID: q.Host.WorkspaceID, ProjectID: q.Host.ProjectID, IssueID: q.Host.IssueID,
			Scope: domain.Scope{Kind: kind, ID: q.Host.IssueID}}
		a.Purpose = []domain.Purpose{domain.IssueDescription, domain.IssueAttachment}[i]
		if kind != domain.ScopeIssue {
			a.Purpose = []domain.Purpose{domain.CommentDescription, domain.CommentAttachment}[i]
			if kind == domain.ScopeComment {
				a.Placement.CommentID, a.Placement.Scope.ID = q.Host.CommentID, q.Host.CommentID
			}
		}
		f.rows[id] = a
	}
	u := NewContentBindings(ContentBindingDeps{Locks: f, Writer: f, Clock: imageClockFunc(func() time.Time {
		f.clockLocks = append(f.clockLocks, len(f.locks))
		return at
	})})
	return u, f, q, at
}

func TestContentBindings(t *testing.T) {
	for _, kind := range []domain.ScopeKind{domain.ScopeIssue, domain.ScopeComment, domain.ScopeNewComment} {
		t.Run(string(kind), func(t *testing.T) {
			u, f, q, want := contentFixture(t, kind)
			old := f.rows[accessID(8)]
			at, err := u.LockContent(t.Context(), q)
			if err != nil || !at.Equal(want) || len(f.writes) != 0 || !reflect.DeepEqual(f.clockLocks, []int{1}) {
				t.Fatalf("lock=%v %v writes=%v clock at lock counts=%v", at, err, f.writes, f.clockLocks)
			}
			if err := u.BindContent(t.Context(), q, at); err != nil {
				t.Fatal(err)
			}
			ids := []uuid.UUID{accessID(6), accessID(8)}
			if !reflect.DeepEqual(f.locks, [][]uuid.UUID{ids, ids}) || !slices.Equal(f.writes, ids) || len(f.clockLocks) != 1 {
				t.Fatalf("complete sorted locks=%v writes=%v clocks=%v", f.locks, f.writes, f.clockLocks)
			}
			for _, id := range ids {
				a := f.rows[id]
				if a.BoundAt == nil || !a.BoundAt.Equal(at) || !a.UpdatedAt.Equal(at) || a.UpdatedByID == nil || *a.UpdatedByID != q.ActorID || a.StorageRevision != 1 {
					t.Fatalf("missing binding audit: %+v", a)
				}
				if a.Placement.UserID != q.ActorID || a.Placement.IssueID != q.Host.IssueID || a.Placement.ProjectID != q.Host.ProjectID {
					t.Fatal("binding changed applicant or parent")
				}
				if kind == domain.ScopeNewComment && (a.Placement.Scope != (domain.Scope{Kind: domain.ScopeComment, ID: q.Host.CommentID}) || a.Placement.CommentID != q.Host.CommentID) {
					t.Fatal("preparation was not moved to the inserted comment")
				}
			}
			got := f.rows[old.ID]
			if got.CreatedAt != old.CreatedAt || !reflect.DeepEqual(got.CreatedByID, old.CreatedByID) || !reflect.DeepEqual(got.Metadata, old.Metadata) {
				t.Fatal("binding changed original creation or object recovery metadata")
			}
		})
	}
}

func TestContentBindingsRejectWholeSet(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*domain.Asset)
		want error
	}{
		{"applicant", func(a *domain.Asset) { a.Placement.UserID = accessID(20) }, domain.ErrAssetNotFound},
		{"workspace", func(a *domain.Asset) { a.Placement.WorkspaceID = accessID(20) }, domain.ErrAssetNotFound},
		{"project", func(a *domain.Asset) { a.Placement.ProjectID = accessID(20) }, domain.ErrAssetNotFound},
		{"issue", func(a *domain.Asset) { a.Placement.IssueID = accessID(20) }, domain.ErrAssetNotFound},
		{"extra parent", func(a *domain.Asset) { a.Placement.DraftID = accessID(20) }, domain.ErrAssetNotFound},
		{"scope", func(a *domain.Asset) { a.Placement.Scope.ID = accessID(20) }, domain.ErrAssetNotFound},
		{"purpose", func(a *domain.Asset) { a.Purpose = domain.UserAvatar }, domain.ErrAssetNotFound},
		{"deleted", func(a *domain.Asset) { a.DeletedAt = &a.CreatedAt }, domain.ErrAssetNotFound},
		{"pending", func(a *domain.Asset) { a.Uploaded = false }, domain.ErrAssetNotUploaded},
		{"missing final", func(a *domain.Asset) { a.Metadata.Final = nil }, domain.ErrAssetNotUploaded},
		{"reclaiming", func(a *domain.Asset) { a.Metadata.Final.Phase = domain.ObjectReclaiming }, domain.ErrAssetNotUploaded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u, f, q, at := contentFixture(t, domain.ScopeIssue)
			a := f.rows[accessID(8)]
			tc.edit(&a)
			f.rows[a.ID] = a
			if _, err := u.LockContent(t.Context(), q); !errors.Is(err, tc.want) {
				t.Fatalf("LockContent=%v want %v", err, tc.want)
			}
			if err := u.BindContent(t.Context(), q, at); !errors.Is(err, tc.want) || len(f.writes) != 0 {
				t.Fatalf("BindContent=%v want %v, partial writes=%v", err, tc.want, f.writes)
			}
		})
	}
}

func TestContentBindingsBoundAndHidden(t *testing.T) {
	t.Run("another author's bound same-host image stays unchanged", func(t *testing.T) {
		u, f, q, at := contentFixture(t, domain.ScopeComment)
		a := f.rows[accessID(8)]
		a.Placement.UserID = accessID(30)
		a.BoundAt = &a.CreatedAt
		f.rows[a.ID] = a
		if _, err := u.LockContent(t.Context(), q); err != nil {
			t.Fatal(err)
		}
		if err := u.BindContent(t.Context(), q, at); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(f.rows[a.ID], a) || !slices.Equal(f.writes, []uuid.UUID{accessID(6)}) {
			t.Fatal("retaining a bound image rewrote its original audit")
		}
	})
	for _, reverse := range []bool{false, true} {
		t.Run(map[bool]string{false: "pending then hidden", true: "hidden then pending"}[reverse], func(t *testing.T) {
			u, f, q, _ := contentFixture(t, domain.ScopeIssue)
			a := f.rows[accessID(8)]
			a.Uploaded = false
			f.rows[a.ID] = a
			delete(f.rows, accessID(6))
			if reverse {
				q.ImageIDs = []uuid.UUID{accessID(6), accessID(8)}
				q.AttachmentIDs = nil
			}
			if _, err := u.LockContent(t.Context(), q); !errors.Is(err, domain.ErrAssetNotFound) || len(f.writes) != 0 {
				t.Fatalf("hidden resource leaked behind state error: %v", err)
			}
		})
	}
}
