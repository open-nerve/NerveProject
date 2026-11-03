package postgresadapter

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func (w assetWorld) contentBinder() *app.ContentBinder {
	return app.NewContentBindings(app.ContentBindingDeps{Locks: w.store, Writer: w.store, Clock: cleanupClock(w.now.Add(time.Second))})
}
func (w assetWorld) draftMover() *app.DraftAssetMover {
	return app.NewDraftAssetTransfer(app.DraftAssetDeps{Events: discardDraftEvents{}, Locks: w.store, Reader: w.store, Writer: w.store, Clock: cleanupClock(w.now.Add(time.Second))})
}
func (w assetWorld) contentRequest() app.ContentBindRequest {
	return app.ContentBindRequest{ActorID: w.user, Host: app.ContentHost{WorkspaceID: w.workspace, ProjectID: w.project, IssueID: w.issue}}
}
func (w assetWorld) draftRequest() app.DraftAssetRequest {
	return app.DraftAssetRequest{ActorID: w.user, WorkspaceID: w.workspace, DraftID: w.draft, OldProjectID: &w.project, NewProjectID: &w.project}
}
func contentTx(t *testing.T, w assetWorld, draft bool, fn func(context.Context) error) error {
	t.Helper()
	return postgres.NewTxManager(w.pool, 5*time.Second).WithinTx(t.Context(), func(ctx context.Context) error {
		tx, _ := postgres.Tx(ctx)
		if draft {
			if _, err := tx.Exec(ctx, "SELECT id FROM workspaces WHERE id=$1 FOR SHARE", w.workspace); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(ctx, "SELECT id FROM projects WHERE id=$1 FOR NO KEY UPDATE", w.project); err != nil {
			return err
		}
		if draft {
			if _, err := tx.Exec(ctx, "SELECT id FROM draft_issues WHERE id=$1 FOR UPDATE", w.draft); err != nil {
				return err
			}
		}
		return fn(ctx)
	})
}
func rejectContentUpdate(t *testing.T, w assetWorld, id uuid.UUID) {
	t.Helper()
	_, err := w.pool.Exec(t.Context(), `CREATE FUNCTION reject_content_update() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 IF NEW.id='`+id.String()+`'::uuid THEN RAISE EXCEPTION 'content binding fault'; END IF; RETURN NEW; END $$;
 CREATE TRIGGER content_binding_fault BEFORE UPDATE ON file_assets FOR EACH ROW EXECUTE FUNCTION reject_content_update()`)
	if err != nil {
		t.Fatal(err)
	}
}
func requireContentResult(t *testing.T, err error, fail bool) {
	t.Helper()
	if fail {
		var pe *pgconn.PgError
		if !errors.As(err, &pe) || pe.Code != "P0001" {
			t.Fatalf("want actual second UPDATE fault, got %v", err)
		}
	} else if err != nil {
		t.Fatal(err)
	}
}

func TestContentBindingsRequireTransaction(t *testing.T) {
	w := newAssetWorld(t)
	a := w.putImage(t, w.preparedImage(t, domain.ScopeIssue, domain.IssueDescription))
	u, q := w.contentBinder(), w.contentRequest()
	for _, ids := range [][]uuid.UUID{nil, {a.ID}} {
		q.ImageIDs = ids
		if _, err := u.LockContent(t.Context(), q); err == nil {
			t.Fatal("unprotected lock succeeded")
		}
		if err := u.BindContent(t.Context(), q, w.now); err == nil {
			t.Fatal("unprotected binding succeeded")
		}
	}
	if _, err := w.store.BindContentAsset(t.Context(), app.ContentBindingWrite{AssetID: a.ID, By: w.user, At: w.now}); err == nil {
		t.Fatal("unprotected writer succeeded")
	}
	got, err := w.store.FindAsset(t.Context(), a.ID)
	if err != nil || !reflect.DeepEqual(got, a) {
		t.Fatalf("unprotected call changed row: %+v %v", got, err)
	}
}

func TestContentBindingsAtomic(t *testing.T) {
	for _, kind := range []domain.ScopeKind{domain.ScopeIssue, domain.ScopeComment, domain.ScopeNewComment} {
		for _, fail := range []bool{false, true} {
			t.Run(string(kind)+map[bool]string{false: "/commit", true: "/second UPDATE rolls back"}[fail], func(t *testing.T) {
				w := newAssetWorld(t)
				u, q := w.contentBinder(), w.contentRequest()
				purposes := []domain.Purpose{domain.IssueAttachment, domain.IssueDescription}
				if kind != domain.ScopeIssue {
					q.Host.CommentID = w.comment
					purposes = []domain.Purpose{domain.CommentAttachment, domain.CommentDescription}
				}
				if kind == domain.ScopeNewComment {
					q.NewComment, q.Host.CommentID = true, uuid.NewV7()
				}
				var before []domain.Asset
				for i, purpose := range purposes {
					a := w.preparedImage(t, domain.ScopeIssue, domain.IssueAttachment)
					a.Purpose, a.Placement.Scope.Kind = purpose, kind
					if kind == domain.ScopeComment {
						a.Placement.Scope.ID, a.Placement.CommentID = w.comment, w.comment
					}
					a = w.putImage(t, a)
					before = append(before, a)
					if i == 0 {
						q.AttachmentIDs = []uuid.UUID{a.ID}
					} else {
						q.ImageIDs = []uuid.UUID{a.ID, a.ID}
					}
				}
				slices.SortFunc(before, func(a, b domain.Asset) int { return a.ID.Compare(b.ID) })
				if fail {
					rejectContentUpdate(t, w, before[1].ID)
				}
				err := contentTx(t, w, false, func(ctx context.Context) error {
					at, err := u.LockContent(ctx, q)
					if err != nil {
						return err
					}
					tx, _ := postgres.Tx(ctx)
					if kind == domain.ScopeNewComment {
						if _, err := tx.Exec(ctx, "INSERT INTO issue_comments(id,workspace_id,project_id,issue_id,actor_id,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$6)", q.Host.CommentID, w.workspace, w.project, w.issue, w.user, at); err != nil {
							return err
						}
					}
					return u.BindContent(ctx, q, at)
				})
				requireContentResult(t, err, fail)
				for _, old := range before {
					got, err := w.store.FindAsset(t.Context(), old.ID)
					if err != nil {
						t.Fatal(err)
					}
					if fail {
						if !reflect.DeepEqual(got, old) {
							t.Fatal("failed transaction left partial binding")
						}
						continue
					}
					at := w.now.Add(time.Second)
					want := old
					want.BoundAt, want.UpdatedAt, want.UpdatedByID, want.StorageRevision = &at, at, &w.user, old.StorageRevision+1
					if q.NewComment {
						want.Placement.Scope, want.Placement.CommentID = domain.Scope{Kind: domain.ScopeComment, ID: q.Host.CommentID}, q.Host.CommentID
					}
					if !reflect.DeepEqual(got, want) {
						t.Fatalf("content binding=%+v want=%+v", got, want)
					}
				}
				if q.NewComment {
					var exists bool
					if err := w.pool.QueryRow(t.Context(), "SELECT EXISTS(SELECT 1 FROM issue_comments WHERE id=$1)", q.Host.CommentID).Scan(&exists); err != nil || exists == fail {
						t.Fatalf("host insertion not atomic: exists=%v error=%v", exists, err)
					}
				}
			})
		}
	}
}

type discardDraftEvents struct{}

func (discardDraftEvents) Publish(context.Context, shared.Event) error { return nil }
