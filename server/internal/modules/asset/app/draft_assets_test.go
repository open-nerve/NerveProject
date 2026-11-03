package app

import (
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func TestDraftAssetTransfer(t *testing.T) {
	for _, mode := range []string{"change project", "clear project", "save", "publish"} {
		t.Run(mode, func(t *testing.T) {
			u, f, q, want := draftAssetFixture(t)
			issue := uuid.Nil()
			switch mode {
			case "clear project":
				q.NewProjectID = nil
			case "save":
				q.NewProjectID = q.OldProjectID
			case "publish":
				q.Publish, q.NewProjectID, issue = true, q.OldProjectID, accessID(42)
			}
			old := f.rows[accessID(9)]
			unreferenced := f.rows[accessID(10)]
			at, err := u.LockDraftAssets(t.Context(), q)
			if err != nil || !at.Equal(want) || len(f.writes) != 0 || !reflect.DeepEqual(f.clockLocks, []int{1}) {
				t.Fatalf("lock=%v %v writes=%v clock=%v", at, err, f.writes, f.clockLocks)
			}
			if err := u.MoveDraftAssets(t.Context(), q, issue, at); err != nil {
				t.Fatal(err)
			}
			ids := []uuid.UUID{accessID(6), accessID(8), accessID(9), accessID(10)}
			if !reflect.DeepEqual(f.locks, [][]uuid.UUID{ids, ids}) || len(f.clockLocks) != 1 {
				t.Fatalf("whole scope lock=%v clock=%v", f.locks, f.clockLocks)
			}
			wantWrites := ids
			if mode == "save" {
				wantWrites = []uuid.UUID{accessID(8)}
			}
			if !slices.Equal(f.writes, wantWrites) {
				t.Fatalf("writes=%v want %v", f.writes, wantWrites)
			}
			for _, id := range ids {
				a := f.rows[id]
				project := uuid.Nil()
				if q.NewProjectID != nil {
					project = *q.NewProjectID
				}
				if a.Placement.ProjectID != project || a.Placement.UserID != q.ActorID || a.Placement.WorkspaceID != q.WorkspaceID {
					t.Fatal("scope parent or original applicant lost")
				}
				if q.Publish && (a.Placement.IssueID != issue || a.Placement.DraftID != uuid.Nil() || a.Placement.Scope != (domain.Scope{Kind: domain.ScopeIssue, ID: issue})) {
					t.Fatalf("not moved to issued host: %+v", a.Placement)
				}
				if !q.Publish && (a.Placement.DraftID != q.DraftID || a.Placement.IssueID != uuid.Nil()) {
					t.Fatal("save changed draft host")
				}
				if slices.Contains(wantWrites, id) && (a.UpdatedAt != at || a.UpdatedByID == nil || *a.UpdatedByID != q.ActorID || a.StorageRevision != 1) {
					t.Fatalf("write stamp/revision=%+v", a)
				}
			}
			if got := f.rows[old.ID]; got.BoundAt == nil || !got.BoundAt.Equal(*old.BoundAt) || got.CreatedAt != old.CreatedAt || !reflect.DeepEqual(got.Metadata, old.Metadata) {
				t.Fatal("old bound image lost original binding/creation or object")
			}
			if got := f.rows[unreferenced.ID]; got.BoundAt != nil || got.CreatedAt != unreferenced.CreatedAt || !reflect.DeepEqual(got.Metadata, unreferenced.Metadata) {
				t.Fatal("unreferenced preparation was published or its age reset")
			}
			image, attachment := f.rows[accessID(8)], f.rows[accessID(6)]
			if image.BoundAt == nil || !image.BoundAt.Equal(at) || (attachment.BoundAt != nil) != q.Publish {
				t.Fatal("current image / published ordinary attachment binding is wrong")
			}
			if q.Publish && (image.Purpose != domain.IssueDescription || attachment.Purpose != domain.IssueAttachment) {
				t.Fatal("draft purpose was not migrated")
			}
		})
	}
}

func TestDraftAssetTransferRefusesCompleteSet(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*domain.Asset)
		want error
	}{
		{"applicant", func(a *domain.Asset) { a.Placement.UserID = accessID(90) }, domain.ErrAssetNotFound},
		{"workspace", func(a *domain.Asset) { a.Placement.WorkspaceID = accessID(90) }, domain.ErrAssetNotFound},
		{"project", func(a *domain.Asset) { a.Placement.ProjectID = accessID(90) }, domain.ErrAssetNotFound},
		{"scope", func(a *domain.Asset) { a.Placement.Scope.Kind = domain.ScopeIssue }, domain.ErrAssetNotFound},
		{"extra parent", func(a *domain.Asset) { a.Placement.CommentID = accessID(90) }, domain.ErrAssetNotFound},
		{"purpose", func(a *domain.Asset) { a.Purpose = domain.IssueDescription }, domain.ErrAssetNotFound},
		{"referenced pending", func(a *domain.Asset) { a.Uploaded = false }, domain.ErrAssetNotUploaded},
		{"reclaiming final", func(a *domain.Asset) { a.Metadata.Final.Phase = domain.ObjectReclaiming }, domain.ErrAssetNotUploaded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u, f, q, at := draftAssetFixture(t)
			a := f.rows[accessID(8)]
			tc.edit(&a)
			f.rows[a.ID] = a
			if _, err := u.LockDraftAssets(t.Context(), q); !errors.Is(err, tc.want) {
				t.Fatalf("lock error=%v want %v", err, tc.want)
			}
			if err := u.MoveDraftAssets(t.Context(), q, uuid.Nil(), at); !errors.Is(err, tc.want) || len(f.writes) != 0 {
				t.Fatalf("move error=%v want %v, partial writes=%v", err, tc.want, f.writes)
			}
		})
	}
}

func TestDraftAssetTransferPublicationStates(t *testing.T) {
	for _, publish := range []bool{false, true} {
		t.Run(map[bool]string{false: "pending moves with project", true: "pending blocks publication"}[publish], func(t *testing.T) {
			u, f, q, at := draftAssetFixture(t)
			a := f.rows[accessID(6)]
			a.Uploaded = false
			a.Metadata.Final = nil
			f.rows[a.ID] = a
			if publish {
				q.Publish, q.NewProjectID = true, q.OldProjectID
			}
			_, err := u.LockDraftAssets(t.Context(), q)
			if publish {
				if !errors.Is(err, domain.ErrAssetNotUploaded) || len(f.writes) != 0 {
					t.Fatalf("pending publish=%v writes=%v", err, f.writes)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := u.MoveDraftAssets(t.Context(), q, uuid.Nil(), at); err != nil {
				t.Fatal(err)
			}
			got := f.rows[a.ID]
			if got.Uploaded || got.BoundAt != nil || got.Placement.ProjectID != *q.NewProjectID {
				t.Fatal("pending asset was lost, published or bound")
			}
		})
	}
	t.Run("published final with losing candidate still publishes", func(t *testing.T) {
		u, f, q, at := draftAssetFixture(t)
		q.Publish, q.NewProjectID = true, q.OldProjectID
		a := f.rows[accessID(6)]
		candidate := *a.Metadata.Final
		candidate.Key, _ = domain.NewObjectKey('o', a.ID, accessID(91))
		candidate.Phase = domain.ObjectReclaiming
		a.Metadata.Candidates = []domain.Candidate{{ObjectRecord: candidate, LeaseUntil: at.Add(time.Hour)}}
		f.rows[a.ID] = a
		if _, err := u.LockDraftAssets(t.Context(), q); err != nil {
			t.Fatal(err)
		}
		if err := u.MoveDraftAssets(t.Context(), q, accessID(42), at); err != nil {
			t.Fatal(err)
		}
		if got := f.rows[a.ID]; got.BoundAt == nil || !reflect.DeepEqual(got.Metadata, a.Metadata) {
			t.Fatal("successful final confused with a losing external attempt")
		}
	})
}
