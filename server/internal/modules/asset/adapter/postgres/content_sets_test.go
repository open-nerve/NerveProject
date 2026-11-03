package postgresadapter

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func TestContentBindingsFullDatabaseSet(t *testing.T) {
	for _, draft := range []bool{false, true} {
		t.Run(map[bool]string{false: "content", true: "draft"}[draft], func(t *testing.T) {
			w := newAssetWorld(t)
			var ids []uuid.UUID
			if err := contentTx(t, w, draft, func(ctx context.Context) error {
				for range 1001 {
					kind, purpose := domain.ScopeIssue, domain.IssueDescription
					if draft {
						kind, purpose = domain.ScopeDraft, domain.DraftIssueDescription
					}
					a, err := w.store.CreateAsset(ctx, w.preparedImage(t, kind, purpose))
					if err != nil {
						return err
					}
					ids = append(ids, a.ID)
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			slices.Reverse(ids)
			if err := contentTx(t, w, draft, func(ctx context.Context) error {
				if draft {
					u, q := w.draftMover(), w.draftRequest()
					q.NewProjectID = nil
					at, err := u.LockDraftAssets(ctx, q)
					if err != nil {
						return err
					}
					return u.MoveDraftAssets(ctx, q, uuid.Nil(), at)
				}
				u, q := w.contentBinder(), w.contentRequest()
				q.ImageIDs = append(slices.Clone(ids), ids[0])
				at, err := u.LockContent(ctx, q)
				if err != nil {
					return err
				}
				return u.BindContent(ctx, q, at)
			}); err != nil {
				t.Fatal(err)
			}
			var count int
			if err := w.pool.QueryRow(t.Context(), "SELECT count(*) FROM file_assets WHERE storage_revision=1 AND ((NOT $1 AND bound_at IS NOT NULL) OR ($1 AND project_id IS NULL AND bound_at IS NULL))", draft).Scan(&count); err != nil || count != 1001 {
				t.Fatalf("full set count=%d want=1001 error=%v", count, err)
			}
		})
	}
}

func TestContentBindingsWriterGuards(t *testing.T) {
	for _, name := range []string{"pending", "deleted", "bound", "wrong revision", "wrong purpose", "missing final", "reclaiming", "missing target"} {
		t.Run(name, func(t *testing.T) {
			w := newAssetWorld(t)
			a := w.preparedImage(t, domain.ScopeIssue, domain.IssueDescription)
			switch name {
			case "pending":
				a.Uploaded = false
			case "deleted":
				a.DeletedAt, a.DeletedByID = &w.now, &w.user
			case "bound":
				a.BoundAt = &w.now
			case "wrong purpose":
				a = w.preparedImage(t, domain.ScopeProject, domain.ProjectCover)
			case "missing final":
				a.Metadata.Final = nil
			case "reclaiming":
				a.Metadata.Final.Phase = domain.ObjectReclaiming
			case "missing target":
				a.Purpose, a.Placement.Scope.Kind = domain.CommentDescription, domain.ScopeNewComment
			}
			a = w.putImage(t, a)
			revision := a.StorageRevision
			if name == "wrong revision" {
				revision++
			}
			err := contentTx(t, w, false, func(ctx context.Context) error {
				_, err := w.store.BindContentAsset(ctx, app.ContentBindingWrite{AssetID: a.ID, Revision: revision, At: w.now, By: w.user})
				return err
			})
			if !errors.Is(err, domain.ErrAssetScopeChanged) {
				t.Fatalf("invalid binding accepted: %v", err)
			}
			got, err := w.store.FindAsset(t.Context(), a.ID)
			if err != nil || !reflect.DeepEqual(got, a) {
				t.Fatalf("refusal changed row: %+v %v", got, err)
			}
		})
	}
}

func TestDraftAssetTransferWriterGuards(t *testing.T) {
	for _, name := range []string{"pending bind", "pending publish", "missing final", "reclaiming", "deleted", "wrong revision", "wrong scope", "publish without project"} {
		t.Run(name, func(t *testing.T) {
			w := newAssetWorld(t)
			a := w.preparedImage(t, domain.ScopeDraft, domain.DraftIssueDescription)
			q := app.DraftAssetWrite{By: w.user, At: w.now, ProjectID: &w.project, Bind: true}
			switch name {
			case "pending bind":
				a.Uploaded = false
			case "pending publish":
				a.Uploaded, q.Bind, q.IssueID = false, false, w.issue
			case "missing final":
				a.Metadata.Final = nil
			case "reclaiming":
				a.Metadata.Final.Phase = domain.ObjectReclaiming
			case "deleted":
				a.DeletedAt, a.DeletedByID = &w.now, &w.user
			case "wrong scope":
				a = w.preparedImage(t, domain.ScopeIssue, domain.IssueDescription)
			case "publish without project":
				q.IssueID, q.ProjectID = w.issue, nil
			}
			a = w.putImage(t, a)
			q.AssetID, q.Revision = a.ID, a.StorageRevision
			if name == "wrong revision" {
				q.Revision++
			}
			err := contentTx(t, w, true, func(ctx context.Context) error {
				_, err := w.store.MoveDraftAsset(ctx, q)
				return err
			})
			if !errors.Is(err, domain.ErrAssetScopeChanged) {
				t.Fatalf("invalid draft move accepted: %v", err)
			}
			got, err := w.store.FindAsset(t.Context(), a.ID)
			if err != nil || !reflect.DeepEqual(got, a) {
				t.Fatalf("refusal changed row: %+v %v", got, err)
			}
		})
	}
}
