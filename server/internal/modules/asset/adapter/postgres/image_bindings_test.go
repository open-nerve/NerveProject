package postgresadapter

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

func (w assetWorld) preparedImage(t *testing.T, kind domain.ScopeKind, purpose domain.Purpose) domain.Asset {
	t.Helper()
	a := w.asset(kind)
	a.Purpose = purpose
	a.Uploaded = true
	key, err := domain.NewObjectKey('o', a.ID, uuid.NewV7())
	if err != nil {
		t.Fatal(err)
	}
	a.Metadata.Final = &domain.ObjectRecord{Key: key, Phase: domain.ObjectActive, SafeAfter: w.now, Size: a.Size, ContentType: a.ContentType, ETag: "etag", SHA256: strings.Repeat("a", 64)}
	return a
}
func (w assetWorld) imageBinder() *app.ImageBinder {
	return app.NewImageBindings(app.ImageBindingDeps{Locks: w.store, Writer: w.store, Bindings: w.store, Clock: cleanupClock(w.now)})
}
func (w assetWorld) putImage(t *testing.T, a domain.Asset) domain.Asset {
	t.Helper()
	a, err := w.store.CreateAsset(t.Context(), a)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestImageBindingsRequireTransaction(t *testing.T) {
	w := newAssetWorld(t)
	a := w.putImage(t, w.preparedImage(t, domain.ScopeUser, domain.UserAvatar))
	req := app.ImageBindingRequest{Host: app.ImageBindingHost{ApplicantID: w.user}, Changes: []app.ImageChange{{Field: "avatar_asset_id", Purpose: domain.UserAvatar, NextID: &a.ID}}}
	u := w.imageBinder()
	if _, err := u.LockImageBindings(t.Context(), req); err == nil {
		t.Fatal("unprotected LockImageBindings succeeded")
	}
	if err := u.ApplyImageBindings(t.Context(), req, w.now); err == nil {
		t.Fatal("unprotected ApplyImageBindings succeeded")
	}
	if _, err := w.store.BindImage(t.Context(), app.ImageBindingWrite{AssetID: a.ID, By: w.user, At: w.now}); err == nil {
		t.Fatal("unprotected BindImage succeeded")
	}
	got, err := w.store.FindAsset(t.Context(), a.ID)
	if err != nil || !reflect.DeepEqual(got, a) {
		t.Fatalf("unprotected call changed row: %+v %v", got, err)
	}
}

func TestImageBindingsAtomicReplacement(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "commit", true: "second binding failure rolls back all"}[fail], func(t *testing.T) {
			w := newAssetWorld(t)
			u := w.imageBinder()
			var old, next []domain.Asset
			req := app.ImageBindingRequest{Host: app.ImageBindingHost{ApplicantID: w.user}}
			for i, purpose := range []domain.Purpose{domain.UserAvatar, domain.UserCover} {
				a := w.preparedImage(t, domain.ScopeUser, purpose)
				past := w.now.Add(-time.Hour)
				a.BoundAt = &past
				old = append(old, w.putImage(t, a))
				next = append(next, w.putImage(t, w.preparedImage(t, domain.ScopeUser, purpose)))
				field := []string{"avatar_asset_id", "cover_image_asset_id"}[i]
				req.Changes = append(req.Changes, app.ImageChange{Field: field, Purpose: purpose, CurrentID: &old[i].ID, NextID: &next[i].ID})
			}
			if _, err := w.pool.Exec(t.Context(), "UPDATE users SET avatar_asset_id=$1,cover_image_asset_id=$2 WHERE id=$3", old[0].ID, old[1].ID, w.user); err != nil {
				t.Fatal(err)
			}
			if fail {
				_, err := w.pool.Exec(t.Context(), `CREATE FUNCTION reject_second_image() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 IF NEW.id='`+next[1].ID.String()+`'::uuid AND NEW.bound_at IS NOT NULL THEN RAISE EXCEPTION 'image binding fault'; END IF; RETURN NEW; END $$;
 CREATE TRIGGER image_binding_fault BEFORE UPDATE ON file_assets FOR EACH ROW EXECUTE FUNCTION reject_second_image()`)
				if err != nil {
					t.Fatal(err)
				}
			}
			err := postgres.NewTxManager(w.pool, time.Second).WithinTx(t.Context(), func(ctx context.Context) error {
				tx, _ := postgres.Tx(ctx)
				if _, err := tx.Exec(ctx, "SELECT id FROM users WHERE id=$1 FOR NO KEY UPDATE", w.user); err != nil {
					return err
				}
				at, err := u.LockImageBindings(ctx, req)
				if err != nil {
					return err
				}
				if _, err := tx.Exec(ctx, "UPDATE users SET avatar_asset_id=$1,cover_image_asset_id=$2 WHERE id=$3", next[0].ID, next[1].ID, w.user); err != nil {
					return err
				}
				return u.ApplyImageBindings(ctx, req, at)
			})
			var pe *pgconn.PgError
			if fail {
				if !errors.As(err, &pe) || pe.Code != "P0001" {
					t.Fatalf("want actual trigger failure, got %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			var avatar, cover uuid.UUID
			if err := w.pool.QueryRow(t.Context(), "SELECT avatar_asset_id,cover_image_asset_id FROM users WHERE id=$1", w.user).Scan(&avatar, &cover); err != nil {
				t.Fatal(err)
			}
			for i := range old {
				gotOld, e1 := w.store.FindAsset(t.Context(), old[i].ID)
				gotNext, e2 := w.store.FindAsset(t.Context(), next[i].ID)
				if e1 != nil || e2 != nil {
					t.Fatalf("read: %v %v", e1, e2)
				}
				if fail {
					if !reflect.DeepEqual(gotOld, old[i]) || !reflect.DeepEqual(gotNext, next[i]) || avatar != old[0].ID || cover != old[1].ID {
						t.Fatal("failed transaction left partial owner/asset changes")
					}
				} else {
					if avatar != next[0].ID || cover != next[1].ID || gotOld.DeletedAt == nil || !gotOld.DeletedAt.Equal(w.now) || gotNext.BoundAt == nil || !gotNext.BoundAt.Equal(w.now) || gotNext.StorageRevision != next[i].StorageRevision+1 {
						t.Fatalf("binding/audit mismatch old=%+v new=%+v", gotOld, gotNext)
					}
					if gotOld.Placement != old[i].Placement || gotNext.Placement != next[i].Placement || !reflect.DeepEqual(gotOld.Metadata, old[i].Metadata) || !reflect.DeepEqual(gotNext.Metadata, next[i].Metadata) {
						t.Fatal("binding changed applicant, parent or recovery metadata")
					}
				}
			}
		})
	}
}
func TestImageBindingsProjectInsertThenBind(t *testing.T) {
	w := newAssetWorld(t)
	a := w.preparedImage(t, domain.ScopeWorkspace, domain.WorkspaceLogo)
	a.Purpose = domain.ProjectCover
	a.Placement.Scope.Kind = domain.ScopeNewProject
	a = w.putImage(t, a)
	projectID := uuid.NewV7()
	req := app.ImageBindingRequest{Host: app.ImageBindingHost{ApplicantID: w.user, WorkspaceID: w.workspace, ProjectID: projectID, NewProject: true},
		Changes: []app.ImageChange{{Field: "cover_image_asset_id", Purpose: domain.ProjectCover, NextID: &a.ID}}}
	err := postgres.NewTxManager(w.pool, time.Second).WithinTx(t.Context(), func(ctx context.Context) error {
		tx, _ := postgres.Tx(ctx)
		if _, err := tx.Exec(ctx, "SELECT id FROM users WHERE id=$1 FOR NO KEY UPDATE", w.user); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, "SELECT id FROM workspaces WHERE id=$1 FOR SHARE", w.workspace); err != nil {
			return err
		}
		at, err := w.imageBinder().LockImageBindings(ctx, req)
		if err != nil {
			return err
		}
		before, err := w.store.FindAsset(ctx, a.ID)
		if err != nil || before.Placement != a.Placement || before.BoundAt != nil {
			t.Fatalf("lock changed preparation: %+v %v", before, err)
		}
		if _, err := tx.Exec(ctx, "INSERT INTO projects(id,workspace_id,name,identifier,cover_image_asset_id,created_at,updated_at) VALUES($1,$2,'New','NEW',$3,$4,$4)", projectID, w.workspace, a.ID, at); err != nil {
			return err
		}
		return w.imageBinder().ApplyImageBindings(ctx, req, at)
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := w.store.FindAsset(t.Context(), a.ID)
	if err != nil || got.Placement.Scope != (domain.Scope{Kind: domain.ScopeProject, ID: projectID}) || got.Placement.ProjectID != projectID || got.Placement.UserID != w.user || got.Placement.WorkspaceID != w.workspace || got.BoundAt == nil {
		t.Fatalf("binding=%+v err=%v", got, err)
	}
	// SaveAsset must retain its old immutability boundary even after adding BindImage.
	got.Placement.ProjectID = w.project
	got.Placement.Scope.ID = w.project
	got.BoundAt = nil
	saved, err := w.store.SaveAsset(t.Context(), got)
	if err != nil || saved.Placement.ProjectID != projectID || saved.BoundAt == nil {
		t.Fatalf("SaveAsset widened ownership: %+v %v", saved, err)
	}
}
func TestImageBindingsWriterRejectsInvalidState(t *testing.T) {
	for _, name := range []string{"pending", "deleted", "bound", "wrong revision", "non-image", "missing final", "reclaiming"} {
		t.Run(name, func(t *testing.T) {
			w := newAssetWorld(t)
			a := w.preparedImage(t, domain.ScopeUser, domain.UserAvatar)
			switch name {
			case "pending":
				a.Uploaded = false
			case "deleted":
				a.DeletedAt = &w.now
				a.DeletedByID = &w.user
			case "bound":
				a.BoundAt = &w.now
			case "non-image":
				a = w.preparedImage(t, domain.ScopeIssue, domain.IssueAttachment)
			case "missing final":
				a.Metadata.Final = nil
			case "reclaiming":
				a.Metadata.Final.Phase = domain.ObjectReclaiming
			}
			a = w.putImage(t, a)
			revision := a.StorageRevision
			if name == "wrong revision" {
				revision++
			}
			err := postgres.NewTxManager(w.pool, time.Second).WithinTx(t.Context(), func(ctx context.Context) error {
				_, err := w.store.BindImage(ctx, app.ImageBindingWrite{AssetID: a.ID, Revision: revision, At: w.now, By: w.user})
				return err
			})
			if !errors.Is(err, domain.ErrAssetScopeChanged) {
				t.Fatalf("unexpected writer result: %v", err)
			}
			got, err := w.store.FindAsset(t.Context(), a.ID)
			if err != nil || !reflect.DeepEqual(got, a) {
				t.Fatalf("rejected binding changed row: %+v %v", got, err)
			}
		})
	}
}
