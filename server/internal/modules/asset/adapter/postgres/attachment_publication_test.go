package postgresadapter

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func TestAttachmentPublicationWriterGuards(t *testing.T) {
	for _, name := range []string{"issue", "comment", "draft", "pending", "deleted", "bound", "wrong revision", "description", "new comment", "missing final", "reclaiming", "without transaction"} {
		t.Run(name, func(t *testing.T) {
			w := newAssetWorld(t)
			a := w.preparedImage(t, domain.ScopeIssue, domain.IssueAttachment)
			switch name {
			case "comment":
				a.Purpose = domain.CommentAttachment
				a.Placement.Scope = domain.Scope{Kind: domain.ScopeComment, ID: w.comment}
				a.Placement.CommentID = w.comment
			case "draft":
				a = w.preparedImage(t, domain.ScopeDraft, domain.DraftIssueAttachment)
			case "pending":
				a.Uploaded = false
			case "deleted":
				a.DeletedAt, a.DeletedByID = &w.now, &w.user
			case "bound":
				a.BoundAt = &w.now
			case "description":
				a.Purpose = domain.IssueDescription
			case "new comment":
				a.Purpose = domain.CommentAttachment
				a.Placement.Scope.Kind = domain.ScopeNewComment
			case "missing final":
				a.Metadata.Final = nil
			case "reclaiming":
				a.Metadata.Final.Phase = domain.ObjectReclaiming
			}
			a = w.putImage(t, a)
			in := app.AttachmentBindingWrite{AssetID: a.ID, By: w.user, At: w.now.Add(time.Second), Revision: a.StorageRevision}
			if name == "wrong revision" {
				in.Revision++
			}
			var got domain.Asset
			var err error
			write := func(ctx context.Context) error { got, err = w.store.BindAttachment(ctx, in); return err }
			if name == "without transaction" {
				err = write(t.Context())
			} else {
				err = contentTx(t, w, name == "draft", write)
			}
			valid := name == "issue" || name == "comment" || name == "draft"
			if valid {
				if err != nil || got.BoundAt == nil || !got.BoundAt.Equal(in.At) || !got.UpdatedAt.Equal(in.At) || got.UpdatedByID == nil || *got.UpdatedByID != in.By || got.StorageRevision != a.StorageRevision+1 || got.Placement != a.Placement || got.Purpose != a.Purpose || !reflect.DeepEqual(got.Metadata, a.Metadata) {
					t.Fatalf("ordinary binding lost stamp/immutability: %+v %v", got, err)
				}
			} else {
				if err == nil || (name != "without transaction" && !errors.Is(err, domain.ErrAssetScopeChanged)) {
					t.Fatalf("invalid publication accepted: %v", err)
				}
				after, err := w.store.FindAsset(t.Context(), a.ID)
				if err != nil || !reflect.DeepEqual(after, a) {
					t.Fatal("invalid publication changed row", err)
				}
			}
		})
	}
}
