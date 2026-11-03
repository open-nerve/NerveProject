package postgresadapter

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

func TestDraftAssetTransferRequiresTransaction(t *testing.T) {
	w := newAssetWorld(t)
	u, q := w.draftMover(), w.draftRequest()
	if _, err := u.LockDraftAssets(t.Context(), q); err == nil {
		t.Fatal("empty draft lock without transaction succeeded")
	}
	if err := u.MoveDraftAssets(t.Context(), q, uuid.Nil(), w.now); err == nil {
		t.Fatal("empty draft move without transaction succeeded")
	}
	if _, err := w.store.DraftAssetIDs(t.Context(), w.draft, uuid.Nil(), 1000); err == nil {
		t.Fatal("directory read outside protected transaction succeeded")
	}
	if _, err := w.store.MoveDraftAsset(t.Context(), app.DraftAssetWrite{AssetID: uuid.NewV7(), By: w.user, At: w.now}); err == nil {
		t.Fatal("unprotected writer succeeded")
	}
}

func TestDraftAssetTransferAtomic(t *testing.T) {
	for _, mode := range []string{"clear project", "save", "publish"} {
		for _, fail := range []bool{false, true} {
			t.Run(mode+map[bool]string{false: "/commit", true: "/second UPDATE rolls back"}[fail], func(t *testing.T) {
				w := newAssetWorld(t)
				u, q := w.draftMover(), w.draftRequest()
				issueID := uuid.Nil()
				switch mode {
				case "publish":
					q.Publish, issueID = true, uuid.NewV7()
				case "clear project":
					q.NewProjectID = nil
				}
				var before []domain.Asset
				for i := range 5 {
					a := w.preparedImage(t, domain.ScopeDraft, domain.DraftIssueDescription)
					if i == 0 && mode != "save" {
						a.Purpose = domain.DraftIssueAttachment
					}
					if i == 2 {
						past := w.now.Add(-30 * 24 * time.Hour)
						a.BoundAt, a.CreatedAt = &past, past
					}
					if i == 4 {
						a.DeletedAt, a.DeletedByID = &w.now, &w.user
					}
					a = w.putImage(t, a)
					before = append(before, a)
					if i == 1 || (i == 0 && mode == "save") {
						q.ImageIDs = append(q.ImageIDs, a.ID)
					}
				}
				if fail {
					rejectContentUpdate(t, w, before[1].ID)
				}
				err := contentTx(t, w, true, func(ctx context.Context) error {
					at, err := u.LockDraftAssets(ctx, q)
					if err != nil {
						return err
					}
					tx, _ := postgres.Tx(ctx)
					if q.Publish {
						if _, err := tx.Exec(ctx, "INSERT INTO issues(id,workspace_id,project_id,state_id,name,sequence_id,created_at,updated_at) VALUES($1,$2,$3,$3,'Published',2,$4,$4)", issueID, w.workspace, w.project, at); err != nil {
							return err
						}
					}
					if err := u.MoveDraftAssets(ctx, q, issueID, at); err != nil {
						return err
					}
					if q.Publish {
						_, err = tx.Exec(ctx, "UPDATE draft_issues SET deleted_at=$1 WHERE id=$2", at, w.draft)
					} else {
						_, err = tx.Exec(ctx, "UPDATE draft_issues SET project_id=$1 WHERE id=$2", q.NewProjectID, w.draft)
					}
					return err
				})
				requireContentResult(t, err, fail)
				at := w.now.Add(time.Second)
				for _, old := range before {
					got, err := w.store.FindAsset(t.Context(), old.ID)
					if err != nil {
						t.Fatal(err)
					}
					want := old
					if !fail && old.DeletedAt == nil {
						bind := old.BoundAt == nil && (slices.Contains(q.ImageIDs, old.ID) || (q.Publish && old.Purpose == domain.DraftIssueAttachment))
						if mode != "save" || bind {
							want.UpdatedAt, want.UpdatedByID, want.StorageRevision = at, &w.user, old.StorageRevision+1
						}
						if bind {
							want.BoundAt = &at
						}
						if mode == "clear project" {
							want.Placement.ProjectID = uuid.Nil()
						}
						if q.Publish {
							want.Placement.Scope = domain.Scope{Kind: domain.ScopeIssue, ID: issueID}
							want.Placement.IssueID, want.Placement.DraftID = issueID, uuid.Nil()
							want.Purpose = domain.IssueDescription
							if old.Purpose == domain.DraftIssueAttachment {
								want.Purpose = domain.IssueAttachment
							}
						}
					}
					if !reflect.DeepEqual(got, want) {
						t.Fatalf("draft asset=%+v want=%+v", got, want)
					}
				}
				var deleted *time.Time
				var project *uuid.UUID
				if err := w.pool.QueryRow(t.Context(), "SELECT project_id,deleted_at FROM draft_issues WHERE id=$1", w.draft).Scan(&project, &deleted); err != nil {
					t.Fatal(err)
				}
				if (deleted != nil) != (q.Publish && !fail) || (project == nil) != (mode == "clear project" && !fail) {
					t.Fatal("draft owner and asset updates disagreed")
				}
				if q.Publish {
					var exists bool
					if err := w.pool.QueryRow(t.Context(), "SELECT EXISTS(SELECT 1 FROM issues WHERE id=$1)", issueID).Scan(&exists); err != nil || exists == fail {
						t.Fatalf("issue insertion not atomic: %v %v", exists, err)
					}
				}
			})
		}
	}
}

func TestDraftAssetTransferPendingAndImmutableSave(t *testing.T) {
	w := newAssetWorld(t)
	a := w.putImage(t, w.asset(domain.ScopeDraft))
	u, q := w.draftMover(), w.draftRequest()
	q.Publish = true
	if err := contentTx(t, w, true, func(ctx context.Context) error {
		_, err := u.LockDraftAssets(ctx, q)
		return err
	}); !errors.Is(err, domain.ErrAssetNotUploaded) {
		t.Fatalf("pending publish=%v", err)
	}
	q.Publish, q.NewProjectID = false, nil
	if err := contentTx(t, w, true, func(ctx context.Context) error {
		at, err := u.LockDraftAssets(ctx, q)
		if err != nil {
			return err
		}
		return u.MoveDraftAssets(ctx, q, uuid.Nil(), at)
	}); err != nil {
		t.Fatal(err)
	}
	got, err := w.store.FindAsset(t.Context(), a.ID)
	if err != nil || got.Uploaded || got.BoundAt != nil || got.Placement.ProjectID != uuid.Nil() || got.Placement.DraftID != w.draft {
		t.Fatalf("pending transfer=%+v %v", got, err)
	}
	unchanged := got
	got.Placement.Scope, got.Placement.DraftID, got.Placement.IssueID = domain.Scope{Kind: domain.ScopeIssue, ID: w.issue}, uuid.Nil(), w.issue
	got.Placement.ProjectID, got.Purpose, got.BoundAt = w.project, domain.IssueAttachment, &w.now
	saved, err := w.store.SaveAsset(t.Context(), got)
	unchanged.StorageRevision++
	if err != nil || !reflect.DeepEqual(saved, unchanged) {
		t.Fatalf("SaveAsset widened ownership/binding: %+v %v", saved, err)
	}
}
